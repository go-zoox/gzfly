package command

import (
	"github.com/go-zoox/core-utils/fmt"
	"github.com/go-zoox/core-utils/object"
	"github.com/go-zoox/core-utils/strings"

	"github.com/go-zoox/cli"
	"github.com/go-zoox/gzfly/core"
	"github.com/go-zoox/logger"
)

// @TODO
type ClientCLIConfig struct {
	Relay  string `config:"relay"`
	Auth   string `config:"auth"`
	Crypto string `config:"crypto"`
	//
	Actions map[string]Action `config:"actions"`
}

type Action struct {
	Target string `config:"target"`
	Bind   string `config:"bind"`
	Socks5 string `config:"socks5"`
}

func RegisterClient(app *cli.MultipleProgram) {
	app.Register("client", &cli.Command{
		Name:  "client",
		Usage: "client for gzfly",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "auth",
				Usage: "auth info, format: client_id:client_secret:client_pairkey",
				// Required: true,
			},
			&cli.StringFlag{
				Name:  "relay",
				Usage: "relay server, format: protocol://host:port",
			},
			&cli.StringFlag{
				Name:    "target",
				Aliases: []string{"peer"},
				Usage:   "pair target",
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
			&cli.StringFlag{
				Name:  "action",
				Usage: "use user custom action for target and bind",
			},
			&cli.StringFlag{
				Name:    "config",
				Usage:   "the filepath for client configuration",
				Aliases: []string{"c"},
			},
		},
		Action: func(ctx *cli.Context) error {
			cliCfg := &ClientCLIConfig{}
			if err := cli.LoadConfig(ctx, cliCfg); err != nil {
				return fmt.Errorf("failed to load config: %v", err)
			}

			if ctx.String("relay") != "" {
				cliCfg.Relay = ctx.String("relay")
			}
			if ctx.String("auth") != "" {
				cliCfg.Auth = ctx.String("auth")
			}
			if ctx.String("crypto") != "" {
				cliCfg.Crypto = ctx.String("crypto")
			}
			if cliCfg.Relay == "" {
				cliCfg.Relay = "wss://gzfly.zcorky.com"
			}

			var targetX string
			var bindX string
			var socks5X string
			if ctx.String("action") != "" && cliCfg.Actions != nil {
				action, ok := cliCfg.Actions[ctx.String("action")]
				if !ok {
					return fmt.Errorf("invalid action: %s, available: %s", ctx.String("action"), strings.Join(object.Keys(cliCfg.Actions), ","))
				}
				if action.Target != "" {
					targetX = action.Target
				}
				if action.Bind != "" {
					bindX = action.Bind
				}
				if action.Socks5 != "" {
					socks5X = action.Socks5
				}
			}
			if ctx.String("target") != "" {
				targetX = ctx.String("target")
			}
			if ctx.String("bind") != "" {
				bindX = ctx.String("bind")
			}
			if ctx.String("socks5") != "" {
				socks5X = ctx.String("socks5")
			}

			fmt.PrintJSON(map[string]any{
				"cliCfg": cliCfg,
				"custom": map[string]any{
					"targetX": targetX,
					"bindX":   bindX,
					"socks5X": socks5X,
				},
			})

			protocol, host, port, path, err := parseRelay(cliCfg.Relay)
			if err != nil {
				return fmt.Errorf("invalid relay: %v", err)
			}

			auth, err := parseAuth(cliCfg.Auth)
			if err != nil {
				return err
			}

			logger.Info("relay: %s", cliCfg.Relay)
			logger.Info("auth: %s", cliCfg.Auth)

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
				Crypto: cliCfg.Crypto,
			})
			if err != nil {
				return err
			}

			var target *core.Target
			if targetX != "" {
				target, err = parseTarget(targetX)
				if err != nil {
					return err
				}
			}

			var socks5 *core.Socks5
			var bind *core.Bind

			if socks5X != "" {
				socks5, err = parseSocks5(socks5X)
				if err != nil {
					return err
				}

				socks5.Target = target
			}

			if bindX != "" {
				bind, err = parseBind(bindX)
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
