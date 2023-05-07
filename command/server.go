package command

import (
	"fmt"
	"net"

	"github.com/go-zoox/cli"
	"github.com/go-zoox/gzfly/core"
	"github.com/go-zoox/logger"
)

func RegisterServer(app *cli.MultipleProgram) {
	app.Register("server", &cli.Command{
		Name:  "server",
		Usage: "server for gzfly",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Usage:   "the filepath for server configuration",
				Aliases: []string{"c"},
				// Required: true,
			},
		},
		Action: func(ctx *cli.Context) error {
			cfg := &core.ServerConfig{}
			if err := cli.LoadConfig(ctx, cfg, &cli.LoadConfigOptions{
				Required: true,
			}); err != nil {
				return fmt.Errorf("failed to load config file: %v", err)
			}

			if cfg.Port == 0 {
				cfg.Port = 8080
			}

			if cfg.Path == "" {
				cfg.Path = "/"
			}

			cfg.OnConnect = func(conn net.Conn, source string, target string) {
				logger.Info("[%s] connect to %s", source, target)
			}

			server := core.NewServer(cfg)

			// // bind
			// go func() {
			// 	bindConfig := &core.BindConfig{
			// 		TargetUserClientID: "id_04aba01",
			// 		TargetUserPairKey:  "pair_3fd01",
			// 		Network:            "tcp",
			// 		LocalHost:          "127.0.0.1",
			// 		LocalPort:          8889,
			// 		RemoteHost:         "127.0.0.1",
			// 		RemotePort:         22,
			// 	}

			// 	if err := server.Bind(bindConfig); err != nil {
			// 		logger.Error(
			// 			"failed to bind with target(%s): %s://%s:%d:%s:%d",
			// 			bindConfig.TargetUserClientID,
			// 			bindConfig.Network,
			// 			bindConfig.LocalHost,
			// 			bindConfig.LocalPort,
			// 			bindConfig.RemoteHost,
			// 			bindConfig.RemotePort,
			// 		)
			// 	}
			// }()

			return server.Run()
		},
	})
}
