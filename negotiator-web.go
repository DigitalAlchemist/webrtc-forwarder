package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/pion/webrtc/v3"
)

type WebNegotiator struct {
	baseURL string
}

func (w *WebNegotiator) recv(id, endpoint string) (*webrtc.SessionDescription, error) {
	sdp := webrtc.SessionDescription{}

	resp, err := http.Get(fmt.Sprintf("%s/%s/%s", w.baseURL, id, endpoint))
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	SDPDecode(string(body), &sdp)

	return &sdp, nil

}

func (w *WebNegotiator) send(sdp *webrtc.SessionDescription, id, endpoint string) error {
	resp, err := http.Post(
		fmt.Sprintf("%s/%s/%s", w.baseURL, id, endpoint),
		"application/json",
		bytes.NewBuffer([]byte(SDPEncode(sdp))),
	)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Received invalid status %d", resp.StatusCode)
	}

	return nil

}

func (w *WebNegotiator) RecvOffer(id string) (*webrtc.SessionDescription, error) {
	return w.recv(id, "offer")
}

func (w *WebNegotiator) SendOffer(id string, sdp *webrtc.SessionDescription) error {
	return w.send(sdp, id, "offer")
}

func (w *WebNegotiator) RecvAnswer(id string) (*webrtc.SessionDescription, error) {
	return w.recv(id, "answer")
}

func (w *WebNegotiator) SendAnswer(id string, sdp *webrtc.SessionDescription) error {
	return w.send(sdp, id, "answer")
}

func (w *WebNegotiator) Complete(id string) error {
	// TODO: Remove entries.
	return nil
}
