package importxui

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// outboundToMap marshals a mapped s-ui outbound (type+tag+options) into a flat
// map so tests can assert the sing-box fields.
func outboundToMap(t *testing.T, ob *model.Outbound) map[string]any {
	t.Helper()
	raw, err := json.Marshal(ob)
	if err != nil {
		t.Fatalf("marshal outbound: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("unmarshal outbound: %v", err)
	}
	return m
}

func mapAt(t *testing.T, m map[string]any, key string) map[string]any {
	t.Helper()
	v, ok := m[key].(map[string]any)
	if !ok {
		t.Fatalf("expected %q to be an object, got %T (%v)", key, m[key], m[key])
	}
	return v
}

// singleOutbound maps an Xray outbound expected to produce exactly one s-ui
// outbound and returns it (nil when none was produced).
func singleOutbound(ob xrayOutbound) (*model.Outbound, []string) {
	outs, w := outboundsFromXray(ob)
	if len(outs) == 0 {
		return nil, w
	}
	return &outs[0], w
}

func TestOutboundFromXray_VLESSReality(t *testing.T) {
	ob := xrayOutbound{
		Tag:            "inbams-p6updfmz",
		Protocol:       "vless",
		Settings:       json.RawMessage(`{"vnext":[{"address":"1.2.3.4","port":443,"users":[{"id":"uuid-1","flow":"xtls-rprx-vision","encryption":"none"}]}]}`),
		StreamSettings: json.RawMessage(`{"network":"tcp","security":"reality","realitySettings":{"publicKey":"PBK","shortId":"abcd","serverName":"example.com","fingerprint":"chrome"}}`),
	}
	out, warnings := singleOutbound(ob)
	if out == nil {
		t.Fatalf("expected an outbound, got nil; warnings=%v", warnings)
	}
	if out.Type != "vless" || out.Tag != "inbams-p6updfmz" {
		t.Fatalf("type/tag = %q/%q", out.Type, out.Tag)
	}
	m := outboundToMap(t, out)
	if m["server"] != "1.2.3.4" || m["server_port"].(float64) != 443 {
		t.Errorf("server/port = %v:%v", m["server"], m["server_port"])
	}
	if m["uuid"] != "uuid-1" || m["flow"] != "xtls-rprx-vision" {
		t.Errorf("uuid/flow = %v/%v", m["uuid"], m["flow"])
	}
	tls := mapAt(t, m, "tls")
	if tls["enabled"] != true || tls["server_name"] != "example.com" {
		t.Errorf("tls enabled/sni = %v/%v", tls["enabled"], tls["server_name"])
	}
	reality := mapAt(t, tls, "reality")
	if reality["public_key"] != "PBK" || reality["short_id"] != "abcd" {
		t.Errorf("reality pbk/sid = %v/%v", reality["public_key"], reality["short_id"])
	}
	utls := mapAt(t, tls, "utls")
	if utls["fingerprint"] != "chrome" {
		t.Errorf("utls fingerprint = %v", utls["fingerprint"])
	}
}

func TestOutboundFromXray_VMessWSTLS(t *testing.T) {
	ob := xrayOutbound{
		Tag:            "vmess-out",
		Protocol:       "vmess",
		Settings:       json.RawMessage(`{"vnext":[{"address":"v.example.com","port":8443,"users":[{"id":"vuuid","alterId":0,"security":"auto"}]}]}`),
		StreamSettings: json.RawMessage(`{"network":"ws","security":"tls","tlsSettings":{"serverName":"v.example.com","fingerprint":"chrome"},"wsSettings":{"path":"/ws","headers":{"Host":"v.example.com"}}}`),
	}
	out, warnings := singleOutbound(ob)
	if out == nil {
		t.Fatalf("expected an outbound, got nil; warnings=%v", warnings)
	}
	m := outboundToMap(t, out)
	if m["type"] != "vmess" || m["uuid"] != "vuuid" || m["security"] != "auto" {
		t.Errorf("type/uuid/security = %v/%v/%v", m["type"], m["uuid"], m["security"])
	}
	if m["alter_id"].(float64) != 0 {
		t.Errorf("alter_id = %v", m["alter_id"])
	}
	tr := mapAt(t, m, "transport")
	if tr["type"] != "ws" || tr["path"] != "/ws" {
		t.Errorf("transport type/path = %v/%v", tr["type"], tr["path"])
	}
	headers := mapAt(t, tr, "headers")
	if headers["Host"] != "v.example.com" {
		t.Errorf("transport host = %v", headers["Host"])
	}
	tls := mapAt(t, m, "tls")
	if tls["enabled"] != true || tls["server_name"] != "v.example.com" {
		t.Errorf("tls = %v", tls)
	}
}

func TestOutboundFromXray_TrojanGRPC(t *testing.T) {
	ob := xrayOutbound{
		Tag:            "trojan-out",
		Protocol:       "trojan",
		Settings:       json.RawMessage(`{"servers":[{"address":"t.example.com","port":443,"password":"tpw"}]}`),
		StreamSettings: json.RawMessage(`{"network":"grpc","security":"tls","tlsSettings":{"serverName":"t.example.com"},"grpcSettings":{"serviceName":"grpcsvc"}}`),
	}
	out, warnings := singleOutbound(ob)
	if out == nil {
		t.Fatalf("expected an outbound, got nil; warnings=%v", warnings)
	}
	m := outboundToMap(t, out)
	if m["type"] != "trojan" || m["password"] != "tpw" {
		t.Errorf("type/password = %v/%v", m["type"], m["password"])
	}
	tr := mapAt(t, m, "transport")
	if tr["type"] != "grpc" || tr["service_name"] != "grpcsvc" {
		t.Errorf("transport = %v", tr)
	}
}

func TestOutboundFromXray_Shadowsocks(t *testing.T) {
	ob := xrayOutbound{
		Tag:      "ss-out",
		Protocol: "shadowsocks",
		Settings: json.RawMessage(`{"servers":[{"address":"s.example.com","port":8388,"method":"aes-256-gcm","password":"sspw"}]}`),
	}
	out, warnings := singleOutbound(ob)
	if out == nil {
		t.Fatalf("expected an outbound, got nil; warnings=%v", warnings)
	}
	m := outboundToMap(t, out)
	if m["type"] != "shadowsocks" || m["method"] != "aes-256-gcm" || m["password"] != "sspw" {
		t.Errorf("ss = %v", m)
	}
	if _, ok := m["tls"]; ok {
		t.Errorf("shadowsocks must not carry a tls block: %v", m["tls"])
	}
	if _, ok := m["transport"]; ok {
		t.Errorf("shadowsocks must not carry a transport block: %v", m["transport"])
	}
}

func TestOutboundFromXray_SocksAndHTTPAuth(t *testing.T) {
	socks := xrayOutbound{
		Tag:      "socks-out",
		Protocol: "socks",
		Settings: json.RawMessage(`{"servers":[{"address":"127.0.0.1","port":1080,"users":[{"user":"u1","pass":"p1"}]}]}`),
	}
	out, _ := singleOutbound(socks)
	if out == nil {
		t.Fatal("expected socks outbound")
	}
	m := outboundToMap(t, out)
	if m["type"] != "socks" || m["version"] != "5" || m["username"] != "u1" || m["password"] != "p1" {
		t.Errorf("socks = %v", m)
	}

	http := xrayOutbound{
		Tag:      "http-out",
		Protocol: "http",
		Settings: json.RawMessage(`{"servers":[{"address":"127.0.0.1","port":8080,"users":[{"user":"hu","pass":"hp"}]}]}`),
	}
	out2, _ := singleOutbound(http)
	if out2 == nil {
		t.Fatal("expected http outbound")
	}
	m2 := outboundToMap(t, out2)
	if m2["type"] != "http" || m2["username"] != "hu" || m2["password"] != "hp" {
		t.Errorf("http = %v", m2)
	}
}

// TestOutboundFromXray_TransportWarningSaysOutbound guards the shared
// mapTransport warning attribution: a transport needing review on an OUTBOUND
// must be reported as "outbound <tag>", not "inbound <tag>".
func TestOutboundFromXray_TransportWarningSaysOutbound(t *testing.T) {
	ob := xrayOutbound{
		Tag:            "split-out",
		Protocol:       "vless",
		Settings:       json.RawMessage(`{"vnext":[{"address":"a.example.com","port":443,"users":[{"id":"u"}]}]}`),
		StreamSettings: json.RawMessage(`{"network":"splithttp","httpupgradeSettings":{"path":"/x"}}`),
	}
	out, warnings := singleOutbound(ob)
	if out == nil {
		t.Fatalf("expected an outbound, got nil; warnings=%v", warnings)
	}
	joined := strings.Join(warnings, "\n")
	if !strings.Contains(joined, "outbound split-out") {
		t.Errorf("transport warning should be attributed to the outbound, got %v", warnings)
	}
	if strings.Contains(joined, "inbound split-out") {
		t.Errorf("transport warning misattributed to inbound: %v", warnings)
	}
}

func TestOutboundFromXray_MissingServerSkipped(t *testing.T) {
	out, warnings := singleOutbound(xrayOutbound{Tag: "empty", Protocol: "trojan", Settings: json.RawMessage(`{}`)})
	if out != nil {
		t.Fatalf("expected nil for trojan with no servers, got %v", out)
	}
	if len(warnings) == 0 || !strings.Contains(warnings[0], "no server") {
		t.Errorf("expected a 'no server' warning, got %v", warnings)
	}
}

// TestOutboundsFromXray_MultiServerGroupAndMux checks that a multi-server proxy
// outbound becomes per-server members + a urltest group carrying the tag, that
// XUDP signals packet_encoding, and that Xray mux is reported (not silently
// enabled as the non-interoperable sing-box multiplex).
func TestOutboundsFromXray_MultiServerGroupAndMux(t *testing.T) {
	ob := xrayOutbound{
		Tag:      "chain",
		Protocol: "vless",
		Settings: json.RawMessage(`{"vnext":[
			{"address":"a.example.com","port":443,"users":[{"id":"u1"}]},
			{"address":"b.example.com","port":443,"users":[{"id":"u2"}]}
		]}`),
		Mux: json.RawMessage(`{"enabled":true,"concurrency":8,"xudpConcurrency":16}`),
	}
	outs, warnings := outboundsFromXray(ob)
	if len(outs) != 3 {
		t.Fatalf("want 3 outbounds (2 members + group), got %d: %#v", len(outs), outs)
	}
	var group *model.Outbound
	memberTags := map[string]bool{}
	for i := range outs {
		o := &outs[i]
		if o.Type == "urltest" {
			group = o
			continue
		}
		memberTags[o.Tag] = true
		if m := outboundToMap(t, o); m["packet_encoding"] != "xudp" {
			t.Errorf("member %s packet_encoding = %v, want xudp", o.Tag, m["packet_encoding"])
		}
	}
	if !memberTags["chain-0"] || !memberTags["chain-1"] {
		t.Errorf("member tags = %v, want chain-0 & chain-1", memberTags)
	}
	if group == nil || group.Tag != "chain" {
		t.Fatalf("expected urltest group tagged 'chain', got %#v", group)
	}
	if gobs, _ := outboundToMap(t, group)["outbounds"].([]any); len(gobs) != 2 {
		t.Errorf("group outbounds = %v, want 2 members", gobs)
	}
	if joined := strings.Join(warnings, "\n"); !strings.Contains(joined, "mux") {
		t.Errorf("expected a mux warning, got %v", warnings)
	}
	for i := range outs {
		if _, ok := outboundToMap(t, &outs[i])["multiplex"]; ok {
			t.Errorf("multiplex must not be enabled (Xray mux is not interoperable): %s", outs[i].Tag)
		}
	}
}

// TestMapXrayOutbounds_AllKinds checks the full classification: proxy outbounds
// become outbounds and route to their own tag, system outbounds become routing
// targets (freedom/blackhole/dns), and loopback/unknown protocols are flagged
// rather than dropped silently.
func TestMapXrayOutbounds_AllKinds(t *testing.T) {
	cfg := `{"outbounds":[
		{"tag":"dns-out","protocol":"dns"},
		{"tag":"loop","protocol":"loopback","settings":{"inboundTag":"x"}},
		{"tag":"hy","protocol":"hysteria"},
		{"tag":"vless-out","protocol":"vless","settings":{"vnext":[{"address":"a.example.com","port":443,"users":[{"id":"u"}]}]}},
		{"tag":"direct","protocol":"freedom"},
		{"tag":"blocked","protocol":"blackhole"}
	]}`
	endpoints, outbounds, targets, warnings := mapXrayOutbounds(cfg)

	if len(endpoints) != 0 {
		t.Errorf("want 0 endpoints, got %d", len(endpoints))
	}
	if len(outbounds) != 1 || outbounds[0].Tag != "vless-out" || outbounds[0].Type != "vless" {
		t.Fatalf("want 1 vless outbound, got %#v", outbounds)
	}
	if targets["direct"] != directOutboundTag || targets["blocked"] != rejectTarget {
		t.Errorf("freedom/blackhole targets = %v", targets)
	}
	if targets["dns-out"] != dnsHijackTarget {
		t.Errorf("dns target = %q, want %q", targets["dns-out"], dnsHijackTarget)
	}
	if targets["vless-out"] != "vless-out" {
		t.Errorf("proxy outbound must route to its own tag, got %q", targets["vless-out"])
	}
	if _, ok := targets["loop"]; ok {
		t.Errorf("loopback must not be a routing target")
	}
	joined := strings.Join(warnings, "\n")
	if !strings.Contains(joined, "loopback has no s-ui equivalent") {
		t.Errorf("expected loopback warning, got %v", warnings)
	}
	if !strings.Contains(joined, `protocol "hysteria" has no automatic`) {
		t.Errorf("expected hysteria warning, got %v", warnings)
	}
}

// TestMapXrayRouting_DNSHijack verifies a rule targeting the dns outbound
// becomes a sing-box hijack-dns action rule rather than an invalid dns outbound.
func TestMapXrayRouting_DNSHijack(t *testing.T) {
	cfg := `{
		"outbounds":[{"tag":"dns-out","protocol":"dns"}],
		"routing":{"rules":[{"type":"field","outboundTag":"dns-out","domain":["geosite:category-ads-all"]}]}
	}`
	_, _, targets, _ := mapXrayOutbounds(cfg)
	mapped, _, mappedCount, manualCount := MapXrayRouting(cfg, targets)
	if mappedCount != 1 || manualCount != 0 {
		t.Fatalf("mapped=%d manual=%d, want 1/0", mappedCount, manualCount)
	}
	route := mapped["route"].(map[string]any)
	rules := route["rules"].([]any)
	if len(rules) != 1 {
		t.Fatalf("want 1 rule, got %d", len(rules))
	}
	rule := rules[0].(map[string]any)
	if rule["action"] != "hijack-dns" {
		t.Errorf("dns rule action = %v, want hijack-dns", rule["action"])
	}
	if _, ok := rule["outbound"]; ok {
		t.Errorf("hijack-dns rule must not set outbound: %v", rule)
	}
}

// TestPlanRoutingDisabledNotice_WarnsAboutOutbounds verifies that disabling
// routing import no longer silently drops proxy outbounds: a warning-only plan
// item is surfaced so the operator knows to enable routing to migrate them.
func TestPlanRoutingDisabledNotice_WarnsAboutOutbounds(t *testing.T) {
	dir := makeImportXUITempDir(t)
	srcPath := filepath.Join(dir, "x-ui.db")
	buildCompatSource(t, forkVariant, srcPath)

	db, err := gorm.Open(sqlite.Open(srcPath), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	xray := `{"outbounds":[{"tag":"chain","protocol":"vless","settings":{"vnext":[{"address":"a.example.com","port":443,"users":[{"id":"u"}]}]}}]}`
	if err := db.Exec("INSERT INTO settings(key, value) VALUES(?, ?)", "xrayConfig", xray).Error; err != nil {
		t.Fatal(err)
	}
	if sqlDB, err := db.DB(); err == nil {
		_ = sqlDB.Close()
	}

	src, err := openSource(srcPath)
	if err != nil {
		t.Fatalf("openSource: %v", err)
	}
	defer src.close()

	plan := &MigrationPlan{}
	if err := planRoutingDisabledNotice(context.Background(), src, plan); err != nil {
		t.Fatalf("planRoutingDisabledNotice: %v", err)
	}
	found := false
	for _, item := range plan.Items {
		if item.Kind != KindRouting || item.Action != ActionSkip {
			continue
		}
		for _, w := range item.Warnings {
			if strings.Contains(w, "routing import is disabled") && strings.Contains(w, "proxy outbound") {
				found = true
			}
		}
	}
	if !found {
		t.Fatalf("expected a routing-disabled notice mentioning proxy outbounds; items=%#v", plan.Items)
	}
}

// TestCreateNewOutbounds_IdempotentNoClobber mirrors the WARP endpoint guard:
// a re-import must not overwrite an operator-tuned outbound of the same tag.
func TestCreateNewOutbounds_IdempotentNoClobber(t *testing.T) {
	initCompatDest(t)
	db := database.GetDB()

	cfg := `{"outbounds":[{"tag":"chain-out","protocol":"trojan","settings":{"servers":[{"address":"t.example.com","port":443,"password":"tpw"}]},"streamSettings":{"network":"tcp","security":"tls","tlsSettings":{"serverName":"t.example.com"}}}]}`
	_, outbounds, _, _ := mapXrayOutbounds(cfg)
	if len(outbounds) != 1 {
		t.Fatalf("want 1 outbound, got %d", len(outbounds))
	}
	report := &Report{}
	if err := createNewOutbounds(db, outbounds, report); err != nil {
		t.Fatalf("first create: %v", err)
	}
	if report.Summary.Outbounds.Imported != 1 {
		t.Fatalf("first run imported=%d, want 1", report.Summary.Outbounds.Imported)
	}

	// Operator edits the outbound after import.
	if err := db.Model(&model.Outbound{}).Where("tag = ?", "chain-out").
		Update("options", json.RawMessage(`{"server":"edited"}`)).Error; err != nil {
		t.Fatal(err)
	}

	_, outbounds2, _, _ := mapXrayOutbounds(cfg)
	report2 := &Report{}
	if err := createNewOutbounds(db, outbounds2, report2); err != nil {
		t.Fatalf("second create: %v", err)
	}
	if report2.Summary.Outbounds.Imported != 0 || report2.Summary.Outbounds.Skipped != 1 {
		t.Fatalf("second run imported=%d skipped=%d, want 0/1", report2.Summary.Outbounds.Imported, report2.Summary.Outbounds.Skipped)
	}
	var ob model.Outbound
	if err := db.Where("tag = ?", "chain-out").First(&ob).Error; err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(ob.Options), "edited") {
		t.Fatalf("operator edit was clobbered: %s", ob.Options)
	}
	var count int64
	db.Model(&model.Outbound{}).Where("tag = ?", "chain-out").Count(&count)
	if count != 1 {
		t.Fatalf("duplicate outbounds: %d", count)
	}
}
