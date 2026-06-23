package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/deposist/s-ui-x-extended/database"

	"gorm.io/gorm"
)

// initFailoverStateTestDB stands up a throwaway on-disk SQLite database using the
// Windows-safe discipline from the error journal: a manually created temp dir, a
// WAL checkpoint (TRUNCATE) + handle close before removal, and retry-deletion -
// t.TempDir's automatic RemoveAll races SQLite's WAL/SHM files on Windows.
func initFailoverStateTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	tempDir, err := os.MkdirTemp("", "s-ui-failover-state-")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("SUI_DB_FOLDER", tempDir)
	closeFailoverStateDB(database.GetDB())
	if err := database.InitDB(filepath.Join(tempDir, "s-ui.db")); err != nil {
		removeFailoverStateTempDir(t, tempDir)
		if strings.Contains(err.Error(), "go-sqlite3 requires cgo") {
			t.Skip(err)
		}
		t.Fatal(err)
	}
	db := database.GetDB()
	t.Cleanup(func() {
		closeFailoverStateDB(db)
		removeFailoverStateTempDir(t, tempDir)
	})
	return db
}

func closeFailoverStateDB(db *gorm.DB) {
	if db == nil {
		return
	}
	_ = db.Exec("PRAGMA wal_checkpoint(TRUNCATE)").Error
	if sqlDB, err := db.DB(); err == nil {
		_ = sqlDB.Close()
	}
}

func removeFailoverStateTempDir(t *testing.T, dir string) {
	t.Helper()
	var err error
	for i := 0; i < 20; i++ {
		err = os.RemoveAll(dir)
		if err == nil || os.IsNotExist(err) {
			return
		}
		time.Sleep(time.Duration(i+1) * 10 * time.Millisecond)
	}
	t.Errorf("remove failover-state temp dir %q: %v", dir, err)
}

// EnsureFailoverSchema is safe to call repeatedly (CREATE TABLE IF NOT EXISTS),
// so boot and any later call never error or destroy existing rows.
func TestEnsureFailoverSchemaIsIdempotent(t *testing.T) {
	db := initFailoverStateTestDB(t)

	if err := EnsureFailoverSchema(db); err != nil {
		t.Fatalf("first EnsureFailoverSchema: %v", err)
	}
	if err := WriteFailoverMemberStates(db, []FailoverMemberState{
		{GroupTag: "g", MemberTag: "a", Healthy: true, ConsecUp: 1, LastProbeAt: 1},
	}); err != nil {
		t.Fatal(err)
	}
	// A second EnsureSchema must not drop or reset the existing row.
	if err := EnsureFailoverSchema(db); err != nil {
		t.Fatalf("second EnsureFailoverSchema: %v", err)
	}
	rows, err := ReadFailoverMemberStates(db, "g")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || !rows[0].Healthy {
		t.Fatalf("rows after re-ensure = %#v, want the surviving healthy row", rows)
	}
}

// Re-writing the same (group, member) key UPDATEs the row in place (PRIMARY KEY
// conflict -> UpdateAll), so health history never churns into duplicate rows and
// a read returns the latest state.
func TestWriteFailoverMemberStatesUpsertsInPlace(t *testing.T) {
	db := initFailoverStateTestDB(t)
	if err := EnsureFailoverSchema(db); err != nil {
		t.Fatal(err)
	}

	if err := WriteFailoverMemberStates(db, []FailoverMemberState{
		{GroupTag: "g", MemberTag: "a", Healthy: false, ConsecDown: 1, LastProbeAt: 100},
		{GroupTag: "g", MemberTag: "b", Healthy: true, ConsecUp: 2, LastProbeAt: 100},
	}); err != nil {
		t.Fatal(err)
	}
	// Member "a" recovers: same key, new health.
	if err := WriteFailoverMemberStates(db, []FailoverMemberState{
		{GroupTag: "g", MemberTag: "a", Healthy: true, ConsecUp: 3, ConsecDown: 0, LastProbeAt: 200},
	}); err != nil {
		t.Fatal(err)
	}

	rows, err := ReadFailoverMemberStates(db, "g")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("row count = %d, want 2 (upsert in place, no duplicate rows)", len(rows))
	}
	byTag := make(map[string]FailoverMemberState, len(rows))
	for _, r := range rows {
		byTag[r.MemberTag] = r
	}
	if a := byTag["a"]; !a.Healthy || a.ConsecUp != 3 || a.ConsecDown != 0 || a.LastProbeAt != 200 {
		t.Fatalf("member a not upserted to latest state: %#v", a)
	}
	if b := byTag["b"]; !b.Healthy || b.ConsecUp != 2 || b.LastProbeAt != 100 {
		t.Fatalf("member b must be untouched by member a's write: %#v", b)
	}
}

// ReadFailoverMemberStates returns only the requested group's rows.
func TestReadFailoverMemberStatesFiltersByGroup(t *testing.T) {
	db := initFailoverStateTestDB(t)
	if err := EnsureFailoverSchema(db); err != nil {
		t.Fatal(err)
	}
	if err := WriteFailoverMemberStates(db, []FailoverMemberState{
		{GroupTag: "g1", MemberTag: "a", LastProbeAt: 1},
		{GroupTag: "g2", MemberTag: "a", LastProbeAt: 1},
	}); err != nil {
		t.Fatal(err)
	}
	rows, err := ReadFailoverMemberStates(db, "g1")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].GroupTag != "g1" {
		t.Fatalf("read g1 = %#v, want exactly one g1 row", rows)
	}
}
