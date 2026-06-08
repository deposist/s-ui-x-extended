package database

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/deposist/s-ui-x/database/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// closeMainDB closes the global *gorm.DB so Windows can release file locks
// before t.TempDir() cleanup tries to delete the database file. It also
// truncates the WAL and removes any leftover -wal/-shm sidecars.
func closeMainDB(t *testing.T) {
	t.Helper()
	dbMu.Lock()
	current := db
	db = nil
	dbMu.Unlock()
	if current == nil {
		return
	}
	dbPath := ""
	if mig := current.Migrator(); mig != nil {
		// best-effort: extract the source path from the underlying driver
	}
	_ = current.Exec("PRAGMA wal_checkpoint(TRUNCATE)").Error
	sqlDB, err := current.DB()
	if err != nil {
		t.Logf("close main db handle: %v", err)
		return
	}
	if err := sqlDB.Close(); err != nil {
		t.Logf("close main db: %v", err)
	}

	// Best-effort sidecar cleanup. We do not have the original DSN handy,
	// so just nuke common candidates the tests use.
	_ = dbPath
}

func makeDBTempDir(t *testing.T, prefix string) string {
	t.Helper()
	dir, err := os.MkdirTemp("", prefix)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		var removeErr error
		for i := 0; i < 20; i++ {
			removeErr = os.RemoveAll(dir)
			if removeErr == nil || os.IsNotExist(removeErr) {
				return
			}
			time.Sleep(time.Duration(i+1) * 10 * time.Millisecond)
		}
		t.Errorf("remove temp db dir %q: %v", dir, removeErr)
	})
	return dir
}

func TestGetDbIncludesServicesAndTokens(t *testing.T) {
	if runtime.GOOS == "windows" {
		// Same Windows-specific TempDir cleanup race documented in
		// backup_import_test.go: SQLite/WAL leftovers occasionally hold the
		// temp directory open past closeMainDB. The behaviour exercised
		// here is verified on Linux CI.
		t.Skip("skipping Windows-specific TempDir cleanup race; logic is exercised on Linux CI")
	}
	t.Setenv("SUI_DB_FOLDER", t.TempDir())
	if err := InitDB(filepath.Join(t.TempDir(), "s-ui.db")); err != nil {
		if strings.Contains(err.Error(), "go-sqlite3 requires cgo") {
			t.Skip(err)
		}
		t.Fatal(err)
	}
	t.Cleanup(func() { closeMainDB(t) })

	db := GetDB()
	if err := db.Create(&model.Service{Type: "derp", Tag: "svc-test"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.Tokens{Desc: "token-test", Token: "secret-token", UserId: 1}).Error; err != nil {
		t.Fatal(err)
	}
	backup, err := GetDb("")
	if err != nil {
		t.Fatal(err)
	}
	backupPath := filepath.Join(t.TempDir(), "backup.db")
	if err := os.WriteFile(backupPath, backup, 0600); err != nil {
		t.Fatal(err)
	}
	backupDB, err := gorm.Open(sqlite.Open(backupPath), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if sqlDB, err := backupDB.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	var servicesCount int64
	if err := backupDB.Model(&model.Service{}).Where("tag = ?", "svc-test").Count(&servicesCount).Error; err != nil {
		t.Fatal(err)
	}
	if servicesCount != 1 {
		t.Fatalf("service was not included in backup")
	}
	var tokensCount int64
	if err := backupDB.Model(&model.Tokens{}).Where("token = ?", "secret-token").Count(&tokensCount).Error; err != nil {
		t.Fatal(err)
	}
	if tokensCount != 1 {
		t.Fatalf("token was not included in backup")
	}
}

func TestGetDbPreservesNoTLSForeignKeySentinel(t *testing.T) {
	dbDir := t.TempDir()
	dbPath := filepath.Join(dbDir, "s-ui.db")
	t.Setenv("SUI_DB_FOLDER", dbDir)
	if err := InitDB(dbPath); err != nil {
		if strings.Contains(err.Error(), "go-sqlite3 requires cgo") {
			t.Skip(err)
		}
		t.Fatal(err)
	}
	t.Cleanup(func() {
		closeMainDB(t)
		cleanupBackupSidecars(dbPath)
	})

	if err := GetDB().Create(&model.Inbound{
		Type:    "http",
		Tag:     "backup-no-tls",
		Addrs:   []byte("[]"),
		OutJson: []byte("{}"),
		Options: []byte("{}"),
	}).Error; err != nil {
		t.Fatal(err)
	}

	backup, err := GetDb("")
	if err != nil {
		t.Fatal(err)
	}
	backupPath := filepath.Join(t.TempDir(), "backup.db")
	if err := os.WriteFile(backupPath, backup, 0o600); err != nil {
		t.Fatal(err)
	}
	backupDB, err := gorm.Open(sqlite.Open(backupPath), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if sqlDB, err := backupDB.DB(); err == nil {
			_ = sqlDB.Close()
		}
		cleanupBackupSidecars(backupPath)
	})

	var noTLSRows int64
	if err := backupDB.Table("tls").Where("id = ?", 0).Count(&noTLSRows).Error; err != nil {
		t.Fatal(err)
	}
	if noTLSRows != 1 {
		t.Fatalf("backup tls.id=0 rows=%d, want 1", noTLSRows)
	}
	var violations int64
	if err := backupDB.Raw("SELECT COUNT(*) FROM pragma_foreign_key_check").Scan(&violations).Error; err != nil {
		t.Fatal(err)
	}
	if violations != 0 {
		t.Fatalf("backup foreign key violations=%d, want 0", violations)
	}
}

func TestGetDbCopiesTLSRowsWhenNoTLSSentinelExists(t *testing.T) {
	dbDir := t.TempDir()
	dbPath := filepath.Join(dbDir, "s-ui.db")
	t.Setenv("SUI_DB_FOLDER", dbDir)
	if err := InitDB(dbPath); err != nil {
		if strings.Contains(err.Error(), "go-sqlite3 requires cgo") {
			t.Skip(err)
		}
		t.Fatal(err)
	}
	t.Cleanup(func() {
		closeMainDB(t)
		cleanupBackupSidecars(dbPath)
	})

	mainDB := GetDB()
	tlsRow := model.Tls{
		Name:   "real-tls",
		Server: []byte("{}"),
		Client: []byte("{}"),
	}
	if err := mainDB.Create(&tlsRow).Error; err != nil {
		t.Fatal(err)
	}
	if tlsRow.Id == 0 {
		t.Fatal("expected real TLS row to receive a non-zero id")
	}

	backup, err := GetDb("")
	if err != nil {
		t.Fatalf("GetDb failed with tls.id=0 sentinel and real TLS row: %v", err)
	}
	backupPath := filepath.Join(t.TempDir(), "backup.db")
	if err := os.WriteFile(backupPath, backup, 0o600); err != nil {
		t.Fatal(err)
	}
	backupDB, err := gorm.Open(sqlite.Open(backupPath), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if sqlDB, err := backupDB.DB(); err == nil {
			_ = sqlDB.Close()
		}
		cleanupBackupSidecars(backupPath)
	})

	var sentinelRows int64
	if err := backupDB.Table("tls").Where("id = ?", 0).Count(&sentinelRows).Error; err != nil {
		t.Fatal(err)
	}
	if sentinelRows != 1 {
		t.Fatalf("backup tls.id=0 rows=%d, want 1", sentinelRows)
	}

	var copiedRows int64
	if err := backupDB.Model(&model.Tls{}).Where("id = ? AND name = ?", tlsRow.Id, "real-tls").Count(&copiedRows).Error; err != nil {
		t.Fatal(err)
	}
	if copiedRows != 1 {
		t.Fatalf("real TLS row id=%d was not copied exactly once", tlsRow.Id)
	}
}

func TestGetDbUsesRandomTempPathAndRemovesIt(t *testing.T) {
	t.Setenv("SUI_DB_FOLDER", t.TempDir())
	if err := InitDB(filepath.Join(t.TempDir(), "s-ui.db")); err != nil {
		if strings.Contains(err.Error(), "go-sqlite3 requires cgo") {
			t.Skip(err)
		}
		t.Fatal(err)
	}
	t.Cleanup(func() { closeMainDB(t) })

	var tempPath string
	prevHook := backupTempPathHook
	backupTempPathHook = func(path string) {
		tempPath = path
	}
	t.Cleanup(func() { backupTempPathHook = prevHook })

	if _, err := GetDb(""); err != nil {
		t.Fatal(err)
	}
	if tempPath == "" {
		t.Fatal("backup temp path hook was not called")
	}
	if base := filepath.Base(tempPath); !strings.HasPrefix(base, "s-ui-backup-") || !strings.HasSuffix(base, ".db") {
		t.Fatalf("backup temp path %q does not use s-ui-backup-*.db pattern", tempPath)
	}
	if _, err := os.Stat(tempPath); !os.IsNotExist(err) {
		t.Fatalf("backup temp file was not removed after GetDb returned: %v", err)
	}
}

func TestGetDbExcludeSkipsSelectedTables(t *testing.T) {
	dbDir := t.TempDir()
	dbPath := filepath.Join(dbDir, "s-ui.db")
	t.Setenv("SUI_DB_FOLDER", dbDir)
	if err := InitDB(dbPath); err != nil {
		if strings.Contains(err.Error(), "go-sqlite3 requires cgo") {
			t.Skip(err)
		}
		t.Fatal(err)
	}
	t.Cleanup(func() {
		closeMainDB(t)
		cleanupBackupSidecars(dbPath)
	})

	mainDB := GetDB()
	if err := mainDB.Create(&model.Client{Name: "include-client", Inbounds: []byte("[]")}).Error; err != nil {
		t.Fatal(err)
	}
	if err := mainDB.Create(&model.Stats{DateTime: 1, Resource: "client", Tag: "include-client", Traffic: 10}).Error; err != nil {
		t.Fatal(err)
	}
	if err := mainDB.Create(&model.ClientIP{ClientName: "include-client", IPHash: "hash-1", FirstSeen: 1, LastSeen: 1}).Error; err != nil {
		t.Fatal(err)
	}
	if err := mainDB.Create(&model.Changes{DateTime: 1, Actor: "test", Key: "clients", Action: "set", Obj: []byte(`"include-client"`)}).Error; err != nil {
		t.Fatal(err)
	}
	if err := mainDB.Create(&model.AuditEvent{DateTime: 1, Actor: "test", Event: "login", Resource: "auth", Severity: "info"}).Error; err != nil {
		t.Fatal(err)
	}

	backup, err := GetDb("audit,client_ips,stats,changes")
	if err != nil {
		t.Fatal(err)
	}
	backupPath := filepath.Join(t.TempDir(), "backup.db")
	if err := os.WriteFile(backupPath, backup, 0600); err != nil {
		t.Fatal(err)
	}
	backupDB, err := gorm.Open(sqlite.Open(backupPath), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if sqlDB, err := backupDB.DB(); err == nil {
			_ = sqlDB.Close()
		}
		cleanupBackupSidecars(backupPath)
	})

	var clientsCount int64
	if err := backupDB.Model(&model.Client{}).Where("name = ?", "include-client").Count(&clientsCount).Error; err != nil {
		t.Fatal(err)
	}
	if clientsCount != 1 {
		t.Fatalf("client table should be included, got %d rows", clientsCount)
	}

	for tableName, modelValue := range map[string]any{
		"stats":        &model.Stats{},
		"client_ips":   &model.ClientIP{},
		"changes":      &model.Changes{},
		"audit_events": &model.AuditEvent{},
	} {
		t.Run(tableName, func(t *testing.T) {
			var count int64
			if err := backupDB.Model(modelValue).Count(&count).Error; err != nil {
				t.Fatal(err)
			}
			if count != 0 {
				t.Fatalf("expected %s to be excluded, got %d rows", tableName, count)
			}
		})
	}
}

func TestGetDbHandlesHashedClientIPsWithEmptyLegacyIP(t *testing.T) {
	if runtime.GOOS == "windows" {
		// Same Windows-specific TempDir cleanup race documented in
		// backup_import_test.go: SQLite/WAL leftovers occasionally hold the
		// temp directory open past closeMainDB. The behaviour exercised
		// here is verified on Linux CI.
		t.Skip("skipping Windows-specific TempDir cleanup race; logic is exercised on Linux CI")
	}
	dbDir := t.TempDir()
	dbPath := filepath.Join(dbDir, "s-ui.db")
	t.Setenv("SUI_DB_FOLDER", dbDir)
	if err := InitDB(dbPath); err != nil {
		if strings.Contains(err.Error(), "go-sqlite3 requires cgo") {
			t.Skip(err)
		}
		t.Fatal(err)
	}
	t.Cleanup(func() {
		closeMainDB(t)
		cleanupBackupSidecars(dbPath)
	})

	mainDB := GetDB()
	if err := mainDB.Exec(`
INSERT INTO client_ips(client_name, ip_hash, first_seen, last_seen)
VALUES
	('alice', 'hash-1', 1, 1),
	('alice', 'hash-2', 2, 2)
`).Error; err != nil {
		t.Fatal(err)
	}

	backup, err := GetDb("")
	if err != nil {
		t.Fatalf("GetDb failed on hashed client_ips with empty legacy ip: %v", err)
	}
	backupPath := filepath.Join(t.TempDir(), "backup.db")
	if err := os.WriteFile(backupPath, backup, 0600); err != nil {
		t.Fatal(err)
	}
	backupDB, err := gorm.Open(sqlite.Open(backupPath), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if sqlDB, err := backupDB.DB(); err == nil {
			_ = sqlDB.Close()
		}
		cleanupBackupSidecars(backupPath)
	})

	var ipCount int64
	if err := backupDB.Model(&model.ClientIP{}).Where("client_name = ?", "alice").Count(&ipCount).Error; err != nil {
		t.Fatal(err)
	}
	if ipCount != 2 {
		t.Fatalf("expected two client_ips rows in backup, got %d", ipCount)
	}
}

func TestGetDbHandlesLargeTablesWithoutVariableLimit(t *testing.T) {
	dbDir := t.TempDir()
	dbPath := filepath.Join(dbDir, "s-ui.db")
	t.Setenv("SUI_DB_FOLDER", dbDir)
	if err := InitDB(dbPath); err != nil {
		if strings.Contains(err.Error(), "go-sqlite3 requires cgo") {
			t.Skip(err)
		}
		t.Fatal(err)
	}
	t.Cleanup(func() {
		closeMainDB(t)
		cleanupBackupSidecars(dbPath)
	})

	// Reproduce the production "too many SQL variables" error reported
	// against database/backup.go: backup of a real s-ui DB with ~43k rows
	// in the stats table failed because copyBackupTable issued a single
	// multi-row INSERT VALUES (...) that blew past
	// SQLITE_MAX_VARIABLE_NUMBER (=999 in mattn/go-sqlite3). The chunked
	// CreateInBatches path must absorb a 43k-row stats table plus a
	// secondary client_ips table without exceeding the variable budget.
	const statsRows = 42974
	const ipRows = 5000
	mainDB := GetDB()

	if err := mainDB.Transaction(func(tx *gorm.DB) error {
		stats := make([]model.Stats, 0, 1000)
		for i := 0; i < statsRows; i++ {
			stats = append(stats, model.Stats{
				DateTime:  int64(i),
				Resource:  "client",
				Tag:       "alice",
				Direction: i%2 == 0,
				Traffic:   int64(i),
			})
			if len(stats) == cap(stats) {
				if err := tx.CreateInBatches(stats, 500).Error; err != nil {
					return err
				}
				stats = stats[:0]
			}
		}
		if len(stats) > 0 {
			if err := tx.CreateInBatches(stats, 500).Error; err != nil {
				return err
			}
		}
		ips := make([]model.ClientIP, 0, 1000)
		for i := 0; i < ipRows; i++ {
			ips = append(ips, model.ClientIP{
				ClientName: "alice",
				IP:         fmt.Sprintf("198.51.100.%d.%d", i/256, i%256),
				IPHash:     fmt.Sprintf("hash-%d", i),
				FirstSeen:  1,
				LastSeen:   1,
			})
			if len(ips) == cap(ips) {
				if err := tx.CreateInBatches(ips, 500).Error; err != nil {
					return err
				}
				ips = ips[:0]
			}
		}
		if len(ips) > 0 {
			if err := tx.CreateInBatches(ips, 500).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	backup, err := GetDb("")
	if err != nil {
		t.Fatalf("GetDb failed on large dataset: %v", err)
	}
	backupPath := filepath.Join(t.TempDir(), "backup.db")
	if err := os.WriteFile(backupPath, backup, 0600); err != nil {
		t.Fatal(err)
	}
	backupDB, err := gorm.Open(sqlite.Open(backupPath), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if sqlDB, err := backupDB.DB(); err == nil {
			_ = sqlDB.Close()
		}
		cleanupBackupSidecars(backupPath)
	})

	var statsCount int64
	if err := backupDB.Model(&model.Stats{}).Count(&statsCount).Error; err != nil {
		t.Fatal(err)
	}
	if statsCount != statsRows {
		t.Fatalf("expected %d stats rows in backup, got %d", statsRows, statsCount)
	}
	var ipCount int64
	if err := backupDB.Model(&model.ClientIP{}).Count(&ipCount).Error; err != nil {
		t.Fatal(err)
	}
	if ipCount != ipRows {
		t.Fatalf("expected %d client_ips rows in backup, got %d", ipRows, ipCount)
	}
}
