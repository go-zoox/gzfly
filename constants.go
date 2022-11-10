package tow

import (
	"github.com/go-zoox/logger"
	"github.com/gorilla/websocket"
)

func init() {
	// logger.SetLevel(logger.LevelError)
	// logger.SetLevel(logger.LevelInfo)
	logger.SetLevel(logger.LevelDebug)
}

const (
	MessageTypeText   = websocket.TextMessage
	MessageTypeBinary = websocket.BinaryMessage
	MessageTypeClose  = websocket.CloseMessage
	MessageTypePing   = websocket.PingMessage
	MessageTypePong   = websocket.PongMessage
	//
	STATUS_OK                     = 0x00
	STATUS_INVALID_USER_CLIENT_ID = 0x01
	STATUS_INVALID_SIGNATURE      = 0x02
	STATUS_USER_NOT_ONLINE        = 0x03
	STATUS_FAILED_TO_PAIR         = 0x04
	STATUS_FAILED_TO_HANDSHAKE    = 0x05
)
