package ipmonitor

import (
	"testing"
	"time"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
)

func TestLoadCacheEntryFailsClosedOnClientIPReadError(t *testing.T) {
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
	if err := database.GetDB().Migrator().DropTable(&model.ClientIP{}); err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	if _, ok := loadCacheEntry("alice", now); ok {
		t.Fatal("ClientIP read error should fail closed and avoid caching an incomplete allow entry")
	}
	if refreshClient("alice", now) {
		t.Fatal("refreshClient should not cache an entry when loadCacheEntry fails")
	}
	if _, ok := cachedClient("alice", now); ok {
		t.Fatal("cache should not contain an entry after ClientIP read error")
	}
}
