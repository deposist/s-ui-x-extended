package migration

import (
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestForeignKeyViolationCountsSyntheticData(t *testing.T) {
	db := openForeignKeyTestDB(t)
	createForeignKeyTestTables(t, db)
	if err := db.Exec("INSERT INTO tokens(id, user_id) VALUES(1, 42)").Error; err != nil {
		t.Fatal(err)
	}

	violations, err := foreignKeyViolations(db)
	if err != nil {
		t.Fatal(err)
	}
	counts := foreignKeyViolationCounts(violations)
	if counts["tokens"] != 1 {
		t.Fatalf("unexpected foreign-key counts: %#v violations=%#v", counts, violations)
	}
}

func TestVerifyForeignKeysRepairsSafeTokenOrphans(t *testing.T) {
	db := openForeignKeyTestDB(t)
	createForeignKeyTestTables(t, db)
	if err := db.Exec("INSERT INTO users(id) VALUES(7)").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("INSERT INTO tokens(id, user_id) VALUES(1, 42), (2, 7)").Error; err != nil {
		t.Fatal(err)
	}

	if err := verifyForeignKeysBeforeMigration(db, Options{RepairForeignKeyOrphans: true}); err != nil {
		t.Fatal(err)
	}

	var tokenCount int64
	if err := db.Table("tokens").Count(&tokenCount).Error; err != nil {
		t.Fatal(err)
	}
	if tokenCount != 1 {
		t.Fatalf("expected one non-orphan token to remain, got %d", tokenCount)
	}
	violations, err := foreignKeyViolations(db)
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 0 {
		t.Fatalf("foreign-key violations remained after repair: %#v", violations)
	}
	var auditCount int64
	if err := db.Table("audit_events").Where("event = ?", "foreign_key_check_failed").Count(&auditCount).Error; err != nil {
		t.Fatal(err)
	}
	if auditCount == 0 {
		t.Fatal("foreign-key repair did not write an audit event")
	}
}

func TestVerifyForeignKeysFailsFastWithoutRepair(t *testing.T) {
	db := openForeignKeyTestDB(t)
	createForeignKeyTestTables(t, db)
	if err := db.Exec("INSERT INTO tokens(id, user_id) VALUES(1, 42)").Error; err != nil {
		t.Fatal(err)
	}

	err := verifyForeignKeysBeforeMigration(db, Options{})
	if err == nil || !strings.Contains(err.Error(), "repair-fk-orphans") {
		t.Fatalf("expected repair guidance, got %v", err)
	}
}

func TestEnsureNoTLSForeignKeyParentRepairsNoTLSInbound(t *testing.T) {
	db := openForeignKeyTestDB(t)
	if err := db.Exec("PRAGMA foreign_keys = OFF").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`
CREATE TABLE tls(
	id integer PRIMARY KEY AUTOINCREMENT,
	name text,
	server blob,
	client blob
)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`
CREATE TABLE inbounds(
	id integer PRIMARY KEY AUTOINCREMENT,
	tag text,
	tls_id integer,
	FOREIGN KEY(tls_id) REFERENCES tls(id)
)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("INSERT INTO inbounds(tag, tls_id) VALUES(?, ?)", "no-tls", 0).Error; err != nil {
		t.Fatal(err)
	}

	before, err := foreignKeyViolations(db)
	if err != nil {
		t.Fatal(err)
	}
	if len(before) != 1 || before[0].Table != "inbounds" {
		t.Fatalf("expected one no-TLS inbound violation before repair, got %#v", before)
	}
	if err := ensureNoTLSForeignKeyParent(db); err != nil {
		t.Fatal(err)
	}
	after, err := foreignKeyViolations(db)
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != 0 {
		t.Fatalf("foreign-key violations remained after no-TLS parent repair: %#v", after)
	}
}

func openForeignKeyTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	name := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	db, err := gorm.Open(sqlite.Open("file:"+name+"?mode=memory&cache=shared&_foreign_keys=off"), &gorm.Config{})
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

func createForeignKeyTestTables(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.Exec("PRAGMA foreign_keys = OFF").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("CREATE TABLE users(id integer PRIMARY KEY)").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("CREATE TABLE tokens(id integer PRIMARY KEY, user_id integer, FOREIGN KEY(user_id) REFERENCES users(id))").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`
CREATE TABLE audit_events (
	id integer PRIMARY KEY AUTOINCREMENT,
	date_time integer,
	actor text,
	event text,
	resource text,
	severity text,
	ip text,
	user_agent text,
	details blob
)`).Error; err != nil {
		t.Fatal(err)
	}
}
