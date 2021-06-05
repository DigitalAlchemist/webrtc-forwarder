package main

import (
	"fmt"
	"io"
	"net"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/urfave/cli/v2"
	"github.com/pion/webrtc/v3"
)

type UDPConnHook struct {
	*net.UDPConn
	addr *net.UDPAddr
}

func (u *UDPConnHook) Read(b []byte) (int, error){
	n, addr, err := u.ReadFromUDP(b)
	u.addr = addr
	return n, err
}

func (u *UDPConnHook) Write(b []byte) (int, error){
	return u.WriteToUDP(b, u.addr)
}

func appRun(ctx *cli.Context) error {

	var err error
	var conn io.ReadWriter
	var negotiator SessionNegotiator

	switch ctx.String("negotiator") {
	case "stdin":
		negotiator = &StdinNegotiator{}
	case "web":
		negotiator = &WebNegotiator{
			baseURL: ctx.String("web-baseurl"),
		}
	case "default":
		return fmt.Errorf("Invalid negotiator selected")

	}

	addr, _ := net.ResolveUDPAddr("udp", ctx.String("socket-addr"))
	if ctx.Bool("dial") {
		conn, err = net.DialUDP("udp", nil, addr)
	} else {
		var udp *net.UDPConn
		udp, err = net.ListenUDP("udp", addr)
		conn = &UDPConnHook{udp, nil}
	}

	if err != nil {
		return err
	}


	for {
		err = handleConnection(conn,
			negotiator,
			ctx.Bool("negotiator-skip-receive"),
			ctx.String("peer-id"))
		if err != nil{
			fmt.Println(err)
		}
	}
	return nil
}

func handleConnection(conn io.ReadWriter,
	negotiator SessionNegotiator,
	skipReceive bool, peerId string) error {

	s := webrtc.SettingEngine{}
	s.DetachDataChannels()

	// Create an API object with the engine
	api := webrtc.NewAPI(webrtc.WithSettingEngine(s))

	// Prepare the configuration
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}

	// Create a new RTCPeerConnection
	peerConnection, err := api.NewPeerConnection(config)
	if err != nil {
		return err
	}
	defer peerConnection.Close()
	closer := make(chan struct {})

	// Set the handler for ICE connection state
	// This will notify you when the peer has connected/disconnected
	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		fmt.Printf("ICE Connection State has changed: %s\n", connectionState.String())
		if connectionState == webrtc.ICEConnectionStateDisconnected ||
			connectionState == webrtc.ICEConnectionStateFailed {
				close(closer)
		}
	})

	const (
		WAIT_FOR_OFFER = iota
		OFFER_NOT_FOUND_CREATE_OFFER
		OFFER_SENT_WAIT_ANSWER

		ANSWER_RECEIVED_ESTABLISHING
		ANSWER_SENT_ESTABLISHING

		CONNECTION_ESTABLISHED
	)
	stateNames := []string{
		"WAIT_FOR_OFFER",
		"OFFER_NOT_FOUND_CREATE_OFFER",
		"OFFER_SENT_WAIT_ANSWER",
		"ANSWER_RECEIVED_ESTABLISHING",
		"ANSWER_SENT_ESTABLISHING",
		"CONNECTION_ESTABLISHED",
	}

	state := WAIT_FOR_OFFER
	if skipReceive {
		fmt.Println("Skipping initial receive.")
		state = OFFER_NOT_FOUND_CREATE_OFFER
	}
	for {
		fmt.Printf("Current State: %s\n", stateNames[state])
		switch state {
		case WAIT_FOR_OFFER:
			// Exponential backoff to get the offer, largely for race conditions.
			var offer *webrtc.SessionDescription
			err = backoff.Retry(func() error {
				var err error
				offer, err = negotiator.RecvOffer(peerId)
				return err
			}, backoff.WithMaxRetries(backoff.NewExponentialBackOff(), 5))
			if err != nil {
				state = OFFER_NOT_FOUND_CREATE_OFFER
				continue
			}

			// Received an offer!
			err = peerConnection.SetRemoteDescription(*offer)
			if err != nil {
				return err
			}

			// Register data channel creation handling
			peerConnection.OnDataChannel(func(dataChannel *webrtc.DataChannel) {
				dataChannel.OnOpen(func() {
					ch, err := dataChannel.Detach()
					if err != nil {
						panic(err)
					}

					go io.Copy(ch, conn)
					io.Copy(conn, ch)
				})
			})

			// Create an answer
			answer, err := peerConnection.CreateAnswer(nil)
			if err != nil {
				return err
			}

			// Create channel that is blocked until ICE Gathering is complete
			gatherComplete := webrtc.GatheringCompletePromise(peerConnection)

			// Sets the LocalDescription, and starts our UDP listeners
			err = peerConnection.SetLocalDescription(answer)
			if err != nil {
				return err
			}

			// Block until ICE Gathering is complete, disabling trickle ICE
			// we do this because we only can exchange one signaling message
			// in a production application you should exchange ICE Candidates via OnICECandidate
			<-gatherComplete

			localDesc := peerConnection.LocalDescription()

			err = negotiator.SendAnswer(peerId, localDesc)
			if err != nil {
				return err
			}

			state = ANSWER_SENT_ESTABLISHING
		case OFFER_NOT_FOUND_CREATE_OFFER:
			dataChannel, err := peerConnection.CreateDataChannel("data", nil)
			if err != nil {
				return err
			}

			dataChannel.OnOpen(func() {
				ch, err := dataChannel.Detach()
				if err != nil {
					//TODO: Recover more correctly
					panic(err)
				}

				go io.Copy(ch, conn)
				io.Copy(conn, ch)
			})

			offer, err := peerConnection.CreateOffer(nil)
			if err != nil {
				return err
			}

			// Sets the LocalDescription, and starts our UDP listeners
			// Note: this will start the gathering of ICE candidates
			if err = peerConnection.SetLocalDescription(offer); err != nil {
				return err
			}

			err = negotiator.SendOffer(peerId, &offer)
			if err != nil {
				dataChannel.Close()
				return err
			}
			state = OFFER_SENT_WAIT_ANSWER
		case OFFER_SENT_WAIT_ANSWER:
			var answer *webrtc.SessionDescription
			for {
				answer, err = negotiator.RecvAnswer(peerId)
				// TODO: Backoff?
				// XXX: Inverted error check
				if err == nil {
					break
				}
			}

			err = peerConnection.SetRemoteDescription(*answer)
			if err != nil {
				return err
			}

			state = ANSWER_RECEIVED_ESTABLISHING
		case ANSWER_SENT_ESTABLISHING, ANSWER_RECEIVED_ESTABLISHING:
			for {
				if peerConnection.ConnectionState() == webrtc.PeerConnectionStateConnected {
					state = CONNECTION_ESTABLISHED
					break
				}
				time.Sleep(5 * time.Second)
			}

		case CONNECTION_ESTABLISHED:
			select {
			case <-closer:
				return nil
			}
		default:
			panic(fmt.Errorf("Invalid State"))
		}
	}
	return nil
}
