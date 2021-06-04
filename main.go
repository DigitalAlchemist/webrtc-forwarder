package main

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/pion/webrtc/v3"
)

func MustReadStdin() string {
	r := bufio.NewReader(os.Stdin)

	var in string
	for {
		var err error
		in, err = r.ReadString('\n')
		if err != io.EOF {
			if err != nil {
				panic(err)
			}
		}
		in = strings.TrimSpace(in)
		if len(in) > 0 {
			break
		}
	}

	fmt.Println("")

	return in
}

func SDPEncode(obj interface{}) string {
	b, err := json.Marshal(obj)
	if err != nil {
		panic(err)
	}

	return base64.StdEncoding.EncodeToString(b)
}

// Decode decodes the input from base64
// It can optionally unzip the input after decoding
func SDPDecode(in string, obj interface{}) {
	b, err := base64.StdEncoding.DecodeString(in)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(b, obj)
	if err != nil {
		panic(err)
	}
}

func main() {
	// Everything below is the Pion WebRTC API! Thanks for using it!
	isOffer := flag.String("offer", "yes", "should generate an offer")
	flag.Parse()

	// Prepare the configuration
	config := webrtc.Configuration{
/*		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
		*/
	}

	// Create a new RTCPeerConnection
	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		panic(err)
	}

	// Set the handler for ICE connection state
	// This will notify you when the peer has connected/disconnected
	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		fmt.Printf("ICE Connection State has changed: %s\n", connectionState.String())
	})

	// Register data channel creation handling
	peerConnection.OnDataChannel(func(d *webrtc.DataChannel) {
		fmt.Printf("New DataChannel %s %d\n", d.Label(), d.ID())

		// Register channel opening handling
		d.OnOpen(func() {
			fmt.Printf("Data channel '%s'-'%d' open.\n", d.Label(), d.ID())

			sendErr := d.Send([]byte("test"))
			if sendErr != nil {
				panic(sendErr)
			}
		})

		// Register text message handling
		d.OnMessage(func(msg webrtc.DataChannelMessage) {
			fmt.Printf("Message from DataChannel '%s': '%s'\n", d.Label(), string(msg.Data))
		})
	})
	/*
		// Attempt to receive X times before sending an offer.
		receive := func() error {
			//TODO: this
		}

		// TODO: Make configurable
		err := Retry(receive, WithMaxRetries(NewExponentialBackoff(), 5)
		if err != nil {
			// Oh shoot.
		}
	*/

	if *isOffer == "yes" {
		// Create a datachannel with label 'data'
		dataChannel, err := peerConnection.CreateDataChannel("data", nil)
		if err != nil {
			panic(err)
		}

		dataChannel.OnOpen(func() {
			fmt.Printf("Data channel '%s'-'%d' open. Random messages will now be sent to any connected DataChannels every 5 seconds\n", dataChannel.Label(), dataChannel.ID())

			for range time.NewTicker(5 * time.Second).C {
				sendErr := dataChannel.SendText("test")
				if sendErr != nil {
					panic(sendErr)
				}
			}
		})

		// Register text message handling
		dataChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
			fmt.Printf("Message from DataChannel '%s': '%s'\n", dataChannel.Label(), string(msg.Data))
		})

		offer, err := peerConnection.CreateOffer(nil)
		if err != nil {
			panic(err)
		}

		// Sets the LocalDescription, and starts our UDP listeners
		// Note: this will start the gathering of ICE candidates
		if err = peerConnection.SetLocalDescription(offer); err != nil {
			panic(err)
		}

		// Send our offer to the HTTP server listening in the other process
		payload := SDPEncode(offer)
		fmt.Println("Offer ----------")
		fmt.Println(payload)

		// Wait for the offer to be pasted
		answer := webrtc.SessionDescription{}
		SDPDecode(MustReadStdin(), &answer)

		err = peerConnection.SetRemoteDescription(answer)
		if err != nil {
			panic(err)
		}

		fmt.Println(SDPEncode(*peerConnection.RemoteDescription()))

	} else {
		// Wait for the offer to be pasted
		offer := webrtc.SessionDescription{}
		SDPDecode(MustReadStdin(), &offer)

		// Set the remote SessionDescription
		err = peerConnection.SetRemoteDescription(offer)
		if err != nil {
			panic(err)
		}

		// Create an answer
		answer, err := peerConnection.CreateAnswer(nil)
		if err != nil {
			panic(err)
		}

		// Create channel that is blocked until ICE Gathering is complete
		gatherComplete := webrtc.GatheringCompletePromise(peerConnection)

		// Sets the LocalDescription, and starts our UDP listeners
		err = peerConnection.SetLocalDescription(answer)
		if err != nil {
			panic(err)
		}

		// Block until ICE Gathering is complete, disabling trickle ICE
		// we do this because we only can exchange one signaling message
		// in a production application you should exchange ICE Candidates via OnICECandidate
		<-gatherComplete

		fmt.Println("Answer----------")
		fmt.Println(SDPEncode(*peerConnection.LocalDescription()))
		fmt.Println("------")
	}

	// Block forever
	select {}
}
