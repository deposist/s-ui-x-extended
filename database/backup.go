package database

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/deposist/s-ui-x-extended/cmd/migration"
	"github.com/deposist/s-ui-x-extended/config"
	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/deposist/s-ui-x-extended/logger"
	"github.com/deposist/s-ui-x-extended/util/common"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type backupTable struct {
	name  string
	model any
}

func backupTables() []backupTable {
	return []backupTable{
		{name: "settings", model: &model.Setting{}},
		{name: "tls", model: &model.Tls{}},
		{name: "inbounds", model: &model.Inbound{}},
		{name: "outbounds", model: &model.Outbound{}},
		{name: "services", model: &model.Service{}},
		{name: "endpoints", model: &model.Endpoint{}},
		{name: "providers", model: &model.Provider{}},
		{name: "users", model: &model.User{}},
		{name: "tokens", model: &model.Tokens{}},
		{name: "stats", model: &model.Stats{}},
		{name: "client_ips", model: &model.ClientIP{}},
		{name: "clients", model: &model.Client{}},
		{name: "changes", model: &model.Changes{}},
		{name: "audit_events", model: &model.AuditEvent{}},
	}
}

func GetDb(exclude string) ([]byte, error) {
	backupPath, cleanup, err := PrepareDbBackup(exclude)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	// Encrypted backup/envelope callers still need a contiguous plaintext buffer
	// because the current authenticated envelope API seals the whole payload at
	// once. Plain HTTP export uses PrepareDbBackup and streams the file instead.
	// #nosec G304 -- backupPath is a freshly-created local backup file path.
	file, err := os.Open(backupPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return io.ReadAll(file)
}

func PrepareDbBackup(exclude string) (backupPath string, cleanup func(), err error) {
	excludedTables := parseBackupExcludes(exclude)

	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return "", nil, err
	}
	tmpFile, err := os.CreateTemp(dir, "s-ui-backup-*.db")
	if err != nil {
		return "", nil, err
	}
	dbPath := tmpFile.Name()
	cleanup = func() { cleanupBackupTempFiles(dbPath) }
	if err := tmpFile.Close(); err != nil {
		cleanup()
		return "", nil, err
	}
	if backupTempPathHook != nil {
		backupTempPathHook(dbPath)
	}
	defer func() {
		if err != nil {
			cleanup()
		}
	}()

	backupDb, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return "", nil, err
	}
	backupSQLDB, err := backupDb.DB()
	if err != nil {
		return "", nil, err
	}
	defer func() { _ = backupSQLDB.Close() }()

	tables := backupTables()
	models := make([]any, 0, len(tables))
	for _, table := range tables {
		models = append(models, table.model)
	}
	if err = backupDb.AutoMigrate(models...); err != nil {
		return "", nil, err
	}

	for _, table := range tables {
		if excludedTables[table.name] {
			continue
		}
		sourceDB := db
		if table.name == "tls" {
			// The no-TLS sentinel uses id=0, which GORM treats as an unset
			// auto-increment key during Create. Copy it explicitly below so it
			// cannot be reinserted with a generated id that collides with real
			// TLS rows.
			sourceDB = db.Where("id <> ?", 0)
		}
		if err := copyBackupTable(sourceDB, backupDb, table.model); err != nil {
			return "", nil, err
		}
	}
	// A no-TLS inbound points at tls.id=0. GORM treats a zero primary key as
	// unset during row copies, so the sentinel must be restored explicitly in
	// the backup or PRAGMA foreign_key_check will reject the restore.
	if err := ensureNoTLSRowOn(backupDb); err != nil {
		return "", nil, err
	}

	// Update WAL with TRUNCATE for compactness; fall back to FULL on failure
	// (e.g. a hot WAL with another writer); fall back to a no-checkpoint
	// path with warning if both fail. SQLite's WAL is still consistent
	// without a checkpoint, the backup is just larger or has a stale -wal
	// sidecar that gets cleaned up by cleanupBackupSidecars below.
	if err := walCheckpointWithFallback(backupDb); err != nil {
		logger.Warning("backup WAL checkpoint failed in both TRUNCATE and FULL modes: ", err, "; continuing without checkpoint")
	}

	if err := backupSQLDB.Close(); err != nil {
		return "", nil, err
	}
	cleanupBackupSidecars(dbPath)
	return dbPath, cleanup, nil
}

func parseBackupExcludes(exclude string) map[string]bool {
	excluded := map[string]bool{}
	for _, table := range strings.Split(exclude, ",") {
		table = strings.TrimSpace(table)
		switch table {
		case "audit":
			excluded["audit_events"] = true
		case "audit_events", "client_ips", "changes", "stats":
			excluded[table] = true
		}
	}
	return excluded
}

func ParseBackupExcludes(exclude string) []string {
	excluded := parseBackupExcludes(exclude)
	ordered := make([]string, 0, len(excluded))
	for _, table := range []string{"stats", "client_ips", "audit_events", "changes"} {
		if excluded[table] {
			ordered = append(ordered, table)
		}
	}
	return ordered
}

func copyBackupTable(sourceDB *gorm.DB, backupDB *gorm.DB, modelValue any) error {
	modelType := reflect.TypeOf(modelValue)
	if modelType.Kind() != reflect.Ptr {
		return common.NewError("backup model must be a pointer")
	}
	// Source-side paging keeps memory bounded for large stats / client_ips
	// tables, and the destination CreateInBatches keeps each generated
	// INSERT below SQLite's compile-time SQLITE_MAX_VARIABLE_NUMBER (=999
	// in mattn/go-sqlite3). Without paging on the destination, GORM tries
	// to emit one INSERT VALUES (...),(...) for the whole result set and
	// fails with "too many SQL variables" the moment row_count*column_count
	// exceeds the budget — the historical 1.5.x backup / xui-import bug.
	batch := SafeSQLiteBatchSize(backupDB, modelValue)
	return backupDB.Transaction(func(tx *gorm.DB) error {
		slicePtr := reflect.New(reflect.SliceOf(modelType.Elem()))
		findResult := sourceDB.Model(modelValue).
			FindInBatches(slicePtr.Interface(), batch, func(_ *gorm.DB, _ int) error {
				if slicePtr.Elem().Len() == 0 {
					return nil
				}
				return tx.CreateInBatches(slicePtr.Elem().Interface(), batch).Error
			})
		return findResult.Error
	})
}

var backupTempPathHook func(string)

func cleanupBackupTempFiles(dbPath string) {
	_ = os.Remove(dbPath)
	cleanupBackupSidecars(dbPath)
}

func cleanupBackupSidecars(dbPath string) {
	_ = os.Remove(dbPath + "-wal")
	_ = os.Remove(dbPath + "-shm")
	_ = os.Remove(dbPath + "-journal")
}

// walCheckpointWithFallback runs PRAGMA wal_checkpoint(TRUNCATE) first,
// then falls back to FULL on error. Returns the FULL error if both fail,
// nil if either succeeded. Production callers log the FULL error as a
// warning and continue; the backup is still valid because SQLite's WAL
// is automatically synchronized when the connection closes.
func walCheckpointWithFallback(db *gorm.DB) error {
	if err := db.Exec("PRAGMA wal_checkpoint(TRUNCATE);").Error; err != nil {
		if fallbackErr := db.Exec("PRAGMA wal_checkpoint(FULL);").Error; fallbackErr != nil {
			return fallbackErr
		}
		logger.Warning("backup WAL TRUNCATE checkpoint failed, fell back to FULL: ", err)
	}
	return nil
}

func ImportDB(file multipart.File) error {
	// Check if the file is a SQLite database.
	isValidDb, err := IsSQLiteDB(file)
	if err != nil {
		return common.NewErrorf("Error checking db file format: %v", err)
	}
	if !isValidDb {
		return common.NewError("Invalid db file format")
	}

	// Reset the file reader to the beginning.
	if _, err = file.Seek(0, 0); err != nil {
		return common.NewErrorf("Error resetting file reader: %v", err)
	}

	dbPath := config.GetDBPath()
	tempPath := dbPath + ".temp"
	fallbackPath := dbPath + ".backup"

	// Best-effort cleanup of any leftovers from a previous failed import.
	cleanupSidecars := func(p string) {
		_ = os.Remove(p + "-wal")
		_ = os.Remove(p + "-shm")
		_ = os.Remove(p + "-journal")
	}
	_ = os.Remove(tempPath)
	cleanupSidecars(tempPath)
	_ = os.Remove(fallbackPath)
	cleanupSidecars(fallbackPath)

	// Stage the uploaded bytes to a temp file. Close the handle before any
	// SQLite open or rename so the OS does not refuse the rename and SQLite
	// does not race against an open-write fd.
	if err := stageBackupToFile(file, tempPath); err != nil {
		return err
	}

	// Make sure the staged file opens read-only and passes SQLite integrity
	// checks before it can replace the live database.
	if err := validateSQLiteBackup(tempPath); err != nil {
		_ = os.Remove(tempPath)
		return err
	}

	// Close the running DB handle so the live database file is no longer
	// busy. Without this, on Windows the rename below fails outright; on
	// Linux it succeeds but stale WAL/SHM files attached to the old fd may
	// be replayed against the new database.
	closeLiveDB()

	// Move the live DB aside as a fallback. Move the WAL/SHM sidecars too,
	// otherwise SQLite would replay them on top of the imported database
	// and corrupt it (this is the historical "1.4.1 backup will not
	// restore" bug). After the rename, also nuke any sidecars that were
	// left behind (rename does not move them, since they are separate
	// files in WAL mode).
	fallbackReady := false
	if _, statErr := os.Stat(dbPath); statErr == nil {
		if err := os.Rename(dbPath, fallbackPath); err != nil {
			return reopenLiveDBAfterImportError(dbPath, "backing up live db file", err)
		}
		fallbackReady = true
	} else if !os.IsNotExist(statErr) {
		return reopenLiveDBAfterImportError(dbPath, "checking live db file", statErr)
	}
	cleanupSidecars(dbPath)

	// Move the staged file into place.
	if err := os.Rename(tempPath, dbPath); err != nil {
		return rollbackImportedDB(dbPath, fallbackPath, fallbackReady, "installing imported db file", err)
	}
	cleanupSidecars(dbPath) // imported file may have brought its own .db-wal/.db-shm if user uploaded a hot copy

	// From here on, on any failure we attempt to restore the fallback so
	// the panel keeps running on the previous data set instead of dying
	// without a database.
	rollback := func(stage string, cause error) error {
		return rollbackImportedDB(dbPath, fallbackPath, fallbackReady, stage, cause)
	}

	// Schema migrations + post-migration adapter for legacy backups.
	if migErr := migration.MigrateDb(); migErr != nil {
		return rollback("migrating imported db", migErr)
	}
	if err := InitDB(dbPath); err != nil {
		return rollback("opening imported db", err)
	}
	if err := ResetCaches(context.Background()); err != nil {
		return rollback("resetting in-memory caches", err)
	}

	// Imported db is healthy and live; drop the on-disk fallback.
	_ = os.Remove(fallbackPath)
	cleanupSidecars(fallbackPath)

	// Trigger an in-process restart. We use SIGHUP for parity with the rest
	// of the codebase; main.go traps SIGHUP and re-runs app.Init -> Start,
	// at which point migration is re-run as a no-op against the now-current
	// DB and the panel starts cleanly.
	if err := SendSighup(); err != nil {
		return common.NewErrorf("Error restarting app: %v", err)
	}
	return nil
}

func closeLiveDB() {
	// Swap the global pointer under dbMu (mirrors OpenDB), so concurrent
	// GetDB() readers — e.g. a cron job firing mid-import — never race the write.
	// The actual Close() is done outside the lock (I/O, touches no global).
	dbMu.Lock()
	current := db
	db = nil
	dbMu.Unlock()
	if current == nil {
		return
	}
	if sqlDB, err := current.DB(); err == nil {
		_ = sqlDB.Close()
	}
}

func rollbackImportedDB(dbPath string, fallbackPath string, fallbackReady bool, stage string, cause error) error {
	closeLiveDB()
	_ = os.Remove(dbPath)
	cleanupBackupSidecars(dbPath)
	if !fallbackReady {
		return common.NewErrorf("Error %s: %v", stage, cause)
	}
	if err := os.Rename(fallbackPath, dbPath); err != nil {
		return common.NewErrorf("Error %s (%v) and restoring fallback failed: %v", stage, cause, err)
	}
	return reopenLiveDBAfterImportError(dbPath, stage, cause)
}

func reopenLiveDBAfterImportError(dbPath string, stage string, cause error) error {
	if err := InitDB(dbPath); err != nil {
		return common.NewErrorf("Error %s (%v) and reopening live db failed: %v", stage, cause, err)
	}
	return common.NewErrorf("Error %s: %v", stage, cause)
}

// stageBackupToFile writes the uploaded multipart body to dst, fsyncs and
// closes the file handle. Closing here is important: any later code path
// that opens or renames dst would otherwise race against an open fd held by
// this process.
func stageBackupToFile(src io.Reader, dst string) error {
	// #nosec G304 -- dst is an internal staging path derived from the configured DB path.
	out, err := os.Create(dst)
	if err != nil {
		return common.NewErrorf("Error creating temporary db file: %v", err)
	}
	if _, err := io.Copy(out, src); err != nil {
		_ = out.Close()
		_ = os.Remove(dst)
		return common.NewErrorf("Error saving db: %v", err)
	}
	if err := out.Sync(); err != nil {
		_ = out.Close()
		_ = os.Remove(dst)
		return common.NewErrorf("Error syncing db: %v", err)
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(dst)
		return common.NewErrorf("Error closing temporary db file: %v", err)
	}
	return nil
}

func validateSQLiteBackup(path string) error {
	probe, openErr := gorm.Open(sqlite.Open(sqliteReadOnlyDSN(path)), &gorm.Config{Logger: gormlogger.Discard})
	if openErr != nil {
		return common.NewErrorf("Error checking db: %v", openErr)
	}
	sqlDB, dbErr := probe.DB()
	if dbErr == nil {
		defer sqlDB.Close()
	}
	var result string
	if err := probe.Raw("PRAGMA integrity_check").Scan(&result).Error; err != nil {
		return common.NewErrorf("Error checking db integrity: %v", err)
	}
	if result != "ok" {
		return common.NewErrorf("Invalid db integrity: %s", result)
	}
	if isXUIDatabase(probe) {
		return common.NewError("this looks like a 3x-ui/x-ui database, not an s-ui backup; use \"Migrate from 3x-ui\" to import it instead of Restore")
	}
	if err := validateVersionedBackupConfig(probe); err != nil {
		return err
	}
	return nil
}

// isXUIDatabase reports whether the probed database is a 3x-ui / x-ui (xray)
// database rather than an s-ui backup. client_traffics and inbound_client_ips
// are 3x-ui tables that no s-ui schema has ever carried, so their presence is a
// definitive signal. Detecting it here lets Restore reject foreign databases
// with a clear, actionable message instead of failing later in schema
// migration with a cryptic "no such table: changes". The check is positive
// (looks-like-3x-ui) rather than negative (lacks-s-ui-tables) so it never
// false-rejects a minimal or legacy s-ui backup.
func isXUIDatabase(probe *gorm.DB) bool {
	for _, table := range []string{"client_traffics", "inbound_client_ips"} {
		if probe.Migrator().HasTable(table) {
			return true
		}
	}
	return false
}

func validateVersionedBackupConfig(probe *gorm.DB) error {
	if !probe.Migrator().HasTable(&model.Setting{}) {
		return nil
	}

	var version string
	if err := probe.Model(&model.Setting{}).Select("value").Where("key = ?", "version").Scan(&version).Error; err != nil {
		return common.NewErrorf("Error checking db settings: %v", err)
	}
	if strings.TrimSpace(version) == "" {
		return nil
	}

	var configRows int64
	if err := probe.Model(&model.Setting{}).Where("key = ?", "config").Count(&configRows).Error; err != nil {
		return common.NewErrorf("Error checking db config: %v", err)
	}
	if configRows == 0 {
		logger.Warning("versioned S-UI backup is missing settings.config; legacy or partial backup, restore will continue")
		return nil
	}
	return nil
}

func sqliteReadOnlyDSN(path string) string {
	urlPath := filepath.ToSlash(path)
	if runtime.GOOS == "windows" && !strings.HasPrefix(urlPath, "/") {
		urlPath = "/" + urlPath
	}
	u := url.URL{
		Scheme: "file",
		Path:   urlPath,
	}
	values := url.Values{}
	values.Set("mode", "ro")
	u.RawQuery = values.Encode()
	return u.String()
}

func IsSQLiteDB(file io.Reader) (bool, error) {
	signature := []byte("SQLite format 3\x00")
	buf := make([]byte, len(signature))
	_, err := file.Read(buf)
	if err != nil {
		return false, err
	}
	return bytes.Equal(buf, signature), nil
}

// sendSighupHook lets tests intercept the restart signal so they don't kill
// the test runner. Production code uses the default no-op override (nil)
// which makes SendSighup execute its normal signal logic.
var sendSighupHook func() error

// sighupTimeout is the delay between SendSighup invocation and the
// actual signal delivery. Tests override this via the package-level
// helper SetSighupTimeoutForTest. Production reads SUI_SIGHUP_TIMEOUT_SECONDS
// from the environment on first call; if unset or invalid, falls back
// to the historical 3s default.
var (
	sighupTimeout     time.Duration
	sighupTimeoutOnce sync.Once
)

func SetSendSighupHook(hook func() error) {
	sendSighupHook = hook
}

func resolvedSighupTimeout() time.Duration {
	sighupTimeoutOnce.Do(func() {
		sighupTimeout = parseSighupTimeoutEnv()
	})
	return sighupTimeout
}

func parseSighupTimeoutEnv() time.Duration {
	const defaultTimeout = 3 * time.Second
	raw := strings.TrimSpace(os.Getenv("SUI_SIGHUP_TIMEOUT_SECONDS"))
	if raw == "" {
		return defaultTimeout
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed < 1 || parsed > 60 {
		logger.Warning("invalid SUI_SIGHUP_TIMEOUT_SECONDS=", raw, ", falling back to 3s")
		return defaultTimeout
	}
	return time.Duration(parsed) * time.Second
}

// SetSighupTimeoutForTest overrides the resolved timeout. Test helpers
// call this in t.Cleanup-bracketed pairs. Production must not call it.
func SetSighupTimeoutForTest(d time.Duration) {
	sighupTimeout = d
	sighupTimeoutOnce.Do(func() {})
}

func SendSighup() error {
	if sendSighupHook != nil {
		return sendSighupHook()
	}
	// Get the current process
	process, err := os.FindProcess(os.Getpid())
	if err != nil {
		return err
	}

	// Send SIGHUP after the configured delay (SUI_SIGHUP_TIMEOUT_SECONDS, default 3s).
	time.AfterFunc(resolvedSighupTimeout(), func() {
		var signalErr error
		if runtime.GOOS == "windows" {
			signalErr = process.Kill()
		} else {
			signalErr = process.Signal(syscall.SIGHUP)
		}
		if signalErr != nil {
			logger.Error("send signal SIGHUP failed:", signalErr)
		}
	})
	return nil
}
