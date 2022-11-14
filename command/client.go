package command

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-zoox/cli"
	"github.com/go-zoox/fly/core"
	"github.com/go-zoox/fly/user"
	"github.com/go-zoox/logger"
)

func RegisterClient(app *cli.MultipleProgram) {
	app.Register("client", &cli.Command{
		Name:  "client",
		Usage: "client for fly",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "auth",
				Usage:    "auth info, format: client_id:client_secret:client_pairkey",
				Required: true,
			},
			&cli.StringFlag{
				Name:  "relay",
				Usage: "relay server, format: protocol://host:port",
				Value: "wss://fly.zcorky.com",
			},
		},
		Action: func(ctx *cli.Context) error {
			relayR := ctx.String("relay")
			relay, err := url.Parse(relayR)
			if err != nil {
				return fmt.Errorf("invalid relay: %v", err)
			}

			port := 443
			if relay.Port() != "" {
				port, err = strconv.Atoi(relay.Port())
				if err != nil {
					return fmt.Errorf("invalid relay port: %v", err)
				}
			}

			authR := ctx.String("auth")
			authS := strings.Split(authR, ":")
			if len(authS) != 3 {
				return fmt.Errorf("invalid auth: %v", err)
			}
			clientID, clientSecret, clientPairKey := authS[0], authS[1], authS[2]

			logger.Info("relay: %s://%s:%d%s", relay.Scheme, relay.Hostname(), port, relay.Path)
			logger.Info("auth: %s", authR)

			client := core.NewClient(&core.ClientConfig{
				// OnConnect: func(conn net.Conn, source string, target string) {
				// 	logger.Info("[%s] connect to %s", source, target)
				// },
				Protocol: relay.Scheme,
				Host:     relay.Hostname(),
				Port:     port,
				Path:     relay.Path,
				// USER
				User: user.New(clientID, clientSecret, clientPairKey),
			})

			return client.Listen()
		},
	})
}
