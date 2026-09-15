package migration

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x-extended/config"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestMigrateDbVersionChainWritesPreMigrationBackup proves that a real version
// upgrade (db version differs from binary) snapshots the database file before
// migrating, so a binary downgrade can be paired with restoring the snapshot.
func TestMigrateDbVersionChainWritesPreMigrationBackup(t *testing.T) {
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
	stmts := []string{
		`CREATE TABLE settings (id integer PRIMARY KEY AUTOINCREMENT, key text, value text)`,
		`INSERT INTO settings(key, value) VALUES('version', '1.4.3')`,
		`CREATE TABLE clients (id integer PRIMARY KEY AUTOINCREMENT, enable boolean, name text)`,
		`INSERT INTO clients(enable, name) VALUES(1, 'alice')`,
		`CREATE TABLE audit_events (id integer PRIMARY KEY AUTOINCREMENT, date_time integer, actor text, event text, resource text, severity text, ip text, user_agent text, details blob)`,
	}
	for _, s := range stmts {
		if err := db.Exec(s).Error; err != nil {
			t.Fatalf("setup %q: %v", s, err)
		}
	}
	if sqlDB, err := db.DB(); err == nil {
		if err := sqlDB.Close(); err != nil {
			t.Fatal(err)
		}
	}

	if err := MigrateDb(); err != nil {
		t.Fatalf("MigrateDb failed: %v", err)
	}

	// A pre-migration snapshot of the 1.4.3 database must exist.
	matches, err := filepath.Glob(dbPath + ".pre-*.bak")
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected exactly one pre-migration backup, got %v", matches)
	}
	if !strings.Contains(matches[0], "pre-1.4.3") {
		t.Fatalf("backup name should carry the pre-migration version, got %s", matches[0])
	}

	// The snapshot must be a valid, restorable database holding the old version.
	backup, err := gorm.Open(sqlite.Open(matches[0]), &gorm.Config{})
	if err != nil {
		t.Fatalf("open backup: %v", err)
	}
	defer func() {
		if b, err := backup.DB(); err == nil {
			_ = b.Close()
		}
	}()
	var oldVersion string
	if err := backup.Raw("SELECT value FROM settings WHERE key = ?", "version").Scan(&oldVersion).Error; err != nil {
		t.Fatalf("read backup version: %v", err)
	}
	if oldVersion != "1.4.3" {
		t.Fatalf("backup version=%q, want 1.4.3 (pre-migration state)", oldVersion)
	}
}
