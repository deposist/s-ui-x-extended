package migrateutil

import (
	"encoding/json"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func openLegacyInboundMigrationTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		if strings.Contains(err.Error(), "go-sqlite3 requires cgo") {
			t.Skip(err)
		}
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

func createLegacyInboundMigrationTables(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.Exec(`
CREATE TABLE inbounds (
	id integer PRIMARY KEY AUTOINCREMENT,
	type text,
	tag text,
	tls_id integer,
	addrs blob,
	out_json blob,
	options blob
)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`
CREATE TABLE settings (
	id integer PRIMARY KEY AUTOINCREMENT,
	key text,
	value text
)`).Error; err != nil {
		t.Fatal(err)
	}
}

func TestMigrateLegacyInboundRuleActionFieldsMovesOptionsToRouteRules(t *testing.T) {
	db := openLegacyInboundMigrationTestDB(t)
	createLegacyInboundMigrationTables(t, db)
	config := `{"route":{"rules":[{"action":"sniff"},{"protocol":["dns"],"action":"hijack-dns"}]}}`
	options := `{"listen":"::","listen_port":443,"sniff":true,"sniff_override_destination":true,"sniff_timeout":"1s","domain_strategy":"prefer_ipv4","udp_disable_domain_unmapping":true,"proxy_protocol":true}`
	if err := db.Exec("INSERT INTO settings(key, value) VALUES(?, ?)", "config", config).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("INSERT INTO inbounds(type, tag, tls_id, addrs, out_json, options) VALUES(?,?,?,?,?,?)", "vless", "vless-in", 0, []byte(`null`), []byte(`{}`), []byte(options)).Error; err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 2; i++ {
		if err := MigrateLegacyInboundRuleActionFields(db); err != nil {
			t.Fatal(err)
		}
	}

	var storedOptions string
	if err := db.Raw("SELECT options FROM inbounds WHERE tag = ?", "vless-in").Scan(&storedOptions).Error; err != nil {
		t.Fatal(err)
	}
	var optionsMap map[string]any
	if err := json.Unmarshal([]byte(storedOptions), &optionsMap); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"sniff", "sniff_override_destination", "sniff_timeout", "domain_strategy", "udp_disable_domain_unmapping"} {
		if _, ok := optionsMap[key]; ok {
			t.Fatalf("legacy key %s was not removed from inbound options: %s", key, storedOptions)
		}
	}
	if optionsMap["proxy_protocol"] != true {
		t.Fatalf("non-legacy inbound option was not preserved: %s", storedOptions)
	}

	var storedConfig string
	if err := db.Raw("SELECT value FROM settings WHERE key = ?", "config").Scan(&storedConfig).Error; err != nil {
		t.Fatal(err)
	}
	var cfg struct {
		Route struct {
			Rules []map[string]any `json:"rules"`
		} `json:"route"`
	}
	if err := json.Unmarshal([]byte(storedConfig), &cfg); err != nil {
		t.Fatal(err)
	}
	if countAction(cfg.Route.Rules, "resolve") != 1 || countAction(cfg.Route.Rules, "sniff") != 2 || countAction(cfg.Route.Rules, "route-options") != 1 {
		t.Fatalf("unexpected migrated route rules: %#v", cfg.Route.Rules)
	}
	if len(cfg.Route.Rules) < 3 || cfg.Route.Rules[0]["action"] != "resolve" || cfg.Route.Rules[1]["action"] != "sniff" || cfg.Route.Rules[2]["action"] != "route-options" {
		t.Fatalf("legacy inbound rule actions must run before existing routing rules: %#v", cfg.Route.Rules)
	}
	if !hasRule(cfg.Route.Rules, map[string]any{"action": "resolve", "inbound": []any{"vless-in"}, "strategy": "prefer_ipv4"}) {
		t.Fatalf("resolve rule was not migrated: %#v", cfg.Route.Rules)
	}
	if !hasRule(cfg.Route.Rules, map[string]any{"action": "sniff", "inbound": []any{"vless-in"}, "timeout": "1s"}) {
		t.Fatalf("sniff rule was not migrated: %#v", cfg.Route.Rules)
	}
	if !hasRule(cfg.Route.Rules, map[string]any{"action": "route-options", "inbound": []any{"vless-in"}, "udp_disable_domain_unmapping": true}) {
		t.Fatalf("route-options rule was not migrated: %#v", cfg.Route.Rules)
	}
}

func TestMigrateLegacyInboundRuleActionFieldsDoesNotTreatNarrowRuleAsDuplicate(t *testing.T) {
	db := openLegacyInboundMigrationTestDB(t)
	createLegacyInboundMigrationTables(t, db)
	config := `{"route":{"rules":[{"inbound":["in-a"],"action":"sniff","timeout":"300ms","sniffer":["tls"]}]}}`
	options := `{"sniff":true,"sniff_timeout":"300ms"}`
	if err := db.Exec("INSERT INTO settings(key, value) VALUES(?, ?)", "config", config).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("INSERT INTO inbounds(type, tag, tls_id, addrs, out_json, options) VALUES(?,?,?,?,?,?)", "mixed", "in-a", 0, []byte(`null`), []byte(`{}`), []byte(options)).Error; err != nil {
		t.Fatal(err)
	}
	if err := MigrateLegacyInboundRuleActionFields(db); err != nil {
		t.Fatal(err)
	}

	var storedConfig string
	if err := db.Raw("SELECT value FROM settings WHERE key = ?", "config").Scan(&storedConfig).Error; err != nil {
		t.Fatal(err)
	}
	var cfg struct {
		Route struct {
			Rules []map[string]any `json:"rules"`
		} `json:"route"`
	}
	if err := json.Unmarshal([]byte(storedConfig), &cfg); err != nil {
		t.Fatal(err)
	}
	if countAction(cfg.Route.Rules, "sniff") != 2 {
		t.Fatalf("narrow sniff rule should not block migration of equivalent inbound sniff: %#v", cfg.Route.Rules)
	}
	if cfg.Route.Rules[0]["action"] != "sniff" || cfg.Route.Rules[0]["sniffer"] != nil {
		t.Fatalf("migrated broad sniff rule should be inserted before existing narrow rule: %#v", cfg.Route.Rules)
	}
}

func TestMigrateLegacyInboundRuleActionFieldsSkipsDuplicateRules(t *testing.T) {
	db := openLegacyInboundMigrationTestDB(t)
	createLegacyInboundMigrationTables(t, db)
	config := `{"route":{"rules":[{"inbound":["in-a"],"action":"sniff","timeout":"300ms"}]}}`
	options := `{"sniff":true,"sniff_timeout":"300ms"}`
	if err := db.Exec("INSERT INTO settings(key, value) VALUES(?, ?)", "config", config).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("INSERT INTO inbounds(type, tag, tls_id, addrs, out_json, options) VALUES(?,?,?,?,?,?)", "mixed", "in-a", 0, []byte(`null`), []byte(`{}`), []byte(options)).Error; err != nil {
		t.Fatal(err)
	}
	if err := MigrateLegacyInboundRuleActionFields(db); err != nil {
		t.Fatal(err)
	}

	var storedConfig string
	if err := db.Raw("SELECT value FROM settings WHERE key = ?", "config").Scan(&storedConfig).Error; err != nil {
		t.Fatal(err)
	}
	var cfg struct {
		Route struct {
			Rules []map[string]any `json:"rules"`
		} `json:"route"`
	}
	if err := json.Unmarshal([]byte(storedConfig), &cfg); err != nil {
		t.Fatal(err)
	}
	if countAction(cfg.Route.Rules, "sniff") != 1 {
		t.Fatalf("duplicate sniff rule was created: %#v", cfg.Route.Rules)
	}
}

func countAction(rules []map[string]any, action string) int {
	count := 0
	for _, rule := range rules {
		if rule["action"] == action {
			count++
		}
	}
	return count
}

func hasRule(rules []map[string]any, candidate map[string]any) bool {
	for _, rule := range rules {
		if routeRulesEquivalent(rule, candidate) {
			return true
		}
	}
	return false
}
