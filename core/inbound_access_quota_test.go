package core

import (
	"context"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/deposist/s-ui-x-extended/ipmonitor"
)

func TestPeerQuotaAndDelayedStartPersist(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SUI_DB_FOLDER", dir)
	if err := database.InitDB(filepath.Join(dir, "s-ui.db")); err != nil {
		t.Fatal(err)
	}
	db := database.GetDB()
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		peerChargeStatement.Lock()
		if peerChargeStatement.db == sqlDB && peerChargeStatement.stmt != nil {
			_ = peerChargeStatement.stmt.Close()
			peerChargeStatement.db, peerChargeStatement.stmt = nil, nil
		}
		peerChargeStatement.Unlock()
		_ = sqlDB.Close()
	})
	ctx := context.Background()
	pool := &peerConnectionPool{policy: ipmonitor.ConnectionPolicy{ClientID: 1, NodeUUID: "quota", Volume: 5}}
	var accepted atomic.Int32
	var workers sync.WaitGroup
	for range 12 {
		workers.Go(func() {
			if err := pool.charge(ctx, nil, 1); err == nil {
				accepted.Add(1)
			} else if err.Error() != "node traffic quota exhausted" {
				t.Error(err)
			}
		})
	}
	workers.Wait()
	var meter model.PeerTraffic
	if err := db.Where("node_uuid = ? AND client_id = ?", "quota", 1).First(&meter).Error; err != nil {
		t.Fatal(err)
	}
	if accepted.Load() != 5 || meter.Used != 5 {
		t.Fatalf("accepted=%d persisted=%d", accepted.Load(), meter.Used)
	}

	rule := ipmonitor.ConnectionPolicy{ClientID: 2, NodeUUID: "quota", DelayStart: true, Period: 60}
	delayed := &peerConnectionPool{policy: rule}
	start := time.Now().Unix()
	if err := delayed.chargeDelayedStart(ctx, sqlDB, start); err != nil {
		t.Fatal(err)
	}
	reconnected := &peerConnectionPool{policy: rule}
	if err := reconnected.chargeDelayedStart(ctx, sqlDB, start+59); err != nil {
		t.Fatal(err)
	}
	if err := reconnected.chargeDelayedStart(ctx, sqlDB, start+60); err == nil {
		t.Fatal("reconnection extended the persisted deadline")
	}
}
