package handshake

import (
	"bytes"
	"fmt"
	"io"
)

// DATA Protocol:
//
// AUTHENTICATE DATA:
// response: STATUS | MESSAGE
//            1     |  -

const (
	LENGTH_STATUS = 1
)

type HandshakeResponse struct {
	Status  uint8
	Message string
}

func EncodeResponse(a *HandshakeResponse) ([]byte, error) {
	buf := bytes.NewBuffer([]byte{})
	buf.WriteByte(a.Status)
	buf.WriteString(a.Message)
	return buf.Bytes(), nil
}

func DecodeResponse(raw []byte) (*HandshakeResponse, error) {
	reader := bytes.NewReader(raw)

	// STATUS
	buf := make([]byte, LENGTH_STATUS)
	n, err := io.ReadFull(reader, buf)
	if n != LENGTH_STATUS || err != nil {
		return nil, fmt.Errorf("failed to read status:  %s", err)
	}
	Status := uint8(buf[0])

	// Message
	buf, err = io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read message:  %s", err)
	}
	Message := string(buf)

	return &HandshakeResponse{
		Status,
		Message,
	}, nil
}
