package tow

import (
	"fmt"
	"net"

	"github.com/go-zoox/logger"
	"github.com/go-zoox/tcp-over-websocket/connection"
	"github.com/go-zoox/tcp-over-websocket/manager"
	"github.com/go-zoox/tcp-over-websocket/protocol"
	"github.com/go-zoox/tcp-over-websocket/protocol/authenticate"
	"github.com/go-zoox/tcp-over-websocket/protocol/handshake"
	"github.com/go-zoox/tcp-over-websocket/protocol/transmission"
	"github.com/go-zoox/tcp-over-websocket/tcp"
	"github.com/go-zoox/tcp-over-websocket/user"
	"github.com/go-zoox/zoox"
	zd "github.com/go-zoox/zoox/default"
)

type Server interface {
	Run(addr string) error
	//
	Bind(cfg *BindConfig) error
}

type server struct {
	Path string

	// store
	// connections *manager.Manager[*connection.WSConn]
	Users                   *manager.Manager[user.User]
	UserPairsByConnectionID *manager.Manager[*user.Pair]

	// listener
	OnConnect func(conn net.Conn, source, target string)
}

type ServerConfig struct {
	Path                    string
	Users                   *manager.Manager[user.User]
	UserPairsByConnectionID *manager.Manager[*user.Pair]
	OnConnect               func(conn net.Conn, source, target string)
}

func NewServer(cfg *ServerConfig) Server {
	Path := "/"
	Users := manager.New[user.User]()
	UserPairsByConnectionID := manager.New[*user.Pair]()
	var OnConnect func(conn net.Conn, source, target string)

	if cfg.Path != "" {
		Path = cfg.Path
	}
	if cfg.Users != nil {
		Users = cfg.Users
	}
	if cfg.UserPairsByConnectionID != nil {
		UserPairsByConnectionID = cfg.UserPairsByConnectionID
	}
	if cfg.OnConnect != nil {
		OnConnect = cfg.OnConnect
	}

	return &server{
		Path,
		Users,
		UserPairsByConnectionID,
		OnConnect,
	}
}

func (s *server) Run(addr string) error {
	core := zd.Default()

	// wsConnsManager := manager.New[*connection.WSConn]()
	// connectionIDTargetUserMap := manager.New[*user.Pair]()
	// usersManager := manager.New[user.User]()

	// @TODO
	s.Users.Set("id_04aba01", user.New("id_04aba01", "29f4e3d3a4302b4d9e01", "pair_3fd01"))
	s.Users.Set("id_04aba02", user.New("id_04aba02", "29f4e3d3a4302b4d9e02", "pair_3fd02"))

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

		// @TODO
		isAuthenticated := false
		userClientID := ""
		var currentUser user.User
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

				user, err := s.Users.Get(authenticatePacket.UserClientID)
				if err != nil {
					writeResponse(STATUS_INVALID_USER_CLIENT_ID, err)
					return
				}

				ok, err := user.Authenticate(authenticatePacket.Timestamp, authenticatePacket.Nonce, authenticatePacket.Signature)
				if !ok || err != nil {
					writeResponse(STATUS_INVALID_SIGNATURE, err)
					return
				}

				// @TODO
				isAuthenticated = true
				userClientID = authenticatePacket.UserClientID
				currentUser = user
				user.SetOnline(client)

				writeResponse(STATUS_OK, nil)

				ctx.Logger.Info("[user: %s][authenticate] succeed to authenticate", userClientID)
				return
			case protocol.COMMAND_HANDSHAKE_REQUEST:
				handshakePacket, err := handshake.DecodeRequest(packet.Data)
				if err != nil {
					ctx.Logger.Error("failed to decode handshake request packet: %v\n", err)
					return
				}

				Network := "tcp"
				switch handshakePacket.Network {
				case protocol.NETWORK_TCP:
					Network = "tcp"
				case protocol.NETWORK_UDP:
					Network = "udp"
				default:
					logger.Errorf("unknown network type: %d, only support 0x01(tcp)/0x02(udp)", handshakePacket.Network)
					return
				}

				writeResponse := func(status uint8, err error) error {
					if status != STATUS_OK {
						ctx.Logger.Error("[connection: %s] failed to handshake(status: %d): %v", handshakePacket.ConnectionID, status, err)
					}

					dataPacket := &handshake.HandshakeResponse{
						ConnectionID: handshakePacket.ConnectionID,
						Status:       status,
					}
					if err != nil {
						dataPacket.Message = err.Error()
					}

					dataBytes, err := handshake.EncodeResponse(dataPacket)
					if err != nil {
						return fmt.Errorf("failed to encode handshake response: %v", err)
					}

					packet := &protocol.Packet{
						Version: protocol.VERSION,
						Command: protocol.COMMAND_HANDSHAKE_RESPONSE,
						Data:    dataBytes,
					}
					if bytes, err := protocol.Encode(packet); err != nil {
						return fmt.Errorf("failed to encode packet %v", err)
					} else {
						return client.WriteBinary(bytes)
					}
				}

				logger.Infof(
					"[user: %s][handshake][connection: %s] start to check target user(%s) ...",
					userClientID,
					handshakePacket.ConnectionID,
					handshakePacket.TargetUserClientID,
				)
				targetUser, err := s.Users.Get(handshakePacket.TargetUserClientID)
				if err != nil {
					writeResponse(STATUS_INVALID_USER_CLIENT_ID, err)
					return
				}

				logger.Infof(
					"[user: %s][handshake][connection: %s] start to check online(user: %s) ...",
					userClientID,
					handshakePacket.ConnectionID,
					handshakePacket.TargetUserClientID,
				)
				if !targetUser.IsOnline() {
					writeResponse(STATUS_USER_NOT_ONLINE, nil)
					return
				}

				logger.Infof(
					"[user: %s][handshake][connection: %s] request target %s at %s://%s:%d",
					userClientID,
					handshakePacket.ConnectionID,
					targetUser.GetClientID(),
					Network,
					handshakePacket.DSTAddr,
					handshakePacket.DSTPort,
				)

				logger.Infof(
					"[user: %s][handshake][connection: %s] start to pair",
					userClientID,
					handshakePacket.ConnectionID,
				)
				ok, err := targetUser.Pair(handshakePacket.TargetUserPairKey)
				if !ok {
					writeResponse(STATUS_FAILED_TO_PAIR, err)
					return
				}

				logger.Infof(
					"[user: %s][handshake][connection: %s] write packet to %s",
					userClientID,
					handshakePacket.ConnectionID,
					targetUser.GetClientID(),
				)
				if err := targetUser.WritePacket(packet); err != nil {
					writeResponse(STATUS_FAILED_TO_HANDSHAKE, err)
					return
				}

				s.UserPairsByConnectionID.Set(handshakePacket.ConnectionID, &user.Pair{
					Source: currentUser,
					Target: targetUser,
				})
				writeResponse(STATUS_OK, nil)
				return
			// case protocol.COMMAND_BIND:
			// 	go func() {
			// 		if err := CreateTCPServer(&CreateTCPServerConfig{
			// 			Port: 8888,
			// 			OnConn: func() (net.Conn, error) {
			// 				id := connection.GenerateID()
			// 				wsConn := connection.New(id, client)
			// 				wsConnsManager.Set(id, wsConn)
			// 				return wsConn, nil
			// 			},
			// 		}); err != nil {

			// 		}
			// 	}()
			// case protocol.COMMAND_CONNECT:
			// 	data := packet.Data
			// 	id, err := connection.DecodeID(data)
			// 	if err != nil {
			// 		fmt.Print("[connect] failed to parse id:", err)
			// 		return
			// 	}

			// 	wsconn, err := wsConnsManager.Get(id)
			// 	if err != nil {
			// 		fmt.Println("[connect] failed to get conn:", err)
			// 		return
			// 	}

			// 	wsconn.Stream <- data
			case protocol.COMMAND_TRANSMISSION:
				transmissionPacket, err := transmission.Decode(packet.Data)
				if err != nil {
					ctx.Logger.Error(
						"[user: %s][transmission][connection: %s] failed to decode transmission request packet: %v\n",
						userClientID,
						transmissionPacket.ConnectionID,
						err,
					)
					return
				}

				logger.Debugf(
					"[user: %s][transmission][connection: %s] start to check user pair ...",
					userClientID,
					transmissionPacket.ConnectionID,
				)
				userPair, err := s.UserPairsByConnectionID.Get(transmissionPacket.ConnectionID)
				if err != nil {
					ctx.Logger.Error(
						"[user: %s][transmission][connection: %s] failed to get target user: %v\n",
						userClientID,
						transmissionPacket.ConnectionID,
						err,
					)
					return
				}

				var targetUser user.User
				if currentUser.GetClientID() == userPair.Source.GetClientID() {
					targetUser = userPair.Target
				} else {
					targetUser = userPair.Source
				}

				logger.Debugf(
					"[user: %s][transmission][connection: %s] start to transmission to target user(%s)",
					currentUser.GetClientID(),
					transmissionPacket.ConnectionID,
					targetUser.GetClientID(),
				)
				// if err := targetUser.WritePacket(packet); err != nil {
				// 	ctx.Logger.Error(
				// 		"[user: %s][transmission][connection: %s] failed to write packet: %v\n",
				// 		userClientID,
				// 		transmissionPacket.ConnectionID,
				// 		err,
				// 	)
				// 	return
				// }
				if err := targetUser.WriteBytes(raw); err != nil {
					ctx.Logger.Error(
						"[user: %s][transmission][connection: %s] failed to write packet: %v\n",
						userClientID,
						transmissionPacket.ConnectionID,
						err,
					)
					return
				}

				logger.Debugf(
					"[user: %s][transmission][connection: %s] succeed to transmission to target user(%s)",
					currentUser.GetClientID(),
					transmissionPacket.ConnectionID,
					targetUser.GetClientID(),
				)
			}
		}
	})

	return core.Run(addr)
}

func (s *server) Bind(cfg *BindConfig) error {
	logger.Info(
		"[bind] start to bind with target(%s): %s://%s:%d:%s:%d",
		cfg.TargetUserClientID,
		cfg.Network,
		cfg.LocalHost,
		cfg.LocalPort,
		cfg.RemoteHost,
		cfg.RemotePort,
	)

	Network := protocol.NETWORK_TCP
	switch cfg.Network {
	case "tcp":
		Network = protocol.NETWORK_TCP
	case "udp":
		Network = protocol.NETWORK_UDP
	default:
		return fmt.Errorf("unknown network type: %s, only support tcp/udp", cfg.Network)
	}

	connections := manager.New[*connection.WSConn]()

	if err := tcp.CreateTCPServer(&tcp.CreateTCPServerConfig{
		Host: cfg.LocalHost,
		Port: cfg.LocalPort,
		OnConn: func() (net.Conn, error) {
			ConnectionID := connection.GenerateID()

			targetUser, err := s.Users.Get(cfg.TargetUserClientID)
			if err != nil {
				return nil, fmt.Errorf("failed to get user(%s): %v", cfg.TargetUserClientID, err)
			}

			if !targetUser.IsOnline() {
				return nil, fmt.Errorf("user(%s) is not online", cfg.TargetUserClientID)
			}

			var wsConn *connection.WSConn
			currentUser := s.GetSystemUser(func(bytes []byte) error {
				packet, _ := protocol.Decode(bytes)
				transmissionPacket, _ := transmission.Decode(packet.Data)
				connectionID := transmissionPacket.ConnectionID

				wsConn, err = connections.Get(connectionID)
				if err != nil {
					return fmt.Errorf("[bind] failed to get connection(%s): %v", connectionID, err)
				}

				wsConn.Stream <- transmissionPacket.Data

				// fmt.Println("write:", transmissionPacket.Data)
				// fmt.Println("fff:", len(bytes))
				return nil

				// return targetUser.WriteBytes(bytes)
			})
			// wsClient := currentUser.GetWSClient()
			wsClient := connection.NewWSClient(func(bytes []byte) error {
				return targetUser.WriteBytes(bytes)
			})
			wsConn = connection.New(ConnectionID, wsClient)
			if err := connections.Set(ConnectionID, wsConn); err != nil {
				return nil, fmt.Errorf("[bind] failed to set connection(%s): %v", ConnectionID, err)
			}

			// c.connections.Set(wsConn.ID, wsConn)
			s.UserPairsByConnectionID.Set(ConnectionID, &user.Pair{
				Source: currentUser,
				Target: targetUser,
			})

			// 1. handshake (request) => create connection
			dataPacket := &handshake.HandshakeRequest{
				ConnectionID:       ConnectionID,
				TargetUserClientID: cfg.TargetUserClientID,
				TargetUserPairKey:  cfg.TargetUserPairKey,
				// @TODO
				Network: uint8(Network),
				// @TODO
				ATyp:    handshake.ATYP_IPv4,
				DSTAddr: cfg.RemoteHost,
				DSTPort: uint16(cfg.RemotePort),
			}
			data, err := handshake.EncodeRequest(dataPacket)
			if err != nil {
				return nil, fmt.Errorf("failed to encode handshake request: %v", err)
			}
			targetUser.WritePacket(&protocol.Packet{
				Version: protocol.VERSION,
				Command: protocol.COMMAND_HANDSHAKE_REQUEST,
				Data:    data,
			})

			return wsConn, nil
		},
	}); err != nil {
		return fmt.Errorf("failed to create tcp server: %v", err)
	}

	return nil
}

func (s *server) GetSystemUser(write func(bytes []byte) error) user.User {
	userClientID := "id_system_"

	systemUser, err := s.Users.GetOrCreate(userClientID, func() user.User {
		wsClient := connection.NewWSClient(write)
		systemUser := user.New("id_system_", "29f4e3d3a4302b4d9e02", "pair_3fd02")
		systemUser.SetOnline(wsClient)
		return systemUser
	})
	if err != nil {
		panic(fmt.Errorf("failed to create system user: %v", err))
	}
	return systemUser
}

// func (s *server) process(client net.Conn) {
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

// func (s *server) authenticate(client net.Conn) error {
// 	return nil
// }

// func (s *server) connect(client net.Conn) (net.Conn, error) {
// 	return nil
// }

// func (s *server) forward(client net.Conn, target net.Conn) {
// 	return nil
// }
