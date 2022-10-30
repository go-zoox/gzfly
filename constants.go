package tow

import "github.com/gorilla/websocket"

const (
	MessageTypeText   = websocket.TextMessage
	MessageTypeBinary = websocket.BinaryMessage
	MessageTypeClose  = websocket.CloseMessage
	MessageTypePing   = websocket.PingMessage
	MessageTypePong   = websocket.PongMessage
)
