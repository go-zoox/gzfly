package main

import (
	"fmt"
	"io"
	"net"
	"time"

	"github.com/go-zoox/logger"
	tow "github.com/go-zoox/tcp-over-websocket"
	"github.com/go-zoox/uuid"
	"github.com/go-zoox/zoox"
)

func main() {
	c := &tow.Client{
		// OnConnect: func(conn net.Conn, source string, target string) {
		// 	logger.Info("[%s] connect to %s", source, target)
		// },
		Host: "127.0.0.1",
		Port: 1080,
		Path: "/",
	}

	if err := c.Connect(); err != nil {
		logger.Fatal("failed to connect server: %s", err)
		return
	}

	// if err := c.WriteMessage(tow.MessageTypeText, []byte("hi")); err != nil {
	// 	logger.Errorf("failed to write message: %s", err)
	// }

	// if err := c.WriteTextMessage([]byte("text message")); err != nil {
	// 	logger.Errorf("failed to write message: %s", err)
	// }

	// if err := c.WriteBinaryMessage([]byte("binary message")); err != nil {
	// 	logger.Errorf("failed to write message: %s", err)
	// }

	if err := c.Emit("authentication", []byte("zero")); err != nil {
		logger.Errorf("failed to emit: %s", err)
	}

	for {
		select {}
	}
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
	msg := &tow.Message{}
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
	msg := &tow.Message{}
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
