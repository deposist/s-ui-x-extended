package util

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/deposist/s-ui-x-extended/core/capabilities"
	"github.com/deposist/s-ui-x-extended/database/model"
)

// globalForbiddenKeys must NEVER appear anywhere in a client-facing out_json or
// subscription body: private key material and server-only operational fields. The
// legitimate client credentials (sudoku `key`, mtproxy `secret`, per-user
// `password`/`uuid`) are deliberately NOT in this set.
var globalForbiddenKeys = map[string]struct{}{
	"private_key":                  {},
	"host_key":                     {},
	"host_key_path":                {},
	"key_path":                     {},
	"server_version":               {},
	"max_auth_tries":               {},
	"handshake_timeout":            {},
	"fallback":                     {},
	"fallback_for_alpn":            {},
	"allow_fallback_on_unknown_dc": {},
}

// ScanForbiddenKeys walks a decoded JSON value and returns the dotted paths of any
// forbidden key. A bare `key` is forbidden ONLY under a tls/reality ancestor (a TLS
// private key); a top-level sudoku `key` is a legitimate client credential.
func ScanForbiddenKeys(v interface{}, path string, underTLS bool) []string {
	var hits []string
	switch t := v.(type) {
	case map[string]interface{}:
		for k, child := range t {
			childPath := path + "." + k
			if _, bad := globalForbiddenKeys[k]; bad {
				hits = append(hits, childPath)
			}
			if k == "key" && underTLS {
				hits = append(hits, childPath)
			}
			hits = append(hits, ScanForbiddenKeys(child, childPath, underTLS || k == "tls" || k == "reality")...)
		}
	case []interface{}:
		for idx, child := range t {
			hits = append(hits, ScanForbiddenKeys(child, fmt.Sprintf("%s[%d]", path, idx), underTLS)...)
		}
	}
	return hits
}

func cloneStringMap(in map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// tlsWithSecrets is a server TLS config carrying private material (key,
// private_key, reality.private_key). addTls must copy only the public parts.
func tlsWithSecrets() *model.Tls {
	return &model.Tls{
		Server: json.RawMessage(`{
			"enabled": true,
			"server_name": "example.com",
			"certificate": ["PUBLIC-CERT"],
			"key": ["PRIVATE-TLS-KEY"],
			"key_path": "/etc/ssl/private.key",
			"reality": {"enabled": true, "private_key": "REALITY-PRIVATE", "public_key": "REALITY-PUBLIC", "short_id": ["abcd"]}
		}`),
		Client: json.RawMessage(`{"reality": {"enabled": false, "public_key": "REALITY-PUBLIC"}}`),
	}
}

// TestForbiddenKeysNeverInOutJson is the central allow-list regression guard: feed
// every delivered inbound type a kitchen-sink of server secrets (in options and, for
// TLS-capable types, in the TLS config) and prove FillOutJson never emits one.
func TestForbiddenKeysNeverInOutJson(t *testing.T) {
	tlsTypes := map[string]bool{}
	for _, in := range capabilities.Inbounds() {
		if in.HasTLSTemplate {
			tlsTypes[in.Type] = true
		}
	}

	secrets := map[string]interface{}{
		// ssh server-side / private material
		"host_key":       []interface{}{"PRIVATE-HOST-KEY"},
		"host_key_path":  []interface{}{"/etc/ssh/ssh_host_ed25519_key"},
		"server_version": "SSH-2.0-secret",
		"max_auth_tries": 3,
		"private_key":    "GENERIC-PRIVATE",
		// sudoku / mtproxy server-only
		"handshake_timeout":            5,
		"fallback":                     "internal.example.com:8443",
		"fallback_for_alpn":            map[string]interface{}{"h2": map[string]interface{}{"server": "x", "server_port": 1}},
		"allow_fallback_on_unknown_dc": true,
		// legitimate builder inputs that SHOULD survive (not secrets)
		"transport":         "TCP",
		"traffic_pattern":   "tls",
		"listen_ports":      []interface{}{"2090:2099"},
		"network":           []interface{}{"tcp", "udp"},
		"quic":              true,
		"key":               "SUDOKU-CLIENT-KEY", // legit at root for sudoku
		"disable_http_mask": false,
		"http_mask_mode":    "stream",
		"path_root":         "/cdn",
	}

	types := []string{
		"ssh", "mieru", "trusttunnel", "sudoku",
		"vless", "vmess", "trojan", "shadowsocks", "shadowtls",
		"hysteria", "hysteria2", "tuic", "naive", "anytls",
		"http", "socks", "mixed",
	}
	for _, typ := range types {
		opts := cloneStringMap(secrets)
		opts["listen_port"] = 443
		raw, _ := json.Marshal(opts)
		i := &model.Inbound{Type: typ, Tag: typ + "-in", Options: raw, OutJson: json.RawMessage("null")}
		if tlsTypes[typ] {
			i.TlsId = 1
			i.Tls = tlsWithSecrets()
		}
		if err := FillOutJson(i, "example.com"); err != nil {
			t.Fatalf("FillOutJson(%s): %v", typ, err)
		}
		var out map[string]interface{}
		if err := json.Unmarshal(i.OutJson, &out); err != nil {
			t.Fatalf("unmarshal out_json(%s): %v", typ, err)
		}
		if hits := ScanForbiddenKeys(out, typ, false); len(hits) > 0 {
			t.Errorf("%s out_json leaked server secret(s): %v\nfull out_json: %s", typ, hits, i.OutJson)
		}
	}
}

// TestNoBuilderRangesOverWholeInbound makes the allow-list invariant enforceable
// rather than conventional: MarshalFull() exposes every inbound Option to the
// builders, so a `for k, v := range inbound` in any builder would copy server
// secrets wholesale. Statically forbid that pattern in the builder file. (The wipe
// loop ranges over the OUTPUT map `outJson`, which is safe and allowed.)
func TestNoBuilderRangesOverWholeInbound(t *testing.T) {
	src, err := os.ReadFile("outJson.go")
	if err != nil {
		t.Fatalf("read outJson.go: %v", err)
	}
	if rangeInbound.Match(src) {
		t.Fatal("a builder ranges over the whole `inbound` map — that defeats the allow-list and can leak server secrets. Copy only explicitly named fields.")
	}
}

var rangeInbound = regexp.MustCompile(`range\s+inbound\b`)

// TestSudokuRootKeyIsNotFlagged sanity-checks the scanner: a sudoku client key at
// the outbound root is legitimate and must not trip the TLS-key rule.
func TestSudokuRootKeyIsNotFlagged(t *testing.T) {
	out := map[string]interface{}{"type": "sudoku", "key": "client-key"}
	if hits := ScanForbiddenKeys(out, "sudoku", false); len(hits) > 0 {
		t.Errorf("sudoku root key must not be flagged, got %v", hits)
	}
	// But a key inside a tls block must be flagged.
	tlsy := map[string]interface{}{"tls": map[string]interface{}{"key": "leak"}}
	if hits := ScanForbiddenKeys(tlsy, "x", false); len(hits) != 1 {
		t.Errorf("tls.key must be flagged exactly once, got %v", hits)
	}
}
