package tow

import (
	"fmt"
	"io"
	"net"

	"github.com/go-zoox/tcp-over-websocket/connection"
	"github.com/go-zoox/tcp-over-websocket/manager"
	"github.com/go-zoox/tcp-over-websocket/protocol"
	"github.com/go-zoox/uuid"
	"github.com/go-zoox/zoox"
	zd "github.com/go-zoox/zoox/default"
)

type Server struct {
	Path      string
	OnConnect func(conn net.Conn, source, target string)
}

func (s *Server) Run(addr string) error {
	core := zd.Default()

	wsConnsManager := manager.New[*connection.WSConn]()

	core.WebSocket(s.Path, func(ctx *zoox.Context, client *zoox.WebSocketClient) {
		client.OnError = func(err error) {
			if e, ok := err.(*zoox.WebSocketCloseError); ok {
				ctx.Logger.Error("[error][client: %s][code: %d] %v", client.ID, e.Code, e)
			} else {
				ctx.Logger.Error("[error][client: %s][code: nocode] %v", client.ID, err)
			}
		}

		client.OnConnect = func() {
			ctx.Logger.Info("[connect] client: %s", client.ID)
		}

		client.OnDisconnect = func() {
			ctx.Logger.Info("[disconnect] client: %s", client.ID)
		}

		client.OnBinaryMessage = func(raw []byte) {
			packet := protocol.New() // &Protocol{}

			if err := packet.Decode(raw); err != nil {
				ctx.Logger.Error("invalid message: %v", err)
				return
			}

			ctx.Logger.Info("received [type: %d] %d", packet.GetCommand(), len(packet.GetData()))

			switch packet.GetCommand() {
			case protocol.COMMAND_BIND:
				go func() {
					if err := CreateTCPServer(&CreateTCPServerConfig{
						Port: 8888,
						OnConn: func(id string) net.Conn {
							// conn := &WSConn{
							// 	ID:     id,
							// 	Client: client,
							// 	Stream: make(chan []byte),
							// }

							conn := connection.New(id, client)

							wsConnsManager.Set(id, conn)

							return conn
						},
					}); err != nil {

					}
				}()
			case protocol.COMMAND_CONNECT:
				data := packet.GetData()
				id, err := connection.DecodeID(data)
				if err != nil {
					fmt.Print("[connect] failed to parse id:", err)
					return
				}

				wsconn, err := wsConnsManager.Get(id)
				if err != nil {
					fmt.Println("[connect] failed to get conn:", err)
					return
				}

				wsconn.Stream <- data
			}
		}
	})

	return core.Run(addr)
}

type CreateTCPServerConfig struct {
	Port   int
	OnConn func(id string) net.Conn
}

func CreateTCPServer(cfg *CreateTCPServerConfig) error {
	addr := fmt.Sprintf(":%d", cfg.Port)
	fmt.Println("listen tcp server at: ", addr)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer listener.Close()
	for {
		source, err := listener.Accept()
		if err != nil {
			continue
		}

		fmt.Println("tcp connected")

		id := uuid.V4()
		target := cfg.OnConn(id)
		go Copy(source, target)
		go Copy(target, source)
	}
}

func Copy(dst io.Writer, src io.Reader) (written int64, err error) {
	return io.Copy(dst, src)

	// buf := make([]byte, 256)
	// return io.CopyBuffer(dst, src, buf)
}

// func (s *Server) process(client net.Conn) {
// 	// 1. 认证
// 	if err := s.authenticate(client); err != nil {
// 		logger.Errorf("auth error: %v", err)
// 		client.Close()
// 		return
// 	}

// 	// 2. 建立连接
// 	target, err := s.connect(client)
// 	if err != nil {
// 		logger.Errorf("connect error: %v", err)
// 		client.Close()
// 		return
// 	}

// 	// 3. 转发数据
// 	s.forward(client, target)
// }

// func (s *Server) authenticate(client net.Conn) error {
// 	return nil
// }

// func (s *Server) connect(client net.Conn) (net.Conn, error) {
// 	return nil
// }

// func (s *Server) forward(client net.Conn, target net.Conn) {
// 	return nil
// }
