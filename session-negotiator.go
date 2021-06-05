package main

import (
	"encoding/base64"
	"encoding/json"

	"github.com/pion/webrtc/v3"
)

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

type SessionNegotiator interface {
	RecvOffer(id string) (*webrtc.SessionDescription, error)
	SendOffer(id string, sdp *webrtc.SessionDescription) error

	RecvAnswer(id string) (*webrtc.SessionDescription, error)
	SendAnswer(id string, sdp *webrtc.SessionDescription) error

	Complete(id string) error
}
