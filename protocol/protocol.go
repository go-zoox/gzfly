package protocol

import (
	"bytes"
	"fmt"
	"io"
)

const (
	VERSION = 1
)

type Protocol interface {
	Encode() ([]byte, error)
	Decode(raw []byte) error
	//
	Bytes() []byte
	//
	GetCommand() uint8
	SetCommand(command uint8) Protocol
	GetData() []byte
	SetData(data []byte) Protocol
}

const COMMAND_AUTHENTICATE = 0x01
const COMMAND_CONNECT = 0x02
const COMMAND_BIND = 0x03
const COMMAND_CLOSE = 0xff

//  VER | CMD | DATA
//   1  |  1  |  -
type protocol struct {
	version uint8
	command uint8
	data    []byte
	//
	bytes []byte
}

func New() Protocol {
	return &protocol{
		version: VERSION,
	}
}

func (m *protocol) Bytes() []byte {
	return m.bytes
}

func (m *protocol) GetCommand() uint8 {
	return m.command
}

func (m *protocol) SetCommand(command uint8) Protocol {
	m.command = command
	return m
}

func (m *protocol) GetData() []byte {
	return m.data
}

func (m *protocol) SetData(data []byte) Protocol {
	m.data = data
	return m
}

func (m *protocol) Encode() ([]byte, error) {
	if m.bytes != nil {
		return m.bytes, nil
	}

	bytes, err := MessageEncode(m.version, m.command, m.data)
	if err != nil {
		return nil, fmt.Errorf("failed to encode message: %v", err)
	}

	m.bytes = bytes
	return m.bytes, nil
}

func (m *protocol) Decode(raw []byte) error {
	version, typ, payload, err := MessageDecode(raw)
	if err != nil {
		return fmt.Errorf("failed to decode message: %v", err)
	}

	if version != VERSION {
		return fmt.Errorf("invalid version: %d, expect %d", version, VERSION)
	}

	m.version = version
	m.command = typ
	m.data = payload
	m.bytes = raw
	return nil
}

func MessageDecode(msg []byte) (version uint8, command uint8, data []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("failed to decode message")
		}
	}()

	var n int

	reader := bytes.NewReader(msg)

	buf := make([]byte, 1)
	n, err = io.ReadFull(reader, buf)
	if n != 1 || err != nil {
		err = fmt.Errorf("read version error:  %s", err)
		return
	}
	version = uint8(buf[0])

	n, err = io.ReadFull(reader, buf)
	if n != 1 {
		err = fmt.Errorf("read command error:  %s", err)
		return
	}
	command = uint8(buf[0])

	data, err = io.ReadAll(reader)
	if err != nil {
		return
	}

	return
}

func MessageEncode(version uint8, command uint8, data []byte) ([]byte, error) {
	buf := bytes.NewBuffer([]byte{})
	buf.WriteByte(version)
	buf.WriteByte(command)
	buf.Write(data)

	return buf.Bytes(), nil
}
