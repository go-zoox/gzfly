package tow

import (
	"fmt"
	"io"
	"net"
	"time"

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

		client.Conn

		client.OnBinaryMessage = func(raw []byte) {
			msg := &Message{}
			if err := msg.Decode(raw); err != nil {
				ctx.Logger.Error("invalid message: %v", err)
				return
			}

			switch msg.Type {
			case "bind":
				go func() {
					if err := CreateTCPServer(&CreateTCPServerConfig{
						Port: 8888,
						OnConn: func(id string) net.Conn {
							return &WSConn{
								Client: Client,
							}
						},
					}); err != nil {

					}
				}()
			}

			ctx.Logger.Info("received [version: %s][type: %s] %s", msg.Version, msg.Type, msg.Payload)
		}
	})

	return core.Run(addr)
}

type WSConn struct {
	Client *zoox.WebSocketClient
}

func (wc *WSConn) Read(b []byte) (n int, err error) {
	wc.Client
}

func (wc *WSConn) Write(b []byte) (n int, err error) {
	msg := &Message{}
	bytes, err := msg.Encode()
	if err != nil {
		return 0, err
	}

	if err := wc.Client.WriteBinary(bytes); err != nil {
		return 0, err
	}

	return len(bytes), nil
}

func (wc *WSConn) Close() error {

}

func (wc *WSConn) LocalAddr() net.Addr {

}

func (wc *WSConn) RemoteAddr() net.Addr {

}

func (wc *WSConn) SetDeadline(t time.Time) error {
	return nil
}

func (wc *WSConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (wc *WSConn) SetWriteDeadline(t time.Time) error {
	return nil
}

type CreateTCPServerConfig struct {
	Port   int
	OnConn func(id string) net.Conn
}

func CreateTCPServer(cfg *CreateTCPServerConfig) error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Port))
	if err != nil {
		return err
	}
	defer listener.Close()
	for {
		source, err := listener.Accept()
		if err != nil {
			continue
		}

		id := uuid.V4()
		target := cfg.OnConn(id)
		go io.Copy(source, target)
		go io.Copy(target, source)
	}
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
