package ipmonitor

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/deposist/s-ui-x-extended/logger"
	"github.com/deposist/s-ui-x-extended/realtime"
	"github.com/deposist/s-ui-x-extended/util/common"
	"golang.org/x/sync/singleflight"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	ModeMonitor = "monitor"
	ModeEnforce = "enforce"

	allowCacheTTL          = 30 * time.Second
	databaseReadTimeout    = 3 * time.Second
	securityEventDebounce  = 60 * time.Second
	securityEventMaxMapAge = time.Hour
	ipMaskPrefix           = 12
)

type pendingIP struct {
	lastSeen int64
	display  *string
}

type lifecycleToken struct {
	generation       uint64
	clientGeneration uint64
}

type clientLifecycle struct {
	generation uint64
	clearing   bool
}

var pending = struct {
	sync.Mutex
	byClient        map[string]map[string]pendingIP
	inFlight        map[*PendingSnapshot]struct{}
	generation      uint64
	clientLifecycle map[string]clientLifecycle
}{
	byClient:        map[string]map[string]pendingIP{},
	inFlight:        map[*PendingSnapshot]struct{}{},
	generation:      1,
	clientLifecycle: map[string]clientLifecycle{},
}

type allowCacheEntry struct {
	limit     int
	mode      string
	ips       map[string]struct{}
	expiresAt time.Time
}

var allowCache = struct {
	sync.Mutex
	byClient map[string]allowCacheEntry
	revision uint64
}{
	byClient: map[string]allowCacheEntry{},
}

var lifecycleGate sync.RWMutex

var allowCacheRefresh singleflight.Group

var loadCacheEntryForAllow = loadCacheEntryContext
var loadPolicyEntriesForWarmUp = loadPolicyEntriesContext

var installSaltLoad singleflight.Group

var securityEvents = struct {
	sync.Mutex
	lastEmittedAt map[string]time.Time
}{
	lastEmittedAt: map[string]time.Time{},
}

var ipHashSalt = struct {
	sync.Mutex
	value      []byte
	generation uint64
}{}

var ipPrivacySettings = struct {
	sync.Mutex
	showRaw    bool
	expiresAt  time.Time
	generation uint64
}{}

func init() {
	database.RegisterResetHook("ipmonitor", func() error {
		ResetCaches()
		return nil
	})
}

func ResetCaches() {
	lifecycleGate.Lock()
	defer lifecycleGate.Unlock()

	pending.Lock()
	pending.generation++
	pending.byClient = map[string]map[string]pendingIP{}
	pending.inFlight = map[*PendingSnapshot]struct{}{}
	pending.clientLifecycle = map[string]clientLifecycle{}
	pending.Unlock()

	allowCache.Lock()
	allowCache.byClient = map[string]allowCacheEntry{}
	allowCache.revision++
	allowCache.Unlock()

	securityEvents.Lock()
	securityEvents.lastEmittedAt = map[string]time.Time{}
	securityEvents.Unlock()

	ipHashSalt.Lock()
	ipHashSalt.value = nil
	ipHashSalt.generation = 0
	ipHashSalt.Unlock()

	ipPrivacySettings.Lock()
	ipPrivacySettings.showRaw = false
	ipPrivacySettings.expiresAt = time.Time{}
	ipPrivacySettings.generation = 0
	ipPrivacySettings.Unlock()
}

func lifecycleTokenForClient(clientName string) lifecycleToken {
	pending.Lock()
	defer pending.Unlock()
	return lifecycleTokenForClientLocked(clientName)
}

func lifecycleTokenForClientLocked(clientName string) lifecycleToken {
	return lifecycleToken{
		generation:       pending.generation,
		clientGeneration: pending.clientLifecycle[clientName].generation,
	}
}

func lifecycleTokenCurrentLocked(clientName string, token lifecycleToken) bool {
	client := pending.clientLifecycle[clientName]
	return token.generation == pending.generation &&
		token.clientGeneration == client.generation &&
		!client.clearing
}

func lifecycleGenerationCurrent(generation uint64) bool {
	pending.Lock()
	defer pending.Unlock()
	return generation == pending.generation
}

func Record(clientName string, ip string) {
	if clientName == "" || ip == "" {
		return
	}
	lifecycleGate.RLock()
	defer lifecycleGate.RUnlock()
	token := lifecycleTokenForClient(clientName)
	ipHash, display, ok := recordIPFieldsContext(context.Background(), ip, token.generation)
	if !ok {
		return
	}
	pending.Lock()
	if !lifecycleTokenCurrentLocked(clientName, token) {
		pending.Unlock()
		return
	}
	if pending.byClient[clientName] == nil {
		pending.byClient[clientName] = map[string]pendingIP{}
	}
	pending.byClient[clientName][ipHash] = pendingIP{lastSeen: time.Now().Unix(), display: display}
	pending.Unlock()
	cacheAddIP(clientName, ipHash, token)
}

// ObserveAndAllow atomically checks an IP against the client's limit and
// reserves accepted observations so concurrent first connections cannot all
// consume the same free slot.
func ObserveAndAllow(clientName string, ip string) bool {
	return ObserveAndAllowContext(context.Background(), clientName, ip)
}

func ObserveAndAllowContext(ctx context.Context, clientName string, ip string) bool {
	if clientName == "" || ip == "" {
		return true
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if ctx.Err() != nil {
		return false
	}
	lifecycleGate.RLock()
	defer lifecycleGate.RUnlock()
	token := lifecycleTokenForClient(clientName)
	ipHash, display, ok := recordIPFieldsContext(ctx, ip, token.generation)
	if !ok {
		return false
	}
	entry, loaded := clientEntryForAllow(ctx, clientName, time.Now())
	if !loaded || ctx.Err() != nil {
		return false
	}

	pending.Lock()
	if !lifecycleTokenCurrentLocked(clientName, token) {
		pending.Unlock()
		return false
	}
	if entry.mode == ModeEnforce && entry.limit > 0 {
		seen := make(map[string]struct{}, len(entry.ips)+len(pending.byClient[clientName])+1)
		for seenHash := range entry.ips {
			seen[seenHash] = struct{}{}
		}
		for seenHash := range pending.byClient[clientName] {
			seen[seenHash] = struct{}{}
		}
		for snapshot := range pending.inFlight {
			if snapshot.clientTokenCurrentLocked(clientName) {
				for seenHash := range snapshot.byClient[clientName] {
					seen[seenHash] = struct{}{}
				}
			}
		}
		seen[ipHash] = struct{}{}
		if len(seen) > entry.limit {
			payload := map[string]any{
				"kind": "ip_enforced_reject", "client": clientName, "ipHash": ipHash,
				"limit": entry.limit, "count": len(seen),
			}
			pending.Unlock()
			publishSecurityEvent(clientName, "ip_enforced_reject", payload)
			return false
		}
	}
	if pending.byClient[clientName] == nil {
		pending.byClient[clientName] = map[string]pendingIP{}
	}
	pending.byClient[clientName][ipHash] = pendingIP{lastSeen: time.Now().Unix(), display: display}
	pending.Unlock()
	cacheAddIP(clientName, ipHash, token)
	return true
}

func Allow(clientName string, ip string) bool {
	return AllowContext(context.Background(), clientName, ip)
}

func AllowContext(ctx context.Context, clientName string, ip string) bool {
	if clientName == "" || ip == "" {
		return true
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if ctx.Err() != nil {
		return false
	}
	lifecycleGate.RLock()
	defer lifecycleGate.RUnlock()
	token := lifecycleTokenForClient(clientName)
	ipHash, err := hashIPContext(ctx, ip, token.generation)
	if err != nil {
		// Without the installation salt the source cannot be compared with the
		// allow set. At cold start the policy is unknown, so fail closed.
		return false
	}
	entry, ok := clientEntryForAllow(ctx, clientName, time.Now())
	if !ok || ctx.Err() != nil {
		return false
	}
	pending.Lock()
	if !lifecycleTokenCurrentLocked(clientName, token) {
		pending.Unlock()
		return false
	}
	if entry.mode != ModeEnforce || entry.limit <= 0 {
		pending.Unlock()
		return true
	}
	seen := map[string]struct{}{ipHash: {}}
	for seenHash := range entry.ips {
		seen[seenHash] = struct{}{}
	}
	for seenHash := range pending.byClient[clientName] {
		seen[seenHash] = struct{}{}
	}
	for snapshot := range pending.inFlight {
		if snapshot.clientTokenCurrentLocked(clientName) {
			for seenHash := range snapshot.byClient[clientName] {
				seen[seenHash] = struct{}{}
			}
		}
	}
	pending.Unlock()
	if len(seen) <= entry.limit {
		return true
	}
	publishSecurityEvent(clientName, "ip_enforced_reject", map[string]any{
		"kind":   "ip_enforced_reject",
		"client": clientName,
		"ipHash": ipHash,
		"limit":  entry.limit,
		"count":  len(seen),
	})
	return false
}

func clientEntryForAllow(ctx context.Context, clientName string, now time.Time) (allowCacheEntry, bool) {
	entry, ok := cachedClient(clientName, now)
	if ok {
		return entry, true
	}
	allowCache.Lock()
	revision := allowCache.revision
	allowCache.Unlock()
	token := lifecycleTokenForClient(clientName)
	refreshKey := fmt.Sprintf("%s:%d:%d:%d", clientName, revision, token.generation, token.clientGeneration)
	refresh := allowCacheRefresh.DoChan(refreshKey, func() (value any, err error) {
		defer func() {
			if recovered := recover(); recovered != nil {
				value = nil
				err = fmt.Errorf("ipmonitor: cache loader panic: %v", recovered)
			}
		}()
		entry, loaded := loadCacheEntryForAllow(ctx, clientName, now)
		if loaded {
			pending.Lock()
			current := lifecycleTokenCurrentLocked(clientName, token)
			allowCache.Lock()
			if revision == allowCache.revision && current {
				allowCache.byClient[clientName] = entry
			} else {
				loaded = false
			}
			allowCache.Unlock()
			pending.Unlock()
		}
		return cacheRefreshResult{entry: entry, loaded: loaded, revision: revision}, nil
	})
	select {
	case <-ctx.Done():
		return allowCacheEntry{}, false
	case refreshed := <-refresh:
		if refreshed.Err != nil {
			return allowCacheEntry{}, false
		}
		result, typeOK := refreshed.Val.(cacheRefreshResult)
		allowCache.Lock()
		currentRevision := allowCache.revision
		allowCache.Unlock()
		pending.Lock()
		currentToken := lifecycleTokenCurrentLocked(clientName, token)
		pending.Unlock()
		if typeOK && refreshed.Err == nil && result.loaded && result.revision == currentRevision && currentToken {
			return result.entry, true
		}
	}
	return staleCachedClient(clientName)
}

func WarmUp() error {
	db := database.GetDB()
	if db == nil {
		return nil
	}
	lifecycleGate.RLock()
	defer lifecycleGate.RUnlock()
	pending.Lock()
	generation := pending.generation
	pending.Unlock()
	allowCache.Lock()
	revision := allowCache.revision
	allowCache.Unlock()
	ctx, cancel := boundedDatabaseContext(context.Background())
	defer cancel()
	if _, err := getInstallSaltContext(ctx, generation); err != nil {
		return err
	}
	entries, err := loadPolicyEntriesForWarmUp(ctx, db, time.Now(), generation)
	if err != nil {
		return err
	}
	pending.Lock()
	allowCache.Lock()
	if generation == pending.generation && revision == allowCache.revision {
		allowCache.byClient = entries
	}
	allowCache.Unlock()
	pending.Unlock()
	return nil
}

// SecurityEventAuditHook, when set by app wiring, mirrors enforced security
// events (e.g. ip_enforced_reject) into the durable audit log. It is a hook
// rather than a direct call to avoid an import cycle (service imports ipmonitor)
// and is debounced upstream by shouldPublishSecurityEvent (no audit flooding).
var SecurityEventAuditHook func(clientName string, kind string, payload map[string]any)

func publishSecurityEvent(clientName string, kind string, payload map[string]any) {
	if !shouldPublishSecurityEvent(clientName, kind, time.Now()) {
		return
	}
	realtime.Publish(realtime.TopicSecurityEvent, payload)
	if hook := SecurityEventAuditHook; hook != nil {
		hook(clientName, kind, payload)
	}
}

func shouldPublishSecurityEvent(clientName string, kind string, now time.Time) bool {
	key := clientName + "|" + kind
	securityEvents.Lock()
	defer securityEvents.Unlock()
	if last, ok := securityEvents.lastEmittedAt[key]; ok && now.Sub(last) < securityEventDebounce {
		return false
	}
	securityEvents.lastEmittedAt[key] = now
	for eventKey, last := range securityEvents.lastEmittedAt {
		if now.Sub(last) > securityEventMaxMapAge {
			delete(securityEvents.lastEmittedAt, eventKey)
		}
	}
	return true
}

func Flush() error {
	db := database.GetDB()
	if db == nil {
		return nil
	}
	snapshot := SnapshotPending()
	if snapshot.empty() {
		return nil
	}
	tx := db.Begin()
	if tx.Error != nil {
		snapshot.Requeue()
		return tx.Error
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			_ = tx.Rollback().Error
			snapshot.Requeue()
			panic(recovered)
		}
	}()
	if err := snapshot.FlushTo(tx); err != nil {
		_ = tx.Rollback().Error
		snapshot.Requeue()
		return err
	}
	if err := tx.Commit().Error; err != nil {
		snapshot.Requeue()
		return err
	}
	snapshot.Ack()
	return nil
}

type PendingSnapshot struct {
	mu                sync.Mutex
	byClient          map[string]map[string]pendingIP
	generation        uint64
	clientGenerations map[string]uint64
	terminal          bool
	flushed           bool
	holdsLifecycle    bool
}

func SnapshotPending() *PendingSnapshot {
	pending.Lock()
	defer pending.Unlock()
	snapshot := &PendingSnapshot{
		byClient:          pending.byClient,
		generation:        pending.generation,
		clientGenerations: make(map[string]uint64, len(pending.byClient)),
	}
	pending.byClient = map[string]map[string]pendingIP{}
	for clientName := range snapshot.byClient {
		snapshot.clientGenerations[clientName] = pending.clientLifecycle[clientName].generation
	}
	if len(snapshot.byClient) == 0 {
		snapshot.terminal = true
		return snapshot
	}
	pending.inFlight[snapshot] = struct{}{}
	return snapshot
}

func (s *PendingSnapshot) empty() bool {
	return s == nil || len(s.byClient) == 0
}

func (s *PendingSnapshot) clientTokenCurrentLocked(clientName string) bool {
	if s == nil || s.generation != pending.generation {
		return false
	}
	client := pending.clientLifecycle[clientName]
	generation, ok := s.clientGenerations[clientName]
	return ok && generation == client.generation && !client.clearing
}

func (s *PendingSnapshot) currentLocked() bool {
	if s == nil || s.generation != pending.generation {
		return false
	}
	for clientName := range s.byClient {
		if !s.clientTokenCurrentLocked(clientName) {
			return false
		}
	}
	return true
}

// FlushTo writes this snapshot into tx but deliberately does not acknowledge
// it. The transaction owner must call Ack only after a successful commit, or
// Requeue after rollback/commit failure. The snapshot remains admission-visible
// until one of those mutually exclusive terminal calls wins.
func (s *PendingSnapshot) FlushTo(tx *gorm.DB) error {
	if s == nil || len(s.byClient) == 0 {
		return nil
	}
	if tx == nil {
		return errors.New("ipmonitor: nil flush transaction")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	lifecycleGate.RLock()
	keepLifecycle := false
	defer func() {
		if !keepLifecycle {
			lifecycleGate.RUnlock()
		}
	}()
	pending.Lock()
	valid := !s.terminal && !s.flushed && s.currentLocked()
	pending.Unlock()
	if !valid {
		return errors.New("ipmonitor: stale or terminal pending snapshot")
	}
	if err := flushSnapshot(tx, s.byClient); err != nil {
		return err
	}
	pending.Lock()
	valid = !s.terminal && s.currentLocked()
	if valid {
		s.flushed = true
		s.holdsLifecycle = true
		keepLifecycle = true
	}
	pending.Unlock()
	if !valid {
		return errors.New("ipmonitor: pending snapshot invalidated during flush")
	}
	return nil
}

func (s *PendingSnapshot) Ack() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	pending.Lock()
	if s.terminal {
		pending.Unlock()
		return
	}
	delete(pending.inFlight, s)
	s.terminal = true
	release := s.holdsLifecycle
	s.holdsLifecycle = false
	pending.Unlock()
	if release {
		lifecycleGate.RUnlock()
	}
}

func (s *PendingSnapshot) Requeue() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	pending.Lock()
	if s.terminal {
		pending.Unlock()
		return
	}
	delete(pending.inFlight, s)
	if s.currentLocked() {
		requeuePendingLocked(s)
	}
	s.terminal = true
	release := s.holdsLifecycle
	s.holdsLifecycle = false
	pending.Unlock()
	if release {
		lifecycleGate.RUnlock()
	}
}

type snapshotTxConnPool struct {
	gorm.ConnPool
	committer gorm.TxCommitter
	snapshot  *PendingSnapshot
}

func (p *snapshotTxConnPool) Commit() error {
	err := p.committer.Commit()
	if err != nil {
		p.snapshot.Requeue()
		return err
	}
	p.snapshot.Ack()
	return nil
}

func (p *snapshotTxConnPool) Rollback() error {
	err := p.committer.Rollback()
	p.snapshot.Requeue()
	return err
}

// FlushTo writes pending observations to a caller-owned transaction and wraps
// that transaction so the snapshot is acknowledged only by a successful
// Commit. Rollback or commit failure requeues it. Callers must always finish tx.
func FlushTo(tx *gorm.DB) error {
	if tx == nil {
		return errors.New("ipmonitor: nil flush transaction")
	}
	snapshot := SnapshotPending()
	if snapshot.empty() {
		return nil
	}
	if err := snapshot.FlushTo(tx); err != nil {
		snapshot.Requeue()
		return err
	}
	committer, ok := tx.Statement.ConnPool.(gorm.TxCommitter)
	if !ok || committer == nil {
		snapshot.Requeue()
		return errors.New("ipmonitor: FlushTo requires an active transaction")
	}
	tx.Statement.ConnPool = &snapshotTxConnPool{
		ConnPool:  tx.Statement.ConnPool,
		committer: committer,
		snapshot:  snapshot,
	}
	return nil
}

func requeuePendingLocked(snapshot *PendingSnapshot) {
	for clientName, ips := range snapshot.byClient {
		if !snapshot.clientTokenCurrentLocked(clientName) {
			continue
		}
		if pending.byClient[clientName] == nil {
			pending.byClient[clientName] = map[string]pendingIP{}
		}
		for ipHash, previous := range ips {
			current, exists := pending.byClient[clientName][ipHash]
			if !exists || previous.lastSeen > current.lastSeen {
				pending.byClient[clientName][ipHash] = previous
			}
		}
	}
}

func flushSnapshot(tx *gorm.DB, snapshot map[string]map[string]pendingIP) error {
	rows := make([]model.ClientIP, 0)
	lastSeenByClient := make(map[string]int64, len(snapshot))
	for clientName, ips := range snapshot {
		for ipHash, pendingIP := range ips {
			if pendingIP.lastSeen > lastSeenByClient[clientName] {
				lastSeenByClient[clientName] = pendingIP.lastSeen
			}
			rows = append(rows, model.ClientIP{
				ClientName: clientName,
				IPHash:     ipHash,
				IPDisplay:  pendingIP.display,
				FirstSeen:  pendingIP.lastSeen,
				LastSeen:   pendingIP.lastSeen,
			})
		}
	}
	if len(rows) == 0 {
		return nil
	}
	// One batched upsert replaces the former per-IP SELECT + INSERT/UPDATE (an
	// N+1 that ran every 10s). Legacy ip-only rows were given an ip_hash by
	// migration 1.5, so the (client_name, ip_hash) conflict target always
	// matches; first_seen is preserved while last_seen/ip_display are refreshed.
	batch := database.SafeSQLiteBatchSize(tx, &model.ClientIP{})
	if err := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "client_name"}, {Name: "ip_hash"}},
		DoUpdates: clause.AssignmentColumns([]string{"last_seen", "ip_display"}),
	}).CreateInBatches(&rows, batch).Error; err != nil {
		return err
	}
	// Refresh each active client's last_online and last_ip_count; the count is
	// folded into the same UPDATE via a correlated subquery (was a separate COUNT).
	for clientName, lastSeen := range lastSeenByClient {
		if err := tx.Model(model.Client{}).Where("name = ?", clientName).Updates(map[string]interface{}{
			"last_online":   lastSeen,
			"last_ip_count": gorm.Expr("(SELECT COUNT(*) FROM client_ips WHERE client_name = ?)", clientName),
		}).Error; err != nil {
			return err
		}
	}
	return nil
}

func History(clientName string, limit int) ([]model.ClientIP, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows := make([]model.ClientIP, 0)
	err := database.GetDB().Model(model.ClientIP{}).
		Where("client_name = ?", clientName).
		Order("last_seen desc").
		Limit(limit).
		Find(&rows).Error
	if err == nil {
		prepareHistoryRows(rows)
	}
	return rows, err
}

func Clear(clientName string) error {
	if clientName == "" {
		return nil
	}
	db := database.GetDB()
	if db == nil {
		return errors.New("database is not initialized")
	}
	lifecycleGate.Lock()
	defer lifecycleGate.Unlock()

	pending.Lock()
	client := pending.clientLifecycle[clientName]
	client.clearing = true
	client.generation++
	for snapshot := range pending.inFlight {
		delete(snapshot.byClient, clientName)
		delete(snapshot.clientGenerations, clientName)
	}
	pending.clientLifecycle[clientName] = client
	delete(pending.byClient, clientName)
	pending.Unlock()
	invalidateCache(clientName)

	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("client_name = ?", clientName).Delete(&model.ClientIP{}).Error; err != nil {
			return err
		}
		return tx.Model(model.Client{}).Where("name = ?", clientName).Updates(map[string]interface{}{
			"last_ip_count": 0,
		}).Error
	})

	pending.Lock()
	client = pending.clientLifecycle[clientName]
	client.clearing = false
	pending.clientLifecycle[clientName] = client
	pending.Unlock()
	return err
}

func cachedClient(clientName string, now time.Time) (allowCacheEntry, bool) {
	allowCache.Lock()
	defer allowCache.Unlock()
	if entry, ok := allowCache.byClient[clientName]; ok && now.Before(entry.expiresAt) {
		return cloneCacheEntry(entry), true
	}
	return allowCacheEntry{}, false
}

func staleCachedClient(clientName string) (allowCacheEntry, bool) {
	allowCache.Lock()
	defer allowCache.Unlock()
	entry, ok := allowCache.byClient[clientName]
	if !ok {
		return allowCacheEntry{}, false
	}
	return cloneCacheEntry(entry), true
}

type cacheRefreshResult struct {
	entry    allowCacheEntry
	loaded   bool
	revision uint64
}

// loadErrLog throttles DB-error logging so an outage cannot flood the log.
var loadErrLog = struct {
	sync.Mutex
	last time.Time
}{}

func logLoadCacheError(context string, err error) {
	loadErrLog.Lock()
	if !loadErrLog.last.IsZero() && time.Since(loadErrLog.last) < 30*time.Second {
		loadErrLog.Unlock()
		return
	}
	loadErrLog.last = time.Now()
	loadErrLog.Unlock()
	logger.Warning("ipmonitor: ip-limit ", context, " lookup failed; keeping stale policy or failing closed: ", err)
}

func loadCacheEntry(clientName string, now time.Time) (allowCacheEntry, bool) {
	return loadCacheEntryContext(context.Background(), clientName, now)
}

func loadCacheEntryContext(ctx context.Context, clientName string, now time.Time) (allowCacheEntry, bool) {
	db := database.GetDB()
	if db == nil {
		return allowCacheEntry{}, false
	}
	ctx, cancel := boundedDatabaseContext(ctx)
	defer cancel()
	db = db.WithContext(ctx)
	var client model.Client
	if err := db.Model(model.Client{}).Select("enable, limit_ip, ip_limit_mode").Where("name = ?", clientName).First(&client).Error; err != nil {
		if !database.IsNotFound(err) {
			logLoadCacheError("client", err)
		}
		return allowCacheEntry{}, false
	}
	if !client.Enable {
		return allowCacheEntry{expiresAt: now.Add(allowCacheTTL)}, true
	}
	entry := allowCacheEntry{
		limit:     client.LimitIP,
		mode:      client.IPLimitMode,
		ips:       map[string]struct{}{},
		expiresAt: now.Add(allowCacheTTL),
	}
	rows := make([]model.ClientIP, 0)
	if err := db.Model(model.ClientIP{}).Select("ip, ip_hash").Where("client_name = ?", clientName).Find(&rows).Error; err != nil {
		logLoadCacheError("client_ips", err)
		return allowCacheEntry{}, false
	}
	for _, row := range rows {
		ipHash := row.IPHash
		if ipHash == "" {
			ipHash = hashLegacyIPValue(row.IP)
		}
		if ipHash != "" {
			entry.ips[ipHash] = struct{}{}
		}
	}
	return entry, true
}

type activeEnforceCacheRow struct {
	ClientName  string
	LimitIP     int
	IPLimitMode string
	IP          sql.NullString
	IPHash      sql.NullString
}

func loadPolicyEntriesContext(ctx context.Context, db *gorm.DB, now time.Time, generation uint64) (map[string]allowCacheEntry, error) {
	ctx, cancel := boundedDatabaseContext(ctx)
	defer cancel()
	db = db.WithContext(ctx)
	rows := make([]activeEnforceCacheRow, 0)
	err := db.Raw(`
		SELECT
			clients.name AS client_name,
			clients.limit_ip,
			clients.ip_limit_mode,
			client_ips.ip,
			client_ips.ip_hash
		FROM clients
		LEFT JOIN client_ips ON client_ips.client_name = clients.name
		WHERE clients.enable = true
			AND clients.ip_limit_mode IN (?, ?)
		ORDER BY clients.name
	`, ModeMonitor, ModeEnforce).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	if !lifecycleGenerationCurrent(generation) {
		return nil, errors.New("ipmonitor: lifecycle changed while loading policies")
	}

	entries := make(map[string]allowCacheEntry)
	for _, row := range rows {
		entry, ok := entries[row.ClientName]
		if !ok {
			entry = allowCacheEntry{
				limit:     row.LimitIP,
				mode:      row.IPLimitMode,
				ips:       map[string]struct{}{},
				expiresAt: now.Add(allowCacheTTL),
			}
		}
		ipHash := ""
		if row.IPHash.Valid {
			ipHash = row.IPHash.String
		}
		if ipHash == "" && row.IP.Valid {
			ipHash = hashLegacyIPValue(row.IP.String)
		}
		if ipHash != "" {
			entry.ips[ipHash] = struct{}{}
		}
		entries[row.ClientName] = entry
	}
	return entries, nil
}

func refreshClient(clientName string, now time.Time) bool {
	token := lifecycleTokenForClient(clientName)
	entry, ok := loadCacheEntry(clientName, now)
	if !ok {
		return false
	}
	pending.Lock()
	defer pending.Unlock()
	if !lifecycleTokenCurrentLocked(clientName, token) {
		return false
	}
	allowCache.Lock()
	allowCache.byClient[clientName] = entry
	allowCache.Unlock()
	return true
}

func cloneCacheEntry(entry allowCacheEntry) allowCacheEntry {
	clone := allowCacheEntry{
		limit:     entry.limit,
		mode:      entry.mode,
		ips:       make(map[string]struct{}, len(entry.ips)),
		expiresAt: entry.expiresAt,
	}
	for ip := range entry.ips {
		clone.ips[ip] = struct{}{}
	}
	return clone
}

func cacheAddIP(clientName string, ip string, token lifecycleToken) {
	pending.Lock()
	defer pending.Unlock()
	if !lifecycleTokenCurrentLocked(clientName, token) {
		return
	}
	allowCache.Lock()
	defer allowCache.Unlock()
	entry, ok := allowCache.byClient[clientName]
	if !ok || time.Now().After(entry.expiresAt) {
		return
	}
	if entry.ips == nil {
		entry.ips = map[string]struct{}{}
	}
	entry.ips[ip] = struct{}{}
	allowCache.byClient[clientName] = entry
}

func invalidateCache(clientName string) {
	allowCache.Lock()
	defer allowCache.Unlock()
	allowCache.revision++
	delete(allowCache.byClient, clientName)
}

func InvalidateAllCache() {
	allowCache.Lock()
	defer allowCache.Unlock()
	allowCache.revision++
	allowCache.byClient = map[string]allowCacheEntry{}
}

func boundedDatabaseContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, ok := ctx.Deadline(); ok {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, databaseReadTimeout)
}

func recordIPFieldsContext(ctx context.Context, ip string, generation uint64) (string, *string, bool) {
	ipHash, err := hashIPContext(ctx, ip, generation)
	if err != nil {
		return "", nil, false
	}
	showRaw, err := getIPShowRawContext(ctx, time.Now(), generation)
	if err != nil || !showRaw {
		return ipHash, nil, true
	}
	display := ip
	return ipHash, &display, true
}

func hashIP(ip string) (string, error) {
	lifecycleGate.RLock()
	defer lifecycleGate.RUnlock()
	pending.Lock()
	generation := pending.generation
	pending.Unlock()
	return hashIPContext(context.Background(), ip, generation)
}

func hashIPContext(ctx context.Context, ip string, generation uint64) (string, error) {
	salt, err := getInstallSaltContext(ctx, generation)
	if err != nil {
		return "", err
	}
	if !lifecycleGenerationCurrent(generation) {
		return "", errors.New("ipmonitor: lifecycle changed while hashing IP")
	}
	h := sha256.New()
	_, _ = h.Write(salt)
	_, _ = h.Write([]byte(ip))
	return hex.EncodeToString(h.Sum(nil)), nil
}

func getInstallSaltContext(ctx context.Context, generation uint64) ([]byte, error) {
	ipHashSalt.Lock()
	if len(ipHashSalt.value) > 0 && ipHashSalt.generation == generation {
		salt := append([]byte(nil), ipHashSalt.value...)
		ipHashSalt.Unlock()
		return salt, nil
	}
	ipHashSalt.Unlock()

	key := strconv.FormatUint(generation, 10)
	value, err, _ := installSaltLoad.Do(key, func() (any, error) {
		db := database.GetDB()
		if db == nil {
			return nil, errors.New("database is not initialized")
		}
		readCtx, cancel := boundedDatabaseContext(ctx)
		defer cancel()
		db = db.WithContext(readCtx)
		var setting model.Setting
		err := db.Model(model.Setting{}).Where("key = ?", "installSalt").First(&setting).Error
		if database.IsNotFound(err) {
			setting = model.Setting{Key: "installSalt", Value: common.Random(32)}
			if createErr := db.Create(&setting).Error; createErr != nil {
				if rereadErr := db.Model(model.Setting{}).Where("key = ?", "installSalt").First(&setting).Error; rereadErr != nil {
					return nil, fmt.Errorf("installSalt create failed (%v) and reread failed: %w", createErr, rereadErr)
				}
			}
		} else if err != nil {
			return nil, err
		}
		return []byte(setting.Value), nil
	})
	if err != nil {
		return nil, err
	}
	salt := append([]byte(nil), value.([]byte)...)
	pending.Lock()
	if generation != pending.generation {
		pending.Unlock()
		return nil, errors.New("ipmonitor: lifecycle changed while loading install salt")
	}
	ipHashSalt.Lock()
	ipHashSalt.value = append([]byte(nil), salt...)
	ipHashSalt.generation = generation
	ipHashSalt.Unlock()
	pending.Unlock()
	return salt, nil
}

func getIPShowRaw(now time.Time) (bool, error) {
	lifecycleGate.RLock()
	defer lifecycleGate.RUnlock()
	pending.Lock()
	generation := pending.generation
	pending.Unlock()
	return getIPShowRawContext(context.Background(), now, generation)
}

func getIPShowRawContext(ctx context.Context, now time.Time, generation uint64) (bool, error) {
	ipPrivacySettings.Lock()
	if generation == ipPrivacySettings.generation && now.Before(ipPrivacySettings.expiresAt) {
		showRaw := ipPrivacySettings.showRaw
		ipPrivacySettings.Unlock()
		return showRaw, nil
	}
	ipPrivacySettings.Unlock()

	showRaw := false
	db := database.GetDB()
	if db != nil {
		readCtx, cancel := boundedDatabaseContext(ctx)
		defer cancel()
		var setting model.Setting
		err := db.WithContext(readCtx).Model(model.Setting{}).Where("key = ?", "ipShowRaw").First(&setting).Error
		if err != nil && !database.IsNotFound(err) {
			return false, err
		}
		if err == nil {
			var parseErr error
			showRaw, parseErr = strconv.ParseBool(setting.Value)
			if parseErr != nil {
				return false, parseErr
			}
		}
	}
	pending.Lock()
	if generation != pending.generation {
		pending.Unlock()
		return false, errors.New("ipmonitor: lifecycle changed while loading privacy settings")
	}
	ipPrivacySettings.Lock()
	ipPrivacySettings.showRaw = showRaw
	ipPrivacySettings.expiresAt = now.Add(allowCacheTTL)
	ipPrivacySettings.generation = generation
	ipPrivacySettings.Unlock()
	pending.Unlock()
	return showRaw, nil
}

func prepareHistoryRows(rows []model.ClientIP) {
	showRaw, err := getIPShowRaw(time.Now())
	if err != nil {
		showRaw = false
	}
	for i := range rows {
		display := maskedIP(rows[i])
		if showRaw {
			if rows[i].IPDisplay != nil && *rows[i].IPDisplay != "" {
				display = *rows[i].IPDisplay
			} else if rows[i].IPHash == "" && !looksLikeSHA256Hex(rows[i].IP) {
				display = rows[i].IP
			}
		}
		rows[i].IP = display
		rows[i].IPHash = ""
		rows[i].IPDisplay = nil
	}
}

func maskedIP(row model.ClientIP) string {
	ipHash := row.IPHash
	if ipHash == "" {
		ipHash = hashLegacyIPValue(row.IP)
	}
	if len(ipHash) < ipMaskPrefix {
		return "masked"
	}
	return "masked:" + ipHash[:ipMaskPrefix]
}

func hashLegacyIPValue(ip string) string {
	if looksLikeSHA256Hex(ip) {
		return ip
	}
	ipHash, err := hashIP(ip)
	if err != nil {
		return ""
	}
	return ipHash
}

func looksLikeSHA256Hex(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
