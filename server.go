package tow

import (
	"fmt"
	"net"

	"github.com/go-zoox/tcp-over-websocket/connection"
	"github.com/go-zoox/tcp-over-websocket/manager"
	"github.com/go-zoox/tcp-over-websocket/protocol"
	"github.com/go-zoox/tcp-over-websocket/protocol/authenticate"
	"github.com/go-zoox/tcp-over-websocket/protocol/handshake"
	"github.com/go-zoox/tcp-over-websocket/user"
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
	usersManager := manager.New(&manager.Options[user.User]{
		Cache: map[string]user.User{
			"id_04aba01": user.New("id_04aba01", "29f4e3d3a4302b4d9e01", "pair_3fd01"),
			"id_04aba02": user.New("id_04aba02", "29f4e3d3a4302b4d9e02", "pair_3fd02"),
		},
	})

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

		isAuthenticated := false
		client.OnBinaryMessage = func(raw []byte) {
			packet, err := protocol.Decode(raw)
			if err != nil {
				ctx.Logger.Error("invalid packet: %v", err)
				return
			}

			if !isAuthenticated && packet.Command != protocol.COMMAND_AUTHENTICATE {
				ctx.Logger.Error("client must authenticate before send command(%d)", packet.Command)
				return
			}

			switch packet.Command {
			case protocol.COMMAND_AUTHENTICATE:
				// decode
				authenticatePacket, err := authenticate.DecodeRequest(packet.Data)
				if err != nil {
					ctx.Logger.Error("failed to decode authenticate request packet: %v\n", err)
					return
				}

				writeResponse := func(status uint8, err error) error {
					if status != STATUS_OK {
						ctx.Logger.Error("[user: %s] failed to connect(status: %d): %v", authenticatePacket.UserClientID, status, err)
					}

					dataPacket := &authenticate.AuthenticateResponse{
						Status: status,
					}
					if err != nil {
						dataPacket.Message = err.Error()
					}

					dataBytes, err := authenticate.EncodeResponse(dataPacket)
					if err != nil {
						return fmt.Errorf("failed to encode authenticate response: %v", err)
					}

					packet := &protocol.Packet{
						Version: protocol.VERSION,
						Command: protocol.COMMAND_AUTHENTICATE,
						Data:    dataBytes,
					}
					if bytes, err := protocol.Encode(packet); err != nil {
						return fmt.Errorf("failed to encode packet %v", err)
					} else {
						return client.WriteBinary(bytes)
					}
				}

				ctx.Logger.Info("[user: %s][authenticate] start to authenticated", authenticatePacket.UserClientID)

				user, err := usersManager.Get(authenticatePacket.UserClientID)
				if err != nil {
					writeResponse(STATUS_INVALID_USER_CLIENT_ID, err)
					return
				}

				ok, err := user.Authenticate(authenticatePacket.Timestamp, authenticatePacket.Nonce, authenticatePacket.Signature)
				if !ok || err != nil {
					writeResponse(STATUS_INVALID_SIGNATURE, err)
					return
				}

				isAuthenticated = true

				writeResponse(STATUS_OK, nil)

				ctx.Logger.Info("[user: %s][authenticate] succeed to authenticate", authenticatePacket.UserClientID)
				return
			case protocol.COMMAND_HANDSHAKE:
				handshakePacket, err := handshake.DecodeRequest(packet.Data)
				if err != nil {
					ctx.Logger.Error("failed to decode handshake request packet: %v\n", err)
					return
				}

				writeResponse := func(status uint8, err error) error {
					if status != STATUS_OK {
						ctx.Logger.Error("[connection: %s] failed to connect(status: %d): %v", handshakePacket.ConnectionID, status, err)
					}

					dataPacket := &authenticate.AuthenticateResponse{
						Status: status,
					}
					if err != nil {
						dataPacket.Message = err.Error()
					}

					dataBytes, err := authenticate.EncodeResponse(dataPacket)
					if err != nil {
						return fmt.Errorf("failed to encode authenticate response: %v", err)
					}

					packet := &protocol.Packet{
						Version: protocol.VERSION,
						Command: protocol.COMMAND_AUTHENTICATE,
						Data:    dataBytes,
					}
					if bytes, err := protocol.Encode(packet); err != nil {
						return fmt.Errorf("failed to encode packet %v", err)
					} else {
						return client.WriteBinary(bytes)
					}
				}

				targetUser, err := usersManager.Get(handshakePacket.TargetUserClientID)
				if err != nil {
					writeResponse(STATUS_INVALID_USER_CLIENT_ID, err)
					return
				}

				if !targetUser.IsOnline() {
					writeResponse(STATUS_USER_NOT_ONLINE, nil)
					return
				}

				ok, err := targetUser.Pair(handshakePacket.TargetUserPairKey)
				if !ok {
					writeResponse(STATUS_FAILED_TO_PAIR, err)
					return
				}

				if err := targetUser.WritePacket(packet); err != nil {
					writeResponse(STATUS_FAILED_TO_HANDSHAKE, err)
					return
				}

				writeResponse(STATUS_OK, nil)
				return
			case protocol.COMMAND_BIND:
				go func() {
					if err := CreateTCPServer(&CreateTCPServerConfig{
						Port: 8888,
						OnConn: func() net.Conn {
							id := connection.GenerateID()
							wsConn := connection.New(id, client)
							wsConnsManager.Set(id, wsConn)
							return wsConn
						},
					}); err != nil {

					}
				}()
			case protocol.COMMAND_CONNECT:
				data := packet.Data
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
	OnConn func() net.Conn
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

		target := cfg.OnConn()
		go Copy(source, target)
		go Copy(target, source)
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
