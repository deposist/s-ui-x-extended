package database

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestPreparePanelUpdateSnapshotIncludesCommittedWALRows(t *testing.T) {
	databasePath := openPanelUpdateTestDatabase(t)
	live := GetDB()
	sqlDB, err := live.DB()
	if err != nil {
		t.Fatal(err)
	}
	connection, err := sqlDB.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()

	if _, err := connection.ExecContext(context.Background(), "CREATE TABLE update_state (value TEXT NOT NULL)"); err != nil {
		t.Fatal(err)
	}
	if _, err := connection.ExecContext(context.Background(), "PRAGMA wal_checkpoint(TRUNCATE)"); err != nil {
		t.Fatal(err)
	}
	if _, err := connection.ExecContext(context.Background(), "PRAGMA wal_autocheckpoint=0"); err != nil {
		t.Fatal(err)
	}
	if _, err := connection.ExecContext(context.Background(), "INSERT INTO update_state(value) VALUES ('committed-in-wal')"); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(databasePath + "-wal"); err != nil {
		t.Fatalf("expected live WAL after committed insert: %v", err)
	} else if info.Size() == 0 {
		t.Fatal("expected committed row to remain in non-empty WAL")
	}

	snapshotPath, sourcePath, err := PreparePanelUpdateSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cleanupBackupTempFiles(snapshotPath) })
	if !samePanelUpdatePath(sourcePath, databasePath) {
		t.Fatalf("source path=%q, want %q", sourcePath, databasePath)
	}

	snapshot := openPanelUpdateSnapshot(t, snapshotPath)
	var value string
	if err := snapshot.Raw("SELECT value FROM update_state").Scan(&value).Error; err != nil {
		t.Fatal(err)
	}
	if value != "committed-in-wal" {
		t.Fatalf("snapshot value=%q, want committed WAL row", value)
	}

	snapshotSQLDB, err := snapshot.DB()
	if err != nil {
		t.Fatal(err)
	}
	if err := snapshotSQLDB.Close(); err != nil {
		t.Fatal(err)
	}

	movedPath := snapshotPath + ".moved"
	if err := os.Rename(snapshotPath, movedPath); err != nil {
		t.Fatalf("returned snapshot was not closed: %v", err)
	}
	if err := os.Rename(movedPath, snapshotPath); err != nil {
		t.Fatal(err)
	}
}

func TestPreparePanelUpdateSnapshotDoesNotBlockUnrelatedOperations(t *testing.T) {
	openPanelUpdateTestDatabase(t)
	if err := GetDB().Exec("CREATE TABLE update_state (value TEXT NOT NULL)").Error; err != nil {
		t.Fatal(err)
	}

	enteredSync := make(chan struct{})
	releaseSync := make(chan struct{})
	previousSync := panelUpdateSyncFile
	panelUpdateSyncFile = func(string) error {
		close(enteredSync)
		<-releaseSync
		return nil
	}
	t.Cleanup(func() { panelUpdateSyncFile = previousSync })

	type snapshotResult struct {
		path string
		err  error
	}
	snapshotDone := make(chan snapshotResult, 1)
	go func() {
		path, _, err := PreparePanelUpdateSnapshot()
		snapshotDone <- snapshotResult{path: path, err: err}
	}()
	select {
	case <-enteredSync:
	case <-time.After(5 * time.Second):
		t.Fatal("snapshot did not reach sync seam")
	}

	operationEntered := make(chan struct{})
	go func() {
		leave := EnterDBOperation()
		close(operationEntered)
		leave()
	}()
	select {
	case <-operationEntered:
	case <-time.After(time.Second):
		t.Fatal("snapshot blocked an unrelated database operation")
	}

	close(releaseSync)
	result := <-snapshotDone
	if result.err != nil {
		t.Fatal(result.err)
	}
	cleanupBackupTempFiles(result.path)
}

func TestPreparePanelUpdateSnapshotCleansFailedOutput(t *testing.T) {
	openPanelUpdateTestDatabase(t)
	if err := GetDB().Exec("CREATE TABLE update_state (value TEXT NOT NULL)").Error; err != nil {
		t.Fatal(err)
	}

	sentinel := errors.New("snapshot sync failed")
	failedPath := ""
	previousSync := panelUpdateSyncFile
	panelUpdateSyncFile = func(path string) error {
		failedPath = path
		return sentinel
	}
	t.Cleanup(func() { panelUpdateSyncFile = previousSync })

	if _, _, err := PreparePanelUpdateSnapshot(); !errors.Is(err, sentinel) {
		t.Fatalf("snapshot error=%v, want sync failure", err)
	}
	if failedPath == "" {
		t.Fatal("sync seam did not receive snapshot path")
	}
	if _, err := os.Stat(failedPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("failed snapshot still exists at %q: %v", failedPath, err)
	}
	for _, suffix := range []string{"-wal", "-shm", "-journal"} {
		if _, err := os.Stat(failedPath + suffix); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("failed snapshot sidecar still exists at %q: %v", failedPath+suffix, err)
		}
	}
}

func TestRestorePanelUpdateSnapshotRestoresExactSchemaAndData(t *testing.T) {
	databasePath := openPanelUpdateTestDatabase(t)
	if err := GetDB().Exec("CREATE TABLE update_state (value TEXT NOT NULL)").Error; err != nil {
		t.Fatal(err)
	}
	if err := GetDB().Exec("INSERT INTO update_state(value) VALUES ('before-update')").Error; err != nil {
		t.Fatal(err)
	}
	snapshotPath, _, err := PreparePanelUpdateSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cleanupBackupTempFiles(snapshotPath) })

	if err := GetDB().Exec("ALTER TABLE update_state ADD COLUMN update_only TEXT").Error; err != nil {
		t.Fatal(err)
	}
	if err := GetDB().Exec("UPDATE update_state SET value='after-update', update_only='new'").Error; err != nil {
		t.Fatal(err)
	}
	if err := GetDB().Exec("CREATE TABLE update_only_table (id INTEGER PRIMARY KEY)").Error; err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(databasePath+"-journal", []byte("stale"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := RestorePanelUpdateSnapshot(snapshotPath, databasePath); err != nil {
		t.Fatal(err)
	}
	var value string
	if err := GetDB().Raw("SELECT value FROM update_state").Scan(&value).Error; err != nil {
		t.Fatal(err)
	}
	if value != "before-update" {
		t.Fatalf("restored value=%q, want pre-update data", value)
	}
	var updateOnlyColumns int64
	if err := GetDB().Raw("SELECT COUNT(*) FROM pragma_table_info('update_state') WHERE name = 'update_only'").Scan(&updateOnlyColumns).Error; err != nil {
		t.Fatal(err)
	}
	if updateOnlyColumns != 0 {
		t.Fatal("post-snapshot column survived exact restore")
	}
	var updateOnlyTables int64
	if err := GetDB().Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='update_only_table'").Scan(&updateOnlyTables).Error; err != nil {
		t.Fatal(err)
	}
	if updateOnlyTables != 0 {
		t.Fatal("post-snapshot table survived exact restore")
	}
	if _, err := os.Stat(databasePath + "-journal"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("stale rollback journal was not removed: %v", err)
	}
}

func TestRestorePanelUpdateSnapshotRestoresBeforeDatabaseOpen(t *testing.T) {
	databasePath := openPanelUpdateTestDatabase(t)
	if err := GetDB().Exec("CREATE TABLE update_state (value TEXT NOT NULL)").Error; err != nil {
		t.Fatal(err)
	}
	if err := GetDB().Exec("INSERT INTO update_state(value) VALUES ('snapshot')").Error; err != nil {
		t.Fatal(err)
	}
	snapshotPath, _, err := PreparePanelUpdateSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cleanupBackupTempFiles(snapshotPath) })
	if err := GetDB().Exec("UPDATE update_state SET value='changed'").Error; err != nil {
		t.Fatal(err)
	}
	closeMainDB(t)

	if err := RestorePanelUpdateSnapshot(snapshotPath, databasePath); err != nil {
		t.Fatal(err)
	}
	if err := validatePanelUpdateSQLiteFile(databasePath); err != nil {
		t.Fatal(err)
	}
	probe := openPanelUpdateSnapshot(t, databasePath)
	var value string
	if err := probe.Raw("SELECT value FROM update_state").Scan(&value).Error; err != nil {
		t.Fatal(err)
	}
	if value != "snapshot" {
		t.Fatalf("offline restored value=%q, want snapshot", value)
	}
}

func TestRestorePanelUpdateSnapshotRejectsCorruptionWithoutChangingLiveDB(t *testing.T) {
	databasePath := openPanelUpdateTestDatabase(t)
	if err := GetDB().Exec("CREATE TABLE update_state (value TEXT NOT NULL)").Error; err != nil {
		t.Fatal(err)
	}
	if err := GetDB().Exec("INSERT INTO update_state(value) VALUES ('live')").Error; err != nil {
		t.Fatal(err)
	}
	corruptPath := filepath.Join(t.TempDir(), "corrupt.db")
	if err := os.WriteFile(corruptPath, []byte("SQLite format 3\x00corrupt"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := RestorePanelUpdateSnapshot(corruptPath, databasePath); err == nil {
		t.Fatal("corrupted snapshot was accepted")
	}
	assertPanelUpdateValue(t, "live")
}

func TestRestorePanelUpdateSnapshotRejectsWrongTarget(t *testing.T) {
	databasePath := openPanelUpdateTestDatabase(t)
	if err := GetDB().Exec("CREATE TABLE update_state (value TEXT NOT NULL)").Error; err != nil {
		t.Fatal(err)
	}
	if err := GetDB().Exec("INSERT INTO update_state(value) VALUES ('live')").Error; err != nil {
		t.Fatal(err)
	}
	snapshotPath, _, err := PreparePanelUpdateSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cleanupBackupTempFiles(snapshotPath) })
	wrongPath := filepath.Join(filepath.Dir(databasePath), "other.db")

	if err := RestorePanelUpdateSnapshot(snapshotPath, wrongPath); err == nil || !strings.Contains(err.Error(), "refusing target") {
		t.Fatalf("wrong-target error=%v", err)
	}
	if _, err := os.Stat(wrongPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("wrong target was changed: %v", err)
	}
	assertPanelUpdateValue(t, "live")
}

func TestRestorePanelUpdateSnapshotRenameFailureRestoresLiveDatabase(t *testing.T) {
	databasePath, snapshotPath := preparePanelUpdateFailureState(t)
	sentinel := errors.New("install rename failed")
	previousRename := panelUpdateRename
	renameCalls := 0
	panelUpdateRename = func(source string, destination string) error {
		renameCalls++
		if renameCalls == 2 {
			return sentinel
		}
		return previousRename(source, destination)
	}
	t.Cleanup(func() { panelUpdateRename = previousRename })

	if err := RestorePanelUpdateSnapshot(snapshotPath, databasePath); !errors.Is(err, sentinel) {
		t.Fatalf("restore error=%v, want rename failure", err)
	}
	if renameCalls < 3 {
		t.Fatalf("rename calls=%d, want fallback rollback rename", renameCalls)
	}
	assertPanelUpdateValue(t, "current-live")
	assertValidPanelUpdateDatabase(t, databasePath)
}

func TestRestorePanelUpdateSnapshotSyncFailureRestoresLiveDatabase(t *testing.T) {
	databasePath, snapshotPath := preparePanelUpdateFailureState(t)
	sentinel := errors.New("restored database sync failed")
	previousSync := panelUpdateSyncFile
	syncCalls := 0
	panelUpdateSyncFile = func(path string) error {
		syncCalls++
		if syncCalls == 1 {
			return sentinel
		}
		return previousSync(path)
	}
	t.Cleanup(func() { panelUpdateSyncFile = previousSync })

	if err := RestorePanelUpdateSnapshot(snapshotPath, databasePath); !errors.Is(err, sentinel) {
		t.Fatalf("restore error=%v, want sync failure", err)
	}
	if syncCalls < 2 {
		t.Fatalf("sync calls=%d, want fallback sync during rollback", syncCalls)
	}
	assertPanelUpdateValue(t, "current-live")
	assertValidPanelUpdateDatabase(t, databasePath)
}

func TestRestorePanelUpdateSnapshotDirectorySyncFailureRestoresLiveDatabase(t *testing.T) {
	databasePath, snapshotPath := preparePanelUpdateFailureState(t)
	sentinel := errors.New("restored database directory sync failed")
	previousSync := panelUpdateSyncDirectory
	syncCalls := 0
	panelUpdateSyncDirectory = func(path string) error {
		syncCalls++
		if syncCalls == 1 {
			return sentinel
		}
		return previousSync(path)
	}
	t.Cleanup(func() { panelUpdateSyncDirectory = previousSync })

	if err := RestorePanelUpdateSnapshot(snapshotPath, databasePath); !errors.Is(err, sentinel) {
		t.Fatalf("restore error=%v, want directory sync failure", err)
	}
	if syncCalls < 2 {
		t.Fatalf("directory sync calls=%d, want fallback sync during rollback", syncCalls)
	}
	assertPanelUpdateValue(t, "current-live")
	assertValidPanelUpdateDatabase(t, databasePath)
}

func openPanelUpdateTestDatabase(t *testing.T) string {
	t.Helper()
	directory := makeDBTempDir(t, "s-ui-panel-update-db-")
	t.Setenv("SUI_DB_FOLDER", directory)
	databasePath := filepath.Join(directory, "s-ui.db")
	if err := OpenDB(databasePath); err != nil {
		if strings.Contains(err.Error(), "go-sqlite3 requires cgo") {
			t.Skip(err)
		}
		t.Fatal(err)
	}
	t.Cleanup(func() {
		closeMainDB(t)
		cleanupBackupSidecars(databasePath)
	})
	return databasePath
}

func openPanelUpdateSnapshot(t *testing.T, path string) *gorm.DB {
	t.Helper()
	snapshot, err := gorm.Open(sqlite.Open(sqliteReadOnlyDSN(path)), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := snapshot.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return snapshot
}

func preparePanelUpdateFailureState(t *testing.T) (string, string) {
	t.Helper()
	databasePath := openPanelUpdateTestDatabase(t)
	if err := GetDB().Exec("CREATE TABLE update_state (value TEXT NOT NULL)").Error; err != nil {
		t.Fatal(err)
	}
	if err := GetDB().Exec("INSERT INTO update_state(value) VALUES ('snapshot')").Error; err != nil {
		t.Fatal(err)
	}
	snapshotPath, _, err := PreparePanelUpdateSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cleanupBackupTempFiles(snapshotPath) })
	if err := GetDB().Exec("UPDATE update_state SET value='current-live'").Error; err != nil {
		t.Fatal(err)
	}
	return databasePath, snapshotPath
}

func assertPanelUpdateValue(t *testing.T, want string) {
	t.Helper()
	var value string
	if err := GetDB().Raw("SELECT value FROM update_state").Scan(&value).Error; err != nil {
		t.Fatalf("query recoverable database: %v", err)
	}
	if value != want {
		t.Fatalf("live value=%q, want %q", value, want)
	}
}

func assertValidPanelUpdateDatabase(t *testing.T, path string) {
	t.Helper()
	if err := validatePanelUpdateSQLiteFile(path); err != nil {
		t.Fatalf("database is not recoverable: %v", err)
	}
}
