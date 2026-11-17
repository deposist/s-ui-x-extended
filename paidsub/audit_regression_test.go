package paidsub

import (
	"errors"
	"testing"

	"gorm.io/gorm"
)

func TestRateLimiterReclaimsExpiredKeysAtCapacity(t *testing.T) {
	limiter := newRateLimiter(1, 60)
	limiter.maxKeys = 2
	for _, id := range []int64{1, 2} {
		if !limiter.allow(id, 100) {
			t.Fatal("initial request was refused")
		}
	}
	if limiter.allow(3, 100) {
		t.Fatal("saturated active limiter accepted a new key")
	}
	if !limiter.allow(3, 160) {
		t.Fatal("expired limiter entries permanently block new users")
	}
}

func TestUniqueClientNameRejectsDatabaseErrors(t *testing.T) {
	db := openTestDB(t)
	lookupErr := errors.New("client lookup unavailable")
	const callback = "test:fail-client-name-count"
	if err := db.Callback().Query().Before("gorm:query").Register(callback, func(tx *gorm.DB) {
		if tx.Statement.Table == "clients" {
			_ = tx.AddError(lookupErr)
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := db.Callback().Query().Remove(callback); err != nil {
			t.Error(err)
		}
	})
	if name, err := newBot().uniqueClientName(db, 123); !errors.Is(err, lookupErr) || name != "" {
		t.Fatalf("name = %q, err = %v; want the database error", name, err)
	}
}
