package cronjob

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
	"github.com/deposist/s-ui-x/ipmonitor"
	"gorm.io/gorm"
)

func initCronJobTestDB(t *testing.T) {
	t.Helper()
	tempDir, err := os.MkdirTemp("", "s-ui-cronjob-test-")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("SUI_DB_FOLDER", tempDir)
	closeCronJobDB(database.GetDB())
	if err := database.InitDB(filepath.Join(tempDir, "s-ui.db")); err != nil {
		removeCronJobTempDir(t, tempDir)
		if strings.Contains(err.Error(), "go-sqlite3 requires cgo") {
			t.Skip(err)
		}
		t.Fatal(err)
	}
	testDB := database.GetDB()
	t.Cleanup(func() {
		closeCronJobDB(testDB)
		ipmonitor.InvalidateAllCache()
		removeCronJobTempDir(t, tempDir)
	})
}

func closeCronJobDB(db *gorm.DB) {
	if db == nil {
		return
	}
	_ = db.Exec("PRAGMA wal_checkpoint(TRUNCATE)").Error
	if sqlDB, err := db.DB(); err == nil {
		_ = sqlDB.Close()
	}
}

func removeCronJobTempDir(t *testing.T, dir string) {
	t.Helper()
	var err error
	for i := 0; i < 20; i++ {
		err = os.RemoveAll(dir)
		if err == nil || os.IsNotExist(err) {
			return
		}
		time.Sleep(time.Duration(i+1) * 10 * time.Millisecond)
	}
	t.Errorf("remove cronjob temp dir %q: %v", dir, err)
}

func TestCronJobTestDBIsIsolatedBetweenInitializations(t *testing.T) {
	initCronJobTestDB(t)
	if err := database.GetDB().Create(&model.ClientIP{
		ClientName: "alice",
		IP:         "198.51.100.10",
		IPHash:     "hash",
		FirstSeen:  1,
		LastSeen:   1,
	}).Error; err != nil {
		t.Fatal(err)
	}

	initCronJobTestDB(t)
	if err := database.GetDB().Create(&model.ClientIP{
		ClientName: "alice",
		IP:         "198.51.100.10",
		IPHash:     "hash",
		FirstSeen:  1,
		LastSeen:   1,
	}).Error; err != nil {
		t.Fatalf("second isolated test DB rejected duplicate unique row: %v", err)
	}
}

func TestAuditGCJobPrunesAuditEventsAndClientIPs(t *testing.T) {
	initCronJobTestDB(t)
	now := time.Now()
	oldTime := now.Add(-31 * 24 * time.Hour).Unix()
	recentTime := now.Unix()
	if err := database.GetDB().Create(&[]model.AuditEvent{
		{DateTime: oldTime, Actor: "admin", Event: "old"},
		{DateTime: recentTime, Actor: "admin", Event: "recent"},
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Create(&[]model.ClientIP{
		{ClientName: "alice", IP: "198.51.100.10", IPHash: "hash-old", FirstSeen: oldTime, LastSeen: oldTime},
		{ClientName: "alice", IP: "198.51.100.11", IPHash: "hash-recent", FirstSeen: recentTime, LastSeen: recentTime},
	}).Error; err != nil {
		t.Fatal(err)
	}

	NewAuditGCJob().Run()

	var auditEvents []model.AuditEvent
	if err := database.GetDB().Order("event asc").Find(&auditEvents).Error; err != nil {
		t.Fatal(err)
	}
	if len(auditEvents) != 1 || auditEvents[0].Event != "recent" {
		t.Fatalf("unexpected audit events after GC: %#v", auditEvents)
	}
	var clientIPs []model.ClientIP
	if err := database.GetDB().Order("ip asc").Find(&clientIPs).Error; err != nil {
		t.Fatal(err)
	}
	if len(clientIPs) != 1 || clientIPs[0].IP != "198.51.100.11" {
		t.Fatalf("unexpected client IPs after GC: %#v", clientIPs)
	}
}

func TestPruneClientIPsInvalidatesIPMonitorAllowCache(t *testing.T) {
	initCronJobTestDB(t)
	ipmonitor.InvalidateAllCache()
	oldTime := time.Now().Add(-31 * 24 * time.Hour).Unix()
	if err := database.GetDB().Create(&model.Client{
		Enable:      true,
		Name:        "alice",
		LimitIP:     1,
		IPLimitMode: ipmonitor.ModeEnforce,
		Inbounds:    []byte("[]"),
		Links:       []byte("[]"),
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Create(&model.ClientIP{
		ClientName: "alice",
		IP:         "198.51.100.10",
		FirstSeen:  oldTime,
		LastSeen:   oldTime,
	}).Error; err != nil {
		t.Fatal(err)
	}

	if !ipmonitor.Allow("alice", "198.51.100.10") {
		t.Fatal("known IP should warm allow cache")
	}
	if err := pruneClientIPs(30); err != nil {
		t.Fatal(err)
	}
	if !ipmonitor.Allow("alice", "198.51.100.11") {
		t.Fatal("new IP should be allowed after pruned cached IP is removed")
	}
}
