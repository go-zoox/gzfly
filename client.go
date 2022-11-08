package tow

import (
	"fmt"
	"net"
	"net/url"

	"github.com/go-zoox/tcp-over-websocket/connection"
	"github.com/go-zoox/tcp-over-websocket/manager"
	"github.com/go-zoox/tcp-over-websocket/protocol"
	"github.com/gorilla/websocket"
)

type Client struct {
	Conn *websocket.Conn

	Protocol string `json:"protocol"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Path     string `json:"path"`

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
	return nil
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

func (c *Client) WriteMessage(messageType int, data []byte) error {
	return c.Conn.WriteMessage(messageType, data)
}

func (c *Client) WriteTextMessage(data []byte) error {
	return c.Conn.WriteMessage(MessageTypeText, data)
}

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

func (c *Client) Emit(command uint8, data []byte) error {
	return c.WritePacket(command, data)
}

func (client *Client) Listen() error {
	wsConnsManager := manager.New[*connection.WSConn]()

	client.OnBinaryMessage = func(raw []byte) {
		packet, err := protocol.Decode(raw)
		if err != nil {
			fmt.Println("invalid message format")
			return
		}

		switch packet.Command {
		case protocol.COMMAND_CONNECT:
			data := packet.Data
			id, err := connection.DecodeID(data)
			if err != nil {
				fmt.Print("[connect] failed to parse id:", err)
				return
			}

			wsConn, err := wsConnsManager.GetOrCreate(id, func() *connection.WSConn {
				wsConn := connection.New(id, client)

				CreateTCPConnection(&CreateTCPConnectionConfig{
					Host: "127.0.0.1",
					Port: 22,
					Conn: wsConn,
				})
				if err != nil {
					return nil
				}

				return wsConn
			})

			if err != nil {
				fmt.Printf("connect error: %v\n", err)
				return
			}

			wsConn.Stream <- data
		}
	}

	for {
		mt, message, err := client.Conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("read err: %s (type: %d)", err, mt)
		}

		switch mt {
		case websocket.TextMessage:
			if client.OnTextMessage != nil {
				client.OnTextMessage(message)
			}
		case websocket.BinaryMessage:
			if client.OnBinaryMessage != nil {
				client.OnBinaryMessage(message)
			}
		case websocket.CloseMessage:
			// @TODO
		case websocket.PingMessage:
			if client.OnPing != nil {
				client.OnPing()
			}
		case websocket.PongMessage:
			if client.OnPong != nil {
				client.OnPong()
			}
		default:
			fmt.Printf("unknown message type: %d\n", mt)
		}

		if client.OnMessage != nil {
			client.OnMessage(mt, message)
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
