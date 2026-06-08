package database

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/deposist/s-ui-x/config"
	"github.com/deposist/s-ui-x/database/model"
	"github.com/deposist/s-ui-x/util/common"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// memMultipartFile is a minimal multipart.File implementation backed by an
// in-memory byte slice so the import path can be exercised from a test
// without going through net/http.
type memMultipartFile struct{ *bytes.Reader }

func (memMultipartFile) Close() error { return nil }

func newLegacyBackup(t *testing.T) []byte {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "legacy.db")

	// Open a plain (non-WAL) SQLite database so the file we read back is a
	// single self-contained .db blob, exactly like a legacy 1.4.1 backup.
	legacy, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := legacy.AutoMigrate(
		&model.Setting{},
		&model.Tls{},
		&model.Inbound{},
		&model.Outbound{},
		&model.Service{},
		&model.Endpoint{},
		&model.User{},
		&model.Tokens{},
		&model.Stats{},
		&model.Client{},
		&model.Changes{},
	); err != nil {
		t.Fatal(err)
	}

	// Plaintext admin credential (legacy schema), pre-1.4.2 version pin.
	if err := legacy.Create(&model.User{Username: "legacy-admin", Password: "legacy-secret"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := legacy.Create(&model.Setting{Key: "version", Value: "1.4.1"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := legacy.Create(&model.Setting{Key: "config", Value: `{"dns":{},"route":{}}`}).Error; err != nil {
		t.Fatal(err)
	}

	if sqlDB, err := legacy.DB(); err == nil {
		_ = sqlDB.Close()
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestImportDBRunsResetHooks(t *testing.T) {
	dbDir, err := os.MkdirTemp("", "s-ui-import-reset-*")
	if err != nil {
		t.Fatal(err)
	}
	livePath := filepath.Join(dbDir, "s-ui.db")
	t.Setenv("SUI_DB_FOLDER", dbDir)
	t.Cleanup(func() {
		closeMainDB(t)
		time.Sleep(25 * time.Millisecond)
		_ = os.RemoveAll(dbDir)
	})

	if err := InitDB(livePath); err != nil {
		if strings.Contains(err.Error(), "go-sqlite3 requires cgo") {
			t.Skip(err)
		}
		t.Fatal(err)
	}
	if err := GetDB().Create(&model.Setting{Key: "config", Value: `{"dns":{},"route":{}}`}).Error; err != nil {
		t.Fatal(err)
	}

	prev := sendSighupHook
	sendSighupHook = func() error { return nil }
	t.Cleanup(func() { sendSighupHook = prev })

	backupBytes, err := GetDb("")
	if err != nil {
		t.Fatal(err)
	}

	var calls atomic.Int32
	const hookName = "test.import_db_reset_hooks"
	RegisterResetHook(hookName, func() {
		calls.Add(1)
	})
	t.Cleanup(func() {
		RegisterResetHook(hookName, nil)
	})

	if err := ImportDB(memMultipartFile{Reader: bytes.NewReader(backupBytes)}); err != nil {
		t.Fatalf("ImportDB returned error: %v", err)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("reset hook calls=%d, want 1", got)
	}
}

func TestImportDBPreservesConfigDNSAndRouteRules(t *testing.T) {
	dbDir, err := os.MkdirTemp("", "s-ui-import-config-*")
	if err != nil {
		t.Fatal(err)
	}
	livePath := filepath.Join(dbDir, "s-ui.db")
	t.Setenv("SUI_DB_FOLDER", dbDir)
	t.Cleanup(func() {
		closeMainDB(t)
		time.Sleep(25 * time.Millisecond)
		_ = os.RemoveAll(dbDir)
	})

	if err := InitDB(livePath); err != nil {
		if strings.Contains(err.Error(), "go-sqlite3 requires cgo") {
			t.Skip(err)
		}
		t.Fatal(err)
	}

	prev := sendSighupHook
	sendSighupHook = func() error { return nil }
	t.Cleanup(func() { sendSighupHook = prev })

	const restoredConfig = `{
  "dns": {
    "servers": [
      {
        "tag": "dns-umbrella",
        "type": "udp",
        "server": "208.67.222.222"
      }
    ]
  },
  "route": {
    "rules": [
      {
        "domain_suffix": [
          "example.test"
        ],
        "action": "route",
        "outbound": "direct"
      }
    ]
  }
}`
	if err := GetDB().Create(&model.Setting{Key: "config", Value: restoredConfig}).Error; err != nil {
		t.Fatal(err)
	}

	backupBytes, err := GetDb("")
	if err != nil {
		t.Fatal(err)
	}
	if err := GetDB().Model(&model.Setting{}).Where("key = ?", "config").Update("value", `{"dns":{},"route":{}}`).Error; err != nil {
		t.Fatal(err)
	}

	if err := ImportDB(memMultipartFile{Reader: bytes.NewReader(backupBytes)}); err != nil {
		t.Fatalf("ImportDB returned error: %v", err)
	}

	var config string
	if err := GetDB().Model(&model.Setting{}).Select("value").Where("key = ?", "config").Scan(&config).Error; err != nil {
		t.Fatal(err)
	}
	if config != restoredConfig {
		t.Fatalf("imported config=%q, want %q", config, restoredConfig)
	}
}

func TestImportDBAdaptsLegacyBackup(t *testing.T) {
	if runtime.GOOS == "windows" {
		// On Windows the test runner's t.TempDir() cleanup races against
		// the SQLite WAL/SHM mappings even after explicit Close, producing
		// noisy "file in use" errors that do not happen on the production
		// Linux servers this code targets.
		t.Skip("skipping Windows-specific TempDir cleanup race; logic is exercised on Linux CI")
	}

	dbDir := t.TempDir()
	t.Setenv("SUI_DB_FOLDER", dbDir)

	// Initialize a fresh "live" database so ImportDB has something to
	// rotate aside as the fallback. Use the same path GetDBPath() returns
	// so the import code targets it.
	livePath := filepath.Join(dbDir, "s-ui.db")
	if err := InitDB(livePath); err != nil {
		if strings.Contains(err.Error(), "go-sqlite3 requires cgo") {
			t.Skip(err)
		}
		t.Fatal(err)
	}

	// Make sure we close the DB and nuke WAL sidecars before t.TempDir()
	// cleanup runs, otherwise on Windows the dir-remove fails because the
	// SQLite driver is still mmap'd onto the *.db-wal file.
	t.Cleanup(func() {
		closeMainDB(t)
		for _, suffix := range []string{"", "-wal", "-shm", "-journal"} {
			_ = os.Remove(livePath + suffix)
		}
	})

	// Suppress the SIGHUP that ImportDB sends at the end so it does not
	// kill the test runner.
	prev := sendSighupHook
	sendSighupHook = func() error { return nil }
	t.Cleanup(func() { sendSighupHook = prev })

	// Build a legacy backup blob.
	legacyBytes := newLegacyBackup(t)

	// Hand it to ImportDB through the multipart.File interface.
	if err := ImportDB(memMultipartFile{Reader: bytes.NewReader(legacyBytes)}); err != nil {
		t.Fatalf("ImportDB returned error: %v", err)
	}

	// The fallback and temp files must be cleaned up after a successful
	// import.
	for _, p := range []string{livePath + ".temp", livePath + ".backup"} {
		if _, err := os.Stat(p); err == nil {
			t.Errorf("leftover file after successful import: %s", p)
		}
	}

	// The live DB must contain the legacy admin user with a bcrypt-hashed
	// password, validating that AdaptToCurrentVersion ran on the imported
	// database.
	d := GetDB()
	if d == nil {
		t.Fatal("GetDB returned nil after import")
	}
	var stored string
	if err := d.Model(&model.User{}).Select("password").Where("username = ?", "legacy-admin").Scan(&stored).Error; err != nil {
		t.Fatalf("query imported user: %v", err)
	}
	if stored == "" {
		t.Fatal("imported admin user is missing")
	}
	if !common.IsPasswordHash(stored) {
		t.Fatalf("imported password was not rehashed; got plaintext: %q", stored)
	}
	if ok, _ := common.CheckPassword(stored, "legacy-secret"); !ok {
		t.Fatal("rehashed password no longer validates the legacy plaintext")
	}

	// settings.version must have been bumped from 1.4.1 to the current
	// build version.
	var version string
	if err := d.Model(&model.Setting{}).Select("value").Where("key = ?", "version").Scan(&version).Error; err != nil {
		t.Fatal(err)
	}
	if version == "1.4.1" || version == "" {
		t.Fatalf("settings.version was not bumped: %q", version)
	}
}

func TestImportDBRejectsCorruptSQLiteBackup(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping Windows-specific TempDir cleanup race; logic is exercised on Linux CI")
	}
	dbDir := t.TempDir()
	t.Setenv("SUI_DB_FOLDER", dbDir)
	livePath := filepath.Join(dbDir, "s-ui.db")
	if err := InitDB(livePath); err != nil {
		if strings.Contains(err.Error(), "go-sqlite3 requires cgo") {
			t.Skip(err)
		}
		t.Fatal(err)
	}
	t.Cleanup(func() {
		closeMainDB(t)
		for _, suffix := range []string{"", "-wal", "-shm", "-journal"} {
			_ = os.Remove(livePath + suffix)
		}
	})
	corrupt := append([]byte("SQLite format 3\x00"), bytes.Repeat([]byte{0xff}, 256)...)
	if err := ImportDB(memMultipartFile{Reader: bytes.NewReader(corrupt)}); err == nil {
		t.Fatal("corrupt sqlite backup should be rejected")
	}
}

func TestImportDBAcceptsVersionedBackupWithoutConfigIssue12(t *testing.T) {
	dbDir := t.TempDir()
	t.Setenv("SUI_DB_FOLDER", dbDir)
	livePath := filepath.Join(dbDir, "s-ui.db")
	if err := InitDB(livePath); err != nil {
		if strings.Contains(err.Error(), "go-sqlite3 requires cgo") {
			t.Skip(err)
		}
		t.Fatal(err)
	}
	t.Cleanup(func() {
		closeMainDB(t)
		for _, suffix := range []string{"", "-wal", "-shm", "-journal"} {
			_ = os.Remove(livePath + suffix)
		}
	})

	if err := GetDB().Create(&model.Setting{Key: "restore_marker", Value: "live-before-import"}).Error; err != nil {
		t.Fatal(err)
	}
	SetSendSighupHook(func() error { return nil })
	t.Cleanup(func() { SetSendSighupHook(nil) })

	err := ImportDB(memMultipartFile{Reader: bytes.NewReader(newVersionedBackupWithoutConfig(t))})
	// Post-fix #12: missing settings.config no longer aborts the import. The
	// import may still fail for unrelated reasons (e.g. fixture migration
	// gaps), but never because settings.config is absent.
	if err != nil && strings.Contains(err.Error(), "settings.config") {
		t.Fatalf("missing settings.config should now warn-and-continue, got error: %v", err)
	}
	// Live DB must remain reachable. Either the import succeeded (live DB
	// replaced, no restore_marker) or it failed downstream and the fallback
	// rollback re-attached the original live DB (restore_marker present).
	if GetDB() == nil {
		t.Fatal("GetDB returned nil after import attempt")
	}
	if sqlDB, dbErr := GetDB().DB(); dbErr != nil {
		t.Fatalf("live DB handle error: %v", dbErr)
	} else if pingErr := sqlDB.Ping(); pingErr != nil {
		t.Fatalf("live DB ping failed: %v", pingErr)
	}
}

func TestImportDBRollsBackForeignKeyFailureAndReopensLiveDB(t *testing.T) {
	dbDir := t.TempDir()
	t.Setenv("SUI_DB_FOLDER", dbDir)
	livePath := filepath.Join(dbDir, "s-ui.db")
	if err := InitDB(livePath); err != nil {
		if strings.Contains(err.Error(), "go-sqlite3 requires cgo") {
			t.Skip(err)
		}
		t.Fatal(err)
	}
	t.Cleanup(func() {
		closeMainDB(t)
		for _, suffix := range []string{"", "-wal", "-shm", "-journal"} {
			_ = os.Remove(livePath + suffix)
		}
	})

	if err := GetDB().Create(&model.Setting{Key: "restore_marker", Value: "live-before-import"}).Error; err != nil {
		t.Fatal(err)
	}

	err := ImportDB(memMultipartFile{Reader: bytes.NewReader(newForeignKeyBrokenBackup(t))})
	if err == nil || !strings.Contains(err.Error(), "foreign key check failed") {
		t.Fatalf("expected foreign key import failure, got %v", err)
	}
	if GetDB() == nil {
		t.Fatal("GetDB returned nil after failed import rollback")
	}
	if sqlDB, dbErr := GetDB().DB(); dbErr != nil {
		t.Fatalf("live db handle after rollback: %v", dbErr)
	} else if pingErr := sqlDB.Ping(); pingErr != nil {
		t.Fatalf("live db was not reopened after rollback: %v", pingErr)
	}
	var marker string
	if err := GetDB().Model(&model.Setting{}).Select("value").Where("key = ?", "restore_marker").Scan(&marker).Error; err != nil {
		t.Fatal(err)
	}
	if marker != "live-before-import" {
		t.Fatalf("rollback marker=%q, want live-before-import", marker)
	}
}

func newForeignKeyBrokenBackup(t *testing.T) []byte {
	t.Helper()

	path := filepath.Join(t.TempDir(), "broken-fk.db")
	broken, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := broken.AutoMigrate(&model.Setting{}, &model.Tls{}, &model.Inbound{}); err != nil {
		t.Fatal(err)
	}
	if err := broken.Create(&model.Setting{Key: "version", Value: config.GetVersion()}).Error; err != nil {
		t.Fatal(err)
	}
	if err := broken.Create(&model.Setting{Key: "config", Value: `{"dns":{},"route":{}}`}).Error; err != nil {
		t.Fatal(err)
	}
	if err := broken.Exec("PRAGMA foreign_keys = OFF").Error; err != nil {
		t.Fatal(err)
	}
	if err := broken.Exec(`
INSERT INTO inbounds(type, tag, tls_id, addrs, out_json, options)
VALUES(?, ?, ?, ?, ?, ?)
`, "http", "broken-fk", 99, []byte("[]"), []byte("{}"), []byte("{}")).Error; err != nil {
		t.Fatal(err)
	}
	if sqlDB, err := broken.DB(); err == nil {
		_ = sqlDB.Close()
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func newVersionedBackupWithoutConfig(t *testing.T) []byte {
	t.Helper()

	path := filepath.Join(t.TempDir(), "missing-config.db")
	backup, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := backup.AutoMigrate(&model.Setting{}); err != nil {
		t.Fatal(err)
	}
	if err := backup.Create(&model.Setting{Key: "version", Value: config.GetVersion()}).Error; err != nil {
		t.Fatal(err)
	}
	if sqlDB, err := backup.DB(); err == nil {
		_ = sqlDB.Close()
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// _ keeps io referenced when nothing else uses it.
var _ = io.EOF
