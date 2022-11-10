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
	"github.com/go-zoox/tcp-over-websocket/user"
	"github.com/gorilla/websocket"
)

type Client struct {
	Conn *websocket.Conn

	Protocol string `json:"protocol"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Path     string `json:"path"`

	// User
	User user.User

	OnConnect       func()
	OnDisconnect    func()
	OnMessage       func(typ int, msg []byte)
	OnTextMessage   func(msg []byte)
	OnBinaryMessage func(msg []byte)
	OnError         func(err error)
	OnPing          func()
	OnPong          func()
}

func (c *Client) authenticate() error {
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

func (c *Client) Connect() error {
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

// func (c *Client) WriteMessage(messageType int, data []byte) error {
// 	return c.Conn.WriteMessage(messageType, data)
// }

// func (c *Client) WriteTextMessage(data []byte) error {
// 	return c.Conn.WriteMessage(MessageTypeText, data)
// }

func (c *Client) WriteBinary(data []byte) error {
	return c.Conn.WriteMessage(MessageTypeBinary, data)
}

func (c *Client) WritePacket(command uint8, data []byte) error {
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

// func (c *Client) Emit(command uint8, data []byte) error {
// 	return c.WritePacket(command, data)
// }

func (c *Client) Listen() error {
	wsConnsManager := manager.New[*connection.WSConn]()

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

			logger.Info("[authenticate] succeed to auth as %s", c.User.GetClientID())
			return
		case protocol.COMMAND_HANDSHAKE:
			handshakePacket, err := handshake.DecodeRequest(packet.Data)
			if err != nil {
				logger.Error("failed to decode handshake request packet: %v", err)
				return
			}

			wsConn := connection.New(handshakePacket.ConnectionID, c)

			CreateTCPConnection(&CreateTCPConnectionConfig{
				// Network: handshakePacket.Network,
				Host: handshakePacket.DSTAddr,
				Port: int(handshakePacket.DSTPort),
				Conn: wsConn,
				ID:   handshakePacket.ConnectionID,
			})
			if err != nil {
				logger.Error("failed to create connection to %s://%s:%d: %v", handshakePacket.Network, handshakePacket.DSTAddr, handshakePacket.DSTPort, err)
				return
			}

			wsConnsManager.Set(handshakePacket.ConnectionID, wsConn)
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
