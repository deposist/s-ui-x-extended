package api

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSecurityValidateRollbackPathRejectsTraversalMissingAndOutsideFiles(t *testing.T) {
	dbDir := t.TempDir()
	t.Setenv("SUI_DB_FOLDER", dbDir)
	outsideDir := t.TempDir()

	tests := []struct {
		name string
		path string
	}{
		{name: "missing", path: filepath.Join(dbDir, "s-ui-pre-xui-import-missing.db")},
		{name: "outside", path: filepath.Join(outsideDir, "s-ui-pre-xui-import-1.db")},
		{name: "path traversal outside", path: filepath.Join(dbDir, "..", filepath.Base(outsideDir), "s-ui-pre-xui-import-1.db")},
		{name: "wrong prefix inside", path: filepath.Join(dbDir, "manual-backup.db")},
	}
	if err := os.WriteFile(filepath.Join(outsideDir, "s-ui-pre-xui-import-1.db"), []byte("SQLite format 3\x00"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dbDir, "manual-backup.db"), []byte("SQLite format 3\x00"), 0o600); err != nil {
		t.Fatal(err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateRollbackPath(tt.path); err == nil {
				t.Fatalf("expected rollback path %q to be rejected", tt.path)
			}
		})
	}
}

func TestSecurityValidateRollbackPathRejectsSymlinkInDatabaseDir(t *testing.T) {
	dbDir := t.TempDir()
	t.Setenv("SUI_DB_FOLDER", dbDir)
	outside := filepath.Join(t.TempDir(), "outside.db")
	if err := os.WriteFile(outside, []byte("SQLite format 3\x00"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dbDir, "s-ui-pre-xui-import-symlink.db")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if err := validateRollbackPath(link); err == nil {
		t.Fatal("expected symlink rollback path to be rejected")
	}
}
