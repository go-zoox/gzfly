package user

import (
	"errors"
	"fmt"

	"github.com/go-zoox/crypto/hmac"
	"github.com/go-zoox/tcp-over-websocket/protocol"
	"github.com/go-zoox/zoox"
)

type User interface {
	// Server
	Authenticate(timestamp, nonce string, signature string) (bool, error)
	Pair(pairKey string) (bool, error)
	// Client
	Sign(timestamp, nonce string) (string, error)
	//
	GetClientID() string
	IsOnline() bool
	WritePacket(packet *protocol.Packet) error
	SetOnline(client *zoox.WebSocketClient) error
	SetOffline(client *zoox.WebSocketClient) error
	//
	WriteBytes(b []byte) error
}

type user struct {
	// Length 10
	ClientID string
	//
	ClientSecret string
	// Length 10
	PairKey string
	//
	isOnline bool
	WSClient *zoox.WebSocketClient
}

func New(clientID, clientSecret, pairKey string) User {
	return &user{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		PairKey:      pairKey,
	}
}

func (u *user) Authenticate(timestamp, nonce, signature string) (ok bool, err error) {
	return u.Verify(timestamp, nonce, signature)
}

func (u *user) Pair(pairKey string) (bool, error) {
	return u.PairKey == pairKey, nil
}

func (u *user) Sign(timestamp, nonce string) (signature string, err error) {
	defer func() {
		if errx := recover(); errx != nil {
			switch v := errx.(type) {
			case error:
				err = v
			case string:
				err = errors.New(v)
			default:
				err = fmt.Errorf("%v", v)
			}
		}
	}()

	return hmac.Sha256(fmt.Sprintf("%s_%s_%s", u.ClientID, timestamp, nonce), u.ClientSecret, "hex"), nil
}

func (u *user) Verify(timestamp, nonce, signature string) (bool, error) {
	if ns, err := u.Sign(timestamp, nonce); err != nil {
		return false, err
	} else {
		return ns == signature, nil
	}
}

//
func (u *user) GetClientID() string {
	return u.ClientID
}

//
func (u *user) IsOnline() bool {
	return u.isOnline
}

func (u *user) WritePacket(packet *protocol.Packet) error {
	if !u.IsOnline() {
		return errors.New("user is not online")
	}

	if bytes, err := protocol.Encode(packet); err != nil {
		return fmt.Errorf("failed to encode packet %v", err)
	} else {
		return u.WSClient.WriteBinary(bytes)
	}
}

func (u *user) WriteBytes(b []byte) error {
	if !u.IsOnline() {
		return errors.New("user is not online")
	}

	return u.WSClient.WriteBinary(b)
}

func (u *user) SetOnline(client *zoox.WebSocketClient) error {
	u.WSClient = client
	u.isOnline = true
	return nil
}

func (u *user) SetOffline(client *zoox.WebSocketClient) error {
	u.WSClient = nil
	u.isOnline = false
	return nil
}
