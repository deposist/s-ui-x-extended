package ipmonitor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/deposist/s-ui-x-extended/realtime"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func initIPMonitorTestDB(t *testing.T) {
	t.Helper()
	pending.Lock()
	pending.byClient = map[string]map[string]pendingIP{}
	pending.inFlight = map[*PendingSnapshot]struct{}{}
	pending.clientLifecycle = map[string]clientLifecycle{}
	pending.generation++
	pending.Unlock()
	allowCache.Lock()
	allowCache.byClient = map[string]allowCacheEntry{}
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
	realtime.CloseAll("test_reset")
	tempDir := makeIPMonitorTempDir(t, "s-ui-ipmonitor-test-")
	t.Setenv("SUI_DB_FOLDER", tempDir)
	closeIPMonitorTestDB(database.GetDB())
	if err := database.InitDB(filepath.Join(tempDir, "s-ui.db")); err != nil {
		if strings.Contains(err.Error(), "go-sqlite3 requires cgo") {
			t.Skip(err)
		}
		t.Fatal(err)
	}
	testDB := database.GetDB()
	t.Cleanup(func() {
		closeIPMonitorTestDB(testDB)
		realtime.CloseAll("test_done")
	})
}

func closeIPMonitorTestDB(db *gorm.DB) {
	if db == nil {
		return
	}
	_ = db.Exec("PRAGMA wal_checkpoint(TRUNCATE)").Error
	if sqlDB, err := db.DB(); err == nil {
		_ = sqlDB.Close()
	}
}

func makeIPMonitorTempDir(tb testing.TB, prefix string) string {
	tb.Helper()
	dir, err := os.MkdirTemp("", prefix)
	if err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(func() {
		var removeErr error
		for i := 0; i < 20; i++ {
			removeErr = os.RemoveAll(dir)
			if removeErr == nil || os.IsNotExist(removeErr) {
				return
			}
			time.Sleep(time.Duration(i+1) * 10 * time.Millisecond)
		}
		tb.Errorf("remove ipmonitor temp dir %q: %v", dir, removeErr)
	})
	return dir
}

func TestRecordFlushAndClear(t *testing.T) {
	initIPMonitorTestDB(t)
	if err := database.GetDB().Create(&model.Client{
		Enable:      true,
		Name:        "alice",
		IPLimitMode: ModeMonitor,
		Inbounds:    []byte("[]"),
		Links:       []byte("[]"),
	}).Error; err != nil {
		t.Fatal(err)
	}
	Record("alice", "198.51.100.10")
	Record("alice", "198.51.100.11")
	if err := Flush(); err != nil {
		t.Fatal(err)
	}
	rows, err := History("alice", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected two IP rows, got %d", len(rows))
	}
	var client model.Client
	if err := database.GetDB().Where("name = ?", "alice").First(&client).Error; err != nil {
		t.Fatal(err)
	}
	if client.LastIPCount != 2 || client.LastOnline == 0 {
		t.Fatalf("client counters not updated: %#v", client)
	}
	if err := Clear("alice"); err != nil {
		t.Fatal(err)
	}
	rows, err = History("alice", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 {
		t.Fatalf("expected cleared history, got %d rows", len(rows))
	}
}

func TestAllowEnforceRejectsNewIPOverLimit(t *testing.T) {
	initIPMonitorTestDB(t)
	if err := database.GetDB().Create(&model.Client{
		Enable:      true,
		Name:        "alice",
		LimitIP:     1,
		IPLimitMode: ModeEnforce,
		Inbounds:    []byte("[]"),
		Links:       []byte("[]"),
	}).Error; err != nil {
		t.Fatal(err)
	}
	warmUpIPMonitorForTest(t)
	Record("alice", "198.51.100.10")
	if !Allow("alice", "198.51.100.10") {
		t.Fatal("known IP should be allowed")
	}
	if Allow("alice", "198.51.100.11") {
		t.Fatal("new IP over limit should be rejected")
	}
	if err := Flush(); err != nil {
		t.Fatal(err)
	}
	if Allow("alice", "198.51.100.11") {
		t.Fatal("new IP over limit should still be rejected after pending flush")
	}
}

func TestAllowEnforceRejectPublishesSecurityEventWithoutRawIP(t *testing.T) {
	initIPMonitorTestDB(t)
	if err := database.GetDB().Create(&model.Client{
		Enable:      true,
		Name:        "alice",
		LimitIP:     1,
		IPLimitMode: ModeEnforce,
		Inbounds:    []byte("[]"),
		Links:       []byte("[]"),
	}).Error; err != nil {
		t.Fatal(err)
	}
	warmUpIPMonitorForTest(t)
	ch := make(chan realtime.Event, 1)
	unregister := realtime.Register(&realtime.ClientHandle{
		User:   "admin",
		Scope:  realtime.ScopeAdmin,
		SendCh: ch,
	})
	defer unregister()

	Record("alice", "198.51.100.10")
	const rejectedIP = "198.51.100.11"
	if Allow("alice", rejectedIP) {
		t.Fatal("new IP over limit should be rejected")
	}

	select {
	case event := <-ch:
		if event.Type != realtime.TopicSecurityEvent {
			t.Fatalf("unexpected event type: %s", event.Type)
		}
		payload, ok := event.Payload.(map[string]any)
		if !ok {
			t.Fatalf("unexpected payload: %#v", event.Payload)
		}
		if payload["kind"] != "ip_enforced_reject" || payload["client"] != "alice" {
			t.Fatalf("unexpected payload values: %#v", payload)
		}
		if payload["ipHash"] == "" || payload["ipHash"] == rejectedIP {
			t.Fatalf("raw IP leaked or hash missing: %#v", payload)
		}
	case <-time.After(time.Second):
		t.Fatal("security_event was not published")
	}
}

func TestAllowEnforceRejectSecurityEventDebounced(t *testing.T) {
	initIPMonitorTestDB(t)
	if err := database.GetDB().Create(&model.Client{
		Enable:      true,
		Name:        "alice",
		LimitIP:     1,
		IPLimitMode: ModeEnforce,
		Inbounds:    []byte("[]"),
		Links:       []byte("[]"),
	}).Error; err != nil {
		t.Fatal(err)
	}
	warmUpIPMonitorForTest(t)
	ch := make(chan realtime.Event, 100)
	unregister := realtime.Register(&realtime.ClientHandle{
		User:   "admin",
		Scope:  realtime.ScopeAdmin,
		SendCh: ch,
	})
	defer unregister()

	Record("alice", "198.51.100.10")
	for i := 0; i < 100; i++ {
		if Allow("alice", "198.51.100.11") {
			t.Fatal("new IP over limit should be rejected")
		}
	}

	got := 0
	for {
		select {
		case event := <-ch:
			if event.Type == realtime.TopicSecurityEvent {
				got++
			}
		default:
			if got != 1 {
				t.Fatalf("expected exactly one debounced security_event, got %d", got)
			}
			return
		}
	}
}

func TestRecordFlushStoresHashedIPAndMasksHistoryByDefault(t *testing.T) {
	initIPMonitorTestDB(t)
	if err := database.GetDB().Create(&model.Client{
		Enable:      true,
		Name:        "alice",
		IPLimitMode: ModeMonitor,
		Inbounds:    []byte("[]"),
		Links:       []byte("[]"),
	}).Error; err != nil {
		t.Fatal(err)
	}

	const rawIP = "198.51.100.10"
	Record("alice", rawIP)
	if err := Flush(); err != nil {
		t.Fatal(err)
	}

	var row model.ClientIP
	if err := database.GetDB().Where("client_name = ?", "alice").First(&row).Error; err != nil {
		t.Fatal(err)
	}
	if row.IP == rawIP {
		t.Fatal("raw IP was stored in legacy ip column")
	}
	if row.IP != "" || row.IPHash == "" {
		t.Fatalf("expected legacy ip column empty and ip_hash populated for new rows: %#v", row)
	}
	if row.IPDisplay != nil {
		t.Fatalf("ip_display must stay NULL while ipShowRaw=false: %#v", row.IPDisplay)
	}

	history, err := History("alice", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 1 {
		t.Fatalf("expected one history row, got %d", len(history))
	}
	if history[0].IP == rawIP || history[0].IPHash != "" || history[0].IPDisplay != nil {
		t.Fatalf("history leaked raw/hash internals: %#v", history[0])
	}
	if !strings.HasPrefix(history[0].IP, "masked:") {
		t.Fatalf("history did not return a masked IP: %#v", history[0])
	}
}

func TestRecordFlushStoresRawDisplayOnlyWhenEnabled(t *testing.T) {
	initIPMonitorTestDB(t)
	if err := database.GetDB().Create(&model.Setting{Key: "ipShowRaw", Value: "true"}).Error; err != nil {
		t.Fatal(err)
	}
	ipPrivacySettings.Lock()
	ipPrivacySettings.expiresAt = time.Time{}
	ipPrivacySettings.Unlock()
	if err := database.GetDB().Create(&model.Client{
		Enable:      true,
		Name:        "alice",
		IPLimitMode: ModeMonitor,
		Inbounds:    []byte("[]"),
		Links:       []byte("[]"),
	}).Error; err != nil {
		t.Fatal(err)
	}

	const rawIP = "198.51.100.10"
	Record("alice", rawIP)
	if err := Flush(); err != nil {
		t.Fatal(err)
	}

	var row model.ClientIP
	if err := database.GetDB().Where("client_name = ?", "alice").First(&row).Error; err != nil {
		t.Fatal(err)
	}
	if row.IPDisplay == nil || *row.IPDisplay != rawIP {
		t.Fatalf("raw display was not stored when ipShowRaw=true: %#v", row)
	}
	if row.IP != "" || row.IPHash == "" {
		t.Fatalf("legacy ip column should be empty and ip_hash populated: %#v", row)
	}

	history, err := History("alice", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 1 || history[0].IP != rawIP {
		t.Fatalf("history should return raw display when explicitly enabled: %#v", history)
	}
	if history[0].IPDisplay != nil || history[0].IPHash != "" {
		t.Fatalf("history leaked storage internals: %#v", history[0])
	}
}

// TestFlushBatchUpsertCountsAndPreservesFirstSeen covers the O1 batched upsert:
// many (client, ip) pairs are written in one flush with correct per-client
// counts/last_online, and re-recording an existing ip updates last_seen while
// preserving first_seen and never creating a duplicate row.
func TestFlushBatchUpsertCountsAndPreservesFirstSeen(t *testing.T) {
	initIPMonitorTestDB(t)
	db := database.GetDB()
	for _, name := range []string{"alice", "bob"} {
		if err := db.Create(&model.Client{Enable: true, Name: name, IPLimitMode: ModeMonitor, Inbounds: []byte("[]"), Links: []byte("[]")}).Error; err != nil {
			t.Fatal(err)
		}
	}

	Record("alice", "198.51.100.1")
	Record("alice", "198.51.100.2")
	Record("bob", "203.0.113.9")
	if err := Flush(); err != nil {
		t.Fatal(err)
	}

	clientIPCount := func(name string) int64 {
		t.Helper()
		var n int64
		if err := db.Model(model.ClientIP{}).Where("client_name = ?", name).Count(&n).Error; err != nil {
			t.Fatal(err)
		}
		return n
	}
	if got := clientIPCount("alice"); got != 2 {
		t.Fatalf("alice client_ips rows = %d, want 2", got)
	}
	if got := clientIPCount("bob"); got != 1 {
		t.Fatalf("bob client_ips rows = %d, want 1", got)
	}

	var alice model.Client
	if err := db.Where("name = ?", "alice").First(&alice).Error; err != nil {
		t.Fatal(err)
	}
	if alice.LastIPCount != 2 {
		t.Fatalf("alice last_ip_count = %d, want 2", alice.LastIPCount)
	}
	if alice.LastOnline == 0 {
		t.Fatal("alice last_online was not set by the batch update")
	}

	// Pin a known-old first_seen on one row, then re-record the SAME ip: the
	// upsert must refresh last_seen, preserve first_seen, and not duplicate.
	hash1, err := hashIP("198.51.100.1")
	if err != nil {
		t.Fatal(err)
	}
	const oldFirstSeen = int64(1000)
	if err := db.Model(model.ClientIP{}).Where("client_name = ? AND ip_hash = ?", "alice", hash1).
		Updates(map[string]interface{}{"first_seen": oldFirstSeen, "last_seen": oldFirstSeen}).Error; err != nil {
		t.Fatal(err)
	}

	Record("alice", "198.51.100.1")
	if err := Flush(); err != nil {
		t.Fatal(err)
	}

	if got := clientIPCount("alice"); got != 2 {
		t.Fatalf("upsert created a duplicate row: alice has %d rows, want 2", got)
	}
	var row model.ClientIP
	if err := db.Where("client_name = ? AND ip_hash = ?", "alice", hash1).First(&row).Error; err != nil {
		t.Fatal(err)
	}
	if row.FirstSeen != oldFirstSeen {
		t.Fatalf("first_seen not preserved on upsert: got %d want %d", row.FirstSeen, oldFirstSeen)
	}
	if row.LastSeen <= oldFirstSeen {
		t.Fatalf("last_seen not refreshed on upsert: got %d (want > %d)", row.LastSeen, oldFirstSeen)
	}
}

func TestWarmUpLoadsActiveEnforceClients(t *testing.T) {
	initIPMonitorTestDB(t)
	if err := database.GetDB().Create(&model.Client{
		Enable:      true,
		Name:        "alice",
		LimitIP:     1,
		IPLimitMode: ModeEnforce,
		Inbounds:    []byte("[]"),
		Links:       []byte("[]"),
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Create(&model.ClientIP{
		ClientName: "alice",
		IP:         "198.51.100.10",
		FirstSeen:  1,
		LastSeen:   1,
	}).Error; err != nil {
		t.Fatal(err)
	}
	warmUpIPMonitorForTest(t)
	queryCounter := &countingGormLogger{}
	database.GetDB().Config.Logger = queryCounter

	if !Allow("alice", "198.51.100.10") {
		t.Fatal("known IP should be allowed")
	}
	for i := 0; i < 100; i++ {
		if !Allow("alice", "198.51.100.10") {
			t.Fatal("known IP should stay allowed")
		}
		if Allow("alice", "198.51.100.11") {
			t.Fatal("new IP over limit should be rejected")
		}
	}
	if got := queryCounter.Count(); got != 0 {
		t.Fatalf("expected warm cache to avoid database queries, got %d", got)
	}
}

func TestAllowEnforceCacheMissRefreshesBeforeDecision(t *testing.T) {
	initIPMonitorTestDB(t)
	if err := database.GetDB().Create(&model.Client{
		Enable:      true,
		Name:        "alice",
		LimitIP:     1,
		IPLimitMode: ModeEnforce,
		Inbounds:    []byte("[]"),
		Links:       []byte("[]"),
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Create(&model.ClientIP{
		ClientName: "alice",
		IP:         "198.51.100.10",
		FirstSeen:  1,
		LastSeen:   1,
	}).Error; err != nil {
		t.Fatal(err)
	}

	if Allow("alice", "198.51.100.11") {
		t.Fatal("first forbidden IP after a cache miss must be rejected")
	}
}

func TestAllowConcurrentCacheMissPerformsOneRefresh(t *testing.T) {
	initIPMonitorTestDB(t)
	if err := database.GetDB().Create(&model.Client{Enable: true, Name: "alice", LimitIP: 1, IPLimitMode: ModeEnforce, Inbounds: []byte("[]"), Links: []byte("[]")}).Error; err != nil {
		t.Fatal(err)
	}
	ipHashSalt.Lock()
	ipHashSalt.value = []byte("test-salt")
	pending.Lock()
	ipHashSalt.generation = pending.generation
	pending.Unlock()
	ipHashSalt.Unlock()
	queryCounter := &countingGormLogger{}
	database.GetDB().Config.Logger = queryCounter
	oldLoad := loadCacheEntryForAllow
	started := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	loadCacheEntryForAllow = func(ctx context.Context, clientName string, now time.Time) (allowCacheEntry, bool) {
		once.Do(func() { close(started) })
		<-release
		return oldLoad(ctx, clientName, now)
	}
	t.Cleanup(func() { loadCacheEntryForAllow = oldLoad })

	const workers = 32
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			Allow("alice", "198.51.100.10")
		}()
	}
	close(start)
	<-started
	time.Sleep(50 * time.Millisecond)
	close(release)
	wg.Wait()
	if got := queryCounter.Count(); got != 2 {
		t.Fatalf("concurrent miss ran %d DB queries, want one two-query refresh", got)
	}
}

func TestObserveAndAllowConcurrentFirstIPsReservesLimit(t *testing.T) {
	initIPMonitorTestDB(t)
	if err := database.GetDB().Create(&model.Client{
		Enable: true, Name: "alice", LimitIP: 1, IPLimitMode: ModeEnforce,
		Inbounds: []byte("[]"), Links: []byte("[]"),
	}).Error; err != nil {
		t.Fatal(err)
	}
	warmUpIPMonitorForTest(t)

	const workers = 32
	start := make(chan struct{})
	var allowed atomic.Int64
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			if ObserveAndAllow("alice", fmt.Sprintf("198.51.100.%d", i+1)) {
				allowed.Add(1)
			}
		}(i)
	}
	close(start)
	wg.Wait()

	if got := allowed.Load(); got != 1 {
		t.Fatalf("concurrent first observations allowed %d IPs; want exactly 1", got)
	}
	pending.Lock()
	reserved := len(pending.byClient["alice"])
	pending.Unlock()
	if reserved != 1 {
		t.Fatalf("concurrent first observations reserved %d IPs; want 1", reserved)
	}
}

func TestAllowRefreshCannotPublishAcrossInvalidation(t *testing.T) {
	initIPMonitorTestDB(t)
	ipHashSalt.Lock()
	ipHashSalt.value = []byte("test-salt")
	pending.Lock()
	ipHashSalt.generation = pending.generation
	pending.Unlock()
	ipHashSalt.Unlock()
	oldLoad := loadCacheEntryForAllow
	started := make(chan struct{})
	release := make(chan struct{})
	loadCacheEntryForAllow = func(context.Context, string, time.Time) (allowCacheEntry, bool) {
		close(started)
		<-release
		return allowCacheEntry{mode: ModeMonitor, limit: 1, ips: map[string]struct{}{}, expiresAt: time.Now().Add(time.Minute)}, true
	}
	t.Cleanup(func() { loadCacheEntryForAllow = oldLoad })

	result := make(chan bool, 1)
	go func() { result <- Allow("alice", "198.51.100.10") }()
	<-started
	invalidateCache("alice")
	close(release)

	if <-result {
		t.Fatal("refresh result from before invalidation was used for admission")
	}
	allowCache.Lock()
	_, published := allowCache.byClient["alice"]
	allowCache.Unlock()
	if published {
		t.Fatal("refresh result from before invalidation repopulated cache")
	}
}

func TestAllowContextCancellationUnblocksRefreshWaiter(t *testing.T) {
	initIPMonitorTestDB(t)
	ipHashSalt.Lock()
	ipHashSalt.value = []byte("test-salt")
	pending.Lock()
	ipHashSalt.generation = pending.generation
	pending.Unlock()
	ipHashSalt.Unlock()
	oldLoad := loadCacheEntryForAllow
	started := make(chan struct{})
	release := make(chan struct{})
	loadCacheEntryForAllow = func(context.Context, string, time.Time) (allowCacheEntry, bool) {
		close(started)
		<-release
		return allowCacheEntry{mode: ModeMonitor, expiresAt: time.Now().Add(time.Minute)}, true
	}
	t.Cleanup(func() { loadCacheEntryForAllow = oldLoad })

	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan bool, 1)
	go func() { result <- AllowContext(ctx, "alice", "198.51.100.10") }()
	<-started
	cancel()
	select {
	case allowed := <-result:
		if allowed {
			t.Fatal("canceled refresh waiter was allowed")
		}
	case <-time.After(time.Second):
		t.Fatal("canceled refresh waiter remained blocked")
	}
	close(release)
}

func TestDetachedSnapshotKeepsAdmissionReservationUntilAck(t *testing.T) {
	initIPMonitorTestDB(t)
	if err := database.GetDB().Create(&model.Client{
		Enable: true, Name: "alice", LimitIP: 1, IPLimitMode: ModeEnforce,
		Inbounds: []byte("[]"), Links: []byte("[]"),
	}).Error; err != nil {
		t.Fatal(err)
	}
	warmUpIPMonitorForTest(t)
	if !ObserveAndAllow("alice", "198.51.100.10") {
		t.Fatal("first IP was rejected")
	}
	snapshot := SnapshotPending()
	invalidateCache("alice")
	if ObserveAndAllow("alice", "198.51.100.11") {
		t.Fatal("detached uncommitted snapshot released the reserved IP slot")
	}
	snapshot.Requeue()
}

func TestAllowDoesNotUseSingleflightResultAfterInvalidation(t *testing.T) {
	initIPMonitorTestDB(t)
	ipHashSalt.Lock()
	ipHashSalt.value = []byte("test-salt")
	pending.Lock()
	ipHashSalt.generation = pending.generation
	pending.Unlock()
	ipHashSalt.Unlock()
	oldLoad := loadCacheEntryForAllow
	started := make(chan struct{})
	release := make(chan struct{})
	loadCacheEntryForAllow = func(context.Context, string, time.Time) (allowCacheEntry, bool) {
		close(started)
		<-release
		return allowCacheEntry{mode: ModeMonitor, limit: 1, ips: map[string]struct{}{}, expiresAt: time.Now().Add(time.Minute)}, true
	}
	t.Cleanup(func() { loadCacheEntryForAllow = oldLoad })

	first := make(chan bool, 1)
	go func() { first <- Allow("alice", "198.51.100.10") }()
	<-started
	allowCache.Lock()
	allowCache.revision++
	close(release)
	time.Sleep(20 * time.Millisecond)
	delete(allowCache.byClient, "alice")
	allowCache.Unlock()
	if <-first {
		t.Fatal("caller used refresh result after its revision was invalidated")
	}
}

func TestAllowRefreshErrorKeepsStaleEnforcePolicy(t *testing.T) {
	initIPMonitorTestDB(t)
	if err := database.GetDB().Create(&model.Client{Enable: true, Name: "alice", LimitIP: 1, IPLimitMode: ModeEnforce, Inbounds: []byte("[]"), Links: []byte("[]")}).Error; err != nil {
		t.Fatal(err)
	}
	Record("alice", "198.51.100.10")
	warmUpIPMonitorForTest(t)
	allowCache.Lock()
	entry := allowCache.byClient["alice"]
	entry.expiresAt = time.Now().Add(-time.Second)
	allowCache.byClient["alice"] = entry
	allowCache.Unlock()
	if err := database.GetDB().Migrator().DropTable(&model.ClientIP{}); err != nil {
		t.Fatal(err)
	}

	if Allow("alice", "198.51.100.11") {
		t.Fatal("refresh failure discarded stale enforce policy")
	}
	allowCache.Lock()
	_, ok := allowCache.byClient["alice"]
	allowCache.Unlock()
	if !ok {
		t.Fatal("refresh failure removed last known policy")
	}
}

func TestAllowColdStartDatabaseFailureFailsClosed(t *testing.T) {
	initIPMonitorTestDB(t)
	closeIPMonitorTestDB(database.GetDB())
	if Allow("unknown", "198.51.100.10") {
		t.Fatal("cold start without policy database or hash salt must fail closed")
	}
}

func TestAllowMonitorModeDoesNotBlockOnRefresh(t *testing.T) {
	initIPMonitorTestDB(t)
	if err := database.GetDB().Create(&model.Client{Enable: true, Name: "alice", LimitIP: 1, IPLimitMode: ModeMonitor, Inbounds: []byte("[]"), Links: []byte("[]")}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Create(&model.ClientIP{ClientName: "alice", IP: "198.51.100.10", FirstSeen: 1, LastSeen: 1}).Error; err != nil {
		t.Fatal(err)
	}
	warmUpIPMonitorForTest(t)
	allowCache.Lock()
	entry := allowCache.byClient["alice"]
	entry.expiresAt = time.Now().Add(-time.Second)
	allowCache.byClient["alice"] = entry
	allowCache.Unlock()
	if err := database.GetDB().Migrator().DropTable(&model.ClientIP{}); err != nil {
		t.Fatal(err)
	}

	if !Allow("alice", "198.51.100.11") {
		t.Fatal("monitor mode blocked traffic after refresh failure")
	}
}

func TestAllowCacheConcurrent10K(t *testing.T) {
	initIPMonitorTestDB(t)
	if err := database.GetDB().Create(&model.Client{
		Enable:      true,
		Name:        "alice",
		LimitIP:     2,
		IPLimitMode: ModeEnforce,
		Inbounds:    []byte("[]"),
		Links:       []byte("[]"),
	}).Error; err != nil {
		t.Fatal(err)
	}
	warmUpIPMonitorForTest(t)
	Record("alice", "198.51.100.10")
	if !Allow("alice", "198.51.100.10") {
		t.Fatal("known pending IP should be allowed")
	}
	queryCounter := &countingGormLogger{}
	database.GetDB().Config.Logger = queryCounter

	const total = 10000
	const workers = 32
	var failed atomic.Int64
	var wg sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		wg.Add(1)
		go func(offset int) {
			defer wg.Done()
			for i := offset; i < total; i += workers {
				if !Allow("alice", "198.51.100.10") {
					failed.Add(1)
				}
			}
		}(worker)
	}
	wg.Wait()
	if failed.Load() != 0 {
		t.Fatalf("%d concurrent Allow calls rejected a known IP", failed.Load())
	}
	if got := queryCounter.Count(); got != 0 {
		t.Fatalf("expected warmed cache to avoid database queries, got %d", got)
	}
}

func TestResetCachesClearsSaltAndAllowState(t *testing.T) {
	pending.Lock()
	pending.byClient = map[string]map[string]pendingIP{
		"alice": {"hash": {lastSeen: 1}},
	}
	pending.Unlock()
	allowCache.Lock()
	allowCache.byClient = map[string]allowCacheEntry{
		"alice": {limit: 1, mode: ModeEnforce, ips: map[string]struct{}{"hash": {}}, expiresAt: time.Now().Add(time.Minute)},
	}
	allowCache.Unlock()
	securityEvents.Lock()
	securityEvents.lastEmittedAt = map[string]time.Time{"alice|reject": time.Now()}
	securityEvents.Unlock()
	ipHashSalt.Lock()
	ipHashSalt.value = []byte("salt")
	ipHashSalt.Unlock()
	ipPrivacySettings.Lock()
	ipPrivacySettings.showRaw = true
	ipPrivacySettings.expiresAt = time.Now().Add(time.Minute)
	ipPrivacySettings.Unlock()

	ResetCaches()

	pending.Lock()
	pendingCount := len(pending.byClient)
	pending.Unlock()
	allowCache.Lock()
	allowCount := len(allowCache.byClient)
	allowCache.Unlock()
	securityEvents.Lock()
	securityCount := len(securityEvents.lastEmittedAt)
	securityEvents.Unlock()
	ipHashSalt.Lock()
	saltLen := len(ipHashSalt.value)
	ipHashSalt.Unlock()
	ipPrivacySettings.Lock()
	showRaw := ipPrivacySettings.showRaw
	privacyExpired := ipPrivacySettings.expiresAt.IsZero()
	ipPrivacySettings.Unlock()

	if pendingCount != 0 || allowCount != 0 || securityCount != 0 || saltLen != 0 || showRaw || !privacyExpired {
		t.Fatalf("reset did not clear caches: pending=%d allow=%d security=%d salt=%d showRaw=%v privacyExpired=%v",
			pendingCount, allowCount, securityCount, saltLen, showRaw, privacyExpired)
	}
}

func warmUpIPMonitorForTest(t *testing.T) {
	t.Helper()
	if err := WarmUp(); err != nil {
		t.Fatal(err)
	}
}

func TestSnapshotPendingAtomicallyMovesToInFlight(t *testing.T) {
	initIPMonitorTestDB(t)
	Record("alice", "198.51.100.10")
	snapshot := SnapshotPending()
	defer snapshot.Requeue()

	pending.Lock()
	_, pendingVisible := pending.byClient["alice"]
	_, inFlightVisible := pending.inFlight[snapshot]
	pending.Unlock()
	if pendingVisible || !inFlightVisible {
		t.Fatalf("snapshot transition was not atomic: pending=%v inFlight=%v", pendingVisible, inFlightVisible)
	}
}

func TestClearDropsPendingAndDetachedSnapshot(t *testing.T) {
	initIPMonitorTestDB(t)
	if err := database.GetDB().Create(&model.Client{
		Enable: true, Name: "alice", IPLimitMode: ModeMonitor,
		Inbounds: []byte("[]"), Links: []byte("[]"),
	}).Error; err != nil {
		t.Fatal(err)
	}
	Record("alice", "198.51.100.10")
	snapshot := SnapshotPending()
	Record("alice", "198.51.100.11")
	if err := Clear("alice"); err != nil {
		t.Fatal(err)
	}
	snapshot.Requeue()
	if err := Flush(); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := database.GetDB().Model(&model.ClientIP{}).Where("client_name = ?", "alice").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("cleared observations were resurrected: %d rows", count)
	}
}

func TestResetInvalidatesDetachedSnapshot(t *testing.T) {
	initIPMonitorTestDB(t)
	Record("alice", "198.51.100.10")
	snapshot := SnapshotPending()
	ResetCaches()
	snapshot.Requeue()
	pending.Lock()
	pendingCount := len(pending.byClient)
	inFlightCount := len(pending.inFlight)
	pending.Unlock()
	if pendingCount != 0 || inFlightCount != 0 {
		t.Fatalf("reset snapshot resurrected state: pending=%d inFlight=%d", pendingCount, inFlightCount)
	}
}

func TestPendingSnapshotAckRequeueOneTerminalOutcome(t *testing.T) {
	initIPMonitorTestDB(t)
	Record("alice", "198.51.100.10")
	snapshot := SnapshotPending()
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); <-start; snapshot.Ack() }()
	go func() { defer wg.Done(); <-start; snapshot.Requeue() }()
	close(start)
	wg.Wait()

	pending.Lock()
	_, tracked := pending.inFlight[snapshot]
	pendingRows := len(pending.byClient["alice"])
	pending.Unlock()
	if tracked || pendingRows > 1 {
		t.Fatalf("snapshot had multiple terminal effects: tracked=%v pendingRows=%d", tracked, pendingRows)
	}
}

func TestAllowLoaderPanicFailsClosed(t *testing.T) {
	initIPMonitorTestDB(t)
	oldLoad := loadCacheEntryForAllow
	loadCacheEntryForAllow = func(context.Context, string, time.Time) (allowCacheEntry, bool) {
		panic("loader failure")
	}
	t.Cleanup(func() { loadCacheEntryForAllow = oldLoad })
	if Allow("alice", "198.51.100.10") {
		t.Fatal("loader panic allowed admission")
	}
}

func TestPackageFlushToAcknowledgesAfterCommit(t *testing.T) {
	initIPMonitorTestDB(t)
	if err := database.GetDB().Create(&model.Client{
		Enable: true, Name: "alice", IPLimitMode: ModeMonitor,
		Inbounds: []byte("[]"), Links: []byte("[]"),
	}).Error; err != nil {
		t.Fatal(err)
	}
	Record("alice", "198.51.100.10")
	tx := database.GetDB().Begin()
	if err := FlushTo(tx); err != nil {
		t.Fatal(err)
	}
	pending.Lock()
	inFlightBeforeCommit := len(pending.inFlight)
	pending.Unlock()
	if inFlightBeforeCommit != 1 {
		t.Fatalf("snapshot acknowledged before commit: inFlight=%d", inFlightBeforeCommit)
	}
	if err := tx.Commit().Error; err != nil {
		t.Fatal(err)
	}
	pending.Lock()
	inFlightAfterCommit := len(pending.inFlight)
	pending.Unlock()
	if inFlightAfterCommit != 0 {
		t.Fatalf("snapshot remained in flight after commit: %d", inFlightAfterCommit)
	}
}

func TestWarmUpCannotPublishAcrossInvalidation(t *testing.T) {
	initIPMonitorTestDB(t)
	oldLoad := loadPolicyEntriesForWarmUp
	started := make(chan struct{})
	release := make(chan struct{})
	loadPolicyEntriesForWarmUp = func(context.Context, *gorm.DB, time.Time, uint64) (map[string]allowCacheEntry, error) {
		close(started)
		<-release
		return map[string]allowCacheEntry{
			"alice": {mode: ModeMonitor, ips: map[string]struct{}{}, expiresAt: time.Now().Add(time.Minute)},
		}, nil
	}
	t.Cleanup(func() { loadPolicyEntriesForWarmUp = oldLoad })

	done := make(chan error, 1)
	go func() { done <- WarmUp() }()
	<-started
	InvalidateAllCache()
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	allowCache.Lock()
	_, published := allowCache.byClient["alice"]
	allowCache.Unlock()
	if published {
		t.Fatal("WarmUp published entries loaded before invalidation")
	}
}

func TestPackageFlushToRollbackRequeues(t *testing.T) {
	initIPMonitorTestDB(t)
	if err := database.GetDB().Create(&model.Client{
		Enable: true, Name: "alice", IPLimitMode: ModeMonitor,
		Inbounds: []byte("[]"), Links: []byte("[]"),
	}).Error; err != nil {
		t.Fatal(err)
	}
	Record("alice", "198.51.100.10")
	tx := database.GetDB().Begin()
	if err := FlushTo(tx); err != nil {
		t.Fatal(err)
	}
	if err := tx.Rollback().Error; err != nil {
		t.Fatal(err)
	}
	pending.Lock()
	queued := len(pending.byClient["alice"])
	inFlight := len(pending.inFlight)
	pending.Unlock()
	if queued != 1 || inFlight != 0 {
		t.Fatalf("rollback lifecycle mismatch: queued=%d inFlight=%d", queued, inFlight)
	}
}

type countingGormLogger struct {
	count atomic.Int64
}

func (l *countingGormLogger) LogMode(logger.LogLevel) logger.Interface {
	return l
}

func (l *countingGormLogger) Info(context.Context, string, ...interface{}) {
}

func (l *countingGormLogger) Warn(context.Context, string, ...interface{}) {
}

func (l *countingGormLogger) Error(context.Context, string, ...interface{}) {
}

func (l *countingGormLogger) Trace(context.Context, time.Time, func() (string, int64), error) {
	l.count.Add(1)
}

func (l *countingGormLogger) Count() int64 {
	return l.count.Load()
}
