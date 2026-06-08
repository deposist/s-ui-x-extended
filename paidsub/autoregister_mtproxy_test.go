package paidsub

import (
	"encoding/hex"
	"strings"
	"testing"
)

// TestRandomMTProtoSecretIsValidFakeTLS proves the generated mtproxy secret passes
// exactly the checks mtglib.Secret.Set enforces (mtg-multi mtglib/secret.go in the
// extended fork): hex-decodable, first byte 0xee (faketls), at least a 16-byte key,
// and a non-empty SNI host. A bare hex key — the panel's historical format — fails
// these and makes the mtproxy inbound refuse to start.
func TestRandomMTProtoSecretIsValidFakeTLS(t *testing.T) {
	for i := 0; i < 50; i++ {
		secret, err := randomMTProtoSecret()
		if err != nil {
			t.Fatalf("randomMTProtoSecret: %v", err)
		}
		if !strings.HasPrefix(secret, "ee") {
			t.Fatalf("secret must start with the faketls 'ee' byte: %s", secret)
		}
		decoded, err := hex.DecodeString(secret)
		if err != nil {
			t.Fatalf("secret must be hex-decodable: %v (%s)", err, secret)
		}
		if len(decoded) < 2 {
			t.Fatalf("secret truncated: %d bytes", len(decoded))
		}
		if decoded[0] != 0xee {
			t.Fatalf("first byte must be 0xee, got %#x", decoded[0])
		}
		rest := decoded[1:]
		if len(rest) < 16 {
			t.Fatalf("secret missing 16-byte key, got %d", len(rest))
		}
		host := string(rest[16:])
		if host == "" {
			t.Fatalf("faketls host must be non-empty: %s", secret)
		}
		// host must be ASCII (a plausible SNI)
		for _, r := range host {
			if r > 127 {
				t.Fatalf("faketls host must be ASCII: %q", host)
			}
		}
	}
}
