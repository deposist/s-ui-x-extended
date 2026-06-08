package migration

import "testing"

func TestIssue7To17CreatesXUISyncPolicyColumnsIdempotently(t *testing.T) {
	db := openMigrationTestDB(t)

	for i := 0; i < 2; i++ {
		if err := to1_7(db); err != nil {
			t.Fatal(err)
		}
	}

	for _, column := range []string{"include_settings", "include_history", "include_routing", "admin_mode"} {
		hasColumn, err := sqliteHasColumn(db, "xui_sync_profiles", column)
		if err != nil {
			t.Fatal(err)
		}
		if !hasColumn {
			t.Fatalf("xui_sync_profiles.%s was not created", column)
		}
	}
	if err := db.Exec("INSERT INTO xui_sync_profiles(name) VALUES(?)", "defaults").Error; err != nil {
		t.Fatal(err)
	}
	var row struct {
		IncludeSettings bool
		IncludeHistory  bool
		IncludeRouting  bool
		AdminMode       string
	}
	if err := db.Raw("SELECT include_settings, include_history, include_routing, admin_mode FROM xui_sync_profiles WHERE name = ?", "defaults").Scan(&row).Error; err != nil {
		t.Fatal(err)
	}
	if row.IncludeSettings || row.IncludeHistory || row.IncludeRouting || row.AdminMode != "skip" {
		t.Fatalf("unexpected policy defaults: %#v", row)
	}
}

func TestIssue7To17AddsXUISyncPolicyColumnsToExistingTable(t *testing.T) {
	db := openMigrationTestDB(t)
	if err := db.Exec(`
CREATE TABLE xui_sync_profiles (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT,
  source_type TEXT,
  source_json BLOB,
  source_salt BLOB,
  strategy TEXT,
  only_new BOOLEAN NOT NULL DEFAULT TRUE,
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  schedule TEXT,
  last_run_at INTEGER,
  last_run_status TEXT,
  last_run_summary JSON,
  created_at INTEGER,
  updated_at INTEGER
)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("INSERT INTO xui_sync_profiles(name) VALUES(?)", "legacy").Error; err != nil {
		t.Fatal(err)
	}

	if err := to1_7(db); err != nil {
		t.Fatal(err)
	}

	var row struct {
		IncludeSettings bool
		IncludeHistory  bool
		IncludeRouting  bool
		AdminMode       string
	}
	if err := db.Raw("SELECT include_settings, include_history, include_routing, admin_mode FROM xui_sync_profiles WHERE name = ?", "legacy").Scan(&row).Error; err != nil {
		t.Fatal(err)
	}
	if row.IncludeSettings || row.IncludeHistory || row.IncludeRouting || row.AdminMode != "skip" {
		t.Fatalf("unexpected backfilled policy defaults: %#v", row)
	}
}
