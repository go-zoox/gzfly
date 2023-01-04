package user

import (
	"errors"
	"fmt"
	"sync"

	"github.com/go-zoox/crypto/hmac"
	"github.com/go-zoox/gzfly/connection"
	"github.com/go-zoox/logger"
	"github.com/go-zoox/packet/socksz/base"
)

// type User interface {
// 	// Server
// 	Authenticate(timestamp, nonce string, signature string) (bool, error)
// 	Pair(connectionID, userClientID, pairSignature string) (bool, error)
// 	// Client
// 	Sign(timestamp, nonce string) (string, error)
// 	//
// 	GetClientID() string
// 	GetWSClient() connection.WSClient
// 	//
// 	IsOnline() bool
// 	WritePacket(packet *base.Base) error
// 	// SetOnline(client *zoox.WebSocketClient) error
// 	// SetOffline(client *zoox.WebSocketClient) error
// 	SetOnline(client connection.WSClient) error
// 	SetOffline(client connection.WSClient) error
// 	//
// 	WriteBytes(b []byte) error
// }

type User struct {
	sync.RWMutex

	// Length 10
	ClientID string
	//
	ClientSecret string
	// Length 10
	PairKey string
	//
	// isOnline bool
	WSClient *connection.WSClient
}

type UserClient struct {
	// Length 10
	ClientID string `config:"client_id"`
	//
	ClientSecret string `config:"client_secret"`
	// Length 10
	PairKey string `config:"pair_key"`
}

func New(clientID, clientSecret, pairKey string) *User {
	return &User{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		PairKey:      pairKey,
	}
}

func (u *User) Authenticate(timestamp, nonce, signature string) (ok bool, err error) {
	return u.Verify(timestamp, nonce, signature)
}

func (u *User) Pair(connectionID, userClientID, pairSignature string) (bool, error) {
	return hmac.Sha256(fmt.Sprintf("%s_%s", connectionID, userClientID), u.PairKey) == pairSignature, nil
}

func (u *User) Sign(timestamp, nonce string) (signature string, err error) {
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

func (u *User) Verify(timestamp, nonce, signature string) (bool, error) {
	if ns, err := u.Sign(timestamp, nonce); err != nil {
		return false, err
	} else {
		return ns == signature, nil
	}
}

func (u *User) GetClientID() string {
	return u.ClientID
}

func (u *User) GetWSClient() *connection.WSClient {
	return u.WSClient
}

func (u *User) GetPairKey() string {
	return u.PairKey
}

func (u *User) IsOnline() bool {
	if u.WSClient == nil {
		return false
	}

	return u.WSClient.IsAlive()
}

func (u *User) WritePacket(packet *base.Base) error {
	if !u.IsOnline() {
		return errors.New("user is not online")
	}

	if bytes, err := packet.Encode(); err != nil {
		return fmt.Errorf("failed to encode packet %v", err)
	} else {
		return u.WSClient.WriteBinary(bytes)
	}
}

func (u *User) WriteBytes(b []byte) error {
	u.Lock()
	defer u.Unlock()

	if !u.IsOnline() {
		return errors.New("user is not online")
	}

	return u.WSClient.WriteBinary(b)
}

func (u *User) SetOnline(client *connection.WSClient) error {
	if u.WSClient != nil {
		if u.WSClient.IsAlive() {
			return fmt.Errorf("not allow login, because user is online before in other place")
		}
	}

	u.WSClient = client
	return nil
}

func (u *User) SetOffline() error {
	if u.IsOnline() {
		if err := u.WSClient.Disconnect(); err != nil {
			logger.Warnf("failed disconnect ws client")
		}
	}

	u.WSClient = nil
	return nil
}
