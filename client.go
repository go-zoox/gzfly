package tow

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"time"

	"github.com/go-zoox/logger"
	"github.com/go-zoox/tcp-over-websocket/connection"
	"github.com/go-zoox/tcp-over-websocket/manager"
	"github.com/go-zoox/tcp-over-websocket/protocol"
	"github.com/go-zoox/tcp-over-websocket/protocol/authenticate"
	"github.com/go-zoox/tcp-over-websocket/protocol/handshake"
	"github.com/go-zoox/tcp-over-websocket/protocol/transmission"
	"github.com/go-zoox/tcp-over-websocket/user"
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
	Conn *websocket.Conn

	Protocol string `json:"protocol"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Path     string `json:"path"`

	// User
	User user.User

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
	Protocol string `json:"protocol"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Path     string `json:"path"`

	// User
	User user.User
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

func New(cfg *ClientConfig) Client {
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
	}
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

	packet := &authenticate.AuthenticateRequest{
		UserClientID: UserClientID,
		Timestamp:    Timestamp,
		Nonce:        Nonce,
		Signature:    Signature,
	}
	bytes, err := authenticate.EncodeRequest(packet)
	if err != nil {
		return err
	}

	return c.WritePacket(protocol.COMMAND_AUTHENTICATE, bytes)
}

func (c *client) Connect() error {
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

// func (c *client) WriteMessage(messageType int, data []byte) error {
// 	return c.Conn.WriteMessage(messageType, data)
// }

// func (c *client) WriteTextMessage(data []byte) error {
// 	return c.Conn.WriteMessage(MessageTypeText, data)
// }

func (c *client) WriteBinary(data []byte) error {
	return c.Conn.WriteMessage(MessageTypeBinary, data)
}

func (c *client) WritePacket(command uint8, data []byte) error {
	packet := &protocol.Packet{
		Version: protocol.VERSION,
		Command: command,
		Data:    data,
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

	if err := c.Connect(); err != nil {
		return err
	}

	c.OnBinaryMessage = func(raw []byte) {
		packet, err := protocol.Decode(raw)
		if err != nil {
			fmt.Println("invalid message format")
			return
		}

		switch packet.Command {
		case protocol.COMMAND_AUTHENTICATE:
			response, err := authenticate.DecodeResponse(packet.Data)
			if err != nil {
				logger.Error("[authenticate] failed to decode authenticate response: %v", err)
				os.Exit(-1)
				return
			}

			if response.Status != STATUS_OK {
				logger.Error("[authenticate] failed to authenticate, status: %d, message: %s", response.Status, response.Message)
				os.Exit(-1)
				return
			}

			if c.onConnect != nil {
				go c.onConnect()
			}

			logger.Info("[authenticate] succeed to auth as %s", c.User.GetClientID())
			return
		case protocol.COMMAND_HANDSHAKE_REQUEST:
			handshakePacket, err := handshake.DecodeRequest(packet.Data)
			if err != nil {
				logger.Errorf("failed to decode handshake request packet: %v", err)
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

			logger.Infof(
				"[handshake][request][connection: %s] request %s://%s:%d",
				handshakePacket.ConnectionID,
				Network,
				handshakePacket.DSTAddr,
				handshakePacket.DSTPort,
			)

			wsConn := connection.New(handshakePacket.ConnectionID, c)
			c.connections.Set(handshakePacket.ConnectionID, wsConn)
			if err := CreateTCPConnection(&CreateTCPConnectionConfig{
				// Network: handshakePacket.Network,
				Host: handshakePacket.DSTAddr,
				Port: int(handshakePacket.DSTPort),
				Conn: wsConn,
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
		case protocol.COMMAND_HANDSHAKE_RESPONSE:
			handshakePacket, err := handshake.DecodeResponse(packet.Data)
			if err != nil {
				logger.Error("failed to decode handshake request packet: %v", err)
				return
			}

			logger.Infof(
				"[handshake][response][connection: %s] response (status: %d, message: %s)",
				handshakePacket.ConnectionID,
				handshakePacket.Status,
				handshakePacket.Message,
			)
			if handshakePacket.Status != STATUS_OK {
				logger.Error("[handshake][response] failed to handshake(connection_id: %s), status: %d, message: %s", handshakePacket.ConnectionID, handshakePacket.Status, handshakePacket.Message)
				os.Exit(-1)
				return
			}

			wsConn, err := c.connections.Get(handshakePacket.ConnectionID)
			if err != nil {
				logger.Error("[handshake][response] failed to get connnection(id: %s)", handshakePacket.ConnectionID)
				os.Exit(-1)
				return
			}

			wsConn.HandshakeCh <- true
		case protocol.COMMAND_TRANSMISSION:
			logger.Infof(
				"[transmission][receive] start to decode",
			)
			transmissionPacket, err := transmission.Decode(packet.Data)
			if err != nil {
				logger.Error("failed to decode transmission packet: %v", err)
				return
			}

			logger.Infof(
				"[transmission][receive][connection: %s] start to check connection",
				transmissionPacket.ConnectionID,
			)
			connection, err := c.connections.Get(transmissionPacket.ConnectionID)
			if err != nil {
				logger.Errorf("[transmission][receive][connection: %s] failed to get connection", transmissionPacket.ConnectionID)
				return
			}

			logger.Infof(
				"[transmission][receive][connection: %s] start to feed data to stream ...",
				transmissionPacket.ConnectionID,
			)
			connection.Stream <- transmissionPacket.Data
			// connection.Stream <- packet.Data
			logger.Infof(
				"[transmission][receive][connection: %s] succeed to feed data to stream ...",
				transmissionPacket.ConnectionID,
			)

			// case protocol.COMMAND_CONNECT:
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
		}
	}

	for {
		mt, message, err := c.Conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("read err: %s (type: %d)", err, mt)
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

func (c *client) OnConnect(fn func()) {
	c.onConnect = fn
}

func (c *client) handshake(dataPacket *handshake.HandshakeRequest, connection *connection.WSConn) error {
	logger.Infof("[handshake] start to handshake ...")

	data, err := handshake.EncodeRequest(dataPacket)
	if err != nil {
		return fmt.Errorf("failed to encode handshake request: %v", err)
	}

	logger.Infof("[handshake] write packet ...")
	if err := c.WritePacket(protocol.COMMAND_HANDSHAKE_REQUEST, data); err != nil {
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

	Network := protocol.NETWORK_TCP
	switch cfg.Network {
	case "tcp":
		Network = protocol.NETWORK_TCP
	case "udp":
		Network = protocol.NETWORK_UDP
	default:
		return fmt.Errorf("unknown network type: %s, only support tcp/udp", cfg.Network)
	}

	if err := CreateTCPServer(&CreateTCPServerConfig{
		Host: cfg.LocalHost,
		Port: cfg.LocalPort,
		OnConn: func() (net.Conn, error) {
			wsConn := connection.New(connection.GenerateID(), c)
			c.connections.Set(wsConn.ID, wsConn)

			if err := c.handshake(&handshake.HandshakeRequest{
				ConnectionID:       wsConn.ID,
				TargetUserClientID: cfg.TargetUserClientID,
				TargetUserPairKey:  cfg.TargetUserPairKey,
				// @TODO
				Network: uint8(Network),
				// @TODO
				ATyp:    handshake.ATYP_IPv4,
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

type CreateTCPConnectionConfig struct {
	Host string
	Port int
	//
	ID   string
	Conn net.Conn
}

func CreateTCPConnection(cfg *CreateTCPConnectionConfig) error {
	addr := net.JoinHostPort(cfg.Host, fmt.Sprintf("%d", cfg.Port))
	fmt.Printf("[%s][tcp] connect to: %s\n", cfg.ID, addr)

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}

	go Copy(cfg.Conn, conn)
	go Copy(conn, cfg.Conn)

	return nil
}

func CloseTCPConnection(conn net.Conn) error {
	return conn.Close()
}
