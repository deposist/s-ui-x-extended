package migration

import (
	"strings"
	"testing"

	"gorm.io/gorm"
)

func TestTo16AddsAuditFilterIndexesAndQueryPlannerUsesThem(t *testing.T) {
	db := openMigrationTestDB(t)
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
	if err := db.Exec(`
INSERT INTO audit_events(date_time, actor, event, resource, severity, ip, user_agent, details)
VALUES
	(3, 'admin', 'login', 'session', 'info', '', '', '{}'),
	(2, 'admin', 'token_create', 'token', 'warning', '', '', '{}'),
	(1, 'admin', 'login', 'session', 'error', '', '', '{}')
`).Error; err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 2; i++ {
		if err := to1_6(db); err != nil {
			t.Fatal(err)
		}
	}

	for _, indexName := range []string{"idx_audit_events_event_dt", "idx_audit_events_severity_dt"} {
		hasIndex, err := sqliteHasIndex(db, "audit_events", indexName)
		if err != nil {
			t.Fatal(err)
		}
		if !hasIndex {
			t.Fatalf("%s was not created", indexName)
		}
	}

	assertQueryUsesIndex(t, db, "idx_audit_events_event_dt",
		"EXPLAIN QUERY PLAN SELECT * FROM audit_events WHERE event = ? ORDER BY date_time DESC LIMIT 50", "login")
	assertQueryUsesIndex(t, db, "idx_audit_events_severity_dt",
		"EXPLAIN QUERY PLAN SELECT * FROM audit_events WHERE severity = ? ORDER BY date_time DESC LIMIT 50", "error")
}

func TestTo16CreatesAuditEventsTableForLegacy14Backups(t *testing.T) {
	db := openMigrationTestDB(t)

	if err := to1_6(db); err != nil {
		t.Fatal(err)
	}
	if !db.Migrator().HasTable("audit_events") {
		t.Fatal("audit_events table was not created")
	}
	for _, indexName := range []string{"idx_audit_events_event_dt", "idx_audit_events_severity_dt"} {
		hasIndex, err := sqliteHasIndex(db, "audit_events", indexName)
		if err != nil {
			t.Fatal(err)
		}
		if !hasIndex {
			t.Fatalf("%s was not created", indexName)
		}
	}
}

func sqliteHasIndex(db *gorm.DB, table string, indexName string) (bool, error) {
	rows, err := db.Raw("PRAGMA index_list(" + table + ")").Rows()
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var (
			seq     int
			name    string
			unique  int
			origin  string
			partial int
		)
		if err := rows.Scan(&seq, &name, &unique, &origin, &partial); err != nil {
			return false, err
		}
		if name == indexName {
			return true, nil
		}
	}
	return false, rows.Err()
}

func assertQueryUsesIndex(t *testing.T, db *gorm.DB, indexName string, query string, args ...any) {
	t.Helper()
	var plan []struct {
		Detail string
	}
	if err := db.Raw(query, args...).Scan(&plan).Error; err != nil {
		t.Fatal(err)
	}
	for _, row := range plan {
		if strings.Contains(row.Detail, indexName) {
			return
		}
	}
	t.Fatalf("query plan did not use %s: %#v", indexName, plan)
}
