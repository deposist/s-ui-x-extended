package importxui

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
)

const warpXrayConfig = `{
  "outbounds": [
    {"tag": "proxy", "protocol": "trojan"},
    {"tag": "direct", "protocol": "freedom"},
    {"tag": "blocked", "protocol": "blackhole"},
    {"tag": "warp", "protocol": "wireguard", "settings": {
      "mtu": 1420,
      "secretKey": "AOrsZLfNlurdFcPYsr9VOPXiOffzbqowVhPmdNCTQ3g=",
      "address": ["172.16.0.2/32", "2606:4700:110:8907:6d4d:12ee:6c58:e9e/128"],
      "workers": 2,
      "domainStrategy": "ForceIPv4",
      "reserved": [227, 0, 191],
      "peers": [{
        "publicKey": "bmXOC+F1FxEMF9dyiK2H5/1SUtzH0JuVo51h2wPfgyo=",
        "allowedIPs": ["0.0.0.0/0", "::/0"],
        "endpoint": "engage.cloudflareclient.com:2408",
        "keepAlive": 0
      }]
    }}
  ],
  "routing": {"rules": [
    {"type": "field", "outboundTag": "blocked", "ip": ["geoip:private"]},
    {"type": "field", "outboundTag": "warp", "domain": ["geosite:google"]}
  ]}
}`

func TestMapXrayOutbounds_WARP(t *testing.T) {
	endpoints, _, targets, _ := mapXrayOutbounds(warpXrayConfig)

	if targets["blocked"] != rejectTarget || targets["direct"] != directOutboundTag || targets["warp"] != "warp" {
		t.Fatalf("targets = %v", targets)
	}
	if len(endpoints) != 1 {
		t.Fatalf("want 1 WARP endpoint, got %d", len(endpoints))
	}
	ep := endpoints[0]
	if ep.Type != "warp" || ep.Tag != "warp" {
		t.Fatalf("endpoint type/tag = %q/%q, want warp/warp", ep.Type, ep.Tag)
	}
	// Type "warp" must render as a sing-box "wireguard" endpoint.
	rawEp, err := json.Marshal(ep)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(rawEp), `"type":"wireguard"`) {
		t.Fatalf("warp endpoint did not render as wireguard: %s", rawEp)
	}

	var opts map[string]any
	if err := json.Unmarshal(ep.Options, &opts); err != nil {
		t.Fatal(err)
	}
	if opts["private_key"] != "AOrsZLfNlurdFcPYsr9VOPXiOffzbqowVhPmdNCTQ3g=" {
		t.Errorf("private_key = %v", opts["private_key"])
	}
	if opts["mtu"].(float64) != 1420 || opts["workers"].(float64) != 2 {
		t.Errorf("mtu/workers = %v/%v", opts["mtu"], opts["workers"])
	}
	peers := opts["peers"].([]any)
	if len(peers) != 1 {
		t.Fatalf("want 1 peer, got %d", len(peers))
	}
	peer := peers[0].(map[string]any)
	if peer["address"] != "engage.cloudflareclient.com" || peer["port"].(float64) != 2408 {
		t.Errorf("peer endpoint split wrong: %v:%v", peer["address"], peer["port"])
	}
	if peer["public_key"] != "bmXOC+F1FxEMF9dyiK2H5/1SUtzH0JuVo51h2wPfgyo=" {
		t.Errorf("peer public_key = %v", peer["public_key"])
	}
	reserved, _ := peer["reserved"].([]any)
	if len(reserved) != 3 || reserved[0].(float64) != 227 {
		t.Errorf("peer reserved = %v, want [227 0 191]", peer["reserved"])
	}
	allowed, _ := peer["allowed_ips"].([]any)
	if len(allowed) != 2 {
		t.Errorf("peer allowed_ips = %v", peer["allowed_ips"])
	}
}

func TestMapXrayRouting_WARPRuleTargetsEndpoint(t *testing.T) {
	_, _, targets, _ := mapXrayOutbounds(warpXrayConfig)
	mapped, warnings, mappedCount, manualCount := MapXrayRouting(warpXrayConfig, targets)

	if mappedCount != 2 || manualCount != 0 {
		t.Fatalf("mapped=%d manual=%d, want 2/0; warnings=%v", mappedCount, manualCount, warnings)
	}
	for _, w := range warnings {
		if strings.Contains(w, "warp") && strings.Contains(w, "manual review") {
			t.Errorf("warp rule should not require manual review: %q", w)
		}
	}
	route := mapped["route"].(map[string]any)
	rules := route["rules"].([]any)
	var warpRule map[string]any
	for _, r := range rules {
		rule := r.(map[string]any)
		if rule["outbound"] == "warp" {
			warpRule = rule
		}
	}
	if warpRule == nil {
		t.Fatalf("no rule targeting the warp endpoint: %#v", rules)
	}
	rs, _ := warpRule["rule_set"].([]string)
	if len(rs) != 1 || rs[0] != "geosite-google" {
		t.Errorf("warp rule rule_set = %v, want [geosite-google]", warpRule["rule_set"])
	}
}

func TestCreateNewEndpoints_IdempotentNoClobber(t *testing.T) {
	initCompatDest(t)
	db := database.GetDB()

	endpoints, _, _, _ := mapXrayOutbounds(warpXrayConfig)
	report := &Report{}
	if err := createNewEndpoints(db, endpoints, report); err != nil {
		t.Fatalf("first create: %v", err)
	}
	if report.Summary.Endpoints.Imported != 1 {
		t.Fatalf("first run imported=%d, want 1", report.Summary.Endpoints.Imported)
	}

	// Operator edits the WARP endpoint after import.
	if err := db.Model(&model.Endpoint{}).Where("tag = ?", "warp").
		Update("options", json.RawMessage(`{"mtu":9999}`)).Error; err != nil {
		t.Fatal(err)
	}

	// A second import / scheduled sync must NOT overwrite the existing endpoint.
	endpoints2, _, _, _ := mapXrayOutbounds(warpXrayConfig)
	report2 := &Report{}
	if err := createNewEndpoints(db, endpoints2, report2); err != nil {
		t.Fatalf("second create: %v", err)
	}
	if report2.Summary.Endpoints.Imported != 0 || report2.Summary.Endpoints.Skipped != 1 {
		t.Fatalf("second run imported=%d skipped=%d, want 0/1", report2.Summary.Endpoints.Imported, report2.Summary.Endpoints.Skipped)
	}
	var ep model.Endpoint
	if err := db.Where("tag = ?", "warp").First(&ep).Error; err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(ep.Options), "9999") {
		t.Fatalf("operator edit was clobbered: %s", ep.Options)
	}
	// And no duplicate row was created.
	var count int64
	db.Model(&model.Endpoint{}).Where("tag = ?", "warp").Count(&count)
	if count != 1 {
		t.Fatalf("duplicate warp endpoints: %d", count)
	}
}

func TestWarpEndpoint_DropsMalformedReserved(t *testing.T) {
	cfg := `{"outbounds":[{"tag":"warp","protocol":"wireguard","settings":{"secretKey":"S","reserved":[1,2,3,4],"peers":[{"publicKey":"P","endpoint":"h:2408"}]}}]}`
	endpoints, _, _, warnings := mapXrayOutbounds(cfg)
	if len(endpoints) != 1 {
		t.Fatalf("want 1 endpoint, got %d", len(endpoints))
	}
	var opts map[string]any
	if err := json.Unmarshal(endpoints[0].Options, &opts); err != nil {
		t.Fatal(err)
	}
	peer := opts["peers"].([]any)[0].(map[string]any)
	if _, ok := peer["reserved"]; ok {
		t.Errorf("malformed 4-byte reserved should have been dropped: %v", peer["reserved"])
	}
	var warned bool
	for _, w := range warnings {
		if strings.Contains(w, "reserved") {
			warned = true
		}
	}
	if !warned {
		t.Errorf("expected a warning about dropped reserved; got %v", warnings)
	}
}

// TestImport_RealXUIBackup_WARP verifies end-to-end against a real x-ui.db that
// the WARP wireguard outbound lands as an s-ui endpoint and its routing rule
// targets that endpoint.
//
//	IMPORT_XUI_REAL_DB="C:\\CheckErrorS-ui\\x-ui (6).db" go test ./database/importxui/ -run RealXUIBackup_WARP -v
func TestImport_RealXUIBackup_WARP(t *testing.T) {
	path := os.Getenv("IMPORT_XUI_REAL_DB")
	if path == "" {
		t.Skip("set IMPORT_XUI_REAL_DB to a real x-ui.db to run this test")
	}
	initCompatDest(t)

	plan, err := Plan(path, PlanOptions{Strategy: StrategyMerge, IncludeRouting: true, AdminMode: AdminModeSkip})
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}
	if _, err := Apply(path, *plan, ApplyOptions{}); err != nil {
		t.Fatalf("Apply failed: %v", err)
	}
	db := database.GetDB()

	var warp model.Endpoint
	if err := db.Where("type = ? AND tag = ?", "warp", "warp").First(&warp).Error; err != nil {
		t.Fatalf("warp endpoint not created: %v", err)
	}
	if !strings.Contains(string(warp.Options), "engage.cloudflareclient.com") {
		t.Errorf("warp endpoint missing cloudflare peer: %s", warp.Options)
	}
	t.Logf("warp endpoint options: %s", warp.Options)

	var cfg model.Setting
	if err := db.Where("key = ?", "config").First(&cfg).Error; err != nil {
		t.Fatalf("live config not written: %v", err)
	}
	if !strings.Contains(cfg.Value, `"outbound": "warp"`) && !strings.Contains(cfg.Value, `"outbound":"warp"`) {
		t.Errorf("live config has no rule targeting warp: %s", cfg.Value)
	}
	t.Logf("live config route: %s", cfg.Value)
}
