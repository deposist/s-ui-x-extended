package migration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestBackupDatabaseBeforeMigrationCreatesRestorableSnapshot proves the
// pre-migration backup is a complete, independently openable database: the
// rollback path for a binary downgrade depends on it.
func TestBackupDatabaseBeforeMigrationCreatesRestorableSnapshot(t *testing.T) {
	db := openMigrationTestDB(t)
	if err := db.Exec(`CREATE TABLE settings (id integer PRIMARY KEY AUTOINCREMENT, key text, value text)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("INSERT INTO settings(key, value) VALUES('version', '1.0.0-beta9'), ('marker', 'pre-migration')").Error; err != nil {
		t.Fatal(err)
	}

	// Resolve the on-disk path the same way the helper does.
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	var seq, path string
	if err := sqlDB.QueryRow("PRAGMA database_list").Scan(new(int), &seq, &path); err != nil {
		// PRAGMA database_list column order: seq, name, file.
		t.Fatalf("resolve db path: %v", err)
	}

	if err := backupDatabaseBeforeMigration(db, path); err != nil {
		t.Fatalf("backup failed: %v", err)
	}

	backupPath := path + ".pre-1.0.0-beta9.bak"
	if _, err := os.Stat(backupPath); err != nil {
		t.Fatalf("backup file not created at %s: %v", backupPath, err)
	}

	// Open the backup as its own database and confirm the data survived.
	backup, err := gorm.Open(sqlite.Open(backupPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open backup: %v", err)
	}
	defer func() {
		if b, err := backup.DB(); err == nil {
			_ = b.Close()
		}
	}()
	var marker string
	if err := backup.Raw("SELECT value FROM settings WHERE key = ?", "marker").Scan(&marker).Error; err != nil {
		t.Fatalf("read backup: %v", err)
	}
	if marker != "pre-migration" {
		t.Fatalf("backup marker=%q, want pre-migration", marker)
	}
}

// TestBackupDatabaseBeforeMigrationSanitizesVersion confirms unusual version
// strings do not produce an unsafe path.
func TestBackupDatabaseBeforeMigrationSanitizesVersion(t *testing.T) {
	db := openMigrationTestDB(t)
	if err := db.Exec(`CREATE TABLE settings (id integer PRIMARY KEY AUTOINCREMENT, key text, value text)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("INSERT INTO settings(key, value) VALUES('version', '1.0/weird;x')").Error; err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	var seq, path string
	if err := sqlDB.QueryRow("PRAGMA database_list").Scan(new(int), &seq, &path); err != nil {
		t.Fatalf("resolve db path: %v", err)
	}
	if err := backupDatabaseBeforeMigration(db, path); err != nil {
		t.Fatalf("backup failed: %v", err)
	}
	// No path separators may leak into the backup filename.
	matches, err := filepath.Glob(path + ".pre-*.bak")
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected exactly one backup, got %v", matches)
	}
	base := filepath.Base(matches[0])
	if strings.ContainsAny(base, "/\\;") {
		t.Fatalf("backup filename contains unsafe chars: %s", base)
	}
}
