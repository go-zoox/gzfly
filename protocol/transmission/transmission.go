package transmission

import (
	"bytes"
	"fmt"
	"io"
)

// DATA Protocol:
//
// TRANSMISIION DATA:
// request:  CONNECTION_ID | DATA
//					       21      |  -

const (
	LENGTH_CONNECTION_ID = 21
)

type Transmission struct {
	ConnectionID string
	Data         []byte
}

func Encode(a *Transmission) ([]byte, error) {
	buf := bytes.NewBuffer([]byte{})
	buf.WriteString(a.ConnectionID)
	buf.Write(a.Data)
	return buf.Bytes(), nil
}

func Decode(raw []byte) (*Transmission, error) {
	reader := bytes.NewReader(raw)

	// CONNECTION_ID
	buf := make([]byte, LENGTH_CONNECTION_ID)
	n, err := io.ReadFull(reader, buf)
	if n != LENGTH_CONNECTION_ID || err != nil {
		return nil, fmt.Errorf("failed to read connection id:  %s", err)
	}
	ConnectionID := string(buf)

	Data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read data:  %s", err)
	}

	return &Transmission{
		ConnectionID,
		Data,
	}, nil
}
