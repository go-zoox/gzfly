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

	// logger.Infof("start socks5 server at: %s ...", "0.0.0.0:1080")
	if err := s.Run(":1080"); err != nil {
		logger.Fatal("failed to start socks5 server: %s", err)
		return
	}
}
