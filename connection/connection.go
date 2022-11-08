package connection

import (
	"fmt"
	"net"
	"time"

	"github.com/go-zoox/tcp-over-websocket/protocol"
)

type WSClient interface {
	WriteBinary(bytes []byte) error
}

type WSConn struct {
	ID     string
	Client WSClient
	Stream chan []byte
}

func New(id string, client WSClient) *WSConn {
	return &WSConn{
		ID:     id,
		Client: client,
		Stream: make(chan []byte),
	}
}

func (wc *WSConn) Read(b []byte) (n int, err error) {
	data := <-wc.Stream
	cursor := 1 + len(wc.ID)
	data = data[cursor:]

	n = copy(b, data)
	fmt.Printf("[%s] read: %d\n", wc.ID, n)
	return
}

func (wc *WSConn) Write(b []byte) (n int, err error) {
	data := EncodeID(wc.ID)
	data = append(data, b...)

	packet := protocol.New() // &Protocol{}
	packet.
		SetCommand(protocol.COMMAND_CONNECT).
		SetData(data)

	bytes, err := packet.Encode()
	if err != nil {
		return 0, err
	}

	fmt.Printf("[%s] write: %d\n", wc.ID, len(b))

	if err := wc.Client.WriteBinary(bytes); err != nil {
		return 0, err
	}

	return len(b), nil
}

func (wc *WSConn) Close() error {
	data := []byte{}
	idBytes := []byte(wc.ID)
	idLength := len(idBytes)
	data = append(data, byte(idLength))
	data = append(data, idBytes...)

	packet := protocol.New() //&Protocol{}
	packet.
		SetCommand(protocol.COMMAND_CLOSE).
		SetData(data)
	bytes, err := packet.Encode()
	if err != nil {
		return err
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
