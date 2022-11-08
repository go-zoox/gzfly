package protocol

import (
	"bytes"
	"fmt"
	"io"
)

func Decode(raw []byte) (packet *Packet, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("failed to decode message")
		}
	}()

	var n int
	reader := bytes.NewReader(raw)

	buf := make([]byte, 1)
	n, err = io.ReadFull(reader, buf)
	if n != 1 || err != nil {
		err = fmt.Errorf("read version error:  %s", err)
		return
	}
	version := uint8(buf[0])

	n, err = io.ReadFull(reader, buf)
	if n != 1 {
		err = fmt.Errorf("read command error:  %s", err)
		return
	}
	command := uint8(buf[0])

	data, err := io.ReadAll(reader)
	if err != nil {
		return
	}

	packet = &Packet{
		Version: version,
		Command: command,
		Data:    data,
	}

	return
}
