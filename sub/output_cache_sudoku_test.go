package sub

import (
	"testing"
	"time"
)

func TestInvalidateClientSubscriptionOutputCacheIsScoped(t *testing.T) {
	ClearSubscriptionOutputCache()
	t.Cleanup(ClearSubscriptionOutputCache)
	now := time.Now()
	for _, key := range []string{"json:default:client-a", "clash:client-a", "base:client-a", "json:default:client-b"} {
		subscriptionCacheSet(key, key, nil, now)
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
