package migration

import "gorm.io/gorm"

func to1_4(db *gorm.DB) error {
	if err := addColumnIfMissing(db, "tokens", "token_hash", "TEXT"); err != nil {
		return err
	}
	if err := addColumnIfMissing(db, "tokens", "token_prefix", "TEXT"); err != nil {
		return err
	}
	if err := addColumnIfMissing(db, "tokens", "scope", "TEXT NOT NULL DEFAULT 'admin'"); err != nil {
		return err
	}
	if err := addColumnIfMissing(db, "tokens", "enabled", "BOOLEAN NOT NULL DEFAULT 1"); err != nil {
		return err
	}
	if err := addColumnIfMissing(db, "tokens", "created_at", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	if err := addColumnIfMissing(db, "tokens", "updated_at", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	if err := addColumnIfMissing(db, "tokens", "last_used_at", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	if err := addColumnIfMissing(db, "tokens", "last_used_ip", "TEXT"); err != nil {
		return err
	}
	if err := createAuditEventsTable(db); err != nil {
		return err
	}
	if err := backfillTokenScopes(db); err != nil {
		return err
	}
	if err := backfillTokenTimestamps(db); err != nil {
		return err
	}
	return nil
}

func backfillTokenScopes(db *gorm.DB) error {
	return db.Exec("UPDATE tokens SET scope = ? WHERE scope IS NULL OR scope = '' OR scope = ?", "admin", "full").Error
}

func backfillTokenTimestamps(db *gorm.DB) error {
	if err := db.Exec("UPDATE tokens SET created_at = strftime('%s','now') WHERE created_at = 0 OR created_at IS NULL").Error; err != nil {
		return err
	}
	return db.Exec("UPDATE tokens SET updated_at = strftime('%s','now') WHERE updated_at = 0 OR updated_at IS NULL").Error
}

func addColumnIfMissing(db *gorm.DB, table string, column string, definition string) error {
	hasColumn, err := sqliteHasColumn(db, table, column)
	if err != nil {
		return err
	}
	if hasColumn {
		return nil
	}
	return db.Exec("ALTER TABLE " + table + " ADD COLUMN " + column + " " + definition).Error
}

func sqliteHasColumn(db *gorm.DB, table string, column string) (bool, error) {
	rows, err := db.Raw("PRAGMA table_info(" + table + ")").Rows()
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var (
			cid       int
			name      string
			ctype     string
			notnull   int
			dfltValue interface{}
			pk        int
		)
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}

func createAuditEventsTable(db *gorm.DB) error {
	// audit_events.details is BLOB on legacy installs, TEXT on fresh installs (AutoMigrate).
	// SQLite is loosely-typed; both work. See remediation-plan-g §B3.
	if err := db.Exec(`
CREATE TABLE IF NOT EXISTS audit_events (
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
		return err
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_audit_events_date_time ON audit_events(date_time)").Error; err != nil {
		return err
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_audit_events_actor ON audit_events(actor)").Error; err != nil {
		return err
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_audit_events_event ON audit_events(event)").Error; err != nil {
		return err
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_audit_events_severity ON audit_events(severity)").Error; err != nil {
		return err
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_audit_events_lookup ON audit_events(date_time, actor, event)").Error; err != nil {
		return err
	}
	return nil
}
