package util

import (
	"encoding/json"
	"net/url"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x-extended/database/model"
)

// wellFormedClientConfig carries one valid user-config block per protocol so
// every link type in InboundTypeWithLink produces at least one link.
// #nosec G101 -- synthetic credentials used only as link-generator test input.
const wellFormedClientConfig = `{
	"socks": {"username":"u","password":"p"},
	"http": {"username":"u","password":"p"},
	"naive": {"username":"u","password":"p"},
	"shadowsocks": {"password":"sspass"},
	"shadowsocks16": {"password":"sspass16"},
	"vmess": {"uuid":"11111111-1111-4111-8111-111111111111"},
	"vless": {"uuid":"11111111-1111-4111-8111-111111111111"},
	"anytls": {"password":"p"},
	"trojan": {"password":"p"},
	"hysteria": {"auth_str":"a"},
	"hysteria2": {"password":"p"},
	"tuic": {"uuid":"11111111-1111-4111-8111-111111111111","password":"p"},
	"mtproxy": {"secret":"ee0011223344556677889900aabbccdd"}
}`

func wellFormedInbound(typ string) *model.Inbound {
	return &model.Inbound{
		Type:    typ,
		Tag:     "node",
		Addrs:   json.RawMessage(`[{"server":"example.com","server_port":443,"remark":"-r"}]`),
		OutJson: json.RawMessage(`{}`),
		Options: json.RawMessage(`{"listen_port":443,"method":"aes-128-gcm"}`),
	}
}

// linkSchemePrefixes maps each protocol to the URI scheme(s) its links must use.
var linkSchemePrefixes = map[string][]string{
	"socks":       {"socks5://"},
	"http":        {"http://"},
	"mixed":       {"socks5://", "http://"},
	"shadowsocks": {"ss://"},
	"naive":       {"http2://"},
	"hysteria":    {"hysteria://"},
	"hysteria2":   {"hysteria2://"},
	"anytls":      {"anytls://"},
	"tuic":        {"tuic://"},
	"vless":       {"vless://"},
	"trojan":      {"trojan://"},
	"vmess":       {"vmess://"},
	"mtproxy":     {"tg://proxy?"},
}

func hasAnyPrefix(s string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

// TestLinkGeneratorDeterministicPerProtocol pins a stated goal of the
// optimization plan: link generation must be deterministic. For every protocol
// in InboundTypeWithLink, the same well-formed input must produce a
// byte-identical link list across runs (no map-iteration order leaking into
// output, no time/random source except the explicitly-randomised reality
// short_id selection, which this fixture avoids by using no TLS).
func TestLinkGeneratorDeterministicPerProtocol(t *testing.T) {
	cfg := json.RawMessage(wellFormedClientConfig)
	for _, typ := range InboundTypeWithLink {
		t.Run(typ, func(t *testing.T) {
			first := LinkGenerator(cfg, wellFormedInbound(typ), "example.com")
			if len(first) == 0 {
				t.Fatalf("protocol %s produced no links for well-formed input", typ)
			}
			for i := range 8 {
				next := LinkGenerator(cfg, wellFormedInbound(typ), "example.com")
				if strings.Join(next, "\n") != strings.Join(first, "\n") {
					t.Fatalf("protocol %s non-deterministic:\nrun0: %#v\nrun%d: %#v", typ, first, i+1, next)
				}
			}
		})
	}
}

// TestLinkGeneratorSchemeAndParseable pins generation correctness: every link
// uses the protocol's expected URI scheme and (for the URL-shaped schemes)
// parses as a valid URL. vmess/ss/naive carry base64 payloads after the scheme
// rather than a host, so only their scheme prefix is asserted.
func TestLinkGeneratorSchemeAndParseable(t *testing.T) {
	cfg := json.RawMessage(wellFormedClientConfig)
	// Schemes whose body is a base64 blob (not a parseable URL authority).
	opaque := map[string]bool{"vmess": true, "shadowsocks": true, "naive": true, "mixed": false}
	for _, typ := range InboundTypeWithLink {
		t.Run(typ, func(t *testing.T) {
			links := LinkGenerator(cfg, wellFormedInbound(typ), "example.com")
			if len(links) == 0 {
				t.Fatalf("protocol %s produced no links", typ)
			}
			prefixes := linkSchemePrefixes[typ]
			for _, link := range links {
				if !hasAnyPrefix(link, prefixes) {
					t.Fatalf("protocol %s link %q does not start with any of %v", typ, link, prefixes)
				}
				if !opaque[typ] {
					if _, err := url.Parse(link); err != nil {
						t.Fatalf("protocol %s link %q failed to parse: %v", typ, link, err)
					}
				}
			}
		})
	}
}
