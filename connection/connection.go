package connection

import (
	"io"
	"net"
	"time"

	"github.com/go-zoox/logger"
	"github.com/go-zoox/packet/socksz"
	"github.com/go-zoox/packet/socksz/base"
	"github.com/go-zoox/packet/socksz/close"
	"github.com/go-zoox/packet/socksz/forward"
	"github.com/go-zoox/zoox/components/context/websocket"
)

type WSClient struct {
	*websocket.WebSocketClient
}

func NewWSClient(client *websocket.WebSocketClient) *WSClient {
	return &WSClient{
		WebSocketClient: client,
	}
}

type WSConn struct {
	ID     string
	Client *WSClient
	// ch
	Stream      chan []byte
	HandshakeCh chan bool
	//
	OnClose func()
	//
	Crypto uint8
	Secret string
	//
	isClosed bool
}

type ConnectionOptions struct {
	ID string
	//
	Crypto uint8
	Secret string
}

func New(client *WSClient, opts ...*ConnectionOptions) *WSConn {
	id := ""
	crypto := uint8(0x00)
	secret := ""
	if len(opts) > 0 && opts[0] != nil {
		if opts[0].ID != "" {
			id = opts[0].ID
		}

		if opts[0].Crypto != 0 {
			crypto = opts[0].Crypto
		}

		if opts[0].Secret != "" {
			secret = opts[0].Secret
		}
	}

	if id == "" {
		id = socksz.GenerateID()
	}

	return &WSConn{
		ID:          id,
		Client:      client,
		Stream:      make(chan []byte),
		HandshakeCh: make(chan bool),
		//
		Crypto: crypto,
		Secret: secret,
	}
}

func (wc *WSConn) Read(b []byte) (n int, err error) {
	if wc.isClosed {
		return 0, io.EOF
	}

	logger.Debugf("[connection][read][connection: %s] start to read ...", wc.ID)

	// data := <-wc.Stream
	// n = copy(b, data[ID_LENGTH:])
	n = copy(b, <-wc.Stream)

	logger.Debugf("[connection][read][connection: %s] succeed to read: %d", wc.ID, n)
	return
}

func (wc *WSConn) Write(b []byte) (n int, err error) {
	if wc.isClosed {
		return 0, io.EOF
	}

	// data, err := EncodeID(wc.ID)
	// if err != nil {
	// 	return 0, err
	// }
	// data = append(data, b...)

	logger.Debugf(
		"[forward][outgoing][connection: %s] start to forward",
		wc.ID,
	)

	logger.Debugf("[connection][write][connection: %s] start to encode", wc.ID)
	dataPacket := &forward.Forward{
		Crypto: wc.Crypto,
		Secret: wc.Secret,
		//
		ConnectionID: wc.ID,
		Data:         b,
	}
	data, err := dataPacket.Encode()
	if err != nil {
		return 0, err
	}

	packet := &base.Base{
		Ver:  socksz.VER,
		Cmd:  socksz.CommandForward,
		Data: data,
		//
		Crypto: wc.Crypto,
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
		"[forward][outgoing][connection: %s] succeed to forward",
		wc.ID,
	)
	return len(b), nil
}

func (wc *WSConn) Close() error {
	// closed
	if wc.isClosed {
		return nil
	}

	dataPacket := &close.Close{
		ConnectionID: wc.ID,
	}
	data, err := dataPacket.Encode()
	if err != nil {
		return err
	}

	packet := &base.Base{
		Ver:  socksz.VER,
		Cmd:  socksz.CommandClose,
		Data: data,
		//
		Crypto: wc.Crypto,
	}
	bytes, err := packet.Encode()
	if err != nil {
		return err
	}

	if wc.OnClose != nil {
		wc.OnClose()
	}

	wc.isClosed = true

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
