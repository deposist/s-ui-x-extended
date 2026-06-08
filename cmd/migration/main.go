package migration

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/deposist/s-ui-x/config"
	"github.com/deposist/s-ui-x/database/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Options struct {
	RepairForeignKeyOrphans bool
}

// MigrateDb runs schema migrations against the SQLite database located at
// `config.GetDBPath()`. The legacy variant terminated the process on any
// error, which made restoring an incompatible backup through the panel kill
// the whole panel. The function now returns an error so callers can decide
// what to do (the CLI prints and exits non-zero, the panel falls back to the
// previous database).
func MigrateDb() error {
	return MigrateDbWithOptions(Options{})
}

func MigrateDbWithOptions(options Options) error {
	// void running on first install
	path := config.GetDBPath()
	if _, err := os.Stat(path); err != nil {
		fmt.Println("Database not found")
		return nil
	}

	db, err := gorm.Open(sqlite.Open(sqliteMigrationDSN(path)))
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("db handle: %w", err)
	}
	defer sqlDB.Close()

	if err := ensureNoTLSForeignKeyParent(db); err != nil {
		return err
	}
	if err := verifyForeignKeysBeforeMigration(db, options); err != nil {
		return err
	}

	tx := db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("begin migration: %w", tx.Error)
	}
	committed := false
	defer func() {
		if !committed {
			tx.Rollback()
		}
	}()

	currentVersion := config.GetVersion()
	dbVersion := ""
	tx.Raw("SELECT value FROM settings WHERE key = ?", "version").Find(&dbVersion)
	fmt.Println("Current version:", currentVersion, "\nDatabase version:", dbVersion)

	if currentVersion == dbVersion {
		fmt.Println("Database is up to date, no need to migrate")
		return nil
	}
	if dbVersion != "" {
		cmp, ok := config.CompareVersions(dbVersion, currentVersion)
		if !ok {
			return fmt.Errorf("database version %q is not semver-compatible", dbVersion)
		}
		if cmp > 0 {
			fmt.Println("Database version is newer than current binary, no migration will run")
			return nil
		}
	}

	fmt.Println("Start migrating database...")

	// Before 1.2 (no version row at all -> very old layout)
	if dbVersion == "" {
		if err = to1_1(tx); err != nil {
			return fmt.Errorf("migration to 1.1: %w", err)
		}
		if err = to1_2(tx); err != nil {
			return fmt.Errorf("migration to 1.2: %w", err)
		}
		dbVersion = "1.2"
	}

	// Before 1.3
	if strings.HasPrefix(dbVersion, "1.2") {
		if err = to1_3(tx); err != nil {
			return fmt.Errorf("migration to 1.3: %w", err)
		}
		dbVersion = "1.3"
	}

	// Before 1.4
	if strings.HasPrefix(dbVersion, "1.3") {
		if err = to1_4(tx); err != nil {
			return fmt.Errorf("migration to 1.4: %w", err)
		}
		dbVersion = "1.4"
	}

	// Before 1.5
	if strings.HasPrefix(dbVersion, "1.4") {
		if err = to1_5(tx); err != nil {
			return fmt.Errorf("migration to 1.5: %w", err)
		}
		dbVersion = "1.5"
	}

	// Before 1.6
	if strings.HasPrefix(dbVersion, "1.5") {
		if err = to1_6(tx); err != nil {
			return fmt.Errorf("migration to 1.6: %w", err)
		}
		dbVersion = "1.6"
	}

	// Before 1.7
	if strings.HasPrefix(dbVersion, "1.6") {
		if err = to1_7(tx); err != nil {
			return fmt.Errorf("migration to 1.7: %w", err)
		}
		dbVersion = "1.7"
	}

	// Persist the new version. The settings row is created lazily in older
	// schemas, so use UPSERT semantics.
	var count int64
	if err = tx.Raw("SELECT COUNT(*) FROM settings WHERE key = ?", "version").Scan(&count).Error; err != nil {
		return fmt.Errorf("count version: %w", err)
	}
	if count == 0 {
		err = tx.Exec("INSERT INTO settings(key, value) VALUES(?, ?)", "version", currentVersion).Error
	} else {
		err = tx.Exec("UPDATE settings SET value = ? WHERE key = ?", currentVersion, "version").Error
	}
	if err != nil {
		return fmt.Errorf("update version: %w", err)
	}
	if err = tx.Commit().Error; err != nil {
		return fmt.Errorf("commit migration: %w", err)
	}
	committed = true
	if err = checkpointWAL(db); err != nil {
		fmt.Println("Warning: WAL checkpoint skipped:", err)
	}
	fmt.Println("Migration done!")
	return nil
}

func sqliteMigrationDSN(path string) string {
	sep := "?"
	if strings.Contains(path, "?") {
		sep = "&"
	}
	return path + sep + "_busy_timeout=10000&_foreign_keys=on"
}

func checkpointWAL(db *gorm.DB) error {
	return db.Exec("PRAGMA wal_checkpoint(FULL)").Error
}

type foreignKeyViolation struct {
	Table  string
	RowID  int64
	Parent string
	FKID   int
}

func verifyForeignKeysBeforeMigration(db *gorm.DB, options Options) error {
	violations, err := foreignKeyViolations(db)
	if err != nil {
		return fmt.Errorf("foreign key check: %w", err)
	}
	if len(violations) == 0 {
		return nil
	}
	fmt.Println("Foreign key check failed:", summarizeForeignKeyViolations(violations))
	if err := recordForeignKeyAudit(db, violations, false); err != nil {
		fmt.Println("Warning: foreign-key audit event skipped:", err)
	}
	if !options.RepairForeignKeyOrphans {
		return fmt.Errorf("foreign key check failed: %s; rerun `s-ui migrate -repair-fk-orphans` to delete safe token orphans, or repair the database manually", summarizeForeignKeyViolations(violations))
	}
	repaired, err := repairSafeForeignKeyOrphans(db, violations)
	if err != nil {
		return fmt.Errorf("repair foreign key orphans: %w", err)
	}
	remaining, err := foreignKeyViolations(db)
	if err != nil {
		return fmt.Errorf("foreign key recheck: %w", err)
	}
	if len(remaining) > 0 {
		_ = recordForeignKeyAudit(db, remaining, false)
		return fmt.Errorf("foreign key check still fails after deleting %d safe token orphans: %s; repair manually", repaired, summarizeForeignKeyViolations(remaining))
	}
	fmt.Printf("Foreign key repair deleted %d safe token orphan(s)\n", repaired)
	if err := recordForeignKeyAudit(db, violations, true); err != nil {
		fmt.Println("Warning: foreign-key repair audit event skipped:", err)
	}
	return nil
}

func ensureNoTLSForeignKeyParent(db *gorm.DB) error {
	if !db.Migrator().HasTable(&model.Tls{}) ||
		!db.Migrator().HasColumn(&model.Tls{}, "server") ||
		!db.Migrator().HasColumn(&model.Tls{}, "client") {
		return nil
	}
	if err := db.Exec("INSERT OR IGNORE INTO tls(id, name, server, client) VALUES(0, ?, ?, ?)", "__none__", []byte("{}"), []byte("{}")).Error; err != nil {
		return fmt.Errorf("ensure no-tls foreign key parent: %w", err)
	}
	return nil
}

func foreignKeyViolations(db *gorm.DB) ([]foreignKeyViolation, error) {
	rows, err := db.Raw("PRAGMA foreign_key_check").Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	violations := make([]foreignKeyViolation, 0)
	for rows.Next() {
		var row foreignKeyViolation
		if err := rows.Scan(&row.Table, &row.RowID, &row.Parent, &row.FKID); err != nil {
			return nil, err
		}
		violations = append(violations, row)
	}
	return violations, rows.Err()
}

func foreignKeyViolationCounts(violations []foreignKeyViolation) map[string]int {
	counts := make(map[string]int)
	for _, violation := range violations {
		counts[violation.Table]++
	}
	return counts
}

func summarizeForeignKeyViolations(violations []foreignKeyViolation) string {
	counts := foreignKeyViolationCounts(violations)
	parts := make([]string, 0, len(counts))
	for table, count := range counts {
		parts = append(parts, fmt.Sprintf("%s=%d", table, count))
	}
	sort.Strings(parts)
	return strings.Join(parts, ", ")
}

func repairSafeForeignKeyOrphans(db *gorm.DB, violations []foreignKeyViolation) (int64, error) {
	counts := foreignKeyViolationCounts(violations)
	if counts["tokens"] == 0 {
		return 0, nil
	}
	if !db.Migrator().HasTable("tokens") || !db.Migrator().HasTable("users") {
		return 0, nil
	}
	result := db.Exec(`
DELETE FROM tokens
WHERE user_id IS NULL
	OR user_id NOT IN (SELECT id FROM users)
`)
	return result.RowsAffected, result.Error
}

func recordForeignKeyAudit(db *gorm.DB, violations []foreignKeyViolation, repaired bool) error {
	if !db.Migrator().HasTable("audit_events") {
		return nil
	}
	details, err := json.Marshal(map[string]any{
		"counts":   foreignKeyViolationCounts(violations),
		"repaired": repaired,
	})
	if err != nil {
		return err
	}
	return db.Create(&model.AuditEvent{
		DateTime: time.Now().Unix(),
		Actor:    "system",
		Event:    "foreign_key_check_failed",
		Resource: "database",
		Severity: "warn",
		Details:  details,
	}).Error
}
