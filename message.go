package tow

import "fmt"

type Message struct {
	Type    []byte `json:"type"`
	Payload []byte `json:"payload"`

	bytes []byte
}

func (m *Message) Encode() ([]byte, error) {
	if m.bytes != nil {
		return m.bytes, nil
	}

	bytes, err := MessageEncode(m.Type, m.Payload)
	if err != nil {
		return nil, fmt.Errorf("failed to encode message: %v", err)
	}

	m.bytes = bytes
	return m.bytes, nil
}

func (m *Message) Decode(raw []byte) error {
	typ, payload, err := MessageDecode(raw)
	if err != nil {
		return fmt.Errorf("failed to decode message: %v", err)
	}

	m.Type = typ
	m.Payload = payload
	m.bytes = raw
	return nil
}

func MessageDecode(msg []byte) (typ []byte, payload []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("failed to decode message")
		}
	}()

	eventTypeLength := msg[0]
	eventType := msg[1 : eventTypeLength+1]
	evenPayloadLength := msg[eventTypeLength+1]
	evenPayloadStart := eventTypeLength + 2
	evenPayload := msg[evenPayloadStart : evenPayloadStart+evenPayloadLength]

	return eventType, evenPayload, nil
}

func MessageEncode(typ []byte, payload []byte) ([]byte, error) {
	b := []byte{byte(len(typ))}
	b = append(b, typ...)
	b = append(b, byte(len(payload)))
	b = append(b, payload...)

	return b, nil
}
