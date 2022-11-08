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

	wsconnManagers := WSConnManager{}

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
							conn := &WSConn{
								ID:     id,
								Client: client,
							}

							wsconnManagers.Set(id, conn)

							return conn
						},
					}); err != nil {

					}
				}()
			case "connect":
				idLength := int(msg.Payload[0])
				id := string(msg.Payload[1 : idLength+1])
				wsconn, err := wsconnManagers.Get(id)
				if err != nil {
					fmt.Errorf("connect error: %v", err)
					return
				}

				wsconn.Stream <- msg.Payload[36:]
			}

			ctx.Logger.Info("received [version: %s][type: %s] %s", msg.Version, msg.Type, msg.Payload)
		}
	})

	return core.Run(addr)
}

type WSConnManager struct {
	cache map[string]*WSConn
}

func (m *WSConnManager) Get(id string) (*WSConn, error) {
	if conn, ok := m.cache[id]; ok {
		return conn, nil
	}

	return nil, fmt.Errorf("id %s not found", id)
}

func (m *WSConnManager) Set(id string, conn *WSConn) error {
	if m.cache == nil {
		m.cache = make(map[string]*WSConn)
	}

	m.cache[id] = conn
	return nil
}

type WSConn struct {
	ID     string
	Client *zoox.WebSocketClient
	Stream chan []byte
}

func (wc *WSConn) Read(b []byte) (n int, err error) {
	n = copy(b, <-wc.Stream)
	return
}

func (wc *WSConn) Write(b []byte) (n int, err error) {
	msg := &Message{}
	msg.Version = "v0.0.0"
	msg.Type = ""

	msg.Payload = []byte{}
	idBytes := []byte(wc.ID)
	idLength := len(idBytes)
	bLength := len(b)
	msg.Payload = append(msg.Payload, byte(idLength))
	msg.Payload = append(msg.Payload, idBytes...)
	msg.Payload = append(msg.Payload, byte(bLength))
	msg.Payload = append(msg.Payload, b...)

	bytes, err := msg.Encode()
	if err != nil {
		return 0, err
	}

	if err := wc.Client.WriteBinary(bytes); err != nil {
		return 0, err
	}

	return bLength, nil
}

func (wc *WSConn) Close() error {
	msg := &Message{}
	msg.Payload = []byte{}
	idBytes := []byte(wc.ID)
	idLength := len(idBytes)
	msg.Payload = append(msg.Payload, byte(idLength))
	msg.Payload = append(msg.Payload, idBytes...)
	// msg.Payload = append(msg.Payload, b...)

	bytes, err := msg.Encode()
	if err != nil {
		return err
	}

	return wc.Client.WriteBinary(bytes)
}

func (wc *WSConn) LocalAddr() net.Addr {
	return wc.Client.Conn.LocalAddr()
}

func (wc *WSConn) RemoteAddr() net.Addr {
	return wc.Client.Conn.RemoteAddr()
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
