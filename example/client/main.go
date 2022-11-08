package main

import (
	"fmt"

	"github.com/go-zoox/logger"
	tow "github.com/go-zoox/tcp-over-websocket"
	"github.com/go-zoox/tcp-over-websocket/protocol"
)

func main() {
	client := &tow.Client{
		// OnConnect: func(conn net.Conn, source string, target string) {
		// 	logger.Info("[%s] connect to %s", source, target)
		// },
		Protocol: "ws",
		Host:     "127.0.0.1",
		Port:     1080,
		Path:     "/",
	}

	if err := client.Connect(); err != nil {
		logger.Fatal("failed to connect server: %s", err)
		return
	}

	if err := client.Emit(protocol.COMMAND_AUTHENTICATE, []byte("zero")); err != nil {
		logger.Errorf("failed to emit: %s", err)
	}

	// client.WritePacket("bind", []byte{})
	client.WritePacket(protocol.COMMAND_BIND, []byte{})

	if err := client.Listen(); err != nil {
		fmt.Println("listen error:", err)
	}
}
