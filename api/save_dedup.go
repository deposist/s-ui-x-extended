package api

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"
)

const (
	// saveDedupWindow is how long after a create COMPLETES an identical create is
	// still treated as an accidental duplicate (catches a slightly-late resend).
	saveDedupWindow = 3 * time.Second
	// saveDedupMaxInFlight bounds how long a claimed-but-never-finished entry
	// blocks duplicates, so a crash mid-save cannot wedge a payload forever.
	saveDedupMaxInFlight = 2 * time.Minute
)

// saveDedupCache is the authoritative, server-side guard against duplicate
// creation: a single logical create cannot insert two rows even if it is
// submitted twice. A key stays a duplicate while the first request is still
// IN-FLIGHT (any duration — a save blocks on a synchronous core restart, which
// can exceed any fixed window) and for saveDedupWindow after it completes. It is
// intentionally narrow (creation actions on entity objects only).
type saveDedupCache struct {
	mu   sync.Mutex
	seen map[string]dedupEntry
}

type dedupEntry struct {
	claimedAt int64 // unixNano when claimed
	doneAt    int64 // unixNano when completed; 0 while in-flight
}

var saveDedup = &saveDedupCache{seen: make(map[string]dedupEntry)}

func saveDedupKey(actor, object, action, data string) string {
	h := sha256.Sum256([]byte(actor + "\x00" + object + "\x00" + action + "\x00" + data))
	return hex.EncodeToString(h[:])
}

// isDedupableSave reports whether a save is a creation action on an entity
// object — the only saves that can produce a duplicate row.
func isDedupableSave(object, action string) bool {
	switch action {
	case "new", "addbulk":
	default:
		return false
	}
	switch object {
	case "clients", "inbounds", "outbounds", "endpoints", "providers", "services", "tls":
		return true
	default:
		return false
	}
}

// claim atomically reserves a key. It returns true if the caller may proceed
// with the save, or false if an identical create is in-flight or completed
// within the window (a duplicate to skip). The check-and-set is under the mutex
// so two concurrent identical requests cannot both proceed.
func (c *saveDedupCache) claim(key string, nowNano int64) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.evict(nowNano)
	if e, ok := c.seen[key]; ok {
		if e.doneAt == 0 || nowNano-e.doneAt <= int64(saveDedupWindow) {
			return false
		}
	}
	c.seen[key] = dedupEntry{claimedAt: nowNano}
	return true
}

// complete marks a claimed key as finished; it keeps deduplicating for
// saveDedupWindow afterwards, then is evicted.
func (c *saveDedupCache) complete(key string, nowNano int64) {
	c.mu.Lock()
	if e, ok := c.seen[key]; ok {
		e.doneAt = nowNano
		c.seen[key] = e
	}
	c.mu.Unlock()
}

// release drops a claimed key so a save that FAILED can be retried immediately
// (a failed create inserted nothing, so its retry must not be treated as a dup).
func (c *saveDedupCache) release(key string) {
	c.mu.Lock()
	delete(c.seen, key)
	c.mu.Unlock()
}

// evict removes completed entries past the window and stuck in-flight entries
// past the safety cap. Caller must hold the mutex.
func (c *saveDedupCache) evict(nowNano int64) {
	for k, e := range c.seen {
		if e.doneAt == 0 {
			if nowNano-e.claimedAt > int64(saveDedupMaxInFlight) {
				delete(c.seen, k)
			}
		} else if nowNano-e.doneAt > int64(saveDedupWindow) {
			delete(c.seen, k)
		}
	}
}

// reset clears the cache (test helper).
func (c *saveDedupCache) reset() {
	c.mu.Lock()
	c.seen = make(map[string]dedupEntry)
	c.mu.Unlock()
}
