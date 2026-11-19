package sub

import (
	"testing"
	"time"
)

func TestSubscriptionOutputCacheHonorsPeerAccessDeadline(t *testing.T) {
	ClearSubscriptionOutputCache()
	t.Cleanup(ClearSubscriptionOutputCache)
	now := time.Now()
	deadline := now.Add(time.Second)
	subscriptionCacheSet("peer", "link", nil, now, subscriptionCacheRevision(), deadline)
	if _, _, ok := subscriptionCacheGet("peer", deadline.Add(-time.Nanosecond)); !ok {
		t.Fatal("cache expired before the peer lease")
	}
	if _, _, ok := subscriptionCacheGet("peer", deadline); ok {
		t.Fatal("cache retained a link after the peer lease expired")
	}
}

func TestInvalidateClientSubscriptionOutputCacheIsScoped(t *testing.T) {
	ClearSubscriptionOutputCache()
	t.Cleanup(ClearSubscriptionOutputCache)
	now := time.Now()
	for _, key := range []string{"json:default:client-a", "clash:client-a", "base:client-a", "json:default:client-b"} {
		subscriptionCacheSet(key, key, nil, now, subscriptionCacheRevision(), time.Time{})
	}
	InvalidateClientSubscriptionOutputCache([]string{"client-a"})
	for _, key := range []string{"json:default:client-a", "clash:client-a", "base:client-a"} {
		if _, _, ok := subscriptionCacheGet(key, now); ok {
			t.Errorf("affected client cache survived: %s", key)
		}
	}
	if body, _, ok := subscriptionCacheGet("json:default:client-b", now); !ok || body != "json:default:client-b" {
		t.Fatalf("unrelated client cache was invalidated: body=%q ok=%v", body, ok)
	}
}
