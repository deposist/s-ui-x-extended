package migration

import "gorm.io/gorm"

func to1_6(db *gorm.DB) error {
	if err := createAuditEventsTable(db); err != nil {
		return err
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_audit_events_event_dt ON audit_events(event, date_time DESC)").Error; err != nil {
		return err
	}
	return db.Exec("CREATE INDEX IF NOT EXISTS idx_audit_events_severity_dt ON audit_events(severity, date_time DESC)").Error
}
