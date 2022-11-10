package protocol

const (
	VERSION = 1
)

const (
	// client <-> server
	COMMAND_AUTHENTICATE = 0x01
	// client <-> client
	COMMAND_HANDSHAKE_REQUEST  = 0x02
	COMMAND_HANDSHAKE_RESPONSE = 0x03
	//
	COMMAND_TRANSMISSION = 0x04
	// COMMAND_CONNECT      = 0x05
	// COMMAND_BIND         = 0x06
	COMMAND_CLOSE = 0xff
)

const (
	NETWORK_TCP = 0x01
	NETWORK_UDP = 0x02
)
