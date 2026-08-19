package sub

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/deposist/s-ui-x-extended/service"
	"github.com/deposist/s-ui-x-extended/util"
)

// scanForbidden mirrors util's forbidden-key scanner (test helper; duplicated to
// avoid exporting test-only code). A bare `key` is forbidden only under tls/reality.
var forbiddenSubKeys = map[string]struct{}{
	"private_key": {}, "host_key": {}, "host_key_path": {}, "key_path": {},
	"server_version": {}, "max_auth_tries": {}, "handshake_timeout": {},
	"fallback": {}, "fallback_for_alpn": {}, "allow_fallback_on_unknown_dc": {},
}

func scanForbidden(v interface{}, path string, underTLS bool) []string {
	var hits []string
	switch t := v.(type) {
	case map[string]interface{}:
		for k, child := range t {
			cp := path + "." + k
			if _, bad := forbiddenSubKeys[k]; bad {
				hits = append(hits, cp)
			}
			if k == "key" && underTLS {
				hits = append(hits, cp)
			}
			hits = append(hits, scanForbidden(child, cp, underTLS || k == "tls" || k == "reality")...)
		}
	case []interface{}:
		for idx, child := range t {
			hits = append(hits, scanForbidden(child, fmt.Sprintf("%s[%d]", path, idx), underTLS)...)
		}
	}
	return hits
}

func deliveredInbound(t *testing.T, typ string, options map[string]interface{}) *model.Inbound {
	t.Helper()
	options["listen_port"] = 443
	raw, _ := json.Marshal(options)
	i := &model.Inbound{Type: typ, Tag: typ + "-in", Options: raw, OutJson: json.RawMessage("null"), Addrs: json.RawMessage("[]")}
	if err := util.FillOutJson(i, "example.com"); err != nil {
		t.Fatalf("FillOutJson(%s): %v", typ, err)
	}
	return i
}

func findOutbound(outs []map[string]interface{}, typ string) map[string]interface{} {
	for _, o := range outs {
		if o["type"] == typ {
			return o
		}
	}
	return nil
}

// TestSubscriptionCredentialMerge proves the field-scoped credential mapping:
// name->username (mieru/trusttunnel), name->user (ssh), password->password; that no
// arbitrary client.config key leaks; and that no server secret reaches the body.
func TestSubscriptionCredentialMerge(t *testing.T) {
	mieru := deliveredInbound(t, "mieru", map[string]interface{}{
		"transport": "TCP", "listen_ports": []interface{}{"2090:2099"},
		"host_key": []interface{}{"PRIV"}, "fallback": "x:1", // server junk that must not leak
	})
	tt := deliveredInbound(t, "trusttunnel", map[string]interface{}{
		"network": []interface{}{"tcp"}, "quic": true, "congestion_controller": "bbr",
	})
	ssh := deliveredInbound(t, "ssh", map[string]interface{}{
		"host_key": []interface{}{"PRIVATE"}, "server_version": "x", "max_auth_tries": 3,
	})
	sudoku := deliveredInbound(t, "sudoku", map[string]interface{}{
		"key": "SHARED-KEY", "aead_method": "chacha20-poly1305",
		"fallback": "y:2", "handshake_timeout": 5, // server-only must not leak
	})

	config := map[string]map[string]interface{}{
		"mieru":       {"name": "alice", "password": "pw1", "junk": "SHOULD-NOT-APPEAR"},
		"trusttunnel": {"name": "bob", "password": "pw2"},
		"ssh":         {"name": "carol", "password": "pw3"},
		"sudoku":      {"key": "PERSONAL-SPLIT-KEY", "server": "evil.example", "server_port": 1, "aead_method": "none", "http_mask": map[string]any{"host": "evil"}, "tag": "evil", "junk": true},
	}
	configRaw, _ := json.Marshal(config)

	outs, _, err := (&JsonService{}).getOutbounds(configRaw, []*model.Inbound{mieru, tt, ssh, sudoku})
	if err != nil {
		t.Fatalf("getOutbounds: %v", err)
	}

	m := findOutbound(*outs, "mieru")
	if m == nil || m["username"] != "alice" || m["password"] != "pw1" {
		t.Errorf("mieru should map name->username, password->password: %v", m)
	}
	if _, leaked := m["junk"]; leaked {
		t.Errorf("mieru must not copy arbitrary client.config key 'junk': %v", m)
	}
	if _, leaked := m["name"]; leaked {
		t.Errorf("mieru must not carry 'name': %v", m)
	}

	tto := findOutbound(*outs, "trusttunnel")
	if tto == nil || tto["username"] != "bob" || tto["password"] != "pw2" {
		t.Errorf("trusttunnel should map name->username: %v", tto)
	}

	so := findOutbound(*outs, "ssh")
	if so == nil || so["user"] != "carol" || so["password"] != "pw3" {
		t.Errorf("ssh should map name->user: %v", so)
	}
	if _, leaked := so["username"]; leaked {
		t.Errorf("ssh must use 'user', not 'username': %v", so)
	}

	su := findOutbound(*outs, "sudoku")
	if su == nil || su["key"] != "PERSONAL-SPLIT-KEY" {
		t.Errorf("sudoku should carry only its personal key override: %v", su)
	}
	if su["server"] != "example.com" || su["server_port"] != float64(443) || su["aead_method"] != "chacha20-poly1305" || su["tag"] == "evil" {
		t.Errorf("sudoku client config overrode inbound-controlled fields: %v", su)
	}
	if _, leaked := su["junk"]; leaked {
		t.Errorf("sudoku must not copy arbitrary client.config fields: %v", su)
	}

	// No server secret anywhere in the assembled outbounds.
	if hits := scanForbidden(*outs, "outbounds", false); len(hits) > 0 {
		body, _ := json.MarshalIndent(*outs, "", "  ")
		t.Errorf("subscription outbounds leaked server secret(s): %v\n%s", hits, body)
	}
}

func TestSudokuSubscriptionKeyIsolationAndFallback(t *testing.T) {
	sudoku := deliveredInbound(t, "sudoku", map[string]interface{}{
		"key": "INBOUND-KEY", "aead_method": "chacha20-poly1305",
	})
	originalOutJSON := append(json.RawMessage(nil), sudoku.OutJson...)
	getKey := func(config string) any {
		t.Helper()
		outs, _, err := (&JsonService{}).getOutbounds(json.RawMessage(config), []*model.Inbound{sudoku})
		if err != nil {
			t.Fatal(err)
		}
		return findOutbound(*outs, "sudoku")["key"]
	}
	if got := getKey(`{"sudoku":{"key":"KEY-A"}}`); got != "KEY-A" {
		t.Fatalf("A key=%v", got)
	}
	if got := getKey(`{"sudoku":{"key":"KEY-B"}}`); got != "KEY-B" {
		t.Fatalf("B key=%v", got)
	}
	for _, config := range []string{`{}`, `{"sudoku":{}}`, `{"sudoku":{"key":""}}`} {
		if got := getKey(config); got != "INBOUND-KEY" {
			t.Fatalf("fallback key for %s = %v", config, got)
		}
	}
	if string(sudoku.OutJson) != string(originalOutJSON) {
		t.Fatalf("subscription merge mutated persisted OutJson: before=%s after=%s", originalOutJSON, sudoku.OutJson)
	}
}

// TestMtproxyExcludedFromJsonSubscription proves mtproxy is delivered ONLY as a
// tg:// link: its out_json is wiped so the subscription's len<5 guard drops it
// (there is no sing-box mtproxy outbound).
func TestMtproxyExcludedFromJsonSubscription(t *testing.T) {
	mt := deliveredInbound(t, "mtproxy", map[string]interface{}{"concurrency": 8})
	if string(mt.OutJson) != "{}" {
		t.Fatalf("mtproxy out_json should be wiped to {}, got %q", mt.OutJson)
	}
	config, _ := json.Marshal(map[string]map[string]interface{}{
		"mtproxy": {"name": "a", "secret": "ee00112233445566778899aabbccddeeff6578616d706c652e636f6d"},
	})
	outs, _, err := (&JsonService{}).getOutbounds(config, []*model.Inbound{mt})
	if err != nil {
		t.Fatalf("getOutbounds: %v", err)
	}
	if len(*outs) != 0 || findOutbound(*outs, "mtproxy") != nil {
		t.Errorf("mtproxy must be excluded from the JSON subscription, got %v", *outs)
	}
}

// TestTrustTunnelSubscriptionCarriesGeneratedPassword reproduces issue #7 end to
// end: a legacy client.config.trusttunnel block had name but NO password, so the
// generated subscription outbound omitted password and the official client
// connected but every tunneled request failed with authorization failed.
//
// The backfill path must now generate a password and the subscription builder
// must carry it through to the outbound.
func TestTrustTunnelSubscriptionCarriesGeneratedPassword(t *testing.T) {
	initSubTestDB(t)

	// Legacy block: name only, no password — exactly the broken shape.
	client := model.Client{
		Name:      "issue7",
		Enable:    true,
		SubSecret: "issue7sub",
		Inbounds:  json.RawMessage(`[]`),
		Config:    json.RawMessage(`{"trusttunnel":{"name":"issue7"}}`),
	}
	if err := database.GetDB().Create(&client).Error; err != nil {
		t.Fatal(err)
	}
	inbound := model.Inbound{Type: "trusttunnel", Tag: "tt-issue7"}
	if err := database.GetDB().Create(&inbound).Error; err != nil {
		t.Fatal(err)
	}
	if err := (&service.ClientService{}).UpdateClientsOnInboundAdd(database.GetDB(), fmt.Sprintf("%d", client.Id), inbound.Id, "host"); err != nil {
		t.Fatal(err)
	}

	var gotClient model.Client
	if err := database.GetDB().First(&gotClient, client.Id).Error; err != nil {
		t.Fatal(err)
	}
	cfg := map[string]map[string]any{}
	if err := json.Unmarshal(gotClient.Config, &cfg); err != nil {
		t.Fatal(err)
	}
	pw, ok := cfg["trusttunnel"]["password"].(string)
	if !ok || strings.TrimSpace(pw) == "" {
		t.Fatalf("backfill did not generate trusttunnel password: %s", gotClient.Config)
	}

	// Now prove the subscription builder carries it to the outbound.
	tt := deliveredInbound(t, "trusttunnel", map[string]interface{}{
		"network": []interface{}{"tcp"}, "quic": true, "congestion_controller": "bbr",
	})
	outs, _, err := (&JsonService{}).getOutbounds(gotClient.Config, []*model.Inbound{tt})
	if err != nil {
		t.Fatal(err)
	}
	tto := findOutbound(*outs, "trusttunnel")
	if tto == nil {
		t.Fatal("no trusttunnel outbound in subscription")
	}
	if tto["username"] != "issue7" {
		t.Errorf("trusttunnel username not mapped: %v", tto)
	}
	if opw, ok := tto["password"].(string); !ok || strings.TrimSpace(opw) == "" {
		t.Errorf("trusttunnel subscription outbound missing password: %v", tto)
	}
}
