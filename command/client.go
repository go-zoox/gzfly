package command

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-zoox/cli"
	"github.com/go-zoox/gzfly/core"
	"github.com/go-zoox/gzfly/user"
	"github.com/go-zoox/logger"
)

func RegisterClient(app *cli.MultipleProgram) {
	app.Register("client", &cli.Command{
		Name:  "client",
		Usage: "client for gzfly",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "auth",
				Usage:    "auth info, format: client_id:client_secret:client_pairkey",
				Required: true,
			},
			&cli.StringFlag{
				Name:  "relay",
				Usage: "relay server, format: protocol://host:port",
				Value: "wss://gzfly.zcorky.com",
			},
			&cli.StringFlag{
				Name:  "bind",
				Usage: "bind remote to local, example: tcp:127.0.0.1:8022:10.0.0.1:22:client_id:pair_key",
				// Value: ""
			},
		},
		Action: func(ctx *cli.Context) error {
			protocol, host, port, path, err := parseRelay(ctx.String("relay"))
			if err != nil {
				return fmt.Errorf("invalid relay: %v", err)
			}

			auth, err := parseAuth(ctx.String("auth"))
			if err != nil {
				return err
			}

			logger.Info("relay: %s", ctx.String("relay"))
			logger.Info("auth: %s", ctx.String("auth"))

			client := core.NewClient(&core.ClientConfig{
				// OnConnect: func(conn net.Conn, source string, target string) {
				// 	logger.Info("[%s] connect to %s", source, target)
				// },
				Protocol: protocol,
				Host:     host,
				Port:     port,
				Path:     path,
				// USER
				User: auth,
			})

			if ctx.String("bind") != "" {
				bindConfig, err := parseBind(ctx.String("bind"))
				if err != nil {
					return err
				}

				client.OnConnect(func() {
					if err := client.Bind(bindConfig); err != nil {
						logger.Error(
							"failed to bind with target(%s): %s://%s:%d:%s:%d",
							bindConfig.TargetUserClientID,
							bindConfig.Network,
							bindConfig.LocalHost,
							bindConfig.LocalPort,
							bindConfig.RemoteHost,
							bindConfig.RemotePort,
						)
					}
				})

			}

			return client.Listen()
		},
	})
}

func parseAuth(auth string) (*user.User, error) {
	parts := strings.Split(auth, ":")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid auth")
	}

	clientID, clientSecret, pairKey := parts[0], parts[1], parts[2]

	return &user.User{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		PairKey:      pairKey,
	}, nil
}

func parseRelay(relayR string) (protocol string, host string, port int, path string, err error) {
	relay, err := url.Parse(relayR)
	if err != nil {
		err = fmt.Errorf("invalid relay: %v", err)
		return
	}

	port = 443
	if relay.Port() != "" {
		port, err = strconv.Atoi(relay.Port())
		if err != nil {
			err = fmt.Errorf("invalid relay port: %v", err)
			return
		}
	}

	protocol = relay.Scheme
	host = relay.Hostname()
	path = relay.Path

	return
}

func parseBind(bind string) (*core.BindConfig, error) {
	parts := strings.Split(bind, ":")
	if len(parts) != 7 {
		return nil, fmt.Errorf("invalid bind")
	}

	LocalPort, err := strconv.Atoi(parts[2])
	if err != nil {
		return nil, fmt.Errorf("failed to parse local port: %v", err)
	}

	RemotePort, err := strconv.Atoi(parts[4])
	if err != nil {
		return nil, fmt.Errorf("failed to parse remote port: %v", err)
	}

	return &core.BindConfig{
		Network:            parts[0],
		LocalHost:          parts[1],
		LocalPort:          LocalPort,
		RemoteHost:         parts[3],
		RemotePort:         RemotePort,
		TargetUserClientID: parts[5],
		TargetUserPairKey:  parts[6],
	}, nil
}
