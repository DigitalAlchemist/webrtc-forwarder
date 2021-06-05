package main

import (
	"log"
	"os"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:   "webrtc-tunneler",
		Action: appRun,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "socket-addr",
				Value:   "127.0.0.1:5000",
				Usage:   "Local socket address to forward to the remote host",
				EnvVars: []string{"SOCKET_ADDR"},
			},
			&cli.StringFlag{
				Name:    "peer-id",
				Value:   "-",
				Usage:   "Identity of remote peer (only one peer supported)",
				EnvVars: []string{"PEER_ID"},
			},
			&cli.BoolFlag{
				Name:    "dial",
				Value:   false,
				Usage:   "Dial a local service on behalf of peer",
				EnvVars: []string{"SHOULD_DIAL"},
			},
			&cli.BoolFlag{
				Name:    "negotiator-skip-receive",
				Value:   false,
				Usage:   "Skip initial receive loop.",
				EnvVars: []string{"NEGOTIATOR_SKIP_RECEIVE"},
			},
			&cli.StringFlag{
				Name:    "negotiator",
				Value:   "stdin",
				Usage:   "Select negotiator to use. Options are stdin,web",
				EnvVars: []string{"NEGOTIATOR"},
			},
			&cli.StringFlag{
				Name:    "web-baseurl",
				Value:   "http://127.0.0.1/negotiate",
				Usage:   "Base URL for web negotiator",
				EnvVars: []string{"WEB_BASEURL"},
			},


		},
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
