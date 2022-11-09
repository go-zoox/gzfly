package main

import (
	"fmt"

	tow "github.com/go-zoox/tcp-over-websocket"
	"github.com/go-zoox/tcp-over-websocket/user"
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
		// USER
		User: user.New("id_04aba6d", "29f4e3d3a4302b4d9e6a", "pair_3fd72"),
	}

	// if err := client.Connect(); err != nil {
	// 	logger.Fatal("failed to connect server: %s", err)
	// 	return
	// }

	// if err := client.Emit(protocol.COMMAND_AUTHENTICATE, []byte("zero")); err != nil {
	// 	logger.Errorf("failed to emit: %s", err)
	// }

	// client.WritePacket("bind", []byte{})
	// client.WritePacket(protocol.COMMAND_BIND, []byte{})

	if err := client.Listen(); err != nil {
		fmt.Println("listen error:", err)
	}
}
