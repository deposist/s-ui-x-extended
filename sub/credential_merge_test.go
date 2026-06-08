package sub

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/deposist/s-ui-x-extended/database/model"
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
	if su == nil || su["key"] != "SHARED-KEY" {
		t.Errorf("sudoku should carry its shared key from out_json: %v", su)
	}

	// No server secret anywhere in the assembled outbounds.
	if hits := scanForbidden(*outs, "outbounds", false); len(hits) > 0 {
		body, _ := json.MarshalIndent(*outs, "", "  ")
		t.Errorf("subscription outbounds leaked server secret(s): %v\n%s", hits, body)
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
