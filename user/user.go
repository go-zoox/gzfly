package user

import (
	"errors"
	"fmt"

	"github.com/go-zoox/crypto/hmac"
)

type User interface {
	// Server
	Authenticate(timestamp, nonce string, signature string) (bool, error)
	Pair(pairKey string) (bool, error)
	// Client
	Sign(timestamp, nonce string) (string, error)
}

type user struct {
	// Length 10
	ClientID string
	//
	ClientSecret string
	// Length 10
	PairKey string
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
