package migration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestTo12MovesLegacyInboundFieldsToRouteRules(t *testing.T) {
	db := openMigrationTestDB(t)
	if err := db.Exec(`
CREATE TABLE tls (
	id integer PRIMARY KEY AUTOINCREMENT,
	name text,
	server blob,
	client blob,
	inbounds blob
)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`
CREATE TABLE inbound_data (
	id integer PRIMARY KEY AUTOINCREMENT,
	tag text,
	addrs blob,
	out_json blob
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
	if err := db.Exec(`
CREATE TABLE clients (
	id integer PRIMARY KEY AUTOINCREMENT,
	enable boolean,
	name text,
	config blob,
	inbounds blob,
	links blob,
	volume integer,
	expiry integer,
	down integer,
	up integer,
	desc text,
	"group" text
)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`
CREATE TABLE changes (
	id integer PRIMARY KEY AUTOINCREMENT,
	date_time integer,
	actor text,
	"key" text,
	action text,
	obj blob,
	"index" integer
)`).Error; err != nil {
		t.Fatal(err)
	}

	binDir := filepath.Join(t.TempDir(), "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(binDir, "config.json")
	oldArgs := os.Args
	oldBinFolder, hadBinFolder := os.LookupEnv("SUI_BIN_FOLDER")
	t.Cleanup(func() {
		os.Args = oldArgs
		if hadBinFolder {
			_ = os.Setenv("SUI_BIN_FOLDER", oldBinFolder)
		} else {
			_ = os.Unsetenv("SUI_BIN_FOLDER")
		}
	})
	os.Args = []string{filepath.Join(filepath.Dir(binDir), "sui")}
	if err := os.Setenv("SUI_BIN_FOLDER", "bin"); err != nil {
		t.Fatal(err)
	}

	configJSON := []byte(`{
		"inbounds": [
			{
				"type": "vless",
				"tag": "legacy-vless",
				"listen": "::",
				"listen_port": 443,
				"sniff": true,
				"sniff_override_destination": true,
				"sniff_timeout": "1s",
				"domain_strategy": "prefer_ipv4",
				"udp_disable_domain_unmapping": true,
				"users": []
			}
		],
		"outbounds": [],
		"route": {
			"rules": [
				{"inbound": ["legacy-vless"], "action": "sniff", "timeout": "1s"}
			]
		},
		"experimental": {}
	}`)
	if err := os.WriteFile(configPath, configJSON, 0o600); err != nil {
		t.Fatal(err)
	}

	if err := to1_2(db); err != nil {
		t.Fatal(err)
	}

	var options string
	if err := db.Raw("SELECT options FROM inbounds WHERE tag = ?", "legacy-vless").Scan(&options).Error; err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"sniff", "sniff_override_destination", "sniff_timeout", "domain_strategy", "udp_disable_domain_unmapping"} {
		if jsonContainsKey(t, options, key) {
			t.Fatalf("legacy key %s remained in inbound options: %s", key, options)
		}
	}

	var configValue string
	if err := db.Raw("SELECT value FROM settings WHERE key = ?", "config").Scan(&configValue).Error; err != nil {
		t.Fatal(err)
	}
	var config struct {
		Route struct {
			Rules []map[string]any `json:"rules"`
		} `json:"route"`
	}
	if err := json.Unmarshal([]byte(configValue), &config); err != nil {
		t.Fatal(err)
	}
	if len(config.Route.Rules) < 3 || config.Route.Rules[0]["action"] != "resolve" || config.Route.Rules[1]["action"] != "route-options" {
		t.Fatalf("legacy inbound rule actions must be inserted before existing routing rules: %#v", config.Route.Rules)
	}
	if !hasMigrationRouteRule(config.Route.Rules, "resolve", "legacy-vless", "strategy", "prefer_ipv4") {
		t.Fatalf("resolve rule missing after migration: %#v", config.Route.Rules)
	}
	if !hasMigrationRouteRule(config.Route.Rules, "sniff", "legacy-vless", "timeout", "1s") {
		t.Fatalf("sniff rule missing after migration: %#v", config.Route.Rules)
	}
	if !hasMigrationRouteRule(config.Route.Rules, "route-options", "legacy-vless", "udp_disable_domain_unmapping", true) {
		t.Fatalf("route-options rule missing after migration: %#v", config.Route.Rules)
	}
	if countMigrationRouteRule(config.Route.Rules, "sniff", "legacy-vless") != 1 {
		t.Fatalf("sniff rule duplicated: %#v", config.Route.Rules)
	}
}

func jsonContainsKey(t *testing.T, raw string, key string) bool {
	t.Helper()
	var object map[string]any
	if err := json.Unmarshal([]byte(raw), &object); err != nil {
		t.Fatal(err)
	}
	_, ok := object[key]
	return ok
}

func hasMigrationRouteRule(rules []map[string]any, action string, inbound string, key string, value any) bool {
	for _, rule := range rules {
		if rule["action"] != action || rule[key] != value {
			continue
		}
		if migrationRuleHasInbound(rule, inbound) {
			return true
		}
	}
	return false
}

func countMigrationRouteRule(rules []map[string]any, action string, inbound string) int {
	count := 0
	for _, rule := range rules {
		if rule["action"] == action && migrationRuleHasInbound(rule, inbound) {
			count++
		}
	}
	return count
}

func migrationRuleHasInbound(rule map[string]any, inbound string) bool {
	switch inbounds := rule["inbound"].(type) {
	case string:
		return inbounds == inbound
	case []any:
		for _, item := range inbounds {
			if item == inbound {
				return true
			}
		}
	}
	return false
}
