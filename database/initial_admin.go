package database

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/deposist/s-ui-x/config"
	"github.com/deposist/s-ui-x/logger"
)

const initialAdminPasswordFile = "initial-admin.txt"

func initialAdminPasswordPath(dbPath string) string {
	dataPath := sqliteDataPath(dbPath)
	if dataPath == "" {
		return filepath.Join(config.GetDBFolderPath(), initialAdminPasswordFile)
	}
	return filepath.Join(filepath.Dir(dataPath), initialAdminPasswordFile)
}

func sqliteDataPath(dbPath string) string {
	if before, _, ok := strings.Cut(dbPath, "?"); ok {
		dbPath = before
	}
	dbPath = strings.TrimPrefix(dbPath, "file:")
	if dbPath == "" || strings.HasPrefix(dbPath, ":memory:") {
		return ""
	}
	return dbPath
}

func writeInitialAdminPassword(path string, password string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".initial-admin-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.WriteString(password + "\n"); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpPath, 0o600); err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
}

func notifyInitialAdminPasswordSaved(path string) {
	fmt.Fprintf(os.Stderr, "initial admin password saved to %s; delete after first login\n", path)
}

func warnIfInitialAdminPasswordFileExists(path string) {
	if _, err := os.Stat(path); err == nil {
		logger.Warning("initial admin password file still exists; delete after first login: ", path)
	}
}
