package authenticate

import (
	"bytes"
	"fmt"
	"io"
)

// DATA Protocol:
//
// AUTHENTICATE DATA:
// request:  USER_CLIENT_ID | TIMESTAMP | NONCE | SIGNATURE
//             10           |    13     |   6   |  64 HMAC_SHA256
// response: STATUS | MESSAGE
//            1     |  -

const (
	LENGTH_USER_CLIENT_ID = 10
	LENGTH_TIMESTAMP      = 13
	LENGTH_NONCE          = 6
	LENGTH_SIGNATURE      = 64
)

type Authenticate struct {
	UserClientID string
	Timestamp    string
	Nonce        string
	Signature    string
}

func Encode(a *Authenticate) ([]byte, error) {
	buf := bytes.NewBuffer([]byte{})
	buf.WriteString(a.UserClientID)
	buf.WriteString(a.Timestamp)
	buf.WriteString(a.Nonce)
	buf.WriteString(a.Signature)
	return buf.Bytes(), nil
}

func Decode(raw []byte) (*Authenticate, error) {
	reader := bytes.NewReader(raw)

	// USER_CLIENT_ID
	buf := make([]byte, LENGTH_USER_CLIENT_ID)
	n, err := io.ReadFull(reader, buf)
	if n != LENGTH_USER_CLIENT_ID || err != nil {
		return nil, fmt.Errorf("failed to read user client id:  %s", err)
	}
	UserClientID := string(buf)

	// TIMESTAMP
	buf = make([]byte, LENGTH_TIMESTAMP)
	n, err = io.ReadFull(reader, buf)
	if n != LENGTH_TIMESTAMP || err != nil {
		return nil, fmt.Errorf("failed to read timestamp:  %s", err)
	}
	Timestamp := string(buf)

	// NONCE
	buf = make([]byte, LENGTH_NONCE)
	n, err = io.ReadFull(reader, buf)
	if n != LENGTH_NONCE || err != nil {
		return nil, fmt.Errorf("failed to read nonce:  %s", err)
	}
	Nonce := string(buf)

	// SIGNATURE
	buf = make([]byte, LENGTH_SIGNATURE)
	n, err = io.ReadFull(reader, buf)
	if n != LENGTH_SIGNATURE || err != nil {
		return nil, fmt.Errorf("failed to read signature:  %s", err)
	}
	Signature := string(buf)

	return &Authenticate{
		UserClientID,
		Timestamp,
		Nonce,
		Signature,
	}, nil
}
