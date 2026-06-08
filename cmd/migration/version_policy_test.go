package migration

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x/config"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestMigrateDbDoesNotDowngradeFutureVersion(t *testing.T) {
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
	if err := db.Exec(`
CREATE TABLE settings (
	id integer PRIMARY KEY AUTOINCREMENT,
	key text,
	value text
)`).Error; err != nil {
		t.Fatal(err)
	}
	const futureVersion = "99.0.0"
	if err := db.Exec("INSERT INTO settings(key, value) VALUES('version', ?)", futureVersion).Error; err != nil {
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
	if version != futureVersion {
		t.Fatalf("future version was downgraded: got %q want %q", version, futureVersion)
	}
}
