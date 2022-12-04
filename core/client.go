package core

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/go-zoox/gzfly/connection"
	"github.com/go-zoox/gzfly/manager"
	"github.com/go-zoox/gzfly/network"
	"github.com/go-zoox/gzfly/user"
	"github.com/go-zoox/logger"
	"github.com/go-zoox/packet/socksz"
	"github.com/go-zoox/packet/socksz/authenticate"
	"github.com/go-zoox/packet/socksz/base"
	"github.com/go-zoox/packet/socksz/close"
	"github.com/go-zoox/packet/socksz/forward"
	"github.com/go-zoox/packet/socksz/handshake"
	"github.com/gorilla/websocket"
)

type Client interface {
	Listen() error
	//
	OnConnect(cb func())
	//
	Bind(cfg *BindConfig) error
}

type client struct {
	sync.RWMutex

	Conn *websocket.Conn

	Protocol string `json:"socksz"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Path     string `json:"path"`

	//
	Crypto uint8
	Secret string

	// User
	User *user.User

	onConnect       func()
	OnDisconnect    func()
	OnMessage       func(typ int, msg []byte)
	OnTextMessage   func(msg []byte)
	OnBinaryMessage func(msg []byte)
	OnError         func(err error)
	OnPing          func()
	OnPong          func()

	// store
	connections *manager.Manager[*connection.WSConn]
}

type ClientConfig struct {
	Protocol string `json:"socksz"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Path     string `json:"path"`

	//
	Crypto string

	// User
	User *user.User
}

type BindConfig struct {
	TargetUserClientID string
	TargetUserPairKey  string
	Network            string
	LocalHost          string
	LocalPort          int
	RemoteHost         string
	RemotePort         int
}

func NewClient(cfg *ClientConfig) (Client, error) {
	Crypto, err := socksz.GetCrypto(cfg.Crypto)
	if err != nil {
		return nil, err
	}

	return &client{
		// store
		connections: manager.New[*connection.WSConn](),
		//
		// OnConnect: func(conn net.Conn, source string, target string) {
		// 	logger.Info("[%s] connect to %s", source, target)
		// },
		Protocol: cfg.Protocol,
		Host:     cfg.Host,
		Port:     cfg.Port,
		Path:     cfg.Path,
		// USER
		User: cfg.User,
		//
		Crypto: Crypto,
		Secret: cfg.User.ClientSecret,
	}, nil
}

func (c *client) authenticate() error {
	logger.Info("[authenticate] start to authenticate(%s)", c.User.GetClientID())

	UserClientID := c.User.GetClientID()
	Timestamp := fmt.Sprintf("%d", time.Now().UnixMilli())
	Nonce := "123456"
	Signature, err := c.User.Sign(Timestamp, Nonce)
	if err != nil {
		return fmt.Errorf("failed to create signature: %v", err)
	}

	packet := &authenticate.Request{
		Secret: c.User.ClientSecret,
		//
		UserClientID: UserClientID,
		Timestamp:    Timestamp,
		Nonce:        Nonce,
		Signature:    Signature,
	}
	bytes, err := packet.Encode()
	if err != nil {
		return err
	}

	return c.writePacket(socksz.CommandAuthenticate, bytes)
}

func (c *client) request() error {
	if c.Conn == nil {
		u := url.URL{Scheme: c.Protocol, Host: net.JoinHostPort(c.Host, fmt.Sprintf("%d", c.Port)), Path: c.Path}
		conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
		if err != nil {
			return err
		}

		c.Conn = conn
	}

	// authentication
	if err := c.authenticate(); err != nil {
		return fmt.Errorf("failed to authenticate: %v", err)
	}

	return nil
}

func (c *client) connect() error {
	if err := c.request(); err != nil {
		return err
	}

	for {
		mt, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				c.Conn.Close()
				return nil
			}

			logger.Errorf("[ws] read err: %s (type: %d)", err, mt)
			return c.reconnect()
		}

		switch mt {
		case websocket.TextMessage:
			if c.OnTextMessage != nil {
				c.OnTextMessage(message)
			}
		case websocket.BinaryMessage:
			if c.OnBinaryMessage != nil {
				c.OnBinaryMessage(message)
			}
		case websocket.CloseMessage:
			// @TODO
		case websocket.PingMessage:
			if c.OnPing != nil {
				c.OnPing()
			}
		case websocket.PongMessage:
			if c.OnPong != nil {
				c.OnPong()
			}
		default:
			fmt.Printf("unknown message type: %d\n", mt)
		}

		if c.OnMessage != nil {
			c.OnMessage(mt, message)
		}
	}
}

func (c *client) reconnect() error {
	logger.Infof("[ws] reconnecting ...")
	if c.Conn != nil {
		c.Conn.Close()
		c.Conn = nil
	}

	return c.connect()
}

// func (c *client) WriteMessage(messageType int, data []byte) error {
// 	return c.Conn.WriteMessage(messageType, data)
// }

// func (c *client) WriteTextMessage(data []byte) error {
// 	return c.Conn.WriteMessage(MessageTypeText, data)
// }

func (c *client) WriteBinary(data []byte) error {
	return c.Write(MessageTypeBinary, data)
}

func (c *client) Write(messageType int, data []byte) error {
	c.Lock()
	defer c.Unlock()

	if c.Conn == nil {
		return fmt.Errorf("conn is not online")
	}

	return c.Conn.WriteMessage(messageType, data)
}

func (c *client) writePacket(command uint8, data []byte) error {
	packet := &base.Base{
		Ver:  socksz.VER,
		Cmd:  command,
		Data: data,
		//
		Crypto: c.Crypto,
	}
	bytes, err := packet.Encode()
	if err != nil {
		return fmt.Errorf("invalid message: %s", err)
	}

	return c.WriteBinary(bytes)
}

// func (c *client) Emit(command uint8, data []byte) error {
// 	return c.WritePacket(command, data)
// }

func (c *client) Listen() error {
	// wsConnsManager := manager.New[*connection.WSConn]()

	c.OnBinaryMessage = func(raw []byte) {
		packet := &base.Base{}
		err := packet.Decode(raw)
		if err != nil {
			fmt.Println("invalid message format")
			return
		}

		forceCloseConn := func(connectionID string) {
			logger.Info("[forward][incomming][connection: %s] force close", connectionID)
			// notify close
			closePacket := &close.Close{
				ConnectionID: connectionID,
			}
			if data, err := closePacket.Encode(); err != nil {
				logger.Errorf("[forward][incomming][connection: %s] failed to encode notify close data", connectionID)
				return
			} else {
				if err := c.writePacket(socksz.CommandForward, data); err != nil {
					logger.Errorf("[forward][incomming][connection: %s] failed to write notify close", connectionID)
					return
				}
			}
		}

		switch packet.Cmd {
		case socksz.CommandAuthenticate:
			authenticatePacket := &authenticate.Response{}
			err := authenticatePacket.Decode(packet.Data)
			if err != nil {
				logger.Error("[authenticate] failed to decode authenticate response: %v", err)
				os.Exit(-1)
				return
			}

			if authenticatePacket.Status != socksz.StatusOK {
				logger.Error("[authenticate] failed to authenticate, status: %d, message: %s", authenticatePacket.Status, authenticatePacket.Message)
				os.Exit(-1)
				return
			}

			if c.onConnect != nil {
				go c.onConnect()
			}

			// heart beat
			go func() {
				for {
					// logger.Info("ping")
					time.Sleep(15 * time.Second)

					if c.Conn != nil {
						if err := c.Write(websocket.PingMessage, []byte{}); err != nil {
							return
						}
					}
				}
			}()

			logger.Info("[authenticate] succeed to auth as %s", c.User.GetClientID())
			return
		case socksz.CommandHandshakeRequest:
			logger.Infof("[handshake] request comming ...")

			handshakePacket := &handshake.Request{
				Secret: c.User.PairKey,
			}
			err := handshakePacket.Decode(packet.Data)
			if err != nil {
				logger.Errorf("failed to decode handshake request packet: %v", err)
				return
			}

			err = handshakePacket.Verify()
			if err != nil {
				logger.Errorf("invalid handshake request packet: %v", err)
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

			logger.Infof(
				"[handshake][request][connection: %s] request %s://%s:%d",
				handshakePacket.ConnectionID,
				Network,
				handshakePacket.DSTAddr,
				handshakePacket.DSTPort,
			)

			wsConn := connection.New(c, &connection.ConnectionOptions{
				Crypto: packet.Crypto,
				Secret: c.Secret,
				//
				ID: handshakePacket.ConnectionID,
			})
			wsConn.OnClose = func() {
				c.connections.Remove(wsConn.ID)
			}
			c.connections.Set(handshakePacket.ConnectionID, wsConn)

			if err := network.Connect(wsConn, &network.ConnectTarget{
				Type: Network,
				Host: handshakePacket.DSTAddr,
				Port: int(handshakePacket.DSTPort),
				ID:   handshakePacket.ConnectionID,
			}); err != nil {
				logger.Error("[handshake][request] failed to create connection to %s://%s:%d: %v", handshakePacket.Network, handshakePacket.DSTAddr, handshakePacket.DSTPort, err)
				return
			}

			logger.Infof(
				"[handshake][request][connection: %s] succeed to request %s://%s:%d",
				handshakePacket.ConnectionID,
				Network,
				handshakePacket.DSTAddr,
				handshakePacket.DSTPort,
			)
		case socksz.CommandHandshakeResponse:
			handshakePacket := &handshake.Response{}
			err := handshakePacket.Decode(packet.Data)
			if err != nil {
				logger.Error("failed to decode handshake request packet: %v", err)
				return
			}

			wsConn, err := c.connections.Get(handshakePacket.ConnectionID)
			logger.Infof(
				"[handshake][response][connection: %s] response (status: %d, message: %s)",
				handshakePacket.ConnectionID,
				handshakePacket.Status,
				handshakePacket.Message,
			)
			if handshakePacket.Status != STATUS_OK {
				logger.Error("[handshake][response] failed to handshake(connection_id: %s), status: %d, message: %s", handshakePacket.ConnectionID, handshakePacket.Status, handshakePacket.Message)
				// os.Exit(-1)
				wsConn.HandshakeCh <- false
				return
			}

			if err != nil {
				logger.Error("[handshake][response] failed to get connnection(id: %s)", handshakePacket.ConnectionID)
				// os.Exit(-1)
				wsConn.HandshakeCh <- false
				return
			}

			wsConn.HandshakeCh <- true
		case socksz.CommandForward:
			logger.Debugf(
				"[forward][incomming] start to decode",
			)
			forwardPacket := &forward.Forward{
				Crypto: packet.Crypto,
				Secret: c.Secret,
			}
			err := forwardPacket.Decode(packet.Data)
			if err != nil {
				logger.Error("failed to decode forward packet: %v", err)
				return
			}

			logger.Debugf(
				"[forward][incomming][connection: %s] start to check connection",
				forwardPacket.ConnectionID,
			)
			connection, err := c.connections.Get(forwardPacket.ConnectionID)
			if err != nil {
				logger.Errorf("[forward][incomming][connection: %s] failed to get connection", forwardPacket.ConnectionID)

				// maybe connection already gone, should force close connection to server
				forceCloseConn(forwardPacket.ConnectionID)
				return
			}

			logger.Debugf(
				"[forward][incomming][connection: %s] start to feed data to stream ...",
				forwardPacket.ConnectionID,
			)
			connection.Stream <- forwardPacket.Data
			// connection.Stream <- packet.Data
			logger.Debugf(
				"[forward][incomming][connection: %s] succeed to feed data to stream ...",
				forwardPacket.ConnectionID,
			)

		// case socksz.COMMAND_CONNECT:
		// 	id, err := connection.DecodeID(packet.Data)
		// 	if err != nil {
		// 		fmt.Println("[connect] failed to parse id:", err)
		// 		return
		// 	}

		// 	wsConn, err := wsConnsManager.GetOrCreate(id, func() *connection.WSConn {
		// 		wsConn := connection.New(id, c)

		// 		CreateTCPConnection(&CreateTCPConnectionConfig{
		// 			Host: "127.0.0.1",
		// 			Port: 22,
		// 			Conn: wsConn,
		// 			ID:   id,
		// 		})
		// 		if err != nil {
		// 			return nil
		// 		}

		// 		return wsConn
		// 	})
		// 	if err != nil {
		// 		fmt.Printf("connect error: %v\n", err)
		// 		return
		// 	}

		// 	wsConn.Stream <- packet.Data
		case socksz.CommandClose:
			logger.Debugf(
				"[close][incomming][cmd] close comming ...",
			)
			closePacket := &close.Close{}
			err := closePacket.Decode(packet.Data)
			if err != nil {
				logger.Error("failed to decode close packet: %v", err)
				return
			}

			logger.Debugf(
				"[close][incomming][connection: %s] start to remove connection",
				closePacket.ConnectionID,
			)
			err = c.connections.Remove(closePacket.ConnectionID)
			if err != nil {
				logger.Errorf("[close][incomming][connection: %s] failed to remove connection", closePacket.ConnectionID)
				return
			}
		default:
			logger.Warnf("[ignore] unknown command %d", packet.Cmd)
		}
	}

	return c.connect()
}

func (c *client) OnConnect(fn func()) {
	c.onConnect = fn
}

func (c *client) handshake(dataPacket *handshake.Request, connection *connection.WSConn) error {
	logger.Infof("[handshake] start to handshake ...")

	data, err := dataPacket.Encode()
	if err != nil {
		return fmt.Errorf("failed to encode handshake request: %v", err)
	}

	logger.Infof("[handshake] write packet ...")
	if err := c.writePacket(socksz.CommandHandshakeRequest, data); err != nil {
		return fmt.Errorf("failed to write packet: %v", err)
	}

	logger.Infof("[handshake] wait handshake response ...")
	if ok := <-connection.HandshakeCh; !ok {
		return fmt.Errorf("failed to wait handshake(connection_id: %s)", dataPacket.ConnectionID)
	}

	logger.Infof("[handshake] succeed to handshake, connected.")
	return nil
}

// func (c *client) waitHandshakeResponse(conectionID string) error {
// 	okCh := make(chan bool)
// 	errCh := make(chan error)

// 	c.connections.Set(conectionID, func(ok bool, err error) {
// 		if err != nil {
// 			errCh <- err
// 			return
// 		}

// 		okCh <- ok
// 	})

// 	for {
// 		select {
// 		case <-okCh:
// 			return nil
// 		case err := <-errCh:
// 			return err
// 		}
// 	}
// }

func (c *client) Bind(cfg *BindConfig) error {
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

	if err := network.Serve(&network.ServeConfig{
		Type: cfg.Network,
		Host: cfg.LocalHost,
		Port: cfg.LocalPort,
		OnConn: func() (net.Conn, error) {
			wsConn := connection.New(c, &connection.ConnectionOptions{
				Crypto: c.Crypto, // packet.Crypto
				Secret: c.Secret,
			})

			wsConn.OnClose = func() {
				logger.Infof("clean connection: %s", wsConn.ID)
				c.connections.Remove(wsConn.ID)
			}
			c.connections.Set(wsConn.ID, wsConn)

			if err := c.handshake(&handshake.Request{
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
			}, wsConn); err != nil {
				return nil, fmt.Errorf("failed to wait handshake(connection_id: %s): %v", wsConn.ID, err)
			}

			return wsConn, nil
		},
	}); err != nil {
		return fmt.Errorf("failed to create tcp server: %v", err)
	}

	return nil
}
