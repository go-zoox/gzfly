package protocol

import "bytes"

func Encode(packet *Packet) ([]byte, error) {
	buf := bytes.NewBuffer([]byte{})
	buf.WriteByte(packet.Version)
	buf.WriteByte(packet.Command)
	buf.Write(packet.Data)

	return buf.Bytes(), nil
}
