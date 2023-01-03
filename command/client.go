package command

import (
	"fmt"

	"github.com/go-zoox/cli"
	"github.com/go-zoox/gzfly/core"
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
			// //
			// &cli.StringFlag{
			// 	Name:  "role",
			// 	Usage: "role, available visitor(consumer) | agent(producer) | both",
			// 	Value: "both",
			// 	// Enums: []string{"visitor", "agent", "both"},
			// },
			//
			&cli.StringFlag{
				Name:  "agent-room-id",
				Usage: "the agent room id that client want to join, as a agent",
			},
			&cli.StringFlag{
				Name:  "agent-room-secret",
				Usage: "the agent room secret that client want to join, as a agent",
			},
			//
			&cli.StringFlag{
				Name:  "target-room-id",
				Usage: "the target room id that client want to join, as a visitor",
			},
			&cli.StringFlag{
				Name:  "target-room-secret",
				Usage: "the target room secret that client want to join, as a visitor",
			},
			//
			&cli.StringFlag{
				Name:    "target",
				Aliases: []string{"peer"},
				Usage:   "pair target",
			},
			&cli.StringFlag{
				Name:  "target-type",
				Usage: "target type, available: user | room, default: user",
				Value: "user",
			},
			&cli.StringFlag{
				Name:  "bind",
				Usage: "bind remote to local, example: tcp:127.0.0.1:8022:10.0.0.1:22",
			},
			&cli.StringFlag{
				Name:  "socks5",
				Usage: "create socks5 server, example: 127.0.0.1:17890",
			},
			&cli.StringFlag{
				Name:  "crypto",
				Usage: "data crypto algorithm, example: aes-128-cfb,aes-192-cfb,aes-256-cfb",
				// Value: ""
			},
		},
		Action: func(ctx *cli.Context) error {
			crypto := ctx.String("crypto")
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

			// role is agent or both
			var agent *core.Agent
			if ctx.String("agent-room-id") != "" {
				agent = &core.Agent{
					// Type: ,
					RoomID:     agent.RoomID,
					RoomSecret: agent.RoomSecret,
				}
			}

			client, err := core.NewClient(&core.ClientConfig{
				// OnConnect: func(conn net.Conn, source string, target string) {
				// 	logger.Info("[%s] connect to %s", source, target)
				// },
				Protocol: protocol,
				Host:     host,
				Port:     port,
				Path:     path,
				// USER
				User: auth,
				//
				Crypto: crypto,
				//
				AsAgent: agent,
			})
			if err != nil {
				return err
			}

			// role is visitor or both
			var target *core.Target
			if ctx.String("target") != "" {
				target, err = parseTarget(
					ctx.String("target"),
					ctx.String("target-room-id"),
					ctx.String("target-room-secret"),
				)
				if err != nil {
					return err
				}
			}

			var socks5 *core.Socks5
			var bind *core.Bind

			if ctx.String("socks5") != "" {
				socks5, err = parseSocks5(ctx.String("socks5"))
				if err != nil {
					return err
				}

				socks5.Target = target
			}

			if ctx.String("bind") != "" {
				bind, err = parseBind(ctx.String("bind"))
				if err != nil {
					return err
				}

				bind.Target = target
			}

			client.OnConnect(func() {
				// bind (port)
				if bind != nil {
					if err := client.BindServe(bind); err != nil {
						logger.Error(
							"failed to bind serve with target(%s): %s://%s:%d:%s:%d (error: %v)",
							bind.Target.UserClientID,
							bind.Network,
							bind.LocalHost,
							bind.LocalPort,
							bind.RemoteHost,
							bind.RemotePort,
							err,
						)
					}
				}

				// socks5
				if socks5 != nil {
					if err := client.Socks5Serve(socks5); err != nil {
						logger.Error(
							"failed to socks serve with target(%s): %s://%s:%d (error: %v)",
							socks5.Target.UserClientID,
							"socks5",
							socks5.IP,
							socks5.Port,
							err,
						)
					}
				}
			})

			return client.Listen()
		},
	})
}
