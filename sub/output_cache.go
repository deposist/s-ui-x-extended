package sub

import (
	"sync"
	"time"

	"github.com/deposist/s-ui-x-extended/service"
)

const subscriptionOutputCacheMaxEntries = 4096

var subscriptionOutputCache = newSubscriptionOutputCache(45 * time.Second)

type subscriptionOutputCacheEntry struct {
	body    string
	headers []string
	expires time.Time
}

type subscriptionOutputCacheStore struct {
	mu      sync.Mutex
	ttl     time.Duration
	entries map[string]subscriptionOutputCacheEntry
}

func newSubscriptionOutputCache(ttl time.Duration) *subscriptionOutputCacheStore {
	return &subscriptionOutputCacheStore{ttl: ttl, entries: map[string]subscriptionOutputCacheEntry{}}
}

func init() {
	service.RegisterSubscriptionCacheInvalidator(ClearSubscriptionOutputCache)
}

func subscriptionCacheGet(key string, now time.Time) (string, []string, bool) {
	subscriptionOutputCache.mu.Lock()
	defer subscriptionOutputCache.mu.Unlock()
	entry, ok := subscriptionOutputCache.entries[key]
	if !ok || !now.Before(entry.expires) {
		delete(subscriptionOutputCache.entries, key)
		return "", nil, false
	}
	return entry.body, append([]string(nil), entry.headers...), true
}

func subscriptionCacheSet(key string, body string, headers []string, now time.Time) {
	subscriptionOutputCache.mu.Lock()
	defer subscriptionOutputCache.mu.Unlock()
	if len(subscriptionOutputCache.entries) >= subscriptionOutputCacheMaxEntries {
		subscriptionOutputCache.entries = map[string]subscriptionOutputCacheEntry{}
	}
	subscriptionOutputCache.entries[key] = subscriptionOutputCacheEntry{
		body:    body,
		headers: append([]string(nil), headers...),
		expires: now.Add(subscriptionOutputCache.ttl),
	}
}

func ClearSubscriptionOutputCache() {
	subscriptionOutputCache.mu.Lock()
	defer subscriptionOutputCache.mu.Unlock()
	subscriptionOutputCache.entries = map[string]subscriptionOutputCacheEntry{}
}
