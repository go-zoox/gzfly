package tow

import (
	"io"

	"github.com/gorilla/websocket"
)

const (
	MessageTypeText   = websocket.TextMessage
	MessageTypeBinary = websocket.BinaryMessage
	MessageTypeClose  = websocket.CloseMessage
	MessageTypePing   = websocket.PingMessage
	MessageTypePong   = websocket.PongMessage
	//
	STATUS_OK                     = 0x01
	STATUS_INVALID_USER_CLIENT_ID = 0x02
	STATUS_INVALID_SIGNATURE      = 0x03
)

func Copy(dst io.Writer, src io.Reader) (written int64, err error) {
	return io.Copy(dst, src)

	// buf := make([]byte, 256)
	// return io.CopyBuffer(dst, src, buf)
}
