package database

import (
	"path/filepath"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// beta1AWGDevicesDDL is the exact awg_devices layout that 1.0.2-beta1 shipped
// (no endpoint_id column yet) together with its indexes.
const beta1AWGDevicesDDL = `CREATE TABLE IF NOT EXISTS awg_devices (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	client_id INTEGER NOT NULL,
	name TEXT NOT NULL,
	create_request_key TEXT NOT NULL DEFAULT '',
	rotate_request_key TEXT NOT NULL DEFAULT '',
	crypto_context BLOB NOT NULL,
	public_key TEXT NOT NULL,
	previous_public_key TEXT,
	private_key_enc BLOB NOT NULL,
	psk_enc BLOB NOT NULL,
	ipv4_address TEXT NOT NULL,
	desired_enabled INTEGER NOT NULL DEFAULT 1,
	sync_state TEXT NOT NULL,
	provisioned INTEGER NOT NULL DEFAULT 0,
	last_error TEXT,
	rx_baseline INTEGER NOT NULL DEFAULT 0,
	tx_baseline INTEGER NOT NULL DEFAULT 0,
	total_rx INTEGER NOT NULL DEFAULT 0,
	total_tx INTEGER NOT NULL DEFAULT 0,
	last_handshake INTEGER NOT NULL DEFAULT 0,
	created_at INTEGER NOT NULL,
	updated_at INTEGER NOT NULL,
	revoked_at INTEGER NOT NULL DEFAULT 0,
	ip_reusable_after INTEGER NOT NULL DEFAULT 0
)`

// The 1.0.2-beta1 -> 1.0.2 upgrade crashed the panel boot loop with "no such
// column: endpoint_id": InitDB AutoMigrated model.AWGDevice over the raw-SQL
// table owned by paidsub.EnsureSchema, and GORM's SQLite migrator rebuilt the
// table because its rendered defaults differ textually from the raw DDL. The
// module-owned tables must not be part of InitDB's AutoMigrate set at all.
func TestInitDBLeavesModuleOwnedAWGTablesAlone(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "s-ui.db")
	t.Setenv("SUI_DB_FOLDER", dir)

	seed, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		if strings.Contains(err.Error(), "go-sqlite3 requires cgo") {
			t.Skip(err)
		}
		t.Fatal(err)
	}
	seedStmts := []string{
		beta1AWGDevicesDDL,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_awg_devices_public_key ON awg_devices(public_key)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_awg_devices_active_ipv4 ON awg_devices(ipv4_address) WHERE desired_enabled = 1`,
		`CREATE INDEX IF NOT EXISTS idx_awg_devices_client_enabled ON awg_devices(client_id, desired_enabled)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_awg_devices_client_create_request
			ON awg_devices(client_id, create_request_key) WHERE create_request_key != ''`,
		`INSERT INTO awg_devices (client_id, name, crypto_context, public_key, private_key_enc, psk_enc,
			ipv4_address, sync_state, created_at, updated_at)
			VALUES (1, 'phone', x'01', 'beta1-key', x'02', x'03', '10.77.0.2', 'in_sync', 1, 1)`,
	}
	for _, stmt := range seedStmts {
		if err := seed.Exec(stmt).Error; err != nil {
			t.Fatal(err)
		}
	}
	if sqlDB, dbErr := seed.DB(); dbErr == nil {
		_ = sqlDB.Close()
	}

	if err := InitDB(dbPath); err != nil {
		t.Fatalf("InitDB over a beta1 database must not fail: %v", err)
	}
	t.Cleanup(func() { closeMainDB(t) })

	// The device row must survive startup untouched.
	var count int64
	if err := GetDB().Table("awg_devices").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("awg_devices rows = %d; want 1", count)
	}
	// InitDB must not have rebuilt the module-owned table: the beta1 layout
	// (still without endpoint_id) is upgraded later by paidsub.EnsureSchema's
	// guarded ALTER, not by GORM AutoMigrate.
	var tableSQL string
	if err := GetDB().Raw(
		`SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'awg_devices'`,
	).Scan(&tableSQL).Error; err != nil {
		t.Fatal(err)
	}
	if strings.Contains(tableSQL, "awg_devices__temp") {
		t.Fatalf("table was rebuilt by AutoMigrate: %s", tableSQL)
	}
}
