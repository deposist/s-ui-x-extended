package util

import (
	"encoding/json"
	"strconv"
	"strings"
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

// TestLinkRoundTripSocksAndHTTP covers the two credential-bearing links that
// carry no per-protocol payload: the panel emits them itself (socks5://, http://,
// https://), so pasting one back as an external link must reproduce the outbound
// instead of failing as an unknown format.
func TestLinkRoundTripSocksAndHTTP(t *testing.T) {
	cfg := json.RawMessage(wellFormedClientConfig)
	for _, typ := range []string{"socks", "http"} {
		t.Run(typ, func(t *testing.T) {
			links := LinkGenerator(cfg, wellFormedInbound(typ), "example.com")
			if len(links) == 0 {
				t.Fatalf("no link generated for %s", typ)
			}
			ob, _, err := GetOutbound(links[0], 0)
			if err != nil {
				t.Fatalf("GetOutbound(%q) error: %v", links[0], err)
			}
			out := *ob
			wantType := typ
			if got, _ := out["type"].(string); got != wantType {
				t.Fatalf("%s: type = %v, want %s", typ, out["type"], wantType)
			}
			if got, _ := out["server"].(string); got != "example.com" {
				t.Errorf("%s: server = %v, want example.com", typ, out["server"])
			}
			if got := portToInt(out["server_port"]); got != 443 {
				t.Errorf("%s: server_port = %v, want 443", typ, out["server_port"])
			}
			if got, _ := out["username"].(string); got != "u" {
				t.Errorf("%s: username = %v, want u", typ, out["username"])
			}
			if got, _ := out["password"].(string); got != "p" {
				t.Errorf("%s: password = %v, want p", typ, out["password"])
			}
		})
	}
}

// TestLinkRoundTripHTTPSTLS pins the https form: an http inbound with TLS is
// advertised as an https link, so the imported outbound has to enable TLS again.
func TestLinkRoundTripHTTPSTLS(t *testing.T) {
	ob, _, err := GetOutbound("https://u:p@example.com:8443", 0)
	if err != nil {
		t.Fatalf("GetOutbound(https) error: %v", err)
	}
	out := *ob
	if got := portToInt(out["server_port"]); got != 8443 {
		t.Errorf("server_port = %v, want 8443 (the explicit port must win over the https default)", out["server_port"])
	}
	tls, ok := out["tls"].(map[string]interface{})
	if !ok {
		t.Fatalf("https link produced no tls block: %#v", out)
	}
	if enabled, _ := tls["enabled"].(bool); !enabled {
		t.Errorf("tls.enabled = %v, want true", tls["enabled"])
	}
	if sni, _ := tls["server_name"].(string); sni != "example.com" {
		t.Errorf("tls.server_name = %v, want example.com", tls["server_name"])
	}
}

// TestGetOutboundNamesUnsupportedScheme checks the reason an operator gets: the
// error has to name the scheme, otherwise a link that vanished from a
// subscription looks like a link the panel never received.
func TestGetOutboundNamesUnsupportedScheme(t *testing.T) {
	_, _, err := GetOutbound("mierus://u:p@example.com?profile=node", 0)
	if err == nil {
		t.Fatal("expected an error for an unsupported link scheme")
	}
	if !strings.Contains(err.Error(), "mierus") {
		t.Fatalf("error %q does not name the unsupported scheme", err.Error())
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
