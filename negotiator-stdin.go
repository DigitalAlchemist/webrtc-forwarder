package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

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

type StdinNegotiator struct {}

func (w *StdinNegotiator) RecvOffer(id string) (*webrtc.SessionDescription, error) {
	offer := webrtc.SessionDescription{}
	fmt.Println("Paste Offer Below ----------")
	SDPDecode(MustReadStdin(), &offer)
	return &offer, nil
}

func (w *StdinNegotiator) SendOffer(id string, sdp *webrtc.SessionDescription) error {
	payload := SDPEncode(sdp)
	fmt.Println("Offer----------")
	fmt.Println(payload)
	return nil
}

func (w *StdinNegotiator) RecvAnswer(id string) (*webrtc.SessionDescription, error) {
	answer := webrtc.SessionDescription{}
	fmt.Println("Paste Answer Below ----------")
	SDPDecode(MustReadStdin(), &answer)
	return &answer, nil
}

func (w *StdinNegotiator) SendAnswer(id string, sdp *webrtc.SessionDescription) error {
	payload := SDPEncode(sdp)
	fmt.Println("Answer----------")
	fmt.Println(payload)
	return nil
}

func (w *StdinNegotiator) Complete(id string) error {
	return nil
}
