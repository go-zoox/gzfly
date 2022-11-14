package main

import (
	"fmt"

	tow "github.com/go-zoox/fly"
	"github.com/go-zoox/fly/user"
	"github.com/go-zoox/logger"
)

func main() {
	client := tow.NewClient(&tow.ClientConfig{
		// OnConnect: func(conn net.Conn, source string, target string) {
		// 	logger.Info("[%s] connect to %s", source, target)
		// },
		Protocol: "ws",
		Host:     "127.0.0.1",
		Port:     1080,
		Path:     "/",
		// USER
		User: user.New("id_04aba02", "29f4e3d3a4302b4d9e02", "pair_3fd02"),
	})

	// if err := client.Connect(); err != nil {
	// 	logger.Fatal("failed to connect server: %s", err)
	// 	return
	// }

	// if err := client.Emit(protocol.COMMAND_AUTHENTICATE, []byte("zero")); err != nil {
	// 	logger.Errorf("failed to emit: %s", err)
	// }

	// client.WritePacket("bind", []byte{})
	// client.WritePacket(protocol.COMMAND_BIND, []byte{})

	client.OnConnect(func() {
		bindConfig := &tow.BindConfig{
			TargetUserClientID: "id_04aba01",
			TargetUserPairKey:  "pair_3fd01",
			Network:            "tcp",
			LocalHost:          "127.0.0.1",
			LocalPort:          8888,
			RemoteHost:         "127.0.0.1",
			RemotePort:         22,
		}

		if err := client.Bind(bindConfig); err != nil {
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
	})

	if err := client.Listen(); err != nil {
		fmt.Println("listen error:", err)
	}
}
