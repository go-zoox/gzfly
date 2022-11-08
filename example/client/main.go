package main

import (
	"fmt"
	"io"
	"net"
	"time"

	"github.com/go-zoox/logger"
	tow "github.com/go-zoox/tcp-over-websocket"
	"github.com/go-zoox/tcp-over-websocket/protocol"
	"github.com/go-zoox/uuid"
)

func main() {
	client := &tow.Client{
		// OnConnect: func(conn net.Conn, source string, target string) {
		// 	logger.Info("[%s] connect to %s", source, target)
		// },
		Protocol: "ws",
		Host:     "127.0.0.1",
		Port:     1080,
		Path:     "/",
	}

	wsConnsManager := WSConnManager{
		Client: client,
		cache:  make(map[string]*WSConn),
	}

	if err := client.Connect(); err != nil {
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

	if err := client.Emit(protocol.COMMAND_AUTHENTICATE, []byte("zero")); err != nil {
		logger.Errorf("failed to emit: %s", err)
	}

	client.OnBinaryMessage = func(raw []byte) {
		packet := protocol.New()
		if err := packet.Decode(raw); err != nil {
			fmt.Println("invalid message format")
			return
		}

		switch packet.GetCommand() {
		case protocol.COMMAND_CONNECT:
			data := packet.GetData()

			cursor := 0
			idLength := int(data[cursor])
			cursor += 1
			id := string(data[cursor : cursor+idLength])
			cursor += idLength
			wsconn, err := wsConnsManager.GetOrCreate(id)
			if err != nil {
				fmt.Errorf("connect error: %v", err)
				return
			}

			// bLength := int(msg.Payload[cursor])
			// cursor += 1
			fmt.Println("receive message:", len(data[cursor:]), len(data), len(raw))
			wsconn.Stream <- data[cursor:]
		}
	}

	// client.WritePacket("bind", []byte{})
	client.WritePacket(protocol.COMMAND_BIND, []byte{})

	if err := client.Listen(); err != nil {
		fmt.Println("listen error:", err)
	}
}

type WSConnManager struct {
	Client *tow.Client
	//
	cache map[string]*WSConn
}

func (m *WSConnManager) Get(id string) (*WSConn, error) {
	if conn, ok := m.cache[id]; ok {
		return conn, nil
	}

	return nil, fmt.Errorf("id %s not found", id)
}

func (m *WSConnManager) GetOrCreate(id string) (*WSConn, error) {
	wsconn, ok := m.cache[id]
	if !ok {
		conn, err := net.Dial("tcp", net.JoinHostPort("10.208.100.204", "22"))
		if err != nil {
			return nil, err
		}

		wsconn = &WSConn{
			ID:     id,
			Client: m.Client,
			Stream: make(chan []byte),
		}

		// buf := make([]byte, 256)
		// conn.Read(buf)

		// fmt.Println("fff:", buf)

		go Copy(wsconn, conn)
		go Copy(conn, wsconn)

		m.cache[wsconn.ID] = wsconn
	}

	return wsconn, nil
}

func Copy(dst io.Writer, src io.Reader) (written int64, err error) {
	return io.Copy(dst, src)

	// buf := make([]byte, 256)
	// return io.CopyBuffer(dst, src, buf)
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
	Client *tow.Client
	Stream chan []byte
}

func (wc *WSConn) Read(b []byte) (n int, err error) {
	n = copy(b, <-wc.Stream)
	fmt.Println("read:", n)
	return
}

func (wc *WSConn) Write(b []byte) (n int, err error) {
	data := []byte{}
	idBytes := []byte(wc.ID)
	idLength := len(idBytes)
	bLength := len(b)
	data = append(data, byte(idLength))
	data = append(data, idBytes...)
	// msg.Payload = append(msg.Payload, byte(bLength))
	data = append(data, b...)

	bytes, err := protocol.
		New().
		SetCommand(protocol.COMMAND_CONNECT).
		SetData(data).
		Encode()
	if err != nil {
		return 0, err
	}

	fmt.Printf("[%s] write message: %d\n", wc.ID, bLength)

	if err := wc.Client.WriteBinary(bytes); err != nil {
		return 0, err
	}

	return bLength, nil
}

func (wc *WSConn) Close() error {
	data := []byte{}
	idBytes := []byte(wc.ID)
	idLength := len(idBytes)
	data = append(data, byte(idLength))
	data = append(data, idBytes...)

	bytes, err := protocol.
		New().
		SetCommand(protocol.COMMAND_CLOSE).
		SetData(data).
		Encode()
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
