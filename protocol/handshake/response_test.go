package handshake

import "testing"

func TestResponseEncodeDecode(t *testing.T) {
	packet := &HandshakeResponse{
		Status:  0x01,
		Message: "hello world",
	}

	encoded, err := EncodeResponse(packet)
	if err != nil {
		t.Fatalf("failed to encode %s", err)
	}

	decoded, err := DecodeResponse(encoded)
	if err != nil {
		t.Fatalf("failed to decode %s", err)
	}

	if decoded.Status != packet.Status {
		t.Fatalf("Status not match, expect %d, but got %d", packet.Status, decoded.Status)
	}

	if decoded.Message != packet.Message {
		t.Fatalf("Message not match, expect %s, but got %s", packet.Message, decoded.Message)
	}
}
