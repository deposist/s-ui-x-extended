package ipmonitor

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"testing"
	"time"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
	"gorm.io/gorm"
)

func TestLegacyIPLookupDoesNotDeadlockDuringReset(t *testing.T) {
	if runResetRegressionChild(t) {
		return
	}

	initIPMonitorTestDB(t)
	db := database.GetDB()
	if err := db.Create(&model.Client{Enable: true, Name: "legacy", LimitIP: 1, IPLimitMode: ModeEnforce}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.ClientIP{ClientName: "legacy", IP: "198.51.100.1"}).Error; err != nil {
		t.Fatal(err)
	}
	resetDone := make(chan struct{})
	const callback = "test:queue-reset-after-legacy-ip-load"
	if err := db.Callback().Query().After("gorm:query").Register(callback, func(tx *gorm.DB) {
		if tx.Statement.Table != "client_ips" {
			return
		}
		go func() { ResetCaches(); close(resetDone) }()
		deadline := time.Now().Add(time.Second)
		for lifecycleGate.TryRLock() {
			lifecycleGate.RUnlock()
			if time.Now().After(deadline) {
				t.Error("cache reset did not queue behind the lookup")
				return
			}
			runtime.Gosched()
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := db.Callback().Query().Remove(callback); err != nil {
			t.Error(err)
		}
	})
	if !ObserveAndAllow("legacy", "198.51.100.1") {
		t.Fatal("existing legacy IP was refused")
	}
	select {
	case <-resetDone:
	case <-time.After(time.Second):
		t.Fatal("cache reset did not finish after the lookup")
	}
}

func TestRepeatedSnapshotFlushDoesNotBlockReset(t *testing.T) {
	if runResetRegressionChild(t) {
		return
	}
	initIPMonitorTestDB(t)
	db := database.GetDB()
	if err := db.Create(&model.Client{Enable: true, Name: "snapshot"}).Error; err != nil {
		t.Fatal(err)
	}
	Record("snapshot", "198.51.100.1")
	snapshot := SnapshotPending()
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	t.Cleanup(func() {
		if err := tx.Rollback().Error; err != nil {
			t.Error(err)
		}
		snapshot.Requeue()
	})
	if err := snapshot.FlushTo(tx); err != nil {
		t.Fatal(err)
	}
	resetDone := make(chan struct{})
	go func() { ResetCaches(); close(resetDone) }()
	deadline := time.Now().Add(time.Second)
	for lifecycleGate.TryRLock() {
		lifecycleGate.RUnlock()
		if time.Now().After(deadline) {
			t.Fatal("cache reset did not queue behind the snapshot")
		}
		runtime.Gosched()
	}
	if err := snapshot.FlushTo(tx); err == nil {
		t.Fatal("flushing the same snapshot twice did not return an error")
	}
	snapshot.Requeue()
	select {
	case <-resetDone:
	case <-time.After(time.Second):
		t.Fatal("cache reset did not finish after releasing the snapshot")
	}
}

func runResetRegressionChild(t *testing.T) bool {
	t.Helper()
	const childEnv = "SUI_IPMONITOR_RESET_REGRESSION_CHILD"
	if os.Getenv(childEnv) == t.Name() {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^"+t.Name()+"$", "-test.count=1")
	cmd.Env = append(os.Environ(), childEnv+"="+t.Name())
	output, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		t.Fatalf("operation and cache reset deadlocked: %s", output)
	}
	if err != nil {
		t.Fatalf("reset regression failed: %v\n%s", err, output)
	}
	return true
}
