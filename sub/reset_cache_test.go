package sub

import (
	"context"
	"testing"
	"time"

	"github.com/deposist/s-ui-x-extended/database"
)

func TestResetCachesClearsDatabaseBackedSubscriptionCaches(t *testing.T) {
	ClearSubscriptionOutputCache()
	clearSubDisplaySettingsCache()
	clearRateLimitSettingCache()
	t.Cleanup(func() {
		ClearSubscriptionOutputCache()
		clearSubDisplaySettingsCache()
		clearRateLimitSettingCache()
	})

	now := time.Now()
	const key = "json:default:client-a"
	subscriptionCacheSet(key, "cached", nil, now)
	subDisplaySettingsCache.Lock()
	subDisplaySettingsCache.value.title = "stale title"
	subDisplaySettingsCache.expiresAt = now.Add(time.Minute)
	subDisplaySettingsCache.Unlock()
	rateLimitSettingMu.Lock()
	rateLimitSetting.limit = 999
	rateLimitSetting.expiresAt = now.Add(time.Minute)
	rateLimitSettingMu.Unlock()

	if err := database.ResetCaches(context.Background()); err != nil {
		t.Fatalf("ResetCaches returned error: %v", err)
	}
	if _, _, ok := subscriptionCacheGet(key, now); ok {
		t.Fatal("subscription output cache entry survived ResetCaches")
	}
	subDisplaySettingsCache.Lock()
	displayTitle, displayExpiry := subDisplaySettingsCache.value.title, subDisplaySettingsCache.expiresAt
	subDisplaySettingsCache.Unlock()
	if displayTitle != "" || !displayExpiry.IsZero() {
		t.Fatalf("subscription display settings cache survived ResetCaches: title=%q expiry=%v", displayTitle, displayExpiry)
	}
	rateLimitSettingMu.Lock()
	limit, limitExpiry := rateLimitSetting.limit, rateLimitSetting.expiresAt
	rateLimitSettingMu.Unlock()
	if limit != 0 || !limitExpiry.IsZero() {
		t.Fatalf("subscription rate-limit setting cache survived ResetCaches: limit=%d expiry=%v", limit, limitExpiry)
	}
}
