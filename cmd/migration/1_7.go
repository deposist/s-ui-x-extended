package migration

import "gorm.io/gorm"

func to1_7(db *gorm.DB) error {
	if err := db.Exec(`
CREATE TABLE IF NOT EXISTS xui_sync_profiles (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT,
  source_type TEXT,
  source_json BLOB,
  source_salt BLOB,
  strategy TEXT,
  only_new BOOLEAN NOT NULL DEFAULT TRUE,
  include_settings BOOLEAN NOT NULL DEFAULT FALSE,
  include_history BOOLEAN NOT NULL DEFAULT FALSE,
  include_routing BOOLEAN NOT NULL DEFAULT FALSE,
  admin_mode TEXT NOT NULL DEFAULT 'skip',
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  schedule TEXT,
  last_run_at INTEGER,
  last_run_status TEXT,
  last_run_summary JSON,
  created_at INTEGER,
  updated_at INTEGER
)`).Error; err != nil {
		return err
	}
	if err := ensureXUISyncProfilePolicyColumns(db); err != nil {
		return err
	}
	if err := db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_xui_sync_profiles_name ON xui_sync_profiles(name)").Error; err != nil {
		return err
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_xui_sync_profiles_enabled ON xui_sync_profiles(enabled, last_run_at)").Error; err != nil {
		return err
	}
	if err := db.Exec(`
CREATE TABLE IF NOT EXISTS xui_known_hosts (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  host TEXT,
  fingerprint TEXT,
  public_key TEXT,
  created_at INTEGER,
  updated_at INTEGER
)`).Error; err != nil {
		return err
	}
	return db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_xui_known_hosts_host ON xui_known_hosts(host)").Error
}

func ensureXUISyncProfilePolicyColumns(db *gorm.DB) error {
	columns := []struct {
		name       string
		definition string
	}{
		{name: "include_settings", definition: "BOOLEAN NOT NULL DEFAULT FALSE"},
		{name: "include_history", definition: "BOOLEAN NOT NULL DEFAULT FALSE"},
		{name: "include_routing", definition: "BOOLEAN NOT NULL DEFAULT FALSE"},
		{name: "admin_mode", definition: "TEXT NOT NULL DEFAULT 'skip'"},
	}
	for _, column := range columns {
		if err := addColumnIfMissing(db, "xui_sync_profiles", column.name, column.definition); err != nil {
			return err
		}
	}
	return nil
}
