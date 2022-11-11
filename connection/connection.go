package connection

import (
	"net"
	"time"

	"github.com/go-zoox/core-utils/fmt"
	"github.com/go-zoox/logger"
	"github.com/go-zoox/tcp-over-websocket/protocol"
	"github.com/go-zoox/tcp-over-websocket/protocol/close"
	"github.com/go-zoox/tcp-over-websocket/protocol/transmission"
)

type WSClient interface {
	WriteBinary(bytes []byte) error
}

type wsClient struct {
	write func(bytes []byte) error
}

func NewWSClient(write func(bytes []byte) error) WSClient {
	return &wsClient{
		write: write,
	}
}

func (c *wsClient) WriteBinary(bytes []byte) error {
	return c.write(bytes)
}

type WSConn struct {
	ID     string
	Client WSClient
	// ch
	Stream      chan []byte
	HandshakeCh chan bool
	//
	OnClose func()
}

func New(id string, client WSClient) *WSConn {
	return &WSConn{
		ID:          id,
		Client:      client,
		Stream:      make(chan []byte),
		HandshakeCh: make(chan bool),
	}
}

func (wc *WSConn) Read(b []byte) (n int, err error) {
	logger.Debugf("[connection][read][connection: %s] start to read ...", wc.ID)

	// data := <-wc.Stream
	// n = copy(b, data[ID_LENGTH:])
	n = copy(b, <-wc.Stream)

	logger.Debugf("[connection][read][connection: %s] succeed to read: %d", wc.ID, n)
	return
}

func (wc *WSConn) Write(b []byte) (n int, err error) {
	// data, err := EncodeID(wc.ID)
	// if err != nil {
	// 	return 0, err
	// }
	// data = append(data, b...)

	logger.Debugf(
		"[transmission][outcomming][connection: %s] start to transmission",
		wc.ID,
	)

	logger.Debugf("[connection][write][connection: %s] start to encode", wc.ID)
	dataPacket := &transmission.Transmission{
		ConnectionID: wc.ID,
		Data:         b,
	}
	data, err := transmission.Encode(dataPacket)
	if err != nil {
		return 0, err
	}

	packet := &protocol.Packet{
		Version: protocol.VERSION,
		Command: protocol.COMMAND_TRANSMISSION,
		Data:    data,
	}
	bytes, err := packet.Encode()
	if err != nil {
		return 0, err
	}

	logger.Debugf("[connection][write][connection: %s] start to write", wc.ID)

	// fmt.Printf("[%s] write: %d\n", wc.ID, len(b))
	if err := wc.Client.WriteBinary(bytes); err != nil {
		return 0, err
	}

	logger.Debugf("[connection][write][connection: %s] succeed to write", wc.ID)

	logger.Debugf(
		"[transmission][outcomming][connection: %s] succeed to transmission",
		wc.ID,
	)
	return len(b), nil
}

func (wc *WSConn) Close() error {
	dataPacket := &close.Close{
		ConnectionID: wc.ID,
	}
	data, err := close.Encode(dataPacket)
	if err != nil {
		return err
	}

	packet := &protocol.Packet{
		Version: protocol.VERSION,
		Command: protocol.COMMAND_CLOSE,
		Data:    data,
	}
	bytes, err := packet.Encode()
	if err != nil {
		return err
	}

	fmt.Println("close:", wc.ID)
	if wc.OnClose != nil {
		wc.OnClose()
	}

	return wc.Client.WriteBinary(bytes)
}

func (wc *WSConn) LocalAddr() net.Addr {
	// return wc.Client.LocalAddr()
	return nil
}

func (wc *WSConn) RemoteAddr() net.Addr {
	// return wc.Client.RemoteAddr()
	return nil
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
