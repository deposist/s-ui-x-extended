package util

import (
	"encoding/json"
	"net/url"
	"testing"

	"github.com/deposist/s-ui-x/database/model"
)

func TestLinkGeneratorTUICIncludesUDPRelayMode(t *testing.T) {
	link := generateTUICLinkForTest(t, `{"udp_relay_mode":"native"}`)
	u, err := url.Parse(link)
	if err != nil {
		t.Fatal(err)
	}

	if got := u.Query().Get("udp_relay_mode"); got != "native" {
		t.Fatalf("expected udp_relay_mode=native, got %q in %s", got, link)
	}
}

func TestLinkGeneratorTUICRoundTripPreservesUDPRelayMode(t *testing.T) {
	link := generateTUICLinkForTest(t, `{"udp_relay_mode":"quic"}`)
	outbound, _, err := GetOutbound(link, 0)
	if err != nil {
		t.Fatal(err)
	}

	if got := (*outbound)["udp_relay_mode"]; got != "quic" {
		t.Fatalf("expected round-trip udp_relay_mode=quic, got %#v", got)
	}
}

func TestLinkGeneratorTUICDefaultsUDPRelayMode(t *testing.T) {
	link := generateTUICLinkForTest(t, `{}`)
	u, err := url.Parse(link)
	if err != nil {
		t.Fatal(err)
	}

	if got := u.Query().Get("udp_relay_mode"); got != defaultTUICUDPRelayMode {
		t.Fatalf("expected default udp_relay_mode=%s, got %q in %s", defaultTUICUDPRelayMode, got, link)
	}
}

// TestLinkGeneratorMalformedAddrsDoesNotPanic feeds an addr map missing
// server/remark and carrying non-bool tls.enabled / non-string alpn elements.
// Before the comma-ok hardening (Q4) these tripped interface-conversion panics
// in the subscription request goroutine; now every link type degrades to a
// partial/empty link instead.
func TestLinkGeneratorMalformedAddrsDoesNotPanic(t *testing.T) {
	malformedAddrs := json.RawMessage(`[
		{"tls":{"enabled":"yes"}},
		{"tls":{"enabled":true,"alpn":[123],"reality":{"enabled":"yes"}}}
	]`)
	clientConfig := json.RawMessage(`{
		"vless": {"uuid":"11111111-1111-4111-8111-111111111111","flow":"xtls-rprx-vision"},
		"trojan": {"password":"secret"},
		"vmess": {"uuid":"11111111-1111-4111-8111-111111111111"},
		"shadowsocks": {"password":"secret"},
		"socks": {"username":"u","password":"p"},
		"http": {"username":"u","password":"p"},
		"naive": {"username":"u","password":"p"},
		"hysteria": {"auth_str":"a"},
		"hysteria2": {"password":"secret"},
		"tuic": {"uuid":"11111111-1111-4111-8111-111111111111","password":"secret"},
		"anytls": {"password":"secret"}
	}`)
	for _, typ := range InboundTypeWithLink {
		t.Run(typ, func(t *testing.T) {
			inbound := &model.Inbound{
				Type:    typ,
				Tag:     "t",
				Addrs:   malformedAddrs,
				OutJson: json.RawMessage(`{}`),
				Options: json.RawMessage(`{"listen_port":443,"method":"aes-128-gcm"}`),
			}
			// The assertion is simply that this does not panic; a malformed
			// addr may legitimately yield empty or partial links.
			_ = LinkGenerator(clientConfig, inbound, "example.com")
		})
	}
}

func generateTUICLinkForTest(t *testing.T, outJSON string) string {
	t.Helper()

	clientConfig := json.RawMessage(`{
		"tuic": {
			"uuid": "11111111-1111-4111-8111-111111111111",
			"password": "secret"
		}
	}`)
	inbound := &model.Inbound{
		Type:    "tuic",
		Tag:     "tuic-test",
		Addrs:   json.RawMessage(`[]`),
		OutJson: json.RawMessage(outJSON),
		Options: json.RawMessage(`{
			"listen_port": 443,
			"congestion_control": "bbr"
		}`),
	}

	links := LinkGenerator(clientConfig, inbound, "example.com")
	if len(links) != 1 {
		t.Fatalf("expected one generated link, got %d: %#v", len(links), links)
	}
	return links[0]
}
