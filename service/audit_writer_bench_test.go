package service

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/deposist/s-ui-x/database/model"
)

func BenchmarkAuditWriter_Push(b *testing.B) {
	writer := newAuditWriter(auditQueueCapacity, auditBatchSize, time.Hour, nil)
	auditDroppedTotal.Store(0)
	event := model.AuditEvent{Event: "phase5_audit_push", Severity: AuditSeverityInfo}
	b.ReportMetric(float64(auditQueueCapacity), "capacity")
	b.ReportMetric(float64(auditBatchSize), "batch")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		writer.push(event)
	}
	b.ReportMetric(float64(auditDroppedTotal.Load()), "drops")
}

func BenchmarkAuditWriter_Overload10000(b *testing.B) {
	for i := 0; i < b.N; i++ {
		auditDroppedTotal.Store(0)
		writer := newAuditWriter(auditQueueCapacity, auditBatchSize, time.Hour, nil)
		for j := 0; j < 10_000; j++ {
			writer.push(auditWriterBenchEvent(j))
		}
		b.ReportMetric(float64(auditDroppedTotal.Load()), "drops/op")
	}
}

func TestAuditWriterOverloadSeverityPriorityAnchorIssue16Phase5(t *testing.T) {
	auditDroppedTotal.Store(0)
	writer := newAuditWriter(auditQueueCapacity, auditBatchSize, time.Hour, nil)
	for i := 0; i < 10_000; i++ {
		writer.push(auditWriterBenchEvent(i))
	}

	kept := map[string]int{}
	writer.mu.Lock()
	for _, event := range writer.queue {
		kept[event.Severity]++
	}
	queueLen := len(writer.queue)
	writer.mu.Unlock()

	// #nosec G115 -- bench drop counter is well within int range.
	dropped := int(auditDroppedTotal.Load())
	wantKeptWarnSecurity := 5000
	if wantKeptWarnSecurity > auditQueueCapacity {
		wantKeptWarnSecurity = auditQueueCapacity
	}
	wantKeptInfo := auditQueueCapacity - wantKeptWarnSecurity
	lostWarnSecurity := 5000 - kept[AuditSeverityWarn]
	lostInfo := 5000 - kept[AuditSeverityInfo]
	if dropped != 10_000-auditQueueCapacity {
		t.Fatalf("unexpected drop count: got %d want %d", dropped, 10_000-auditQueueCapacity)
	}
	if queueLen != auditQueueCapacity {
		t.Fatalf("queue len=%d want %d", queueLen, auditQueueCapacity)
	}
	if kept[AuditSeverityWarn] != wantKeptWarnSecurity || kept[AuditSeverityInfo] != wantKeptInfo {
		t.Fatalf("severity priority eviction kept warn/security=%d info=%d; want warn/security=%d info=%d; kept=%v",
			kept[AuditSeverityWarn], kept[AuditSeverityInfo], wantKeptWarnSecurity, wantKeptInfo, kept)
	}
	t.Logf("phase5 issue16 anchor: dropped=%d lost_warn_security=%d lost_info=%d kept=%v", dropped, lostWarnSecurity, lostInfo, kept)
}

func auditWriterBenchEvent(i int) model.AuditEvent {
	severity := AuditSeverityInfo
	resource := "routine"
	if i < 5000 {
		severity = AuditSeverityWarn
		resource = "security"
	}
	return model.AuditEvent{
		Event:    fmt.Sprintf("phase5_audit_%05d", i),
		Resource: resource,
		Severity: severity,
	}
}

func BenchmarkAuditWriter_EnqueueAndFlush(b *testing.B) {
	var written atomic.Int64
	writer := newAuditWriter(auditQueueCapacity, auditBatchSize, time.Millisecond, func(events []model.AuditEvent) error {
		written.Add(int64(len(events)))
		return nil
	})
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = writer.Stop(ctx)
	}()
	event := model.AuditEvent{Event: "phase5_audit_enqueue", Severity: AuditSeverityInfo}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		writer.Enqueue(event)
	}
	b.StopTimer()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_ = writer.Stop(ctx)
	b.ReportMetric(float64(written.Load()), "written")
}
