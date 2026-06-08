package service

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/deposist/s-ui-x/database/model"
	"github.com/deposist/s-ui-x/logger"
)

const (
	auditQueueCapacity = 4096
	auditBatchSize     = 64
	auditFlushInterval = 200 * time.Millisecond
)

var auditDroppedTotal atomic.Uint64

type auditWriter struct {
	capacity      int
	batchSize     int
	flushInterval time.Duration
	write         func([]model.AuditEvent) error

	mu      sync.Mutex
	queue   []model.AuditEvent
	notify  chan struct{}
	stopCh  chan struct{}
	done    chan struct{}
	started bool
	stopped bool
}

func newAuditWriter(capacity int, batchSize int, flushInterval time.Duration, write func([]model.AuditEvent) error) *auditWriter {
	if capacity <= 0 {
		capacity = auditQueueCapacity
	}
	if batchSize <= 0 {
		batchSize = auditBatchSize
	}
	if flushInterval <= 0 {
		flushInterval = auditFlushInterval
	}
	return &auditWriter{
		capacity:      capacity,
		batchSize:     batchSize,
		flushInterval: flushInterval,
		write:         write,
		queue:         make([]model.AuditEvent, 0, capacity),
		notify:        make(chan struct{}, 1),
		stopCh:        make(chan struct{}),
		done:          make(chan struct{}),
	}
}

func StopAuditWriter(ctx context.Context) error {
	runtime := DefaultRuntime()
	writer := runtime.audit()
	if writer == nil {
		return nil
	}

	err := writer.Stop(ctx)
	runtime.replaceAuditWriterIfCurrent(writer)
	return err
}

func AuditDroppedTotal() uint64 {
	return auditDroppedTotal.Load()
}

func (w *auditWriter) Enqueue(event model.AuditEvent) {
	if !w.Start() {
		return
	}
	w.push(event)
}

func (w *auditWriter) push(event model.AuditEvent) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.stopped {
		return
	}
	if len(w.queue) >= w.capacity {
		victim := auditOverflowVictimIndex(w.queue, event)
		if victim >= 0 {
			copy(w.queue[victim:], w.queue[victim+1:])
			w.queue[len(w.queue)-1] = event
		}
		auditDroppedTotal.Add(1)
		w.signalLocked()
		return
	}
	w.queue = append(w.queue, event)
	w.signalLocked()
}

func auditOverflowVictimIndex(queue []model.AuditEvent, event model.AuditEvent) int {
	for i, queued := range queue {
		if auditLowPriority(queued) {
			return i
		}
	}
	if auditLowPriority(event) {
		return -1
	}
	return 0
}

func auditLowPriority(event model.AuditEvent) bool {
	return event.Severity == "" || event.Severity == AuditSeverityInfo
}

func (w *auditWriter) Start() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.stopped {
		return false
	}
	if w.started {
		return true
	}
	w.started = true
	go w.run()
	return true
}

func (w *auditWriter) Stop(ctx context.Context) error {
	w.mu.Lock()
	if !w.started {
		w.stopped = true
		w.mu.Unlock()
		return nil
	}
	if !w.stopped {
		w.stopped = true
		close(w.stopCh)
	}
	done := w.done
	w.mu.Unlock()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (w *auditWriter) run() {
	defer close(w.done)
	for {
		batch := w.popBatch(w.batchSize)
		if len(batch) == 0 {
			select {
			case <-w.notify:
				continue
			case <-w.stopCh:
				w.flushRemaining()
				return
			}
		}

		timer := time.NewTimer(w.flushInterval)
		flush := false
		for len(batch) < w.batchSize && !flush {
			more := w.popBatch(w.batchSize - len(batch))
			if len(more) > 0 {
				batch = append(batch, more...)
				continue
			}
			select {
			case <-w.notify:
			case <-timer.C:
				flush = true
			case <-w.stopCh:
				stopTimer(timer)
				w.writeBatch(batch)
				w.flushRemaining()
				return
			}
		}
		stopTimer(timer)
		w.writeBatch(batch)
	}
}

func (w *auditWriter) popBatch(limit int) []model.AuditEvent {
	w.mu.Lock()
	defer w.mu.Unlock()
	if limit <= 0 || len(w.queue) == 0 {
		return nil
	}
	if limit > len(w.queue) {
		limit = len(w.queue)
	}
	batch := make([]model.AuditEvent, limit)
	copy(batch, w.queue[:limit])
	copy(w.queue, w.queue[limit:])
	clear(w.queue[len(w.queue)-limit:])
	w.queue = w.queue[:len(w.queue)-limit]
	return batch
}

func (w *auditWriter) flushRemaining() {
	for {
		batch := w.popBatch(w.batchSize)
		if len(batch) == 0 {
			return
		}
		w.writeBatch(batch)
	}
}

func (w *auditWriter) writeBatch(batch []model.AuditEvent) {
	if len(batch) == 0 || w.write == nil {
		return
	}
	if err := w.write(batch); err != nil {
		logger.Warning("audit writer flush failed:", err)
	}
}

func (w *auditWriter) signalLocked() {
	select {
	case w.notify <- struct{}{}:
	default:
	}
}

func stopTimer(timer *time.Timer) {
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
}
