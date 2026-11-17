package util

import (
	"encoding/json"
	"net/url"
	"testing"

	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/sagernet/sing-box/option"
)

func TestGetOutboundRejectsMalformedAuthorityWithoutPanic(t *testing.T) {
	tests := []string{
		"vless://example.com:443",
		"trojan://example.com:443",
		"hysteria2://example.com:443",
		"anytls://example.com:443",
		"tuic://example.com:443",
		"ss://example.com:443",
		"vless://id@:443",
		"trojan://secret@example.com:not-a-port",
		"hysteria://example.com:70000",
		"naive+https://user:pass@:443",
		"vless://id@example.com:443?bad=%zz",
		"vmess://bnVsbA==",
	}
	for _, link := range tests {
		t.Run(link, func(t *testing.T) {
			for range 2 {
				outbound, _, err := GetOutbound(link, 0)
				if err == nil || outbound != nil {
					t.Fatalf("GetOutbound(%q) = %#v, %v; want nil outbound and error", link, outbound, err)
				}
			}
		})
	}
}

func TestPrepareTLSAcceptsNullClientMap(t *testing.T) {
	got := prepareTls(&model.Tls{Client: json.RawMessage(`null`), Server: json.RawMessage(`{"enabled":true}`)})
	if got == nil || got["enabled"] != true {
		t.Fatalf("prepareTls(null client) = %#v", got)
	}
}

func TestShareURLRoundTripsEscapingAndIPv6(t *testing.T) {
	userinfo := url.UserPassword("user@:/?#%", "pass@:/?#%")
	params := []LinkParam{{Key: "mport", Value: "1&x=#%"}, {Key: "alpn", Value: "h2&x=#%"}}
	link := shareURL("tuic", userinfo, "2001:db8::1", 443, params, "remark@:/?#%")

	parsed, err := url.Parse(link)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Hostname() != "2001:db8::1" || parsed.Port() != "443" {
		t.Fatalf("authority did not round-trip: %q", link)
	}
	password, ok := parsed.User.Password()
	if parsed.User.Username() != "user@:/?#%" || !ok || password != "pass@:/?#%" {
		t.Fatalf("userinfo did not round-trip: %#v", parsed.User)
	}
	if parsed.Query().Get("mport") != "1&x=#%" || parsed.Query().Get("alpn") != "h2&x=#%" {
		t.Fatalf("query did not round-trip: %#v", parsed.Query())
	}
	if parsed.Fragment != "remark@:/?#%" {
		t.Fatalf("fragment did not round-trip: %q", parsed.Fragment)
	}
}

func TestHTTPLinkProtocolIsPerAddress(t *testing.T) {
	links := httpLink(map[string]interface{}{"username": "u", "password": "p"}, []map[string]interface{}{
		{"server": "tls.example", "server_port": float64(443), "tls": map[string]interface{}{"enabled": true}},
		{"server": "plain.example", "server_port": float64(80)},
	})
	if len(links) != 2 {
		t.Fatalf("httpLink returned %#v", links)
	}
	first, err := url.Parse(links[0])
	if err != nil {
		t.Fatal(err)
	}
	second, err := url.Parse(links[1])
	if err != nil {
		t.Fatal(err)
	}
	if first.Scheme != "https" || second.Scheme != "http" {
		t.Fatalf("protocol leaked across addresses: %#v", links)
	}
}

func TestVMessInsecureFlagValues(t *testing.T) {
	for _, tc := range []struct {
		value any
		want  bool
	}{{false, false}, {true, true}, {0, false}, {1, true}, {"false", false}, {"true", true}, {"0", false}, {"1", true}} {
		data, err := json.Marshal(map[string]any{"tls": "tls", "allowInsecure": tc.value, "add": "example.com", "port": 443})
		if err != nil {
			t.Fatal(err)
		}
		outbound, _, err := GetOutbound("vmess://"+ByteToB64Str(data), 0)
		if err != nil {
			t.Fatal(err)
		}
		tls := (*outbound)["tls"].(map[string]interface{})
		if got, _ := tls["insecure"].(bool); got != tc.want {
			t.Fatalf("allowInsecure=%v (%T): insecure=%v, want %v", tc.value, tc.value, got, tc.want)
		}
	}
}

func TestImportedTCPLinksDecodeWithCoreOptions(t *testing.T) {
	for _, protocol := range []string{"vmess", "vless", "trojan"} {
		t.Run(protocol, func(t *testing.T) {
			links := LinkGenerator(json.RawMessage(wellFormedClientConfig), wellFormedInbound(protocol), "example.com")
			if len(links) == 0 {
				t.Fatal("no generated link")
			}
			outbound, _, err := GetOutbound(links[0], 0)
			if err != nil {
				t.Fatal(err)
			}
			data, err := json.Marshal(outbound)
			if err != nil {
				t.Fatal(err)
			}
			var options any
			switch protocol {
			case "vmess":
				options = &option.VMessOutboundOptions{}
			case "vless":
				options = &option.VLESSOutboundOptions{}
			case "trojan":
				options = &option.TrojanOutboundOptions{}
			}
			if err := json.Unmarshal(data, options); err != nil {
				t.Fatalf("imported %s link does not decode as core options: %v", protocol, err)
			}
		})
	}
}
