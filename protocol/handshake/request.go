package handshake

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

// DATA Protocol:
//
// Handshake DATA:
// request:  CONNECTION_ID | TARGET_USER_CLIENT_ID | TARGET_USER_PAIR_KEY |  NETWORK   | ATYP                 | DST.ADDR 							 | DST.PORT
//					       21      |       10              |					10          | 1(tcp/udp) | 1(IPv4/IPv6/Domain)  |   4 or 16 or domain    |    2
// response: STATUS | MESSAGE
//            1     |  -

const (
	LENGTH_CONNECTION_ID         = 21
	LENGTH_TARGET_USER_CLIENT_ID = 10
	LENGTH_TARGET_USER_PAIR_KEY  = 10
	LENGTH_NETWORK               = 1
	LENGTH_ATYP                  = 1
	// LENGTH_DST_ADDR = 4
	LENGTH_DST_PORT = 2

	ATYP_IPv4   = 0x01
	ATYP_IPv6   = 0x04
	ATYP_DOMAIN = 0x03
)

type HandshakeRequest struct {
	ConnectionID       string
	TargetUserClientID string
	TargetUserPairKey  string
	Network            uint8
	ATyp               uint8
	DSTAddr            string
	DSTPort            uint16
}

func EncodeRequest(a *HandshakeRequest) ([]byte, error) {
	buf := bytes.NewBuffer([]byte{})
	buf.WriteString(a.ConnectionID)
	buf.WriteString(a.TargetUserClientID)
	buf.WriteString(a.TargetUserPairKey)
	buf.WriteByte(a.Network)
	buf.WriteByte(a.ATyp)

	// switch a.ATyp {
	// case ATYP_IPv4:
	// 	// 1.1.1.1
	// 	for _, p := range strings.Split(a.DSTAddr, ".") {
	// 		if v, err := strconv.Atoi(p); err != nil {
	// 			return nil, fmt.Errorf("invalid atyp IPv4 dst addr(%s)")
	// 		} else {
	// 			buf.WriteByte(byte(v))
	// 		}
	// 	}
	// }

	DSTAddrLength := len(a.DSTAddr)
	buf.WriteByte(byte(DSTAddrLength))
	buf.WriteString(a.DSTAddr)
	portBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(portBytes, a.DSTPort)
	buf.Write(portBytes)

	return buf.Bytes(), nil
}

func DecodeRequest(raw []byte) (*HandshakeRequest, error) {
	reader := bytes.NewReader(raw)

	// CONNECTION_ID
	buf := make([]byte, LENGTH_CONNECTION_ID)
	n, err := io.ReadFull(reader, buf)
	if n != LENGTH_CONNECTION_ID || err != nil {
		return nil, fmt.Errorf("failed to read connection id:  %s", err)
	}
	ConnectionID := string(buf)

	// TARGET_USER_CLIENT_ID
	buf = make([]byte, LENGTH_TARGET_USER_CLIENT_ID)
	n, err = io.ReadFull(reader, buf)
	if n != LENGTH_TARGET_USER_CLIENT_ID || err != nil {
		return nil, fmt.Errorf("failed to read target user client id:  %s", err)
	}
	TargetUserClientID := string(buf)

	// TARGET_USER_PAIR_KEY
	buf = make([]byte, LENGTH_TARGET_USER_PAIR_KEY)
	n, err = io.ReadFull(reader, buf)
	if n != LENGTH_TARGET_USER_PAIR_KEY || err != nil {
		return nil, fmt.Errorf("failed to read target user pair key:  %s", err)
	}
	TargetUserPairKey := string(buf)

	// NETWORK
	buf = make([]byte, LENGTH_NETWORK)
	n, err = io.ReadFull(reader, buf)
	if n != LENGTH_NETWORK || err != nil {
		return nil, fmt.Errorf("failed to read signature:  %s", err)
	}
	Network := uint8(buf[0])

	// ATYP
	buf = make([]byte, LENGTH_ATYP)
	n, err = io.ReadFull(reader, buf)
	if n != LENGTH_ATYP || err != nil {
		return nil, fmt.Errorf("failed to read atyp:  %s", err)
	}
	ATyp := uint8(buf[0])

	// DSTAddr
	buf = make([]byte, 1)
	n, err = io.ReadFull(reader, buf)
	if n != 1 || err != nil {
		return nil, fmt.Errorf("failed to read dst addr length:  %s", err)
	}
	dstAddrLength := int(buf[0])
	buf = make([]byte, dstAddrLength)
	n, err = io.ReadFull(reader, buf)
	if n != dstAddrLength || err != nil {
		return nil, fmt.Errorf("failed to read dst addr:  %s", err)
	}
	DSTAddr := string(buf)

	// DSTAddrPort
	buf = make([]byte, LENGTH_DST_PORT)
	n, err = io.ReadFull(reader, buf)
	if n != LENGTH_DST_PORT || err != nil {
		return nil, fmt.Errorf("failed to read atyp:  %s", err)
	}
	DSTPort := binary.BigEndian.Uint16(buf[:2])

	return &HandshakeRequest{
		ConnectionID,
		TargetUserClientID,
		TargetUserPairKey,
		Network,
		ATyp,
		DSTAddr,
		DSTPort,
	}, nil
}
