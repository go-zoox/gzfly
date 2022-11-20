package core

import (
	"fmt"
	"net"

	"github.com/go-zoox/gzfly/connection"
	"github.com/go-zoox/gzfly/manager"
	"github.com/go-zoox/gzfly/network/tcp"
	"github.com/go-zoox/gzfly/user"
	"github.com/go-zoox/logger"
	"github.com/go-zoox/packet/socksz"
	"github.com/go-zoox/packet/socksz/authenticate"
	"github.com/go-zoox/packet/socksz/base"
	"github.com/go-zoox/packet/socksz/close"
	"github.com/go-zoox/packet/socksz/forward"
	"github.com/go-zoox/packet/socksz/handshake"
	"github.com/go-zoox/zoox"
	zd "github.com/go-zoox/zoox/default"
)

type Server interface {
	Run() error
	//
	Bind(cfg *BindConfig) error
}

type server struct {
	Port int64
	Path string

	// store
	// connections *manager.Manager[*connection.WSConn]
	Users                   *manager.Manager[*user.User]
	UserPairsByConnectionID *manager.Manager[*user.Pair]

	// listener
	OnConnect func(conn net.Conn, source, target string)
}

type ServerConfig struct {
	Port  int64             `config:"port"`
	Path  string            `config:"path"`
	Users []user.UserClient `config:"clients"`
	//
	UserPairsByConnectionID *manager.Manager[*user.Pair]
	OnConnect               func(conn net.Conn, source, target string)
}

func NewServer(cfg *ServerConfig) Server {
	var Port int64 = 8080
	Path := "/"
	Users := manager.New[*user.User]()
	UserPairsByConnectionID := manager.New[*user.Pair]()
	var OnConnect func(conn net.Conn, source, target string)

	if cfg.Port != 0 {
		Port = cfg.Port
	}
	if cfg.Path != "" {
		Path = cfg.Path
	}
	if cfg.Users != nil {
		for _, u := range cfg.Users {
			Users.Set(u.ClientID, user.New(u.ClientID, u.ClientSecret, u.PairKey))
		}
	}
	if cfg.UserPairsByConnectionID != nil {
		UserPairsByConnectionID = cfg.UserPairsByConnectionID
	}
	if cfg.OnConnect != nil {
		OnConnect = cfg.OnConnect
	}

	return &server{
		Port,
		Path,
		Users,
		//
		UserPairsByConnectionID,
		OnConnect,
	}
}

func (s *server) Run() error {
	core := zd.Default()

	// wsConnsManager := manager.New[*connection.WSConn]()
	// connectionIDTargetUserMap := manager.New[*user.Pair]()
	// usersManager := manager.New[user.User]()

	// @TODO
	// s.Users.Set("id_04aba01", user.New("id_04aba01", "29f4e3d3a4302b4d9e01", "pair_3fd01"))
	// s.Users.Set("id_04aba02", user.New("id_04aba02", "29f4e3d3a4302b4d9e02", "pair_3fd02"))

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
		var currentUser *user.User

		client.OnBinaryMessage = func(raw []byte) {
			packet := &base.Base{}
			err := packet.Decode(raw)
			if err != nil {
				ctx.Logger.Error("invalid packet: %v", err)
				return
			}

			if !isAuthenticated && packet.Cmd != socksz.CommandAuthenticate {
				ctx.Logger.Error("client must authenticate before send command(%d)", packet.Cmd)
				return
			}

			switch packet.Cmd {
			case socksz.CommandAuthenticate:
				// decode
				authenticatePacket := &authenticate.Request{}
				err := authenticatePacket.Decode(packet.Data)
				if err != nil {
					ctx.Logger.Error("failed to decode authenticate request packet: %v\n", err)
					return
				}

				writeResponse := func(status uint8, err error) error {
					if status != STATUS_OK {
						ctx.Logger.Error("[user: %s] failed to connect(status: %d): %v", authenticatePacket.UserClientID, status, err)
					}

					dataPacket := &authenticate.Response{
						Status: status,
					}
					if err != nil {
						dataPacket.Message = err.Error()
					}

					dataBytes, err := dataPacket.Encode()
					if err != nil {
						return fmt.Errorf("failed to encode authenticate response: %v", err)
					}

					npacket := &base.Base{
						Ver:  socksz.VER,
						Cmd:  socksz.CommandAuthenticate,
						Data: dataBytes,
						//
						Crypto: packet.Crypto,
					}
					if bytes, err := npacket.Encode(); err != nil {
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
				authenticatePacket.Secret = user.ClientSecret

				if err := authenticatePacket.Verify(); err != nil {
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
			case socksz.CommandHandshakeRequest:
				handshakePacket := handshake.Request{}
				err := handshakePacket.Decode(packet.Data)
				if err != nil {
					ctx.Logger.Error("failed to decode handshake request packet: %v\n", err)
					return
				}

				Network := "tcp"
				switch handshakePacket.Network {
				case handshake.NetworkTCP:
					Network = "tcp"
				case handshake.NetworkUDP:
					Network = "udp"
				default:
					logger.Errorf("unknown network type: %d, only support 0x01(tcp)/0x02(udp)", handshakePacket.Network)
					return
				}

				writeResponse := func(status uint8, err error) error {
					if status != STATUS_OK {
						ctx.Logger.Error("[connection: %s] failed to handshake(status: %d): %v", handshakePacket.ConnectionID, status, err)
					}

					dataPacket := &handshake.Response{
						ConnectionID: handshakePacket.ConnectionID,
						Status:       status,
					}
					if err != nil {
						dataPacket.Message = err.Error()
					}

					dataBytes, err := dataPacket.Encode()
					if err != nil {
						return fmt.Errorf("failed to encode handshake response: %v", err)
					}

					npacket := &base.Base{
						Ver:    socksz.VER,
						Cmd:    socksz.CommandHandshakeResponse,
						Data:   dataBytes,
						Crypto: packet.Crypto,
					}
					if bytes, err := npacket.Encode(); err != nil {
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

				handshakePacket.Secret = targetUser.PairKey
				err = handshakePacket.Verify()
				if err != nil {
					ctx.Logger.Error("invalid handshake request packet: %v - %s\n", err)
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
				ok, err := targetUser.Pair(
					handshakePacket.ConnectionID,
					handshakePacket.TargetUserClientID,
					handshakePacket.TargetUserPairSignature,
				)
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
			// case socksz.COMMAND_BIND:
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
			// case socksz.COMMAND_CONNECT:
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
			case socksz.CommandForward:
				// fmt.Println("forward aes:", packet.Crypto)

				forwardPacket := &forward.Forward{
					Crypto: packet.Crypto,
					Secret: currentUser.ClientSecret,
				}
				err := forwardPacket.Decode(packet.Data)
				if err != nil {
					ctx.Logger.Error(
						"[user: %s][forward][connection: %s] failed to decode forward request packet: %v\n",
						userClientID,
						forwardPacket.ConnectionID,
						err,
					)
					return
				}

				logger.Debugf(
					"[user: %s][forward][connection: %s] start to check user pair ...",
					userClientID,
					forwardPacket.ConnectionID,
				)
				userPair, err := s.UserPairsByConnectionID.Get(forwardPacket.ConnectionID)
				if err != nil {
					ctx.Logger.Error(
						"[user: %s][forward][connection: %s] failed to get target user: %v\n",
						userClientID,
						forwardPacket.ConnectionID,
						err,
					)
					return
				}

				var targetUser *user.User
				if currentUser.GetClientID() == userPair.Source.GetClientID() {
					targetUser = userPair.Target
				} else {
					targetUser = userPair.Source
				}

				logger.Debugf(
					"[user: %s][forward][connection: %s] start to forward to target user(%s)",
					currentUser.GetClientID(),
					forwardPacket.ConnectionID,
					targetUser.GetClientID(),
				)
				// if err := targetUser.WritePacket(packet); err != nil {
				// 	ctx.Logger.Error(
				// 		"[user: %s][forward][connection: %s] failed to write packet: %v\n",
				// 		userClientID,
				// 		forwardPacket.ConnectionID,
				// 		err,
				// 	)
				// 	return
				// }

				forwardPacket.Crypto = packet.Crypto
				forwardPacket.Secret = targetUser.ClientSecret
				forwardBytes, err := forwardPacket.Encode()
				if err != nil {
					ctx.Logger.Error(
						"[user: %s][forward][connection: %s] failed to encode forward request packet: %v\n",
						userClientID,
						forwardPacket.ConnectionID,
						err,
					)
					return
				}

				packet.Data = forwardBytes
				cipher, err := packet.Encode()
				if err != nil {
					ctx.Logger.Error(
						"[user: %s][forward][connection: %s] failed to encode request packet: %v\n",
						userClientID,
						forwardPacket.ConnectionID,
						err,
					)
					return
				}

				if err := targetUser.WriteBytes(cipher); err != nil {
					ctx.Logger.Error(
						"[user: %s][forward][connection: %s] failed to write packet: %v\n",
						userClientID,
						forwardPacket.ConnectionID,
						err,
					)
					return
				}

				logger.Debugf(
					"[user: %s][forward][connection: %s] succeed to forward to target user(%s)",
					currentUser.GetClientID(),
					forwardPacket.ConnectionID,
					targetUser.GetClientID(),
				)
			case socksz.CommandClose:
				closePacket := &close.Close{}
				err := closePacket.Decode(packet.Data)
				if err != nil {
					ctx.Logger.Error(
						"[user: %s][close][connection: %s] failed to decode close request packet: %v\n",
						userClientID,
						closePacket.ConnectionID,
						err,
					)
					return
				}

				logger.Debugf(
					"[user: %s][close][connection: %s] start to check user pair ...",
					userClientID,
					closePacket.ConnectionID,
				)
				userPair, err := s.UserPairsByConnectionID.Get(closePacket.ConnectionID)
				if err != nil {
					ctx.Logger.Error(
						"[user: %s][close][connection: %s] failed to get target user: %v\n",
						userClientID,
						closePacket.ConnectionID,
						err,
					)
					return
				}

				var targetUser *user.User
				if currentUser.GetClientID() == userPair.Source.GetClientID() {
					targetUser = userPair.Target
				} else {
					targetUser = userPair.Source
				}

				logger.Debugf(
					"[user: %s][close][connection: %s] start to close to target user(%s)",
					currentUser.GetClientID(),
					closePacket.ConnectionID,
					targetUser.GetClientID(),
				)
				// if err := targetUser.WritePacket(packet); err != nil {
				// 	ctx.Logger.Error(
				// 		"[user: %s][close][connection: %s] failed to write packet: %v\n",
				// 		userClientID,
				// 		closePacket.ConnectionID,
				// 		err,
				// 	)
				// 	return
				// }
				if err := targetUser.WriteBytes(raw); err != nil {
					ctx.Logger.Error(
						"[user: %s][close][connection: %s] failed to write packet: %v\n",
						userClientID,
						closePacket.ConnectionID,
						err,
					)
					return
				}

				logger.Debugf(
					"[user: %s][close][connection: %s] succeed to close to target user(%s)",
					currentUser.GetClientID(),
					closePacket.ConnectionID,
					targetUser.GetClientID(),
				)
			default:
				logger.Warnf("[ignore] unknown command %d", packet.Cmd)
			}
		}
	})

	return core.Run(fmt.Sprintf(":%d", s.Port))
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

	Network := handshake.NetworkTCP
	switch cfg.Network {
	case "tcp":
		Network = handshake.NetworkTCP
	case "udp":
		Network = handshake.NetworkUDP
	default:
		return fmt.Errorf("unknown network type: %s, only support tcp/udp", cfg.Network)
	}

	connections := manager.New[*connection.WSConn]()
	// mu := &sync.RWMutex{}

	if err := tcp.Serve(&tcp.ServeConfig{
		Host: cfg.LocalHost,
		Port: cfg.LocalPort,
		OnConn: func() (net.Conn, error) {
			targetUser, err := s.Users.Get(cfg.TargetUserClientID)
			if err != nil {
				return nil, fmt.Errorf("failed to get user(%s): %v", cfg.TargetUserClientID, err)
			}

			if !targetUser.IsOnline() {
				return nil, fmt.Errorf("user(%s) is not online", cfg.TargetUserClientID)
			}

			var wsConn *connection.WSConn
			currentUser := s.GetSystemUser(func(bytes []byte) error {
				packet := &base.Base{}
				_ = packet.Decode(bytes)

				forwardPacket := &forward.Forward{
					// Crypto: packet.Crypto,
					// Secret: currentUser.ClientSecret,
				}
				_ = forwardPacket.Decode(packet.Data)

				wsConn, err = connections.Get(forwardPacket.ConnectionID)
				if err != nil {
					return fmt.Errorf("[bind] failed to get connection(%s): %v", forwardPacket.ConnectionID, err)
				}

				wsConn.Stream <- forwardPacket.Data

				// fmt.Println("write:", forwardPacket.Data)
				// fmt.Println("fff:", len(bytes))
				return nil

				// return targetUser.WriteBytes(bytes)
			})
			// wsClient := currentUser.GetWSClient()
			wsClient := connection.NewWSClient(func(bytes []byte) error {
				// mu.Lock()
				// defer mu.Unlock()

				return targetUser.WriteBytes(bytes)
			})
			wsConn = connection.New(wsClient, &connection.ConnectionOptions{
				// Crypto: c.Crypto, // packet.Crypto
				// Secret: c.Secret,
			})
			wsConn.OnClose = func() {
				connections.Remove(wsConn.ID)
			}
			if err := connections.Set(wsConn.ID, wsConn); err != nil {
				return nil, fmt.Errorf("[bind] failed to set connection(%s): %v", wsConn.ID, err)
			}

			// c.connections.Set(wsConn.ID, wsConn)
			s.UserPairsByConnectionID.Set(wsConn.ID, &user.Pair{
				Source: currentUser,
				Target: targetUser,
			})

			// TargetUserPairSignature := hmac.Sha256(fmt.Sprintf("%s_%s", wsConn.ID, cfg.TargetUserClientID), cfg.TargetUserPairKey)

			// 1. handshake (request) => create connection
			dataPacket := &handshake.Request{
				Secret: cfg.TargetUserPairKey,
				//
				ConnectionID:       wsConn.ID,
				TargetUserClientID: cfg.TargetUserClientID,
				// TargetUserPairSignature: TargetUserPairSignature,
				// @TODO
				Network: uint8(Network),
				// @TODO
				ATyp:    handshake.ATypIPv4,
				DSTAddr: cfg.RemoteHost,
				DSTPort: uint16(cfg.RemotePort),
			}
			data, err := dataPacket.Encode()
			if err != nil {
				return nil, fmt.Errorf("failed to encode handshake request: %v", err)
			}
			targetUser.WritePacket(&base.Base{
				Ver:  socksz.VER,
				Cmd:  socksz.CommandHandshakeRequest,
				Data: data,
			})

			return wsConn, nil
		},
	}); err != nil {
		return fmt.Errorf("failed to create tcp server: %v", err)
	}

	return nil
}

func (s *server) GetSystemUser(write func(bytes []byte) error) *user.User {
	userClientID := "id_system_"

	systemUser, err := s.Users.GetOrCreate(userClientID, func() *user.User {
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
