package sub

import (
	"strconv"
	"testing"
	"time"
)

func TestSubscriptionOutputCacheEvictsOnlyPartOfFullCache(t *testing.T) {
	ClearSubscriptionOutputCache()
	t.Cleanup(ClearSubscriptionOutputCache)
	now := time.Now()
	revision := subscriptionCacheRevision()
	for i := range subscriptionOutputCacheMaxEntries {
		expires := now.Add(time.Duration(i+1) * time.Millisecond)
		subscriptionCacheSet("base:"+strconv.Itoa(i), "body", nil, now, revision, expires)
	}
	subscriptionCacheSet("base:new", "new", nil, now, revision, time.Time{})
	if got, _, ok := subscriptionCacheGet("base:new", now); !ok || got != "new" {
		t.Fatal("new entry was not cached")
	}
	if _, _, ok := subscriptionCacheGet("base:0", now); ok {
		t.Fatal("earliest expiring entry was retained")
	}
	if _, _, ok := subscriptionCacheGet("base:"+strconv.Itoa(subscriptionOutputCacheMaxEntries-1), now); !ok {
		t.Fatal("a later entry was evicted")
	}
}
