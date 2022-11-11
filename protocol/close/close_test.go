package close

import "testing"

func TestEncodeDecode(t *testing.T) {
	packet := &Close{
		ConnectionID: "20ed2884bde9d7565dbf1",
	}

	encoded, err := Encode(packet)
	if err != nil {
		t.Fatalf("failed to encode %s", err)
	}

	decoded, err := Decode(encoded)
	if err != nil {
		t.Fatalf("failed to decode %s", err)
	}

	if decoded.ConnectionID != packet.ConnectionID {
		t.Fatalf("ConnectionID not match, expect %s, but got %s", packet.ConnectionID, decoded.ConnectionID)
	}
}
