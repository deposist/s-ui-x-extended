package service

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
	"github.com/deposist/s-ui-x/util/redact"
)

const (
	AuditSeverityInfo = "info"
	AuditSeverityWarn = "warn"
)

type AuditService struct {
	Runtime *Runtime
}

func (s *AuditService) runtime() *Runtime {
	if s != nil {
		return runtimeOrDefault(s.Runtime)
	}
	return DefaultRuntime()
}

type AuditEvent struct {
	Actor     string
	Event     string
	Resource  string
	Severity  string
	IP        string
	UserAgent string
	Details   map[string]any
}

var AuditSyncForTest bool

func (s *AuditService) Record(event AuditEvent) error {
	record, err := buildAuditRecord(event)
	if err != nil {
		return err
	}
	if AuditSyncForTest {
		return writeAuditEvents([]model.AuditEvent{record})
	}
	writeAuditRuntime(s.runtime().audit(), record)
	return nil
}

func buildAuditRecord(event AuditEvent) (model.AuditEvent, error) {
	if event.Severity == "" {
		event.Severity = AuditSeverityInfo
	}
	details, err := json.Marshal(redact.Value(event.Details))
	if err != nil {
		return model.AuditEvent{}, err
	}
	return model.AuditEvent{
		DateTime:  time.Now().Unix(),
		Actor:     event.Actor,
		Event:     event.Event,
		Resource:  event.Resource,
		Severity:  event.Severity,
		IP:        event.IP,
		UserAgent: event.UserAgent,
		Details:   details,
	}, nil
}

func writeAuditEvents(events []model.AuditEvent) error {
	if len(events) == 0 {
		return nil
	}
	db := database.GetDB()
	if db == nil {
		return errors.New("audit database is not initialized")
	}
	return db.Create(&events).Error
}

func (s *AuditService) List(limit int) ([]model.AuditEvent, error) {
	events, _, err := s.ListPage(0, limit)
	return events, err
}

func (s *AuditService) ListPage(cursor uint64, limit int) ([]model.AuditEvent, uint64, error) {
	return s.ListPageFiltered(cursor, limit, "", "", 0, 0)
}

func (s *AuditService) ListPageFiltered(cursor uint64, limit int, event string, severity string, since int64, until int64) ([]model.AuditEvent, uint64, error) {
	if limit <= 0 {
		limit = 200
	}
	if limit > 200 {
		limit = 200
	}
	events := make([]model.AuditEvent, 0, limit+1)
	query := database.GetDB().Model(model.AuditEvent{})
	if cursor > 0 {
		query = query.Where("id < ?", cursor)
	}
	if event != "" {
		query = query.Where("event = ?", event)
	}
	if severity != "" {
		query = query.Where("severity = ?", severity)
	}
	if since > 0 {
		query = query.Where("date_time >= ?", since)
	}
	if until > 0 {
		query = query.Where("date_time <= ?", until)
	}
	err := query.
		Order("id desc").
		Limit(limit + 1).
		Find(&events).Error
	if err != nil {
		return nil, 0, err
	}
	var nextCursor uint64
	if len(events) > limit {
		events = events[:limit]
		nextCursor = events[len(events)-1].Id
	}
	return events, nextCursor, nil
}

func (s *AuditService) Prune(retentionDays int) error {
	if retentionDays <= 0 {
		return nil
	}
	before := time.Now().Add(-time.Duration(retentionDays) * 24 * time.Hour).Unix()
	return database.GetDB().Where("date_time < ?", before).Delete(&model.AuditEvent{}).Error
}
