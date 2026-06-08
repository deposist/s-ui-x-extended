package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
	"github.com/deposist/s-ui-x/util/redact"
)

func TestAuditRecordRedactsDetails(t *testing.T) {
	auditService := &AuditService{}
	initSettingTestDB(t)

	if err := auditService.Record(AuditEvent{
		Actor:    "admin",
		Event:    "api_token_created",
		Resource: "api_token",
		Details: map[string]any{
			"token": "raw-token",
			"desc":  "automation",
		},
	}); err != nil {
		t.Fatal(err)
	}
	events, err := auditService.List(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("expected one event, got %d", len(events))
	}
	var details map[string]any
	if err := json.Unmarshal(events[0].Details, &details); err != nil {
		t.Fatal(err)
	}
	if details["token"] != redact.Marker {
		t.Fatalf("token was not redacted: %#v", details["token"])
	}
	if details["desc"] != "automation" {
		t.Fatalf("non-secret detail changed: %#v", details["desc"])
	}
}

func TestAuditPruneDeletesOldEvents(t *testing.T) {
	auditService := &AuditService{}
	initSettingTestDB(t)

	old := model.AuditEvent{
		DateTime: time.Now().Add(-48 * time.Hour).Unix(),
		Actor:    "admin",
		Event:    "old",
	}
	recent := model.AuditEvent{
		DateTime: time.Now().Unix(),
		Actor:    "admin",
		Event:    "recent",
	}
	if err := database.GetDB().Create(&[]model.AuditEvent{old, recent}).Error; err != nil {
		if strings.Contains(err.Error(), "no such table") {
			t.Skip(err)
		}
		t.Fatal(err)
	}
	if err := auditService.Prune(1); err != nil {
		t.Fatal(err)
	}
	events, err := auditService.List(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Event != "recent" {
		t.Fatalf("unexpected events after prune: %#v", events)
	}
}

func TestAuditWriterDropsOldestWhenFull(t *testing.T) {
	wrote := []model.AuditEvent{}
	auditDroppedTotal.Store(0)
	writer := newAuditWriter(2, 10, time.Hour, func(events []model.AuditEvent) error {
		t.Fatal("drop-oldest test should not start the writer")
		return nil
	})

	writer.push(model.AuditEvent{Event: "first"})
	writer.push(model.AuditEvent{Event: "second"})
	writer.push(model.AuditEvent{Event: "third"})

	if AuditDroppedTotal() != 1 {
		t.Fatalf("audit dropped total=%d, want 1", AuditDroppedTotal())
	}
	writer.mu.Lock()
	wrote = append(wrote, writer.queue...)
	writer.mu.Unlock()
	if len(wrote) != 2 || wrote[0].Event != "second" || wrote[1].Event != "third" {
		t.Fatalf("unexpected queued events after drop-oldest: %#v", wrote)
	}
}

func TestAuditWriterFlushesBatchAtBatchSize(t *testing.T) {
	wrote := make(chan []model.AuditEvent, 1)
	writer := newAuditWriter(10, 2, time.Hour, func(events []model.AuditEvent) error {
		copied := append([]model.AuditEvent(nil), events...)
		wrote <- copied
		return nil
	})

	writer.Enqueue(model.AuditEvent{Event: "one"})
	writer.Enqueue(model.AuditEvent{Event: "two"})
	defer func() {
		if err := writer.Stop(context.Background()); err != nil {
			t.Fatal(err)
		}
	}()

	select {
	case events := <-wrote:
		if len(events) != 2 || events[0].Event != "one" || events[1].Event != "two" {
			t.Fatalf("unexpected batch: %#v", events)
		}
	case <-time.After(time.Second):
		t.Fatal("audit writer did not flush at batch size")
	}
}

func TestAuditWriterStopIsIdempotentAndRejectsLateEnqueue(t *testing.T) {
	wrote := make(chan []model.AuditEvent, 2)
	writer := newAuditWriter(10, 1, time.Hour, func(events []model.AuditEvent) error {
		wrote <- append([]model.AuditEvent(nil), events...)
		return nil
	})

	writer.Enqueue(model.AuditEvent{Event: "before_stop"})
	select {
	case events := <-wrote:
		if len(events) != 1 || events[0].Event != "before_stop" {
			t.Fatalf("unexpected initial write: %#v", events)
		}
	case <-time.After(time.Second):
		t.Fatal("audit writer did not flush before stop")
	}

	if err := writer.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := writer.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	writer.Enqueue(model.AuditEvent{Event: "after_stop"})

	select {
	case events := <-wrote:
		t.Fatalf("late enqueue flushed after stop: %#v", events)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestAuditWriterStopBeforeStartPreventsStart(t *testing.T) {
	writer := newAuditWriter(10, 1, time.Hour, func(events []model.AuditEvent) error {
		t.Fatalf("stopped writer flushed events: %#v", events)
		return nil
	})
	if err := writer.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	writer.Enqueue(model.AuditEvent{Event: "after_stop"})
	writer.mu.Lock()
	defer writer.mu.Unlock()
	if !writer.stopped || writer.started || len(writer.queue) != 0 {
		t.Fatalf("unexpected writer state after stop-before-start: started=%v stopped=%v queue=%d", writer.started, writer.stopped, len(writer.queue))
	}
}

func TestAuditRecordSyncForTestWritesImmediately(t *testing.T) {
	auditService := &AuditService{}
	initSettingTestDB(t)

	if err := auditService.Record(AuditEvent{Event: "sync_test"}); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := database.GetDB().Model(model.AuditEvent{}).Where("event = ?", "sync_test").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("sync audit record count=%d, want 1", count)
	}
}

func TestAuditRecordReturnsMarshalErrorOnly(t *testing.T) {
	auditService := &AuditService{}
	initSettingTestDB(t)
	prevSync := AuditSyncForTest
	AuditSyncForTest = false
	t.Cleanup(func() { AuditSyncForTest = prevSync })

	if err := auditService.Record(AuditEvent{
		Event:   "bad_details",
		Details: map[string]any{"bad": func() {}},
	}); err == nil {
		t.Fatal("expected marshal error")
	}
}

func TestRecordListenFallbackAudit(t *testing.T) {
	initSettingTestDB(t)

	if err := RecordListenFallbackAudit("web", "192.0.2.10:2095", ":2095", nil); err != nil {
		t.Fatal(err)
	}

	var event model.AuditEvent
	if err := database.GetDB().Where("event = ?", "listen_fallback").First(&event).Error; err != nil {
		t.Fatal(err)
	}
	if event.Actor != "system" || event.Resource != "network" || event.Severity != AuditSeverityWarn {
		t.Fatalf("unexpected audit event: %#v", event)
	}
	var details map[string]any
	if err := json.Unmarshal(event.Details, &details); err != nil {
		t.Fatal(err)
	}
	if details["component"] != "web" || details["requested_addr"] != "192.0.2.10:2095" || details["fallback_addr"] != ":2095" {
		t.Fatalf("unexpected details: %#v", details)
	}
}
