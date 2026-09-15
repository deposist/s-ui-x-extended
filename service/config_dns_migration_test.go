package service

import (
	"strings"
	"testing"

	"github.com/deposist/s-ui-x-extended/core"
	"github.com/deposist/s-ui-x-extended/database/migrateutil"
	"github.com/deposist/s-ui-x-extended/database/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// openDNSMigrationDB builds an in-memory DB with just the settings table.
func openDNSMigrationDB(t *testing.T) *gorm.DB {
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
	if err := db.Exec(`CREATE TABLE settings (id integer PRIMARY KEY AUTOINCREMENT, key text, value text)`).Error; err != nil {
		t.Fatal(err)
	}
	return db
}

// TestDNSMigrationOutputValidatesAgainstCore is the load-bearing acceptance
// check: a legacy 1.13 config using independent_cache, store_rdrc and a legacy
// address filter must, after migration, be accepted by the 1.14 core — meaning
// it no longer trips the legacy-DNS-mode conflict and parses cleanly.
func TestDNSMigrationOutputValidatesAgainstCore(t *testing.T) {
	db := openDNSMigrationDB(t)

	legacy := `{
		"log": {"disabled": true},
		"dns": {
			"servers": [
				{"type": "local", "tag": "local"},
				{"type": "local", "tag": "remote"}
			],
			"independent_cache": true,
			"rules": [
				{"domain_suffix": ["example.cn"], "ip_cidr": ["223.0.0.0/8"], "action": "route", "server": "local"},
				{"action": "route", "server": "remote"}
			]
		},
		"route": {"final": "direct"},
		"experimental": {"cache_file": {"enabled": true, "store_rdrc": true}}
	}`
	if err := db.Exec("INSERT INTO settings(key, value) VALUES('config', ?)", legacy).Error; err != nil {
		t.Fatal(err)
	}

	if err := migrateutil.MigrateDNS114(db); err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	var row model.Setting
	if err := db.Where("key = ?", "config").First(&row).Error; err != nil {
		t.Fatal(err)
	}

	// The migrated config must validate against the 1.14 core. Before migration
	// the address filter rule would have needed legacy mode; after conversion to
	// evaluate+match_response it is a native 1.14 config.
	if err := core.ValidateConfig([]byte(row.Value)); err != nil {
		t.Fatalf("migrated config rejected by 1.14 core: %v\n%s", err, row.Value)
	}

	// Sanity: deprecated fields must be gone from the validated blob.
	for _, gone := range []string{`"independent_cache"`, `"store_rdrc"`} {
		if strings.Contains(row.Value, gone) {
			t.Fatalf("deprecated field %s still present after migration:\n%s", gone, row.Value)
		}
	}
	// The address filter must now be expressed via match_response.
	if !strings.Contains(row.Value, `"match_response"`) {
		t.Fatalf("address filter not converted to match_response:\n%s", row.Value)
	}
}

// TestDNSMigrationSameRuleIPVersionAndFilter covers the tightest conflict: one
// rule carrying BOTH ip_version and a legacy address filter. ip_version disables
// legacy DNS mode while the address filter requires it, so 1.14 must see the
// filter converted to match_response for the config to start.
func TestDNSMigrationSameRuleIPVersionAndFilter(t *testing.T) {
	db := openDNSMigrationDB(t)

	legacy := `{
		"log": {"disabled": true},
		"dns": {
			"servers": [
				{"type": "local", "tag": "local"},
				{"type": "local", "tag": "remote"}
			],
			"rules": [
				{"ip_version": 4, "ip_cidr": ["203.0.113.0/24"], "action": "route", "server": "local"},
				{"action": "route", "server": "remote"}
			]
		},
		"route": {"final": "direct"}
	}`
	if err := db.Exec("INSERT INTO settings(key, value) VALUES('config', ?)", legacy).Error; err != nil {
		t.Fatal(err)
	}

	if err := migrateutil.MigrateDNS114(db); err != nil {
		t.Fatalf("migration failed: %v", err)
	}
	var row model.Setting
	if err := db.Where("key = ?", "config").First(&row).Error; err != nil {
		t.Fatal(err)
	}
	// ip_version is a query-match field shared by both the evaluate and the
	// match_response rule; the address filter moved to match_response.
	if err := core.ValidateConfig([]byte(row.Value)); err != nil {
		t.Fatalf("same-rule ip_version+filter config rejected after migration: %v\n%s", err, row.Value)
	}
}

// TestDNSMigrationLegacyConfigWithoutConflictStillValid confirms that a
// legacy address-filter config, once migrated, resolves the legacy-mode
// conflict (the pre-1.14 mixed form would be rejected at core start).
func TestDNSMigrationResolvesLegacyModeConflict(t *testing.T) {
	db := openDNSMigrationDB(t)

	// This mixes a legacy address filter with ip_version on another rule — the
	// exact combination the 1.14 core rejects at startup. After migration the
	// address filter is a match_response rule and the conflict is gone.
	legacy := `{
		"log": {"disabled": true},
		"dns": {
			"servers": [
				{"type": "local", "tag": "local"},
				{"type": "local", "tag": "remote"}
			],
			"rules": [
				{"domain_suffix": ["example.cn"], "ip_is_private": true, "action": "route", "server": "local"},
				{"ip_version": 6, "action": "route", "server": "remote"}
			]
		},
		"route": {"final": "direct"}
	}`
	if err := db.Exec("INSERT INTO settings(key, value) VALUES('config', ?)", legacy).Error; err != nil {
		t.Fatal(err)
	}

	// Confirm the unmigrated form is indeed rejected by the core (guards the
	// test against a future core that stops rejecting the legacy mix).
	if err := core.ValidateConfig([]byte(legacy)); err == nil {
		t.Skipf("core no longer rejects the legacy address-filter + ip_version mix; migration target moved")
	}

	if err := migrateutil.MigrateDNS114(db); err != nil {
		t.Fatalf("migration failed: %v", err)
	}
	var row model.Setting
	if err := db.Where("key = ?", "config").First(&row).Error; err != nil {
		t.Fatal(err)
	}
	if err := core.ValidateConfig([]byte(row.Value)); err != nil {
		t.Fatalf("migrated config still rejected: %v\n%s", err, row.Value)
	}
}
