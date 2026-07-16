package util

import (
	"encoding/json"
	"net/url"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x-extended/database/model"
)

func mieruInboundForTest(t *testing.T, options string) *model.Inbound {
	t.Helper()
	return &model.Inbound{
		Type:    "mieru",
		Tag:     "mieru-test",
		Addrs:   json.RawMessage(`[]`),
		OutJson: json.RawMessage("null"),
		Options: json.RawMessage(options),
	}
}

// TestMieruLinkCarriesPerUserCredentials proves the mierus:// link uses the
// client's own name/password (mieru is not keyless) and carries host, profile,
// port range and protocol from the inbound.
func TestMieruLinkCarriesPerUserCredentials(t *testing.T) {
	inbound := mieruInboundForTest(t, `{
		"listen_ports": ["2090:2099"],
		"transport": "TCP"
	}`)
	clientConfig := json.RawMessage(`{"mieru":{"name":"alice","password":"s3cret"}}`)

	links := LinkGenerator(clientConfig, inbound, "example.com")
	if len(links) != 1 {
		t.Fatalf("expected one mieru link, got %d: %#v", len(links), links)
	}
	u, err := url.Parse(links[0])
	if err != nil {
		t.Fatalf("mieru link must be a valid URL: %v", err)
	}
	if u.Scheme != "mierus" {
		t.Errorf("mieru simple link must use the mierus scheme, got %q", u.Scheme)
	}
	if u.User.Username() != "alice" {
		t.Errorf("username must be the client name, got %q", u.User.Username())
	}
	if pw, _ := u.User.Password(); pw != "s3cret" {
		t.Errorf("password must be the client password, got %q", pw)
	}
	if u.Hostname() != "example.com" {
		t.Errorf("host must be the advertised server, got %q", u.Hostname())
	}
	q := u.Query()
	if q.Get("profile") != "mieru-test" {
		t.Errorf("profile must default to the inbound tag, got %q", q.Get("profile"))
	}
	if q.Get("port") != "2090-2099" {
		t.Errorf("listen_ports range 2090:2099 must become port=2090-2099, got %q", q.Get("port"))
	}
	if q.Get("protocol") != "TCP" {
		t.Errorf("protocol must mirror the transport, got %q", q.Get("protocol"))
	}
}

// TestMieruLinkRequiresCredentials proves a client with no mieru credentials
// yields no link rather than an anonymous one.
func TestMieruLinkRequiresCredentials(t *testing.T) {
	inbound := mieruInboundForTest(t, `{"listen_ports":["2090:2099"],"transport":"TCP"}`)
	for _, cfg := range []string{`{}`, `{"mieru":{"name":"alice"}}`, `{"mieru":{"password":"p"}}`} {
		links := LinkGenerator(json.RawMessage(cfg), inbound, "example.com")
		if len(links) != 0 {
			t.Fatalf("config %s must produce no mieru link, got %#v", cfg, links)
		}
	}
}

// TestMieruLinkFallsBackToListenPort proves a single listen_port (no range list)
// is used when listen_ports is absent.
func TestMieruLinkFallsBackToListenPort(t *testing.T) {
	inbound := mieruInboundForTest(t, `{"listen_port": 9443, "transport": "UDP"}`)
	clientConfig := json.RawMessage(`{"mieru":{"name":"bob","password":"pw"}}`)
	links := LinkGenerator(clientConfig, inbound, "example.com")
	if len(links) != 1 {
		t.Fatalf("expected one link, got %d", len(links))
	}
	q, _ := url.Parse(links[0])
	if q.Query().Get("port") != "9443" {
		t.Errorf("single listen_port should be used verbatim, got %q", q.Query().Get("port"))
	}
	if q.Query().Get("protocol") != "UDP" {
		t.Errorf("protocol should be UDP, got %q", q.Query().Get("protocol"))
	}
}

// TestMieruLinkMultipleAddrs proves one link is emitted per advertised address.
func TestMieruLinkMultipleAddrs(t *testing.T) {
	inbound := mieruInboundForTest(t, `{"listen_ports":["2090:2099"],"transport":"TCP"}`)
	inbound.Addrs = json.RawMessage(`[
		{"server":"a.example.com"},
		{"server":"b.example.com"}
	]`)
	clientConfig := json.RawMessage(`{"mieru":{"name":"alice","password":"p"}}`)
	links := LinkGenerator(clientConfig, inbound, "ignored")
	if len(links) != 2 {
		t.Fatalf("expected one link per addr, got %d: %#v", len(links), links)
	}
	if !strings.Contains(links[0], "a.example.com") || !strings.Contains(links[1], "b.example.com") {
		t.Errorf("links must target their own addr: %#v", links)
	}
}

// TestMieruLinkMultiplePortRanges proves each listen_ports entry becomes its own
// port/protocol pair (mieru servers can bind several ranges).
func TestMieruLinkMultiplePortRanges(t *testing.T) {
	inbound := mieruInboundForTest(t, `{"listen_ports":["2090:2099","3000:3009"],"transport":"TCP"}`)
	clientConfig := json.RawMessage(`{"mieru":{"name":"alice","password":"p"}}`)
	links := LinkGenerator(clientConfig, inbound, "example.com")
	if len(links) != 1 {
		t.Fatalf("expected one link, got %d", len(links))
	}
	u, _ := url.Parse(links[0])
	ports := u.Query()["port"]
	if len(ports) != 2 || ports[0] != "2090-2099" || ports[1] != "3000-3009" {
		t.Errorf("each range must map to its own port token, got %#v", ports)
	}
	if protos := u.Query()["protocol"]; len(protos) != 2 {
		t.Errorf("each port must carry a protocol, got %#v", protos)
	}
}
