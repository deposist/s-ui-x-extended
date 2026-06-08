package util

import (
	"net/url"
	"strings"
	"testing"
)

// TestMtproxyLinkBuildsTgDeepLink proves the tg://proxy link carries the server,
// port and the verbatim faketls secret, URL-escaped.
func TestMtproxyLinkBuildsTgDeepLink(t *testing.T) {
	// A valid faketls secret: ee + 16-byte key (32 hex) + hex("example.com").
	secret := "ee" + strings.Repeat("ab", 16) + "6578616d706c652e636f6d"
	uc := map[string]interface{}{"secret": secret}
	addrs := []map[string]interface{}{
		{"server": "1.2.3.4", "server_port": float64(8443), "remark": "node"},
	}

	links := mtproxyLink(uc, addrs)
	if len(links) != 1 {
		t.Fatalf("expected 1 link, got %d: %v", len(links), links)
	}
	link := links[0]
	if !strings.HasPrefix(link, "tg://proxy?") {
		t.Fatalf("link should be a tg://proxy deep link: %s", link)
	}
	u, err := url.Parse(link)
	if err != nil {
		t.Fatalf("link is not a valid URL: %v", err)
	}
	q := u.Query()
	if q.Get("server") != "1.2.3.4" {
		t.Errorf("server param wrong: %s", q.Get("server"))
	}
	if q.Get("port") != "8443" {
		t.Errorf("port param wrong: %s", q.Get("port"))
	}
	if q.Get("secret") != secret {
		t.Errorf("secret param wrong: got %s want %s", q.Get("secret"), secret)
	}
}

// TestMtproxyLinkEmptySecret proves a missing secret yields no link (rather than a
// broken tg link).
func TestMtproxyLinkEmptySecret(t *testing.T) {
	links := mtproxyLink(map[string]interface{}{}, []map[string]interface{}{{"server": "h", "server_port": float64(1)}})
	if len(links) != 0 {
		t.Errorf("expected no link for empty secret, got %v", links)
	}
}
