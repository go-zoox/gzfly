package close

import (
	"bytes"
	"fmt"
	"io"
)

// DATA Protocol:
//
// CONNECTION CLOSE DATA:
// request:  CONNECTION_ID
//                 21

const (
	LENGTH_CONNECTION_ID = 21
)

type Close struct {
	ConnectionID string
}

func Encode(a *Close) ([]byte, error) {
	buf := bytes.NewBuffer([]byte{})
	buf.WriteString(a.ConnectionID)
	return buf.Bytes(), nil
}

func Decode(raw []byte) (*Close, error) {
	reader := bytes.NewReader(raw)

	// CONNECTION_ID
	buf := make([]byte, LENGTH_CONNECTION_ID)
	n, err := io.ReadFull(reader, buf)
	if n != LENGTH_CONNECTION_ID || err != nil {
		return nil, fmt.Errorf("failed to read connection id:  %s", err)
	}
	ConnectionID := string(buf)

	return &Close{
		ConnectionID,
	}, nil
}
