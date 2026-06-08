package migration

import (
	"path/filepath"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func openMigrationTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "migration.db")), &gorm.Config{})
	if err != nil {
		if strings.Contains(err.Error(), "go-sqlite3 requires cgo") {
			t.Skip(err)
		}
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

func TestTo14AddsTokenColumnsAndAuditEventsIdempotently(t *testing.T) {
	db := openMigrationTestDB(t)
	if err := db.Exec(`
CREATE TABLE tokens (
	id integer PRIMARY KEY AUTOINCREMENT,
	desc text,
	token text,
	expiry integer,
	user_id integer
)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("INSERT INTO tokens(desc, token, expiry, user_id) VALUES('legacy', 'raw-token', 0, 1)").Error; err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 2; i++ {
		if err := to1_4(db); err != nil {
			t.Fatal(err)
		}
	}

	for _, column := range []string{
		"token_hash",
		"token_prefix",
		"scope",
		"enabled",
		"created_at",
		"updated_at",
		"last_used_at",
		"last_used_ip",
	} {
		hasColumn, err := sqliteHasColumn(db, "tokens", column)
		if err != nil {
			t.Fatal(err)
		}
		if !hasColumn {
			t.Fatalf("tokens.%s was not added", column)
		}
	}
	if !db.Migrator().HasTable("audit_events") {
		t.Fatal("audit_events table was not created")
	}
	if err := db.Exec(`
INSERT INTO audit_events(date_time, actor, event, resource, severity, ip, user_agent, details)
VALUES(1, 'admin', 'migration_test', 'audit', 'info', '127.0.0.1', 'test', '{}')
`).Error; err != nil {
		t.Fatal(err)
	}
	var token struct {
		Scope     string
		CreatedAt int64
		UpdatedAt int64
	}
	if err := db.Raw("SELECT scope, created_at, updated_at FROM tokens WHERE desc = ?", "legacy").Scan(&token).Error; err != nil {
		t.Fatal(err)
	}
	if token.Scope != "admin" {
		t.Fatalf("legacy token scope was not backfilled to admin: %q", token.Scope)
	}
	if token.CreatedAt <= 0 || token.UpdatedAt <= 0 {
		t.Fatalf("legacy token timestamps were not backfilled: %#v", token)
	}
}

func TestTo14BackfillsExistingFullScopesToAdmin(t *testing.T) {
	db := openMigrationTestDB(t)
	if err := db.Exec(`
CREATE TABLE tokens (
	id integer PRIMARY KEY AUTOINCREMENT,
	desc text,
	token text,
	scope text,
	expiry integer,
	user_id integer
)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`
INSERT INTO tokens(desc, token, scope, expiry, user_id)
VALUES('full', 'full-token', 'full', 0, 1), ('empty', 'empty-token', '', 0, 1)
`).Error; err != nil {
		t.Fatal(err)
	}

	if err := to1_4(db); err != nil {
		t.Fatal(err)
	}

	var scopes []string
	if err := db.Raw("SELECT scope FROM tokens ORDER BY desc").Scan(&scopes).Error; err != nil {
		t.Fatal(err)
	}
	if len(scopes) != 2 || scopes[0] != "admin" || scopes[1] != "admin" {
		t.Fatalf("existing token scopes were not backfilled: %#v", scopes)
	}
}
