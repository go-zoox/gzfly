package protocol

import "testing"

func TestPacketEncodeDecode(t *testing.T) {
	packet := &Packet{
		Version: VERSION,
		Command: COMMAND_AUTHENTICATE,
		Data:    []byte("hello zero"),
	}

	encoded, err := Encode(packet)
	if err != nil {
		t.Fatalf("failed to encode %s", err)
	}

	decoded, err := Decode(encoded)
	if err != nil {
		t.Fatalf("failed to decode %s", err)
	}

	if decoded.Version != packet.Version {
		t.Fatalf("Version not match, expect %d, but got %d", packet.Version, decoded.Version)
	}

	if decoded.Command != packet.Command {
		t.Fatalf("Command not match, expect %d, but got %d", packet.Command, decoded.Command)
	}

	if string(decoded.Data) != string(packet.Data) {
		t.Fatalf("Data not match, expect %v, but got %v", packet.Data, decoded.Data)
	}
}
