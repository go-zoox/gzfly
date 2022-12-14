package main

import (
	"net"

	tow "github.com/go-zoox/gzfly/core"
	"github.com/go-zoox/logger"
)

func main() {
	s := tow.NewServer(&tow.ServerConfig{
		Port: 1080,
		Path: "/",
		OnConnect: func(conn net.Conn, source string, target string) {
			logger.Info("[%s] connect to %s", source, target)
		},
	})

	// bind
	go func() {
		target := &tow.Target{
			UserClientID: "id_04aba01",
			UserPairKey:  "pair_3fd01",
		}
		bindConfig := &tow.Bind{
			Target:     target,
			Network:    "tcp",
			LocalHost:  "127.0.0.1",
			LocalPort:  8889,
			RemoteHost: "127.0.0.1",
			RemotePort: 22,
		}

		if err := s.Bind(bindConfig); err != nil {
			logger.Error(
				"failed to bind with target(%s): %s://%s:%d:%s:%d (error: %v)",
				bindConfig.Target.UserClientID,
				bindConfig.Network,
				bindConfig.LocalHost,
				bindConfig.LocalPort,
				bindConfig.RemoteHost,
				bindConfig.RemotePort,
				err,
			)
		}
	}()

	// logger.Infof("start socks5 server at: %s ...", "0.0.0.0:1080")
	if err := s.Run(); err != nil {
		logger.Fatal("failed to start socks5 server: %s", err)
		return
	}
}
