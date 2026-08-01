package database

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/deposist/s-ui-x-extended/config"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// Filesystem operations are variables so failure paths can be exercised
// without relying on platform-specific rename and fsync failures.
var (
	panelUpdateRename        = os.Rename
	panelUpdateSyncFile      = syncPanelUpdateFile
	panelUpdateSyncDirectory = syncPanelUpdateDirectory
)

// PreparePanelUpdateSnapshot creates a transactionally-consistent,
// self-contained image of the live SQLite database for the panel updater.
//
// VACUUM INTO reads one SQLite snapshot and includes committed pages that are
// still in the source WAL. In WAL mode it does not stop unrelated readers or
// writers for the duration of the copy. The maintenance read lock only keeps a
// restore from closing the source handle while the snapshot is being made.
// The returned file is closed and fsynced; ownership passes to the caller.
func PreparePanelUpdateSnapshot() (snapshotPath string, databasePath string, err error) {
	leaveOperation := EnterDBOperation()
	defer leaveOperation()

	live := GetDB()
	if live == nil {
		return "", "", errors.New("panel update database snapshot: database is not open")
	}

	databasePath, err = sqliteMainDatabasePath(live)
	if err != nil {
		return "", "", fmt.Errorf("panel update database snapshot: locate live database: %w", err)
	}
	expectedPath, err := canonicalPanelUpdatePath(config.GetDBPath())
	if err != nil {
		return "", "", fmt.Errorf("panel update database snapshot: resolve configured database: %w", err)
	}
	if !samePanelUpdatePath(databasePath, expectedPath) {
		return "", "", fmt.Errorf("panel update database snapshot: live database %q is not configured database %q", databasePath, expectedPath)
	}

	executablePath, err := os.Executable()
	if err != nil {
		return "", "", fmt.Errorf("panel update database snapshot: locate executable: %w", err)
	}
	snapshotPath, err = reserveRemovedTempPath(filepath.Dir(executablePath), ".sui-update-db-*.snapshot")
	if err != nil {
		return "", "", fmt.Errorf("panel update database snapshot: reserve destination: %w", err)
	}
	cleanupPath := snapshotPath
	complete := false
	defer func() {
		if !complete {
			cleanupBackupTempFiles(cleanupPath)
		}
	}()

	if execErr := live.Exec("VACUUM INTO ?", snapshotPath).Error; execErr != nil {
		return "", "", fmt.Errorf("panel update database snapshot: copy live database: %w", execErr)
	}
	if chmodErr := os.Chmod(snapshotPath, 0o600); chmodErr != nil {
		return "", "", fmt.Errorf("panel update database snapshot: secure snapshot: %w", chmodErr)
	}
	cleanupBackupSidecars(snapshotPath)
	if validateErr := validatePanelUpdateSQLiteFile(snapshotPath); validateErr != nil {
		return "", "", fmt.Errorf("panel update database snapshot: validate snapshot: %w", validateErr)
	}
	if syncErr := panelUpdateSyncFile(snapshotPath); syncErr != nil {
		return "", "", fmt.Errorf("panel update database snapshot: sync snapshot: %w", syncErr)
	}
	if syncErr := panelUpdateSyncDirectory(filepath.Dir(snapshotPath)); syncErr != nil {
		return "", "", fmt.Errorf("panel update database snapshot: sync snapshot directory: %w", syncErr)
	}

	complete = true
	return snapshotPath, databasePath, nil
}

// RestorePanelUpdateSnapshot installs an updater-owned SQLite snapshot at the
// configured database path. It deliberately does not run migrations: startup
// recovery calls this function before the old binary's migration step.
func RestorePanelUpdateSnapshot(snapshotPath string, databasePath string) error {
	targetPath, err := validatePanelUpdateRestorePaths(snapshotPath, databasePath)
	if err != nil {
		return err
	}
	if err := validatePanelUpdateSQLiteFile(snapshotPath); err != nil {
		return fmt.Errorf("panel update database restore: validate snapshot: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(targetPath), 0o750); err != nil {
		return fmt.Errorf("panel update database restore: create database directory: %w", err)
	}
	stagedPath, err := stagePanelUpdateSnapshot(snapshotPath, targetPath)
	if err != nil {
		return fmt.Errorf("panel update database restore: stage snapshot: %w", err)
	}
	defer cleanupBackupTempFiles(stagedPath)
	if err := validatePanelUpdateSQLiteFile(stagedPath); err != nil {
		return fmt.Errorf("panel update database restore: validate staged snapshot: %w", err)
	}

	leaveMaintenance, err := beginRestore()
	if err != nil {
		return err
	}
	defer leaveMaintenance()

	live, wasLive, err := panelUpdateLiveDatabase(targetPath)
	if err != nil {
		return err
	}
	originalExists, err := panelUpdatePathExists(targetPath)
	if err != nil {
		return fmt.Errorf("panel update database restore: inspect live database: %w", err)
	}
	if wasLive && !originalExists {
		return errors.New("panel update database restore: open database file is missing")
	}

	if originalExists {
		if err := checkpointPanelUpdateDatabase(live, targetPath); err != nil {
			return fmt.Errorf("panel update database restore: checkpoint live database: %w", err)
		}
	}
	if wasLive {
		closeLiveDB()
	}

	fallbackPath := ""
	fallbackReady := false
	replacementInstalled := false
	rollback := func(stage string, cause error) error {
		return rollbackPanelUpdateRestore(targetPath, fallbackPath, fallbackReady, replacementInstalled, wasLive, stage, cause)
	}

	if err := removePanelUpdateSidecars(targetPath); err != nil {
		return rollback("remove live database sidecars", err)
	}
	if originalExists {
		fallbackPath, err = reserveRemovedTempPath(filepath.Dir(targetPath), filepath.Base(targetPath)+".update-rollback-*.bak")
		if err != nil {
			return rollback("reserve database fallback", err)
		}
		if err := panelUpdateRename(targetPath, fallbackPath); err != nil {
			return rollback("move live database to fallback", err)
		}
		fallbackReady = true
	}

	if err := panelUpdateRename(stagedPath, targetPath); err != nil {
		return rollback("install database snapshot", err)
	}
	replacementInstalled = true
	if err := removePanelUpdateSidecars(targetPath); err != nil {
		return rollback("remove restored database sidecars", err)
	}
	if err := panelUpdateSyncFile(targetPath); err != nil {
		return rollback("sync restored database", err)
	}
	if err := panelUpdateSyncDirectory(filepath.Dir(targetPath)); err != nil {
		return rollback("sync restored database directory", err)
	}
	if wasLive {
		if err := OpenDB(targetPath); err != nil {
			return rollback("reopen restored database", err)
		}
	}

	// All fallible commit steps are complete. Removing the fallback is cleanup,
	// not part of the transaction: an inability to remove it must not turn a
	// successfully installed and durable database into a reported rollback.
	if fallbackReady {
		_ = os.Remove(fallbackPath)
		cleanupBackupSidecars(fallbackPath)
		_ = panelUpdateSyncDirectory(filepath.Dir(targetPath))
	}
	return nil
}

func validatePanelUpdateRestorePaths(snapshotPath string, databasePath string) (string, error) {
	if strings.TrimSpace(snapshotPath) == "" {
		return "", errors.New("panel update database restore: snapshot path is empty")
	}
	requestedPath, err := canonicalPanelUpdatePath(databasePath)
	if err != nil {
		return "", fmt.Errorf("panel update database restore: resolve target database: %w", err)
	}
	expectedPath, err := canonicalPanelUpdatePath(config.GetDBPath())
	if err != nil {
		return "", fmt.Errorf("panel update database restore: resolve configured database: %w", err)
	}
	if !samePanelUpdatePath(requestedPath, expectedPath) {
		return "", fmt.Errorf("panel update database restore: refusing target %q; configured database is %q", requestedPath, expectedPath)
	}
	canonicalSnapshot, err := canonicalPanelUpdatePath(snapshotPath)
	if err != nil {
		return "", fmt.Errorf("panel update database restore: resolve snapshot: %w", err)
	}
	if samePanelUpdatePath(canonicalSnapshot, requestedPath) {
		return "", errors.New("panel update database restore: snapshot and target paths are identical")
	}
	return requestedPath, nil
}

func sqliteMainDatabasePath(target *gorm.DB) (string, error) {
	var databases []struct {
		Name string
		File string
	}
	if err := target.Raw("PRAGMA database_list").Scan(&databases).Error; err != nil {
		return "", err
	}
	for _, database := range databases {
		if database.Name != "main" {
			continue
		}
		if strings.TrimSpace(database.File) == "" || database.File == ":memory:" {
			return "", errors.New("main database is not file-backed")
		}
		return canonicalPanelUpdatePath(database.File)
	}
	return "", errors.New("main database path was not reported by SQLite")
}

func canonicalPanelUpdatePath(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", errors.New("path is empty")
	}
	absolute, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return "", err
	}
	return filepath.Clean(absolute), nil
}

func samePanelUpdatePath(left string, right string) bool {
	if runtime.GOOS == "windows" {
		return strings.EqualFold(filepath.Clean(left), filepath.Clean(right))
	}
	return filepath.Clean(left) == filepath.Clean(right)
}

func reserveRemovedTempPath(dir string, pattern string) (string, error) {
	file, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return "", err
	}
	path := file.Name()
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return "", err
	}
	if err := os.Remove(path); err != nil {
		return "", err
	}
	return path, nil
}

func validatePanelUpdateSQLiteFile(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return errors.New("SQLite snapshot is not a regular file")
	}
	// #nosec G304 -- snapshot paths are updater-owned and the target is checked
	// against the configured database path before replacement.
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	valid, signatureErr := IsSQLiteDB(file)
	closeErr := file.Close()
	if signatureErr != nil {
		return signatureErr
	}
	if closeErr != nil {
		return closeErr
	}
	if !valid {
		return errors.New("invalid SQLite database signature")
	}
	return validateSQLiteBackup(path)
}

func stagePanelUpdateSnapshot(sourcePath string, targetPath string) (string, error) {
	// #nosec G304 -- sourcePath is the updater-owned, pre-validated snapshot.
	source, err := os.Open(sourcePath)
	if err != nil {
		return "", err
	}
	defer source.Close()

	staged, err := os.CreateTemp(filepath.Dir(targetPath), filepath.Base(targetPath)+".update-restore-*.tmp")
	if err != nil {
		return "", err
	}
	stagedPath := staged.Name()
	removeOnError := func(cause error) (string, error) {
		_ = staged.Close()
		cleanupBackupTempFiles(stagedPath)
		return "", cause
	}
	if _, err := io.Copy(staged, source); err != nil {
		return removeOnError(err)
	}
	if err := staged.Sync(); err != nil {
		return removeOnError(err)
	}
	if err := staged.Close(); err != nil {
		cleanupBackupTempFiles(stagedPath)
		return "", err
	}
	return stagedPath, nil
}

func panelUpdateLiveDatabase(targetPath string) (*gorm.DB, bool, error) {
	live := GetDB()
	if live == nil {
		return nil, false, nil
	}
	livePath, err := sqliteMainDatabasePath(live)
	if err != nil {
		return nil, false, fmt.Errorf("panel update database restore: locate open database: %w", err)
	}
	if !samePanelUpdatePath(livePath, targetPath) {
		return nil, false, fmt.Errorf("panel update database restore: open database %q does not match target %q", livePath, targetPath)
	}
	return live, true, nil
}

func panelUpdatePathExists(path string) (bool, error) {
	info, err := os.Lstat(path)
	if err == nil {
		if !info.Mode().IsRegular() {
			return false, fmt.Errorf("unsafe database target is not a regular file: %s", path)
		}
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

func checkpointPanelUpdateDatabase(live *gorm.DB, targetPath string) error {
	if live != nil {
		return runPanelUpdateCheckpoint(live)
	}

	probe, err := gorm.Open(sqlite.Open(panelUpdateWritableDSN(targetPath)), &gorm.Config{Logger: gormlogger.Discard})
	if err != nil {
		return err
	}
	sqlDB, err := probe.DB()
	if err != nil {
		return err
	}
	checkpointErr := runPanelUpdateCheckpoint(probe)
	closeErr := sqlDB.Close()
	return errors.Join(checkpointErr, closeErr)
}

func runPanelUpdateCheckpoint(target *gorm.DB) error {
	var result struct {
		Busy         int
		Log          int
		Checkpointed int
	}
	if err := target.Raw("PRAGMA wal_checkpoint(TRUNCATE)").Scan(&result).Error; err != nil {
		return err
	}
	if result.Busy != 0 {
		return fmt.Errorf("WAL checkpoint remained busy (%d frame(s), %d checkpointed)", result.Log, result.Checkpointed)
	}
	return nil
}

func panelUpdateWritableDSN(path string) string {
	separator := "?"
	if strings.Contains(path, "?") {
		separator = "&"
	}
	return path + separator + "_busy_timeout=10000"
}

func rollbackPanelUpdateRestore(targetPath string, fallbackPath string, fallbackReady bool, replacementInstalled bool, wasLive bool, stage string, cause error) error {
	if wasLive {
		closeLiveDB()
	}

	var rollbackErr error
	if replacementInstalled {
		if err := os.Remove(targetPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			rollbackErr = errors.Join(rollbackErr, fmt.Errorf("remove partial replacement: %w", err))
		} else if err := removePanelUpdateSidecars(targetPath); err != nil {
			rollbackErr = errors.Join(rollbackErr, fmt.Errorf("remove partial replacement sidecars: %w", err))
		}
	}

	if fallbackReady {
		if _, statErr := os.Stat(targetPath); statErr == nil {
			rollbackErr = errors.Join(rollbackErr, fmt.Errorf("restore fallback: target %q still exists; original remains at %q", targetPath, fallbackPath))
		} else if !errors.Is(statErr, os.ErrNotExist) {
			rollbackErr = errors.Join(rollbackErr, fmt.Errorf("inspect partial replacement: %w", statErr))
		} else if err := panelUpdateRename(fallbackPath, targetPath); err != nil {
			rollbackErr = errors.Join(rollbackErr, fmt.Errorf("restore fallback from %q: %w", fallbackPath, err))
		} else {
			if err := removePanelUpdateSidecars(targetPath); err != nil {
				rollbackErr = errors.Join(rollbackErr, fmt.Errorf("clean restored fallback sidecars: %w", err))
			}
			if err := panelUpdateSyncFile(targetPath); err != nil {
				rollbackErr = errors.Join(rollbackErr, fmt.Errorf("sync restored fallback: %w", err))
			}
			if err := panelUpdateSyncDirectory(filepath.Dir(targetPath)); err != nil {
				rollbackErr = errors.Join(rollbackErr, fmt.Errorf("sync restored fallback directory: %w", err))
			}
		}
	}

	if wasLive {
		if _, statErr := os.Stat(targetPath); statErr == nil {
			if err := OpenDB(targetPath); err != nil {
				rollbackErr = errors.Join(rollbackErr, fmt.Errorf("reopen recoverable database: %w", err))
			}
		}
	}

	primaryErr := fmt.Errorf("panel update database restore: %s: %w", stage, cause)
	if rollbackErr != nil {
		return errors.Join(primaryErr, fmt.Errorf("database rollback: %w", rollbackErr))
	}
	return primaryErr
}

func removePanelUpdateSidecars(path string) error {
	var result error
	for _, suffix := range []string{"-wal", "-shm", "-journal"} {
		if err := os.Remove(path + suffix); err != nil && !errors.Is(err, os.ErrNotExist) {
			result = errors.Join(result, err)
		}
	}
	return result
}

func syncPanelUpdateFile(path string) error {
	// #nosec G304 -- path is either an updater-created snapshot or the checked
	// configured database target.
	file, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	syncErr := file.Sync()
	closeErr := file.Close()
	return errors.Join(syncErr, closeErr)
}

func syncPanelUpdateDirectory(path string) error {
	// #nosec G304 -- path is the directory containing an internally managed
	// updater snapshot or the configured database.
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	syncErr := directory.Sync()
	closeErr := directory.Close()
	// os.File.Sync does not expose FlushFileBuffers for directories on Windows.
	if runtime.GOOS == "windows" {
		syncErr = nil
	}
	return errors.Join(syncErr, closeErr)
}
