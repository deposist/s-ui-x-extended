package util

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x-extended/database/model"
)

// decodeSudokuLink parses a sudoku:// short link back into its raw payload map so
// tests can assert the single-letter fields the upstream SUDOKU-ASCII core reads.
func decodeSudokuLink(t *testing.T, link string) map[string]interface{} {
	t.Helper()
	if !strings.HasPrefix(link, "sudoku://") {
		t.Fatalf("link must use the sudoku:// scheme, got %q", link)
	}
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(link, "sudoku://"))
	if err != nil {
		t.Fatalf("sudoku link is not base64url(RawURLEncoding): %v", err)
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("sudoku link payload is not JSON: %v", err)
	}
	return payload
}

func sudokuInboundForTest(t *testing.T, options string) *model.Inbound {
	t.Helper()
	return &model.Inbound{
		Type:    "sudoku",
		Tag:     "sudoku-test",
		Addrs:   json.RawMessage(`[]`),
		OutJson: json.RawMessage("null"),
		Options: json.RawMessage(options),
	}
}

// TestSudokuLinkCarriesSharedKeyAndTransport proves LinkGenerator emits a
// sudoku:// link whose payload carries the shared inbound key, host, port and the
// transport-shaping fields the client needs. Sudoku is keyless, so the key comes
// from the inbound, not from a per-user config block.
func TestSudokuLinkCarriesSharedKeyAndTransport(t *testing.T) {
	inbound := sudokuInboundForTest(t, `{
		"listen_port": 8443,
		"key": "SHARED-KEY",
		"aead_method": "chacha20-poly1305",
		"table_type": "prefer_ascii",
		"enable_pure_downlink": true,
		"http_mask_mode": "stream",
		"path_root": "/cdn"
	}`)

	links := LinkGenerator(json.RawMessage(`{}`), inbound, "example.com")
	if len(links) != 1 {
		t.Fatalf("expected one sudoku link, got %d: %#v", len(links), links)
	}
	p := decodeSudokuLink(t, links[0])

	if p["h"] != "example.com" {
		t.Errorf("host (h) should be the advertised server, got %#v", p["h"])
	}
	if p["p"] != float64(8443) {
		t.Errorf("port (p) should be the listen_port, got %#v", p["p"])
	}
	if p["k"] != "SHARED-KEY" {
		t.Errorf("key (k) should be the shared inbound key, got %#v", p["k"])
	}
	if p["e"] != "chacha20-poly1305" {
		t.Errorf("aead (e) should mirror aead_method, got %#v", p["e"])
	}
	if p["a"] != "ascii" {
		t.Errorf("prefer_ascii table_type should encode to a=ascii, got %#v", p["a"])
	}
	if p["hm"] != "stream" {
		t.Errorf("http_mask_mode should map to hm, got %#v", p["hm"])
	}
	if p["hy"] != "/cdn" {
		t.Errorf("path_root should map to hy, got %#v", p["hy"])
	}
	// enable_pure_downlink=true -> packed downlink is the inverse, so x must be
	// absent (omitempty false).
	if _, has := p["x"]; has {
		t.Errorf("enable_pure_downlink=true must omit packed flag x, got %#v", p["x"])
	}
}

// TestSudokuLinkOmitsLegacyModeAndPackedFalse locks the two omitempty edges that
// match the upstream encoder: legacy mode is never written (client uses its
// default) and pure-downlink=false means packed=true is emitted.
func TestSudokuLinkOmitsLegacyModeAndPackedFalse(t *testing.T) {
	inbound := sudokuInboundForTest(t, `{
		"listen_port": 8443,
		"key": "K",
		"http_mask_mode": "legacy",
		"enable_pure_downlink": false
	}`)
	links := LinkGenerator(json.RawMessage(`{}`), inbound, "example.com")
	if len(links) != 1 {
		t.Fatalf("expected one sudoku link, got %d", len(links))
	}
	p := decodeSudokuLink(t, links[0])

	if _, has := p["hm"]; has {
		t.Errorf("legacy mode must not be written to the link (hm), got %#v", p["hm"])
	}
	if p["x"] != true {
		t.Errorf("enable_pure_downlink=false should emit packed x=true, got %#v", p["x"])
	}
}

// TestSudokuLinkRequiresKey proves a keyless-but-unconfigured inbound (no shared
// key) yields no link rather than a broken one.
func TestSudokuLinkRequiresKey(t *testing.T) {
	inbound := sudokuInboundForTest(t, `{"listen_port": 8443}`)
	links := LinkGenerator(json.RawMessage(`{}`), inbound, "example.com")
	if len(links) != 0 {
		t.Fatalf("sudoku inbound without a key must produce no link, got %#v", links)
	}
}

// TestSudokuLinkUsesExplicitAddrs proves per-address links are generated when the
// inbound advertises multiple Addrs, each with its own host/port.
func TestSudokuLinkUsesExplicitAddrs(t *testing.T) {
	inbound := sudokuInboundForTest(t, `{"listen_port": 8443, "key": "K"}`)
	inbound.Addrs = json.RawMessage(`[
		{"server": "a.example.com", "server_port": 8443},
		{"server": "b.example.com", "server_port": 9443}
	]`)
	links := LinkGenerator(json.RawMessage(`{}`), inbound, "ignored.example.com")
	if len(links) != 2 {
		t.Fatalf("expected one link per addr, got %d: %#v", len(links), links)
	}
	first := decodeSudokuLink(t, links[0])
	second := decodeSudokuLink(t, links[1])
	if first["h"] != "a.example.com" || first["p"] != float64(8443) {
		t.Errorf("first addr mismatch: %#v", first)
	}
	if second["h"] != "b.example.com" || second["p"] != float64(9443) {
		t.Errorf("second addr mismatch: %#v", second)
	}
}
