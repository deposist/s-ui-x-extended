package util

import (
	"encoding/json"
	"testing"

	"github.com/deposist/s-ui-x-extended/database/model"
)

// fillInbound builds a minimal model.Inbound for FillOutJson: Options carries the
// listen_port (and any protocol fields) the builders read via MarshalFull.
func fillInbound(t *testing.T, typ string, options map[string]interface{}) *model.Inbound {
	t.Helper()
	options["listen_port"] = 443
	raw, err := json.Marshal(options)
	if err != nil {
		t.Fatalf("marshal options: %v", err)
	}
	// Mirror production: model.Inbound.UnmarshalJSON stores out_json as "null" when
	// absent (never empty), so FillOutJson always sees valid JSON.
	return &model.Inbound{Type: typ, Tag: typ + "-in", Options: raw, OutJson: json.RawMessage("null")}
}

func fillAndParse(t *testing.T, i *model.Inbound) map[string]interface{} {
	t.Helper()
	if err := FillOutJson(i, "example.com"); err != nil {
		t.Fatalf("FillOutJson(%s): %v", i.Type, err)
	}
	var out map[string]interface{}
	if err := json.Unmarshal(i.OutJson, &out); err != nil {
		t.Fatalf("unmarshal out_json(%s): %v", i.Type, err)
	}
	return out
}

// TestFillOutJsonBaseKeepsCommonFieldsOnly characterises the "base" no-op builder
// (http/socks/mixed/anytls): only the common fields survive, no protocol data.
func TestFillOutJsonBaseKeepsCommonFieldsOnly(t *testing.T) {
	out := fillAndParse(t, fillInbound(t, "http", map[string]interface{}{}))
	for _, k := range []string{"type", "tag", "server", "server_port"} {
		if _, ok := out[k]; !ok {
			t.Errorf("http out_json missing %q: %v", k, out)
		}
	}
	if out["type"] != "http" || out["server"] != "example.com" {
		t.Errorf("unexpected base out_json: %v", out)
	}
	if len(out) != 4 {
		t.Errorf("base out_json should keep exactly 4 common fields, got %d: %v", len(out), out)
	}
}

// TestFillOutJsonProtocolBuilderRuns characterises a protocol builder (vless):
// the builder copies the transport through.
func TestFillOutJsonProtocolBuilderRuns(t *testing.T) {
	out := fillAndParse(t, fillInbound(t, "vless", map[string]interface{}{
		"transport": map[string]interface{}{"type": "ws"},
	}))
	if out["type"] != "vless" {
		t.Errorf("vless out_json type: %v", out["type"])
	}
	tr, ok := out["transport"].(map[string]interface{})
	if !ok || tr["type"] != "ws" {
		t.Errorf("vless out_json should carry the transport: %v", out)
	}
}

// TestFillOutJsonWipesNotYetDeliveredTypes: mtproxy has no sing-box outbound and is
// Telegram-only, so it stays wiped to {} (the subscription len<5 guard drops it).
// mieru/sudoku/trusttunnel/ssh moved to real builders in Phase 1 (covered below).
func TestFillOutJsonWipesNotYetDeliveredTypes(t *testing.T) {
	for _, typ := range []string{"mtproxy"} {
		i := fillInbound(t, typ, map[string]interface{}{"some_server_field": "secret"})
		if err := FillOutJson(i, "example.com"); err != nil {
			t.Fatalf("FillOutJson(%s): %v", typ, err)
		}
		if got := string(i.OutJson); got != "{}" {
			t.Errorf("%s out_json should be wiped to {}, got %q", typ, got)
		}
	}
}

// --- Phase 1 builders -------------------------------------------------------

// TestSshOutCopiesNothingServerSide proves the ssh builder is a zero-size
// allow-list: the private host keys and server-only fields never reach out_json.
func TestSshOutCopiesNothingServerSide(t *testing.T) {
	out := fillAndParse(t, fillInbound(t, "ssh", map[string]interface{}{
		"host_key":       []interface{}{"PRIVATE-HOST-KEY"},
		"host_key_path":  []interface{}{"/etc/ssh/ssh_host_ed25519_key"},
		"server_version": "SSH-2.0-secret",
		"max_auth_tries": 3,
	}))
	for _, forbidden := range []string{"host_key", "host_key_path", "server_version", "max_auth_tries"} {
		if _, leaked := out[forbidden]; leaked {
			t.Errorf("ssh out_json leaked server field %q: %v", forbidden, out)
		}
	}
	if len(out) != 4 { // only type/tag/server/server_port
		t.Errorf("ssh out_json should carry only the 4 common fields, got %v", out)
	}
}

// TestMieruOutMapsListenPorts proves listen_ports -> server_ports and that only
// the allow-listed transport fields are copied.
func TestMieruOutMapsListenPorts(t *testing.T) {
	out := fillAndParse(t, fillInbound(t, "mieru", map[string]interface{}{
		"transport":              "TCP",
		"traffic_pattern":        "tls",
		"listen_ports":           []interface{}{"2090:2099"},
		"user_hint_is_mandatory": true,
	}))
	if out["transport"] != "TCP" || out["traffic_pattern"] != "tls" {
		t.Errorf("mieru out_json missing transport fields: %v", out)
	}
	sp, ok := out["server_ports"].([]interface{})
	if !ok || len(sp) != 1 || sp[0] != "2090:2099" {
		t.Errorf("mieru out_json should map listen_ports->server_ports: %v", out)
	}
	if _, leaked := out["listen_ports"]; leaked {
		t.Errorf("mieru out_json must not carry listen_ports: %v", out)
	}
	if _, leaked := out["user_hint_is_mandatory"]; leaked {
		t.Errorf("mieru out_json must not carry server-only user_hint_is_mandatory: %v", out)
	}
}

// TestTrustTunnelOutCopiesTransportNotOutOnly proves the transport client fields are
// copied but the out-direction-only fields (health_check/multiplex/username/password)
// are never copied from the inbound.
func TestTrustTunnelOutCopiesTransportNotOutOnly(t *testing.T) {
	out := fillAndParse(t, fillInbound(t, "trusttunnel", map[string]interface{}{
		"network":               []interface{}{"tcp", "udp"},
		"quic":                  true,
		"congestion_controller": "bbr",
		"bbr_profile":           "standard",
		"cwnd":                  32,
		"health_check":          true,
		"multiplex":             map[string]interface{}{"enabled": true},
		"username":              "server-side-should-not-copy",
		"password":              "server-side-should-not-copy",
	}))
	for _, want := range []string{"network", "quic", "congestion_controller", "bbr_profile", "cwnd"} {
		if _, ok := out[want]; !ok {
			t.Errorf("trusttunnel out_json missing %q: %v", want, out)
		}
	}
	for _, forbidden := range []string{"health_check", "multiplex", "username", "password"} {
		if _, leaked := out[forbidden]; leaked {
			t.Errorf("trusttunnel out_json must not copy out-only %q from inbound: %v", forbidden, out)
		}
	}
}

// TestSudokuOutBuildsHttpMaskMergePreservesCSide proves the keyless sudoku builder
// carries the client `key`, builds a nested http_mask from the flat inbound fields,
// preserves operator-set C-side host/multiplex, and never copies server-only fields.
func TestSudokuOutBuildsHttpMaskMergePreservesCSide(t *testing.T) {
	i := fillInbound(t, "sudoku", map[string]interface{}{
		"key":               "CLIENT-SHARED-KEY",
		"aead_method":       "chacha20-poly1305",
		"table_type":        "prefer_ascii",
		"disable_http_mask": false,
		"http_mask_mode":    "stream",
		"path_root":         "/cdn",
		"fallback":          "example.com:443",
		"handshake_timeout": 5,
	})
	// Operator-edited C-side out_json (host/multiplex are outbound-only).
	i.OutJson = json.RawMessage(`{"http_mask":{"host":"cdn.example.com","multiplex":"auto"}}`)

	out := fillAndParse(t, i)
	if out["key"] != "CLIENT-SHARED-KEY" || out["aead_method"] != "chacha20-poly1305" {
		t.Errorf("sudoku out_json missing client params: %v", out)
	}
	for _, forbidden := range []string{"fallback", "handshake_timeout"} {
		if _, leaked := out[forbidden]; leaked {
			t.Errorf("sudoku out_json must not copy server-only %q: %v", forbidden, out)
		}
	}
	hm, ok := out["http_mask"].(map[string]interface{})
	if !ok {
		t.Fatalf("sudoku out_json should have a nested http_mask object: %v", out)
	}
	if hm["enabled"] != true || hm["mode"] != "stream" || hm["path_root"] != "/cdn" {
		t.Errorf("http_mask should mirror the inbound: %v", hm)
	}
	if hm["host"] != "cdn.example.com" || hm["multiplex"] != "auto" {
		t.Errorf("http_mask should preserve C-side host/multiplex: %v", hm)
	}
}

// TestSudokuOutDisableHttpMask proves disable_http_mask=true -> http_mask.enabled=false.
func TestSudokuOutDisableHttpMask(t *testing.T) {
	out := fillAndParse(t, fillInbound(t, "sudoku", map[string]interface{}{
		"key":               "K",
		"disable_http_mask": true,
	}))
	hm, _ := out["http_mask"].(map[string]interface{})
	if hm == nil || hm["enabled"] != false {
		t.Errorf("disable_http_mask=true should set http_mask.enabled=false: %v", out)
	}
}

// TestFillOutJsonSkipsTransparentInbounds locks the early-return set
// (direct/tun/redirect/tproxy): out_json is left untouched.
func TestFillOutJsonSkipsTransparentInbounds(t *testing.T) {
	for _, typ := range []string{"direct", "tun", "redirect", "tproxy"} {
		sentinel := json.RawMessage(`{"untouched":true}`)
		i := &model.Inbound{Type: typ, Tag: typ, OutJson: sentinel}
		if err := FillOutJson(i, "example.com"); err != nil {
			t.Fatalf("FillOutJson(%s): %v", typ, err)
		}
		if string(i.OutJson) != `{"untouched":true}` {
			t.Errorf("%s out_json should be untouched, got %q", typ, string(i.OutJson))
		}
	}
}
