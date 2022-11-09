package protocol

type Packet struct {
	Version     uint8
	Command     uint8
	Cryto       uint8
	Compression uint8
	Data        []byte
}

func (p *Packet) Encode() ([]byte, error) {
	return Encode(p)
}

func (m *Packet) Decode(raw []byte) (packet *Packet, err error) {
	return Decode(raw)
}
