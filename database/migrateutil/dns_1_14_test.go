package migrateutil

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x-extended/database/model"
	"gorm.io/gorm"
)

func createSettingTable(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.Exec(`
CREATE TABLE settings (
	id integer PRIMARY KEY AUTOINCREMENT,
	key text,
	value text
)`).Error; err != nil {
		t.Fatal(err)
	}
}

func setConfigBlob(t *testing.T, db *gorm.DB, config string) {
	t.Helper()
	if err := db.Exec("INSERT INTO settings(key, value) VALUES('config', ?)", config).Error; err != nil {
		t.Fatal(err)
	}
}

func getConfigBlob(t *testing.T, db *gorm.DB) map[string]any {
	t.Helper()
	var row model.Setting
	if err := db.Where("key = ?", "config").First(&row).Error; err != nil {
		t.Fatal(err)
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(row.Value), &parsed); err != nil {
		t.Fatal(err)
	}
	return parsed
}

func TestMigrateDNS114RemovesIndependentCache(t *testing.T) {
	db := openLegacyInboundMigrationTestDB(t)
	createSettingTable(t, db)
	setConfigBlob(t, db, `{"dns": {"servers": [], "rules": [], "independent_cache": true}}`)

	if err := MigrateDNS114(db); err != nil {
		t.Fatalf("migration failed: %v", err)
	}
	got := getConfigBlob(t, db)
	dns := got["dns"].(map[string]any)
	if _, present := dns["independent_cache"]; present {
		t.Fatalf("independent_cache not removed: %v", dns)
	}
	if _, present := dns["servers"]; !present {
		t.Fatalf("servers lost: %v", dns)
	}
}

func TestMigrateDNS114StoreRDRCToStoreDNS(t *testing.T) {
	db := openLegacyInboundMigrationTestDB(t)
	createSettingTable(t, db)
	setConfigBlob(t, db, `{"experimental": {"cache_file": {"enabled": true, "store_rdrc": true, "rdrc_timeout": "168h"}}}`)

	if err := MigrateDNS114(db); err != nil {
		t.Fatalf("migration failed: %v", err)
	}
	got := getConfigBlob(t, db)
	cf := got["experimental"].(map[string]any)["cache_file"].(map[string]any)
	if _, present := cf["store_rdrc"]; present {
		t.Fatalf("store_rdrc not removed: %v", cf)
	}
	if cf["store_dns"] != true {
		t.Fatalf("store_dns=%v, want true", cf["store_dns"])
	}
	if _, present := cf["rdrc_timeout"]; present {
		t.Fatalf("rdrc_timeout not removed: %v", cf)
	}
	if cf["enabled"] != true {
		t.Fatalf("enabled lost")
	}
}

func TestMigrateDNS114StoreRDRCRespectsExistingStoreDNS(t *testing.T) {
	db := openLegacyInboundMigrationTestDB(t)
	createSettingTable(t, db)
	// store_dns already explicitly false; store_rdrc true must not overwrite it.
	setConfigBlob(t, db, `{"experimental": {"cache_file": {"store_rdrc": true, "store_dns": false}}}`)

	if err := MigrateDNS114(db); err != nil {
		t.Fatalf("migration failed: %v", err)
	}
	got := getConfigBlob(t, db)
	cf := got["experimental"].(map[string]any)["cache_file"].(map[string]any)
	if _, present := cf["store_rdrc"]; present {
		t.Fatalf("store_rdrc not removed")
	}
	if cf["store_dns"] != false {
		t.Fatalf("existing store_dns overwritten: %v", cf["store_dns"])
	}
}

func TestMigrateDNS114ConvertsAddressFilterRule(t *testing.T) {
	db := openLegacyInboundMigrationTestDB(t)
	createSettingTable(t, db)
	setConfigBlob(t, db, `{"dns": {"servers": [], "rules": [
		{"domain_suffix": ["example.com"], "ip_cidr": ["10.0.0.0/8"], "action": "route", "server": "local"},
		{"action": "route", "server": "remote"}
	]}}`)

	if err := MigrateDNS114(db); err != nil {
		t.Fatalf("migration failed: %v", err)
	}
	got := getConfigBlob(t, db)
	rules := got["dns"].(map[string]any)["rules"].([]any)
	// Original single rule becomes evaluate + match_response; fallback untouched.
	if len(rules) != 3 {
		t.Fatalf("rules count=%d, want 3 (evaluate, match_response, fallback)", len(rules))
	}
	evalRule := rules[0].(map[string]any)
	if evalRule["action"] != "evaluate" {
		t.Fatalf("rule[0].action=%v, want evaluate", evalRule["action"])
	}
	if evalRule["server"] != "local" {
		t.Fatalf("rule[0].server=%v, want local (preserved resolver)", evalRule["server"])
	}
	if _, present := evalRule["ip_cidr"]; present {
		t.Fatalf("evaluate rule must not carry the address filter: %v", evalRule)
	}
	// query match condition shared with evaluate
	if _, present := evalRule["domain_suffix"]; !present {
		t.Fatalf("evaluate rule lost domain_suffix: %v", evalRule)
	}

	filterRule := rules[1].(map[string]any)
	if filterRule["match_response"] != true {
		t.Fatalf("rule[1].match_response=%v, want true", filterRule["match_response"])
	}
	if filterRule["action"] != "route" || filterRule["server"] != "local" {
		t.Fatalf("rule[1] action/server wrong: %v", filterRule)
	}
	if _, present := filterRule["ip_cidr"]; !present {
		t.Fatalf("rule[1] lost ip_cidr filter: %v", filterRule)
	}

	fallback := rules[2].(map[string]any)
	if fallback["action"] != "route" || fallback["server"] != "remote" {
		t.Fatalf("fallback rule disturbed: %v", fallback)
	}
}

func TestMigrateDNS114SkipsNewFormRules(t *testing.T) {
	db := openLegacyInboundMigrationTestDB(t)
	createSettingTable(t, db)
	// A rule already using match_response must not be double-converted.
	setConfigBlob(t, db, `{"dns": {"rules": [
		{"action": "evaluate", "server": "remote"},
		{"match_response": true, "ip_cidr": ["10.0.0.0/8"], "action": "route", "server": "local"}
	]}}`)

	if err := MigrateDNS114(db); err != nil {
		t.Fatalf("migration failed: %v", err)
	}
	got := getConfigBlob(t, db)
	rules := got["dns"].(map[string]any)["rules"].([]any)
	if len(rules) != 2 {
		t.Fatalf("new-form rules were re-converted, count=%d", len(rules))
	}
}

func TestMigrateDNS114Idempotent(t *testing.T) {
	db := openLegacyInboundMigrationTestDB(t)
	createSettingTable(t, db)
	setConfigBlob(t, db, `{"dns": {"independent_cache": true, "rules": [
		{"ip_is_private": true, "action": "route", "server": "local"}
	]}, "experimental": {"cache_file": {"store_rdrc": true}}}`)

	if err := MigrateDNS114(db); err != nil {
		t.Fatalf("first pass failed: %v", err)
	}
	first := getConfigBlob(t, db)
	if err := MigrateDNS114(db); err != nil {
		t.Fatalf("second pass failed: %v", err)
	}
	second := getConfigBlob(t, db)
	fj, _ := json.Marshal(first)
	sj, _ := json.Marshal(second)
	if string(fj) != string(sj) {
		t.Fatalf("not idempotent:\nfirst=%s\nsecond=%s", fj, sj)
	}
}

func TestMigrateDNS114RejectsAmbiguousShape(t *testing.T) {
	db := openLegacyInboundMigrationTestDB(t)
	createSettingTable(t, db)
	// address filter with no server: cannot know which resolver to evaluate with.
	setConfigBlob(t, db, `{"dns": {"rules": [{"ip_cidr": ["10.0.0.0/8"], "action": "route"}]}}`)

	err := MigrateDNS114(db)
	if err == nil || !strings.Contains(err.Error(), "without a server") {
		t.Fatalf("expected no-server error, got %v", err)
	}
}

func TestMigrateDNS114RejectsLogicalFilter(t *testing.T) {
	db := openLegacyInboundMigrationTestDB(t)
	createSettingTable(t, db)
	setConfigBlob(t, db, `{"dns": {"rules": [{"type": "logical", "mode": "and", "ip_cidr": ["10.0.0.0/8"], "rules": [], "action": "route", "server": "local"}]}}`)

	err := MigrateDNS114(db)
	if err == nil || !strings.Contains(err.Error(), "logical rule") {
		t.Fatalf("expected logical-rule rejection, got %v", err)
	}
}

func TestMigrateDNS114NoConfigIsNoOp(t *testing.T) {
	db := openLegacyInboundMigrationTestDB(t)
	createSettingTable(t, db)
	// no config row at all
	if err := MigrateDNS114(db); err != nil {
		t.Fatalf("migration on empty settings failed: %v", err)
	}
}

func TestMigrateDNS114LeavesCleanConfigUntouched(t *testing.T) {
	db := openLegacyInboundMigrationTestDB(t)
	createSettingTable(t, db)
	setConfigBlob(t, db, `{"dns": {"servers": [], "rules": [{"action": "route", "server": "local"}]}}`)

	before := getConfigBlob(t, db)
	if err := MigrateDNS114(db); err != nil {
		t.Fatalf("migration failed: %v", err)
	}
	after := getConfigBlob(t, db)
	bj, _ := json.Marshal(before)
	aj, _ := json.Marshal(after)
	if string(bj) != string(aj) {
		t.Fatalf("clean config modified:\nbefore=%s\nafter=%s", bj, aj)
	}
}
