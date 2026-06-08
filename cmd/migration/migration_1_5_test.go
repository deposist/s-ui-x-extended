package migration

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x/config"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestTo15AddsClientIPSchemaAndBackfillsSubSecretsIdempotently(t *testing.T) {
	db := openMigrationTestDB(t)
	if err := db.Exec(`
CREATE TABLE clients (
	id integer PRIMARY KEY AUTOINCREMENT,
	enable boolean,
	name text
)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("INSERT INTO clients(enable, name) VALUES(1, 'alice'), (1, 'bob')").Error; err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 2; i++ {
		if err := to1_5(db); err != nil {
			t.Fatal(err)
		}
	}

	for _, column := range []string{
		"limit_ip",
		"ip_limit_mode",
		"last_online",
		"last_ip_count",
		"sub_secret",
	} {
		hasColumn, err := sqliteHasColumn(db, "clients", column)
		if err != nil {
			t.Fatal(err)
		}
		if !hasColumn {
			t.Fatalf("clients.%s was not added", column)
		}
	}
	if !db.Migrator().HasTable("client_ips") {
		t.Fatal("client_ips table was not created")
	}
	for _, column := range []string{"ip_hash", "ip_display"} {
		hasColumn, err := sqliteHasColumn(db, "client_ips", column)
		if err != nil {
			t.Fatal(err)
		}
		if !hasColumn {
			t.Fatalf("client_ips.%s was not added", column)
		}
	}
	if hasIndex, err := sqliteHasIndex(db, "client_ips", "idx_client_ips_client_ip"); err != nil {
		t.Fatal(err)
	} else if hasIndex {
		t.Fatal("obsolete unique client/ip index should not be created")
	}
	if hasIndex, err := sqliteHasIndex(db, "client_ips", "idx_client_ips_client_legacy_ip"); err != nil {
		t.Fatal(err)
	} else if !hasIndex {
		t.Fatal("client_ips legacy client/ip lookup index was not created")
	}
	if err := db.Exec(`
INSERT INTO client_ips(client_name, ip, ip_hash, first_seen, last_seen)
VALUES('alice', '', 'hash-1', 1, 1), ('alice', '', 'hash-2', 1, 1)
`).Error; err != nil {
		t.Fatalf("client_ips should allow multiple empty legacy ip values for one client: %v", err)
	}
	if err := db.Exec(`
INSERT INTO client_ips(client_name, ip, ip_hash, first_seen, last_seen)
VALUES('alice', '', 'hash-1', 1, 1)
`).Error; err == nil {
		t.Fatal("client_ips unique client/hash index was not created")
	}

	var clients []struct {
		Name      string
		SubSecret string
	}
	if err := db.Raw("SELECT name, sub_secret FROM clients ORDER BY name").Scan(&clients).Error; err != nil {
		t.Fatal(err)
	}
	if len(clients) != 2 {
		t.Fatalf("expected two clients, got %d", len(clients))
	}
	if clients[0].SubSecret == "" || clients[1].SubSecret == "" || clients[0].SubSecret == clients[1].SubSecret {
		t.Fatalf("sub_secret backfill failed: %#v", clients)
	}
}

func TestTo15BackfillsClientIPHashesIdempotently(t *testing.T) {
	db := openMigrationTestDB(t)
	if err := db.Exec(`
CREATE TABLE settings (
	id integer PRIMARY KEY AUTOINCREMENT,
	key text,
	value text
)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("INSERT INTO settings(key, value) VALUES('installSalt', 'test-salt')").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`
CREATE TABLE clients (
	id integer PRIMARY KEY AUTOINCREMENT,
	enable boolean,
	name text
)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("INSERT INTO clients(enable, name) VALUES(1, 'alice')").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`
CREATE TABLE client_ips (
	id integer PRIMARY KEY AUTOINCREMENT,
	client_name text,
	ip text,
	first_seen integer,
	last_seen integer
)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`
INSERT INTO client_ips(client_name, ip, first_seen, last_seen)
VALUES('alice', '198.51.100.10', 1, 1)
`).Error; err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 2; i++ {
		if err := to1_5(db); err != nil {
			t.Fatal(err)
		}
	}

	var row struct {
		IP        string
		IPHash    string
		IPDisplay *string
	}
	if err := db.Raw("SELECT ip, ip_hash, ip_display FROM client_ips WHERE client_name = ?", "alice").Scan(&row).Error; err != nil {
		t.Fatal(err)
	}
	if row.IP != "198.51.100.10" {
		t.Fatalf("legacy ip column should remain additive, got %q", row.IP)
	}
	if row.IPHash == "" || row.IPHash == row.IP {
		t.Fatalf("ip_hash was not backfilled: %#v", row)
	}
	if row.IPDisplay != nil {
		t.Fatalf("ip_display should stay NULL by default: %#v", row.IPDisplay)
	}
}

func TestMigrateDbFrom14RunsCheckpointAfterCommit(t *testing.T) {
	dbDir := t.TempDir()
	t.Setenv("SUI_DB_FOLDER", dbDir)
	dbPath := filepath.Join(dbDir, config.GetName()+".db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		if strings.Contains(err.Error(), "go-sqlite3 requires cgo") {
			t.Skip(err)
		}
		t.Fatal(err)
	}
	if err := db.Exec("PRAGMA journal_mode=WAL").Error; err != nil {
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
	if err := db.Exec("INSERT INTO settings(key, value) VALUES('version', '1.4.3')").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`
CREATE TABLE clients (
	id integer PRIMARY KEY AUTOINCREMENT,
	enable boolean,
	name text
)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("INSERT INTO clients(enable, name) VALUES(1, 'alice')").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`
CREATE TABLE audit_events (
	id integer PRIMARY KEY AUTOINCREMENT,
	date_time integer,
	actor text,
	event text,
	resource text,
	severity text,
	ip text,
	user_agent text,
	details blob
)`).Error; err != nil {
		t.Fatal(err)
	}
	if sqlDB, err := db.DB(); err == nil {
		if err := sqlDB.Close(); err != nil {
			t.Fatal(err)
		}
	}

	if err := MigrateDb(); err != nil {
		t.Fatal(err)
	}

	db = openMigrationDBAtPath(t, dbPath)
	var version string
	if err := db.Raw("SELECT value FROM settings WHERE key = ?", "version").Scan(&version).Error; err != nil {
		t.Fatal(err)
	}
	if version != config.GetVersion() {
		t.Fatalf("version was not updated: got %q want %q", version, config.GetVersion())
	}
	var subSecret string
	if err := db.Raw("SELECT sub_secret FROM clients WHERE name = ?", "alice").Scan(&subSecret).Error; err != nil {
		t.Fatal(err)
	}
	if subSecret == "" {
		t.Fatal("sub_secret was not backfilled")
	}
}

func openMigrationDBAtPath(t *testing.T, dbPath string) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}
