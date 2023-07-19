package command

import (
	"fmt"
	"net"
	"time"

	"github.com/go-zoox/cli"
	"github.com/go-zoox/gzfly/core"
	"github.com/go-zoox/gzfly/user"
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
				FilePath: ctx.String("config"),
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

			// admin
			if cfg.Admin.ClientID != "" {
				go func() {
					connectAdminClient(cfg.Port, cfg.Path, &cfg.Admin)
				}()
			}

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

func connectAdminClient(port int64, path string, adminUser *user.UserClient) error {
	logger.Infof("try to connect admin client ...")

	time.Sleep(1 * time.Second)

	client, err := core.NewClient(&core.ClientConfig{
		Protocol: "ws",
		Host:     "127.0.0.1",
		Port:     int(port),
		Path:     path,
		User: &user.User{
			ClientID:     adminUser.ClientID,
			ClientSecret: adminUser.ClientSecret,
			PairKey:      adminUser.PairKey,
		},
	})
	if err != nil {
		panic(fmt.Errorf("failed to connect admin client: %v", err))
	}

	if err := client.Listen(); err != nil {
		logger.Errorf("failed to listen admin client: %v", err)
		return connectAdminClient(port, path, adminUser)
	}

	logger.Errorf("admin client closed, no error")
	return connectAdminClient(port, path, adminUser)
}
