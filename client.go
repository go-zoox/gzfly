package tow

import (
	"fmt"
	"net"
	"net/url"

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
	packet := protocol.New()
	packet.
		SetCommand(command).
		SetData(data)

	bytes, err := packet.Encode()
	if err != nil {
		return fmt.Errorf("invalid message: %s", err)
	}

	return c.WriteBinary(bytes)
}

func (c *Client) Emit(command uint8, data []byte) error {
	packet := protocol.New()
	packet.
		SetCommand(command).
		SetData(data)
	b, err := packet.Encode()
	if err != nil {
		return err
	}

	return c.WriteBinary(b)
}

func (client *Client) Listen() error {
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
