package main

import (
	"github.com/go-zoox/logger"
	tow "github.com/go-zoox/tcp-over-websocket"
)

func main() {
	c := &tow.Client{
		// OnConnect: func(conn net.Conn, source string, target string) {
		// 	logger.Info("[%s] connect to %s", source, target)
		// },
		Host: "127.0.0.1",
		Port: 1080,
		Path: "/tow",
	}

	if err := c.Connect(); err != nil {
		logger.Fatal("failed to connect server: %s", err)
		return
	}

	// if err := c.WriteMessage(tow.MessageTypeText, []byte("hi")); err != nil {
	// 	logger.Errorf("failed to write message: %s", err)
	// }

	// if err := c.WriteTextMessage([]byte("text message")); err != nil {
	// 	logger.Errorf("failed to write message: %s", err)
	// }

	// if err := c.WriteBinaryMessage([]byte("binary message")); err != nil {
	// 	logger.Errorf("failed to write message: %s", err)
	// }

	if err := c.Emit("authentication", []byte("zero")); err != nil {
		logger.Errorf("failed to emit: %s", err)
	}
}
