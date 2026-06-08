package database

import (
	"path/filepath"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestWALCheckpointFallbackHelperHandlesTruncateFailureIssue11(t *testing.T) {
	probe, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "checkpoint-closed.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := probe.DB()
	if err != nil {
		t.Fatal(err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatal(err)
	}

	if err := walCheckpointWithFallback(probe); err == nil {
		t.Fatal("expected FULL checkpoint error after TRUNCATE fails on a closed database")
	}
}

func TestWALCheckpointFallbackHelperSucceedsIssue11(t *testing.T) {
	probe, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "checkpoint-open.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := probe.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	if err := walCheckpointWithFallback(probe); err != nil {
		t.Fatalf("expected WAL checkpoint helper to succeed on an open database: %v", err)
	}
}
