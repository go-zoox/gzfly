package tow

import "fmt"

type Message struct {
	Version string `json:"version"`
	Type    string `json:"type"`
	Payload []byte `json:"payload"`

	bytes []byte
}

func (m *Message) Encode() ([]byte, error) {
	if m.bytes != nil {
		return m.bytes, nil
	}

	bytes, err := MessageEncode([]byte(m.Type), m.Payload)
	if err != nil {
		return nil, fmt.Errorf("failed to encode message: %v", err)
	}

	m.bytes = bytes
	return m.bytes, nil
}

func (m *Message) Decode(raw []byte) error {
	version, typ, payload, err := MessageDecode(raw)
	if err != nil {
		return fmt.Errorf("failed to decode message: %v", err)
	}

	m.Version = version
	m.Type = typ
	m.Payload = payload
	m.bytes = raw
	return nil
}

func MessageDecode(msg []byte) (version string, typ string, payload []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("failed to decode message")
		}
	}()

	cursor := 0
	versionLength := int(msg[cursor])
	cursor = cursor + 1

	verx := msg[cursor : cursor+versionLength]
	cursor = cursor + versionLength

	eventTypeLength := int(msg[cursor])
	cursor = cursor + 1

	typx := msg[cursor : cursor+eventTypeLength]
	cursor = cursor + eventTypeLength

	evenPayloadLength := int(msg[cursor])
	cursor = cursor + 1

	payload = msg[cursor : cursor+evenPayloadLength]

	return string(verx), string(typx), payload, nil
}

func MessageEncode(typ []byte, payload []byte) ([]byte, error) {
	b := []byte{}
	b = append(b, byte(len(Version)))
	b = append(b, Version...)
	b = append(b, byte(len(typ)))
	b = append(b, typ...)
	b = append(b, byte(len(payload)))
	b = append(b, payload...)

	return b, nil
}
