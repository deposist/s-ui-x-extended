package importxui

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// firstMappedRule runs MapXrayRouting over a single-rule config and returns the
// produced sing-box rule (or nil) plus counts.
func firstMappedRule(t *testing.T, raw string) (map[string]any, int, int, []string) {
	t.Helper()
	mapped, warnings, mappedCount, manualCount := MapXrayRouting(raw, map[string]string{"out": "direct"})
	route := mapped["route"].(map[string]any)
	rules := route["rules"].([]any)
	var rule map[string]any
	if len(rules) > 0 {
		rule = rules[0].(map[string]any)
	}
	return rule, mappedCount, manualCount, warnings
}

func TestRoutingMatchers_PortsNetworkProtocol(t *testing.T) {
	raw := `{"routing":{"rules":[{"outboundTag":"out","port":"443,1000-2000","network":"tcp,udp","protocol":["tls","http"]}]}}`
	rule, mapped, manual, _ := firstMappedRule(t, raw)
	if mapped != 1 || manual != 0 {
		t.Fatalf("mapped=%d manual=%d, want 1/0", mapped, manual)
	}
	ports, _ := rule["port"].([]int)
	if len(ports) != 1 || ports[0] != 443 {
		t.Errorf("port = %v, want [443]", rule["port"])
	}
	pr := rule["port_range"].([]string)
	if len(pr) != 1 || pr[0] != "1000:2000" {
		t.Errorf("port_range = %v, want [1000:2000]", rule["port_range"])
	}
	nets := rule["network"].([]string)
	if len(nets) != 2 || nets[0] != "tcp" || nets[1] != "udp" {
		t.Errorf("network = %v, want [tcp udp]", rule["network"])
	}
	protos := rule["protocol"].([]string)
	if len(protos) != 2 || protos[0] != "tls" {
		t.Errorf("protocol = %v", rule["protocol"])
	}
	if _, ok := rule["outbound"]; !ok {
		t.Errorf("rule missing outbound: %#v", rule)
	}
}

func TestRoutingMatchers_SourceInboundUser(t *testing.T) {
	raw := `{"routing":{"rules":[{"outboundTag":"out","source":["geoip:private","10.0.0.0/8"],"sourcePort":"50000-60000","inboundTag":["in-1","in-2"],"user":["alice@example.com"]}]}}`
	rule, mapped, manual, _ := firstMappedRule(t, raw)
	if mapped != 1 || manual != 0 {
		t.Fatalf("mapped=%d manual=%d, want 1/0", mapped, manual)
	}
	// geoip:private (source) becomes a remote geoip rule set matched on the source.
	if rs, _ := rule["rule_set"].([]string); len(rs) != 1 || rs[0] != "geoip-private" {
		t.Errorf("rule_set = %v, want [geoip-private]", rule["rule_set"])
	}
	if rule["rule_set_ip_cidr_match_source"] != true {
		t.Errorf("rule_set_ip_cidr_match_source = %v, want true", rule["rule_set_ip_cidr_match_source"])
	}
	if sc := rule["source_ip_cidr"].([]string); len(sc) != 1 || sc[0] != "10.0.0.0/8" {
		t.Errorf("source_ip_cidr = %v", rule["source_ip_cidr"])
	}
	if spr := rule["source_port_range"].([]string); len(spr) != 1 || spr[0] != "50000:60000" {
		t.Errorf("source_port_range = %v", rule["source_port_range"])
	}
	if inb := rule["inbound"].([]string); len(inb) != 2 || inb[0] != "in-1" {
		t.Errorf("inbound = %v", rule["inbound"])
	}
	if au := rule["auth_user"].([]string); len(au) != 1 || au[0] != "alice@example.com" {
		t.Errorf("auth_user = %v", rule["auth_user"])
	}
}

func TestRoutingMatchers_ExtGeoipAndBareIP(t *testing.T) {
	// Regression: an Xray external geoip reference (ext:<file>:<code>) and a bare
	// IP must never land in ip_cidr verbatim — sing-box's ParsePrefix rejects a
	// value without a mask, which used to make the whole migrated config fail to
	// load ("ipcidr: parse: no '/'").
	raw := `{"routing":{"rules":[{"outboundTag":"out","ip":["ext:geoip_RU.dat:ru","8.8.8.8","1.2.3.0/24","not-an-ip"]}]}}`
	rule, mapped, manual, warnings := firstMappedRule(t, raw)
	if mapped != 1 || manual != 0 {
		t.Fatalf("mapped=%d manual=%d, want 1/0", mapped, manual)
	}
	// ext:geoip_RU.dat:ru -> geoip-ru rule set (not ip_cidr).
	if rs, _ := rule["rule_set"].([]string); len(rs) != 1 || rs[0] != "geoip-ru" {
		t.Errorf("rule_set = %v, want [geoip-ru]", rule["rule_set"])
	}
	// bare IP normalised to /32, CIDR kept as-is, the garbage value dropped.
	ipc, _ := rule["ip_cidr"].([]string)
	if len(ipc) != 2 {
		t.Fatalf("ip_cidr = %v, want exactly the two valid prefixes", ipc)
	}
	want := map[string]bool{"8.8.8.8/32": true, "1.2.3.0/24": true}
	for _, c := range ipc {
		if !strings.Contains(c, "/") {
			t.Errorf("ip_cidr entry %q has no mask — sing-box would refuse to start", c)
		}
		if !want[c] {
			t.Errorf("unexpected ip_cidr entry %q", c)
		}
	}
	hasWarn := func(sub string) bool {
		for _, w := range warnings {
			if strings.Contains(w, sub) {
				return true
			}
		}
		return false
	}
	if !hasWarn("ext:geoip_RU.dat:ru") {
		t.Errorf("warnings %v should note the external geoip mapping", warnings)
	}
	if !hasWarn("not-an-ip") {
		t.Errorf("warnings %v should note the dropped non-IP value", warnings)
	}
}

func TestRoutingMatchers_DomainPrefixes(t *testing.T) {
	raw := `{"routing":{"rules":[{"outboundTag":"out","domain":["full:exact.com","domain:sub.com","keyword:ads","regexp:.*\\.evil\\.com","bare.com"]}]}}`
	rule, mapped, manual, _ := firstMappedRule(t, raw)
	if mapped != 1 || manual != 0 {
		t.Fatalf("mapped=%d manual=%d, want 1/0", mapped, manual)
	}
	if d := rule["domain"].([]string); len(d) != 1 || d[0] != "exact.com" {
		t.Errorf("domain = %v, want [exact.com]", rule["domain"])
	}
	ds := rule["domain_suffix"].([]string)
	if len(ds) != 2 {
		t.Errorf("domain_suffix = %v, want [sub.com bare.com]", ds)
	}
	if dk := rule["domain_keyword"].([]string); len(dk) != 1 || dk[0] != "ads" {
		t.Errorf("domain_keyword = %v", rule["domain_keyword"])
	}
	if dr := rule["domain_regex"].([]string); len(dr) != 1 {
		t.Errorf("domain_regex = %v", rule["domain_regex"])
	}
}

// TestRouting_BlackholeBecomesRejectAction is a regression test: a rule routing
// to an Xray blackhole outbound must become a reject *action*, not a dangling
// outbound:"block" reference. The migration never creates an outbound tagged
// "block" and sing-box no longer auto-provides one, so the reference would make
// sing-box drop the matched connection at route time with "outbound not found:
// block".
func TestRouting_BlackholeBecomesRejectAction(t *testing.T) {
	raw := `{"outbounds":[{"tag":"blocked","protocol":"blackhole"}],` +
		`"routing":{"rules":[{"outboundTag":"blocked","domain":["full:ads.example"]}]}}`
	_, _, targets, _ := mapXrayOutbounds(raw)
	mapped, _, mappedCount, manualCount := MapXrayRouting(raw, targets)
	if mappedCount != 1 || manualCount != 0 {
		t.Fatalf("mapped=%d manual=%d", mappedCount, manualCount)
	}
	rule := mapped["route"].(map[string]any)["rules"].([]any)[0].(map[string]any)
	if rule["action"] != "reject" {
		t.Errorf("blackhole rule action = %v, want reject", rule["action"])
	}
	if _, has := rule["outbound"]; has {
		t.Errorf("reject rule must not carry an outbound: %#v", rule)
	}
}

// TestRouting_ProxyTaggedBlockKeepsRouting guards the reject sentinel against a
// tag collision: a real proxy outbound legitimately named "block" must keep
// routing to itself (outbound:"block"), not be turned into a reject action. This
// fails if rejectTarget is the literal string "block".
func TestRouting_ProxyTaggedBlockKeepsRouting(t *testing.T) {
	raw := `{"outbounds":[{"tag":"block","protocol":"vless","settings":{"vnext":[{"address":"a.example.com","port":443,"users":[{"id":"u"}]}]}}],` +
		`"routing":{"rules":[{"outboundTag":"block","domain":["full:x.example"]}]}}`
	_, _, targets, _ := mapXrayOutbounds(raw)
	mapped, _, mappedCount, manualCount := MapXrayRouting(raw, targets)
	if mappedCount != 1 || manualCount != 0 {
		t.Fatalf("mapped=%d manual=%d", mappedCount, manualCount)
	}
	rule := mapped["route"].(map[string]any)["rules"].([]any)[0].(map[string]any)
	if rule["outbound"] != "block" {
		t.Errorf("proxy named \"block\" must keep routing to itself, got %#v", rule)
	}
	if _, isAction := rule["action"]; isAction {
		t.Errorf("a rule routing to a real proxy must not become an action: %#v", rule)
	}
}

// TestEnsureDirectOutbound_SkipsWhenSeededInDB is a regression test: when a
// direct outbound already exists in the DB (the s-ui default InitDB seeds), the
// migration must NOT inject a duplicate — otherwise createNewOutbounds reports a
// misleading "outbound \"direct\" already exists; left unchanged" skip on every
// routing import that references direct.
func TestEnsureDirectOutbound_SkipsWhenSeededInDB(t *testing.T) {
	initCompatDest(t)
	db := database.GetDB()
	mapped := map[string]any{"route": map[string]any{"rules": []any{map[string]any{"outbound": directOutboundTag}}}}

	// Seed present (the normal destination): nothing must be injected.
	if got := ensureDirectOutbound(db, nil, mapped); len(got) != 0 {
		t.Errorf("with a seeded direct outbound, ensureDirectOutbound must inject nothing, got %d", len(got))
	}
	// Seed removed: a direct outbound must be injected so the reference resolves.
	if err := db.Where("tag = ?", directOutboundTag).Delete(&model.Outbound{}).Error; err != nil {
		t.Fatal(err)
	}
	got := ensureDirectOutbound(db, nil, mapped)
	if len(got) != 1 || got[0].Tag != directOutboundTag || got[0].Type != directOutboundTag {
		t.Errorf("with no direct outbound, ensureDirectOutbound must inject one, got %#v", got)
	}
}

// TestApply_DNSOnlyConfigIsNotSkipped is a regression test for a pre-existing
// bug: a source whose only migratable content is DNS (no routing rules, no proxy
// outbounds, no endpoints) was marked ActionSkip and its DNS silently dropped.
func TestApply_DNSOnlyConfigIsNotSkipped(t *testing.T) {
	initCompatDest(t)
	dir := makeImportXUITempDir(t)
	src := filepath.Join(dir, "x-ui.db")
	buildCompatSource(t, forkVariant, src)

	db, err := gorm.Open(sqlite.Open(src), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	// Only freedom/blackhole system outbounds (which become routing targets, not
	// s-ui outbounds) + a DNS block; no routing.rules, so the routing item used
	// to be skipped and the DNS dropped.
	xray := `{"outbounds":[{"tag":"direct","protocol":"freedom"},{"tag":"blocked","protocol":"blackhole"}],` +
		`"dns":{"servers":["1.1.1.1"]}}`
	if err := db.Exec("INSERT INTO settings(key, value) VALUES(?, ?)", "xrayConfig", xray).Error; err != nil {
		t.Fatal(err)
	}
	if sqlDB, err := db.DB(); err == nil {
		_ = sqlDB.Close()
	}

	plan, err := Plan(src, PlanOptions{Strategy: StrategyMerge, AdminMode: AdminModeSkip, IncludeRouting: true})
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	for _, it := range plan.Items {
		if it.Kind == KindRouting && it.Action == ActionSkip {
			t.Fatalf("dns-only config must not be ActionSkip (its DNS would be silently dropped)")
		}
	}
	if _, err := Apply(src, *plan, ApplyOptions{}); err != nil {
		t.Fatalf("apply: %v", err)
	}

	dest := database.GetDB()
	var cfg model.Setting
	if err := dest.Where("key = ?", "config").First(&cfg).Error; err != nil {
		t.Fatal(err)
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(cfg.Value), &parsed); err != nil {
		t.Fatal(err)
	}
	dns, _ := parsed["dns"].(map[string]any)
	if servers, _ := dns["servers"].([]any); len(servers) == 0 {
		t.Errorf("dns servers were not merged into the live config: %v", dns)
	}
}

// TestApply_BlackholeRejectAndDirectResolves is a regression test for the
// sing-box 1.11+ migration: after a full import the live config must contain no
// dangling outbound references. A blackhole outbound becomes a reject action
// (never outbound:"block"), a freedom outbound yields a real direct outbound so
// outbound:"direct" resolves, and every route rule that routes to an outbound
// must point at an outbound/endpoint that actually exists.
func TestApply_BlackholeRejectAndDirectResolves(t *testing.T) {
	initCompatDest(t)
	// InitDB seeds a {Type:"direct",Tag:"direct"} outbound; delete it so the
	// post-import assertion that a direct outbound exists proves ensureDirectOutbound
	// injected one, instead of passing vacuously on the seed.
	if err := database.GetDB().Where("tag = ?", directOutboundTag).Delete(&model.Outbound{}).Error; err != nil {
		t.Fatal(err)
	}
	dir := makeImportXUITempDir(t)
	src := filepath.Join(dir, "x-ui.db")
	buildCompatSource(t, forkVariant, src)

	db, err := gorm.Open(sqlite.Open(src), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	xray := `{"outbounds":[` +
		`{"tag":"direct","protocol":"freedom"},` +
		`{"tag":"blocked","protocol":"blackhole"},` +
		`{"tag":"proxy","protocol":"vless","settings":{"vnext":[{"address":"a.example.com","port":443,"users":[{"id":"u"}]}]}}` +
		`],"routing":{"rules":[` +
		`{"outboundTag":"blocked","ip":["geoip:cn"]},` +
		`{"outboundTag":"direct","domain":["geosite:google"]},` +
		`{"outboundTag":"proxy","domain":["full:netflix.com"]}` +
		`]}}`
	if err := db.Exec("INSERT INTO settings(key, value) VALUES(?, ?)", "xrayConfig", xray).Error; err != nil {
		t.Fatal(err)
	}
	if sqlDB, err := db.DB(); err == nil {
		_ = sqlDB.Close()
	}

	plan, err := Plan(src, PlanOptions{Strategy: StrategyMerge, AdminMode: AdminModeSkip, IncludeRouting: true})
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if _, err := Apply(src, *plan, ApplyOptions{}); err != nil {
		t.Fatalf("apply: %v", err)
	}

	dest := database.GetDB()
	var cfg model.Setting
	if err := dest.Where("key = ?", "config").First(&cfg).Error; err != nil {
		t.Fatal(err)
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(cfg.Value), &parsed); err != nil {
		t.Fatalf("merged live config is not valid JSON: %v", err)
	}
	rules := parsed["route"].(map[string]any)["rules"].([]any)

	var sawReject, sawDirectRule bool
	for _, r := range rules {
		m, ok := r.(map[string]any)
		if !ok {
			continue
		}
		if m["outbound"] == "block" {
			t.Errorf("migrated rule still references the removed block outbound: %#v", m)
		}
		if m["action"] == "reject" {
			sawReject = true
			if _, has := m["outbound"]; has {
				t.Errorf("reject rule must not carry an outbound: %#v", m)
			}
		}
		if m["outbound"] == "direct" {
			sawDirectRule = true
		}
	}
	if !sawReject {
		t.Errorf("blackhole rule was not migrated to a reject action: %v", rules)
	}
	if !sawDirectRule {
		t.Errorf("freedom rule did not produce outbound:direct: %v", rules)
	}

	// Collect the outbound/endpoint tags that exist after import.
	tags := map[string]bool{}
	var obs []model.Outbound
	if err := dest.Model(model.Outbound{}).Scan(&obs).Error; err != nil {
		t.Fatal(err)
	}
	for _, o := range obs {
		tags[o.Tag] = true
	}
	var eps []model.Endpoint
	if err := dest.Model(model.Endpoint{}).Scan(&eps).Error; err != nil {
		t.Fatal(err)
	}
	for _, e := range eps {
		tags[e.Tag] = true
	}
	if !tags["direct"] {
		t.Errorf("no direct outbound was created; existing tags = %v", tags)
	}

	// No dangling references: every rule that routes to an outbound must point at
	// an outbound/endpoint that exists.
	for _, r := range rules {
		m, ok := r.(map[string]any)
		if !ok {
			continue
		}
		ob, ok := m["outbound"].(string)
		if !ok || ob == "" {
			continue
		}
		if !tags[ob] {
			t.Errorf("route rule references outbound %q with no matching outbound/endpoint; tags = %v", ob, tags)
		}
	}
}

// TestApply_RoutingMergesIntoLiveConfig drives the full Plan/Apply path and
// verifies routing is merged into the live `config` setting (not a dead side
// setting): the default rules are preserved, the migrated rule is appended, and
// geosite/geoip become valid remote rule sets with a URL.
func TestApply_RoutingMergesIntoLiveConfig(t *testing.T) {
	initCompatDest(t)
	dir := makeImportXUITempDir(t)
	src := filepath.Join(dir, "x-ui.db")
	buildCompatSource(t, forkVariant, src)

	db, err := gorm.Open(sqlite.Open(src), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	xray := `{"outbounds":[{"tag":"direct","protocol":"freedom"},{"tag":"blocked","protocol":"blackhole"}],` +
		`"routing":{"rules":[{"outboundTag":"direct","domain":["geosite:google"]},{"outboundTag":"blocked","ip":["geoip:cn"]}]},` +
		`"dns":{"servers":["1.1.1.1"]}}`
	if err := db.Exec("INSERT INTO settings(key, value) VALUES(?, ?)", "xrayConfig", xray).Error; err != nil {
		t.Fatal(err)
	}
	if sqlDB, err := db.DB(); err == nil {
		_ = sqlDB.Close()
	}

	plan, err := Plan(src, PlanOptions{Strategy: StrategyMerge, AdminMode: AdminModeSkip, IncludeRouting: true})
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if _, err := Apply(src, *plan, ApplyOptions{}); err != nil {
		t.Fatalf("apply: %v", err)
	}

	dest := database.GetDB()
	var cfg model.Setting
	if err := dest.Where("key = ?", "config").First(&cfg).Error; err != nil {
		t.Fatal(err)
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(cfg.Value), &parsed); err != nil {
		t.Fatalf("merged live config is not valid JSON: %v", err)
	}
	route := parsed["route"].(map[string]any)
	if rules := route["rules"].([]any); len(rules) < 4 {
		t.Errorf("expected default rules preserved + 2 migrated (>=4), got %d: %v", len(rules), rules)
	}
	ruleSets := route["rule_set"].([]any)
	var gs map[string]any
	for _, rs := range ruleSets {
		if m, ok := rs.(map[string]any); ok && m["tag"] == "geosite-google" {
			gs = m
		}
	}
	if gs == nil {
		t.Fatalf("geosite-google rule set not merged into live config: %v", ruleSets)
	}
	if u, _ := gs["url"].(string); !strings.Contains(u, ".srs") {
		t.Errorf("geosite rule set missing a usable url: %v", gs)
	}
	if gs["type"] != "remote" || gs["download_detour"] != "direct" {
		t.Errorf("geosite rule set is not a valid remote rule set: %v", gs)
	}
	dns := parsed["dns"].(map[string]any)
	if servers, _ := dns["servers"].([]any); len(servers) == 0 {
		t.Errorf("dns server not merged into live config: %v", dns)
	}

	// Re-import must be idempotent: the scheduled sync re-applies routing every
	// run, so applying the same plan again must not grow rules, rule sets, or DNS
	// servers (sing-box also rejects a config with duplicate rule-set tags).
	firstRuleCount := len(route["rules"].([]any))
	firstRuleSetCount := len(route["rule_set"].([]any))
	firstDNSServerCount := len(dns["servers"].([]any))
	if _, err := Apply(src, *plan, ApplyOptions{}); err != nil {
		t.Fatalf("second apply: %v", err)
	}
	var cfg2 model.Setting
	if err := dest.Where("key = ?", "config").First(&cfg2).Error; err != nil {
		t.Fatal(err)
	}
	var parsed2 map[string]any
	if err := json.Unmarshal([]byte(cfg2.Value), &parsed2); err != nil {
		t.Fatal(err)
	}
	route2 := parsed2["route"].(map[string]any)
	dns2 := parsed2["dns"].(map[string]any)
	if got := len(route2["rules"].([]any)); got != firstRuleCount {
		t.Errorf("re-import grew route rules from %d to %d (not idempotent)", firstRuleCount, got)
	}
	if got := len(route2["rule_set"].([]any)); got != firstRuleSetCount {
		t.Errorf("re-import grew rule_set from %d to %d (not idempotent)", firstRuleSetCount, got)
	}
	if got := len(dns2["servers"].([]any)); got != firstDNSServerCount {
		t.Errorf("re-import grew dns servers from %d to %d (not idempotent)", firstDNSServerCount, got)
	}
}

func TestRoutingMatchers_AttrsAndNoMatcherAreManual(t *testing.T) {
	// attrs -> whole rule manual (cannot represent in sing-box).
	rawAttrs := `{"routing":{"rules":[{"outboundTag":"out","attrs":{":method":"GET"},"domain":["full:x.com"]}]}}`
	_, mapped, manual, warnings := firstMappedRule(t, rawAttrs)
	if mapped != 0 || manual != 1 {
		t.Fatalf("attrs rule: mapped=%d manual=%d, want 0/1", mapped, manual)
	}
	if len(warnings) == 0 {
		t.Error("attrs rule should warn")
	}

	// outboundTag resolvable but no matchers -> manual (cannot be a sing-box rule).
	rawEmpty := `{"routing":{"rules":[{"outboundTag":"out"}]}}`
	_, mapped2, manual2, _ := firstMappedRule(t, rawEmpty)
	if mapped2 != 0 || manual2 != 1 {
		t.Fatalf("empty-matcher rule: mapped=%d manual=%d, want 0/1", mapped2, manual2)
	}
}
