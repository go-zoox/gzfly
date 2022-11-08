package protocol

const (
	VERSION = 1
)

const (
	COMMAND_AUTHENTICATE = 0x01
	COMMAND_CONNECT      = 0x02
	COMMAND_BIND         = 0x03
	COMMAND_CLOSE        = 0xff
)
