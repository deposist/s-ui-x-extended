package model

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAWGDeviceJSONDoesNotExposeSecrets(t *testing.T) {
	device := AWGDevice{
		CryptoContext:     []byte("crypto-context"),
		PreviousPublicKey: "previous-public-key",
		PrivateKeyEnc:     []byte("encrypted-private-key"),
		PSKEnc:            []byte("encrypted-preshared-key"),
		LastError:         "sensitive runtime error",
	}
	encoded, err := json.Marshal(device)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{
		"crypto-context",
		"previous-public-key",
		"encrypted-private-key",
		"encrypted-preshared-key",
		"sensitive runtime error",
	} {
		if strings.Contains(string(encoded), secret) {
			t.Fatalf("AWGDevice JSON exposed %q: %s", secret, encoded)
		}
	}
}
