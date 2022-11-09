package authenticate

import "testing"

func TestRequestEncodeDecode(t *testing.T) {
	packet := &AuthenticateRequest{
		UserClientID: "0123456789",
		Timestamp:    "1667982806000",
		Nonce:        "123456",
		Signature:    "8665ebcb30adc07590ae3209e8cb0c2b9b43762cf6656d95ddb52fbc2a45e39c",
	}

	encoded, err := EncodeRequest(packet)
	if err != nil {
		t.Fatalf("failed to encode %s", err)
	}

	decoded, err := DecodeRequest(encoded)
	if err != nil {
		t.Fatalf("failed to decode %s", err)
	}

	if decoded.UserClientID != packet.UserClientID {
		t.Fatalf("UserClientID not match, expect %s, but got %s", packet.UserClientID, decoded.UserClientID)
	}

	if decoded.Timestamp != packet.Timestamp {
		t.Fatalf("Timestamp not match, expect %s, but got %s", packet.Timestamp, decoded.Timestamp)
	}

	if decoded.Nonce != packet.Nonce {
		t.Fatalf("Nonce not match, expect %s, but got %s", packet.Nonce, decoded.Nonce)
	}

	if decoded.Signature != packet.Signature {
		t.Fatalf("Signature not match, expect %s, but got %s", packet.Signature, decoded.Signature)
	}
}
