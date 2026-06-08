package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/logger"
)

const (
	tokenUseFlushInterval  = time.Minute
	tokenUseBatchSize      = 100
	tokenUseResetQuiet     = 5 * time.Second
	tokenUseFailureBackoff = 5 * time.Minute
)

type tokenUseUpdate struct {
	ip string
	ts int64
}

type tokenUseDebouncer struct {
	mu             sync.Mutex
	writeMu        sync.Mutex
	pending        map[uint]tokenUseUpdate
	timer          *time.Timer
	epoch          uint64
	interval       time.Duration
	failureBackoff time.Duration
	circuitUntil   time.Time
	flush          func(map[uint]tokenUseUpdate) error
}

func newTokenUseDebouncer(interval time.Duration, flush func(map[uint]tokenUseUpdate) error) *tokenUseDebouncer {
	if interval <= 0 {
		interval = tokenUseFlushInterval
	}
	return &tokenUseDebouncer{
		pending:        make(map[uint]tokenUseUpdate),
		interval:       interval,
		failureBackoff: tokenUseFailureBackoff,
		flush:          flush,
	}
}

func getTokenUseDebouncer() *tokenUseDebouncer {
	return DefaultRuntime().tokenUseDebouncer()
}

func resetTokenUseDebouncerForTest() {
	DefaultRuntime().resetTokenUseDebouncer()
	resumeTokenUseFlush()
}

func StopTokenUseDebouncer(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	finishReset, ok := beginTokenUseStop()
	if !ok {
		return ctx.Err()
	}
	defer finishReset()
	debouncer := getTokenUseDebouncer()
	if debouncer == nil {
		return ctx.Err()
	}
	return debouncer.flushNow(ctx, true)
}

func (d *tokenUseDebouncer) Record(id uint, ip string, ts int64) {
	if id == 0 {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.pending[id] = tokenUseUpdate{ip: ip, ts: ts}
	d.scheduleLocked()
}

func (d *tokenUseDebouncer) scheduleLocked() {
	if d.timer != nil {
		return
	}
	now := time.Now()
	epoch := d.epoch
	delay := d.scheduleDelayLocked(now)
	d.timer = time.AfterFunc(delay, func() {
		d.flushTimer(epoch)
	})
}

func (d *tokenUseDebouncer) flushTimer(epoch uint64) {
	releaseFlush, ok := beginTokenUseFlush()
	if !ok {
		d.mu.Lock()
		if epoch == d.epoch {
			d.timer = nil
			if len(d.pending) > 0 {
				d.scheduleLocked()
			}
		}
		d.mu.Unlock()
		return
	}
	defer releaseFlush()
	if d.timerCircuitOpen(epoch, time.Now()) {
		return
	}
	updates := d.takePending(epoch)
	if len(updates) == 0 {
		return
	}
	if err := d.write(updates); err != nil {
		logger.Warning("token use flush failed:", err)
		d.requeueAfterWriteError(updates, time.Now())
		return
	}
	d.closeFailureCircuit()
	d.mu.Lock()
	if len(d.pending) > 0 {
		d.scheduleLocked()
	}
	d.mu.Unlock()
}

func (d *tokenUseDebouncer) Flush(ctx context.Context) error {
	return d.flushNow(ctx, false)
}

func (d *tokenUseDebouncer) flushNow(ctx context.Context, force bool) error {
	var releaseFlush func()
	if !force {
		var ok bool
		releaseFlush, ok = beginTokenUseFlush()
		if !ok {
			return nil
		}
		defer releaseFlush()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	d.mu.Lock()
	d.epoch++
	if d.timer != nil {
		d.timer.Stop()
		d.timer = nil
	}
	updates := d.pending
	d.pending = make(map[uint]tokenUseUpdate)
	d.mu.Unlock()
	if len(updates) == 0 {
		return nil
	}

	err := d.write(updates)
	if err == nil {
		d.closeFailureCircuit()
	} else if !force {
		d.requeueAfterWriteError(updates, time.Now())
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	return err
}

func (d *tokenUseDebouncer) takePending(epoch uint64) map[uint]tokenUseUpdate {
	d.mu.Lock()
	defer d.mu.Unlock()
	if epoch != d.epoch {
		return nil
	}
	updates := d.pending
	d.pending = make(map[uint]tokenUseUpdate)
	d.timer = nil
	return updates
}

func (d *tokenUseDebouncer) write(updates map[uint]tokenUseUpdate) error {
	d.writeMu.Lock()
	defer d.writeMu.Unlock()
	if d.flush == nil || len(updates) == 0 {
		return nil
	}
	return d.flush(updates)
}

func (d *tokenUseDebouncer) scheduleDelayLocked(now time.Time) time.Duration {
	if d.circuitUntil.After(now) {
		return d.circuitUntil.Sub(now)
	}
	return d.interval
}

func (d *tokenUseDebouncer) timerCircuitOpen(epoch uint64, now time.Time) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if epoch != d.epoch {
		return true
	}
	if !d.circuitUntil.After(now) {
		return false
	}
	if d.timer != nil {
		d.timer.Stop()
		d.timer = nil
	}
	if len(d.pending) > 0 {
		d.scheduleLocked()
	}
	return true
}

func (d *tokenUseDebouncer) requeueAfterWriteError(updates map[uint]tokenUseUpdate, now time.Time) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.mergeTokenUseUpdatesLocked(updates)
	d.circuitUntil = now.Add(d.failureBackoff)
	d.epoch++
	if d.timer != nil {
		d.timer.Stop()
		d.timer = nil
	}
	if len(d.pending) > 0 {
		d.scheduleLocked()
	}
}

func (d *tokenUseDebouncer) closeFailureCircuit() {
	d.mu.Lock()
	hadCircuit := !d.circuitUntil.IsZero()
	d.circuitUntil = time.Time{}
	if hadCircuit && len(d.pending) > 0 {
		if d.timer != nil {
			d.timer.Stop()
			d.timer = nil
			d.epoch++
		}
		d.scheduleLocked()
	}
	d.mu.Unlock()
}

func (d *tokenUseDebouncer) mergeTokenUseUpdatesLocked(updates map[uint]tokenUseUpdate) {
	for id, update := range updates {
		if current, ok := d.pending[id]; !ok || update.ts > current.ts {
			d.pending[id] = update
		}
	}
}

var tokenUseFlushGate = struct {
	sync.RWMutex
	suspended bool
	timer     *time.Timer
}{}

func beginTokenUseFlush() (func(), bool) {
	tokenUseFlushGate.RLock()
	if tokenUseFlushGate.suspended {
		tokenUseFlushGate.RUnlock()
		return nil, false
	}
	return tokenUseFlushGate.RUnlock, true
}

func beginTokenUseReset() func() {
	tokenUseFlushGate.Lock()
	tokenUseFlushGate.suspended = true
	if tokenUseFlushGate.timer != nil {
		tokenUseFlushGate.timer.Stop()
		tokenUseFlushGate.timer = nil
	}
	return func() {
		tokenUseFlushGate.timer = time.AfterFunc(tokenUseResetQuiet, resumeTokenUseFlush)
		tokenUseFlushGate.Unlock()
	}
}

func beginTokenUseStop() (func(), bool) {
	tokenUseFlushGate.Lock()
	if tokenUseFlushGate.suspended {
		if tokenUseFlushGate.timer != nil {
			tokenUseFlushGate.timer.Stop()
		}
		tokenUseFlushGate.timer = time.AfterFunc(tokenUseResetQuiet, resumeTokenUseFlush)
		tokenUseFlushGate.Unlock()
		return nil, false
	}
	tokenUseFlushGate.suspended = true
	if tokenUseFlushGate.timer != nil {
		tokenUseFlushGate.timer.Stop()
		tokenUseFlushGate.timer = nil
	}
	return func() {
		tokenUseFlushGate.timer = time.AfterFunc(tokenUseResetQuiet, resumeTokenUseFlush)
		tokenUseFlushGate.Unlock()
	}, true
}

func resumeTokenUseFlush() {
	tokenUseFlushGate.Lock()
	tokenUseFlushGate.suspended = false
	if tokenUseFlushGate.timer != nil {
		tokenUseFlushGate.timer.Stop()
		tokenUseFlushGate.timer = nil
	}
	tokenUseFlushGate.Unlock()
}

func flushTokenUseUpdates(updates map[uint]tokenUseUpdate) error {
	db := database.GetDB()
	if db == nil || len(updates) == 0 {
		return nil
	}
	ids := make([]uint, 0, len(updates))
	for id := range updates {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	for start := 0; start < len(ids); start += tokenUseBatchSize {
		end := start + tokenUseBatchSize
		if end > len(ids) {
			end = len(ids)
		}
		if err := flushTokenUseBatch(ids[start:end], updates); err != nil {
			return err
		}
	}
	return nil
}

func flushTokenUseBatch(ids []uint, updates map[uint]tokenUseUpdate) error {
	if len(ids) == 0 {
		return nil
	}
	var query strings.Builder
	args := make([]any, 0, len(ids)*5)
	query.WriteString("UPDATE tokens SET last_used_at = CASE id")
	for _, id := range ids {
		update := updates[id]
		query.WriteString(" WHEN ? THEN ?")
		args = append(args, id, update.ts)
	}
	query.WriteString(" END, last_used_ip = CASE id")
	for _, id := range ids {
		update := updates[id]
		query.WriteString(" WHEN ? THEN ?")
		args = append(args, id, update.ip)
	}
	query.WriteString(" END WHERE id IN (")
	for i, id := range ids {
		if i > 0 {
			query.WriteByte(',')
		}
		query.WriteByte('?')
		args = append(args, id)
	}
	query.WriteByte(')')
	if err := database.GetDB().Exec(query.String(), args...).Error; err != nil {
		return fmt.Errorf("flush token use batch: %w", err)
	}
	return nil
}
