package main

import (
	"fmt"

	"github.com/go-zoox/gzfly/core"
	tow "github.com/go-zoox/gzfly/core"
	"github.com/go-zoox/gzfly/user"
	"github.com/go-zoox/logger"
)

func main() {
	client, _ := tow.NewClient(&tow.ClientConfig{
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
		target := &core.Target{
			UserClientID: "id_04aba01",
			UserPairKey:  "pair_3fd01",
		}
		bindConfig := &tow.Bind{
			Target:     target,
			Network:    "tcp",
			LocalHost:  "127.0.0.1",
			LocalPort:  8888,
			RemoteHost: "127.0.0.1",
			RemotePort: 22,
		}

		if err := client.BindServe(bindConfig); err != nil {
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
	})

	if err := client.Listen(); err != nil {
		fmt.Println("listen error:", err)
	}
}
