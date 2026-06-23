package util

import (
	"encoding/json"
	"strconv"
	"testing"
)

func portToInt(v interface{}) int {
	switch p := v.(type) {
	case int:
		return p
	case int64:
		return int(p)
	case float64:
		return int(p)
	case string:
		n, _ := strconv.Atoi(p)
		return n
	default:
		return -1
	}
}

// TestLinkRoundTripPreservesIdentity pins generation correctness across the
// link<->json boundary: a link produced by LinkGenerator must parse back through
// GetOutbound into an outbound that preserves the protocol type, server,
// port, and the credential identity (uuid/password/auth). This guards both
// directions against an accidental regression during later perf edits.
func TestLinkRoundTripPreservesIdentity(t *testing.T) {
	cfg := json.RawMessage(wellFormedClientConfig)
	cases := []struct {
		typ     string
		outType string
		credKey string
		credVal string
	}{
		{"vless", "vless", "uuid", "11111111-1111-4111-8111-111111111111"},
		{"trojan", "trojan", "password", "p"},
		{"tuic", "tuic", "uuid", "11111111-1111-4111-8111-111111111111"},
		{"hysteria", "hysteria", "auth_str", "a"},
		{"hysteria2", "hysteria2", "password", "p"},
		{"anytls", "anytls", "password", "p"},
		{"shadowsocks", "shadowsocks", "password", "sspass"},
		{"vmess", "vmess", "uuid", "11111111-1111-4111-8111-111111111111"},
		{"naive", "naive", "password", "p"},
	}
	for _, c := range cases {
		t.Run(c.typ, func(t *testing.T) {
			links := LinkGenerator(cfg, wellFormedInbound(c.typ), "example.com")
			if len(links) == 0 {
				t.Fatalf("no link generated for %s", c.typ)
			}
			ob, tag, err := GetOutbound(links[0], 0)
			if err != nil {
				t.Fatalf("GetOutbound(%q) error: %v", links[0], err)
			}
			if ob == nil {
				t.Fatalf("GetOutbound(%q) returned nil outbound", links[0])
			}
			if tag == "" {
				t.Errorf("round-trip produced empty tag for %s", c.typ)
			}
			out := *ob
			if got, _ := out["type"].(string); got != c.outType {
				t.Errorf("%s: type = %v, want %s", c.typ, out["type"], c.outType)
			}
			if got, _ := out["server"].(string); got != "example.com" {
				t.Errorf("%s: server = %v, want example.com", c.typ, out["server"])
			}
			if got := portToInt(out["server_port"]); got != 443 {
				t.Errorf("%s: server_port = %v, want 443", c.typ, out["server_port"])
			}
			if got, _ := out[c.credKey].(string); got != c.credVal {
				t.Errorf("%s: %s = %v, want %s", c.typ, c.credKey, out[c.credKey], c.credVal)
			}
		})
	}
}
