package importxui

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// TestPrunePreImportBackups_KeepsNewest guards the runaway-backup fix: a slow
// import that the client resubmits used to leave dozens of
// s-ui-pre-xui-import-*.db files in the db directory. Pruning must keep only
// the newest N and never touch unrelated files.
func TestPrunePreImportBackups_KeepsNewest(t *testing.T) {
	dir := makeImportXUITempDir(t)
	const total = 15
	for i := 1; i <= total; i++ {
		name := filepath.Join(dir, fmt.Sprintf("s-ui-pre-xui-import-%010d.db", 1700000000+i))
		if err := os.WriteFile(name, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	// Unrelated files must survive pruning.
	other := filepath.Join(dir, "s-ui.db")
	if err := os.WriteFile(other, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}

	prunePreImportBackups(dir, preImportBackupRetention)

	matches, err := filepath.Glob(filepath.Join(dir, "s-ui-pre-xui-import-*.db"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != preImportBackupRetention {
		t.Fatalf("kept %d backups, want %d", len(matches), preImportBackupRetention)
	}
	newest := filepath.Join(dir, fmt.Sprintf("s-ui-pre-xui-import-%010d.db", 1700000000+total))
	if _, err := os.Stat(newest); err != nil {
		t.Fatalf("newest backup was pruned: %v", err)
	}
	oldest := filepath.Join(dir, fmt.Sprintf("s-ui-pre-xui-import-%010d.db", 1700000001))
	if _, err := os.Stat(oldest); !os.IsNotExist(err) {
		t.Fatalf("oldest backup should have been pruned, stat err=%v", err)
	}
	if _, err := os.Stat(other); err != nil {
		t.Fatalf("unrelated file was removed: %v", err)
	}
}

// TestPrunePreImportBackups_NoopUnderLimit verifies nothing is deleted when the
// count is at or below the retention limit.
func TestPrunePreImportBackups_NoopUnderLimit(t *testing.T) {
	dir := makeImportXUITempDir(t)
	for i := 1; i <= 3; i++ {
		name := filepath.Join(dir, fmt.Sprintf("s-ui-pre-xui-import-%010d.db", 1700000000+i))
		if err := os.WriteFile(name, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	prunePreImportBackups(dir, preImportBackupRetention)
	matches, _ := filepath.Glob(filepath.Join(dir, "s-ui-pre-xui-import-*.db"))
	if len(matches) != 3 {
		t.Fatalf("kept %d backups, want 3 (no pruning under limit)", len(matches))
	}
}
