package connection

import (
	"bytes"
	"fmt"
	"io"

	nanoid "github.com/matoous/go-nanoid/v2"
)

// UUID
// const ID_LENGTH = 36
// func GenerateID() string {
// 	return uuid.V4()
// }

// NANO ID
const ID_LENGTH = 21

func GenerateID() string {
	id, _ := nanoid.New()
	return id
}

func DecodeID(data []byte) (s string, err error) {
	defer func() {
		if errx := recover(); err != nil {
			err = fmt.Errorf("%v", errx)
		}
	}()

	var n int
	reader := bytes.NewReader(data)
	buf := make([]byte, ID_LENGTH)
	n, err = io.ReadFull(reader, buf)
	if err != nil {
		err = fmt.Errorf("read id error:  %s", err)
		return
	} else if n != ID_LENGTH {
		err = fmt.Errorf("invalid id length(%d), expect %d", n, ID_LENGTH)
		return
	}

	return string(buf), nil
}

func EncodeID(id string) ([]byte, error) {
	if len(id) != ID_LENGTH {
		return nil, fmt.Errorf("invalid id length(%d), expect %d", len(id), ID_LENGTH)
	}

	return []byte(id), nil
}
