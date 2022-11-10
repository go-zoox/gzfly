package main

import (
	"net"

	"github.com/go-zoox/logger"
	tow "github.com/go-zoox/tcp-over-websocket"
)

func main() {
	s := tow.NewServer(&tow.ServerConfig{
		Path: "/",
		OnConnect: func(conn net.Conn, source string, target string) {
			logger.Info("[%s] connect to %s", source, target)
		},
	})

	// bind
	go func() {
		bindConfig := &tow.BindConfig{
			TargetUserClientID: "id_04aba01",
			TargetUserPairKey:  "pair_3fd01",
			Network:            "tcp",
			LocalHost:          "127.0.0.1",
			LocalPort:          8889,
			RemoteHost:         "127.0.0.1",
			RemotePort:         22,
		}

		if err := s.Bind(bindConfig); err != nil {
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
	}()

	// logger.Infof("start socks5 server at: %s ...", "0.0.0.0:1080")
	if err := s.Run(":1080"); err != nil {
		logger.Fatal("failed to start socks5 server: %s", err)
		return
	}
}
