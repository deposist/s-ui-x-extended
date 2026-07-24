package service

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/deposist/s-ui-x-extended/database/model"
)

func TestAuditWriterExtraOverflowIncrementsDroppedTotal(t *testing.T) {
	auditDroppedTotal.Store(0)
	writer := newAuditWriter(1, 10, time.Hour, nil)

	writer.push(model.AuditEvent{Event: "first"})
	writer.push(model.AuditEvent{Event: "second"})
	writer.push(model.AuditEvent{Event: "third"})

	if got := AuditDroppedTotal(); got != 2 {
		t.Fatalf("dropped total=%d, want 2", got)
	}
	writer.mu.Lock()
	defer writer.mu.Unlock()
	if len(writer.queue) != 1 || writer.queue[0].Event != "third" {
		t.Fatalf("overflow should retain newest event, queue=%#v", writer.queue)
	}
}

func TestAuditWriterExtraFlushesPartialBatchOnInterval(t *testing.T) {
	wrote := make(chan []model.AuditEvent, 1)
	writer := newAuditWriter(10, 5, 20*time.Millisecond, func(events []model.AuditEvent) error {
		wrote <- append([]model.AuditEvent(nil), events...)
		return nil
	})
	defer func() {
		if err := writer.Stop(context.Background()); err != nil {
			t.Fatal(err)
		}
	}()

	writer.Enqueue(model.AuditEvent{Event: "one"})
	writer.Enqueue(model.AuditEvent{Event: "two"})

	select {
	case events := <-wrote:
		if len(events) != 2 || events[0].Event != "one" || events[1].Event != "two" {
			t.Fatalf("unexpected interval batch: %#v", events)
		}
	case <-time.After(time.Second):
		t.Fatal("partial audit batch was not flushed on interval")
	}
}

func TestAuditWriterExtraRetriesFailedBatchWithoutLosingEvents(t *testing.T) {
	attempts := make(chan []model.AuditEvent, 3)
	var calls atomic.Int32
	writer := newAuditWriter(10, 1, time.Hour, func(events []model.AuditEvent) error {
		attempts <- append([]model.AuditEvent(nil), events...)
		if calls.Add(1) == 1 {
			return context.DeadlineExceeded
		}
		return nil
	})
	defer func() {
		if err := writer.Stop(context.Background()); err != nil {
			t.Fatal(err)
		}
	}()

	writer.Enqueue(model.AuditEvent{Event: "durable"})

	for attempt := 1; attempt <= 2; attempt++ {
		select {
		case events := <-attempts:
			if len(events) != 1 || events[0].Event != "durable" {
				t.Fatalf("attempt %d wrote unexpected batch: %#v", attempt, events)
			}
		case <-time.After(time.Second):
			t.Fatalf("attempt %d did not write the failed batch", attempt)
		}
	}
}

func TestAuditWriterExtraStopReturnsErrorWhenEventsRemainUnsaved(t *testing.T) {
	writer := newAuditWriter(10, 1, time.Hour, func([]model.AuditEvent) error {
		return context.DeadlineExceeded
	})
	writer.Enqueue(model.AuditEvent{Event: "unsaved"})

	if err := writer.Stop(context.Background()); err == nil {
		t.Fatal("Stop returned nil while unsaved audit events remained")
	}
	writer.mu.Lock()
	defer writer.mu.Unlock()
	if len(writer.queue) != 1 || writer.queue[0].Event != "unsaved" {
		t.Fatalf("unsaved events were lost: %#v", writer.queue)
	}
}

func TestAuditWriterExtraStopFlushesUnsentBatch(t *testing.T) {
	wrote := make(chan []model.AuditEvent, 1)
	writer := newAuditWriter(10, 10, time.Hour, func(events []model.AuditEvent) error {
		wrote <- append([]model.AuditEvent(nil), events...)
		return nil
	})

	writer.Enqueue(model.AuditEvent{Event: "pending"})
	if err := writer.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}

	select {
	case events := <-wrote:
		if len(events) != 1 || events[0].Event != "pending" {
			t.Fatalf("unexpected stop flush batch: %#v", events)
		}
	default:
		t.Fatal("Stop did not flush pending audit batch")
	}
}
