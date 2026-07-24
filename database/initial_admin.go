package database

import (
	"crypto/subtle"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/deposist/s-ui-x-extended/config"
	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/deposist/s-ui-x-extended/logger"
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

// removeInitialAdminPasswordIfBootstrapCredential removes the plaintext
// credential only after a successful password change for the exact fresh admin
// it created. The marker is the still-present credential file plus a constant-
// time check against the old password; migrated users and unrelated accounts
// therefore cannot consume it.
func ConsumeInitialAdminPasswordIfBootstrapCredential(username, oldPassword string) error {
	return removeInitialAdminPasswordIfBootstrapCredential("", username, oldPassword)
}

func removeInitialAdminPasswordIfBootstrapCredential(dbPath, username, oldPassword string) error {
	path := initialAdminPasswordPath(dbPath)
	// #nosec G304 -- path is derived solely from the configured SQLite data path
	// and the fixed initialAdminPasswordFile basename.
	credential, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if username != "admin" || subtle.ConstantTimeCompare([]byte(strings.TrimSpace(string(credential))), []byte(oldPassword)) != 1 {
		return nil
	}

	var users []model.User
	if err := GetDB().Order("id").Find(&users).Error; err != nil {
		return err
	}
	if len(users) != 1 {
		return nil
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func warnIfInitialAdminPasswordFileExists(path string) {
	if _, err := os.Stat(path); err == nil {
		logger.Warning("initial admin password file still exists; delete after first login: ", path)
	}
}
