package tow

import (
	"fmt"
	"net"
	"net/url"

	"github.com/gorilla/websocket"
)

type Client struct {
	conn *websocket.Conn

	Host string `json:"host"`
	Port int    `json:"port"`
	Path string `json:"path"`
}

func (c *Client) authenticate() error {
	return nil
}

func (c *Client) Connect() error {
	if c.conn == nil {
		u := url.URL{Scheme: "ws", Host: net.JoinHostPort(c.Host, fmt.Sprintf("%d", c.Port)), Path: c.Path}
		conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
		if err != nil {
			return err
		}

		c.conn = conn
	}

	// authentication
	if err := c.authenticate(); err != nil {
		return fmt.Errorf("failed to authenticate: %v", err)
	}

	return nil
}

func (c *Client) WriteMessage(messageType int, data []byte) error {
	return c.conn.WriteMessage(messageType, data)
}

func (c *Client) WriteTextMessage(data []byte) error {
	return c.conn.WriteMessage(MessageTypeText, data)
}

func (c *Client) WriteBinaryMessage(data []byte) error {
	return c.conn.WriteMessage(MessageTypeBinary, data)
}

func (c *Client) Emit(typ []byte, payload []byte) error {
	msg := &Message{Type: typ, Payload: payload}
	b, err := msg.Encode()
	if err != nil {
		return err
	}

	return c.WriteBinaryMessage(b)
}
