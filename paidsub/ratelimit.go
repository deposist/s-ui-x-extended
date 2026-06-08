package paidsub

import "sync"

// rateLimiter is a fixed-window per-key limiter (keyed by Telegram user id),
// mirroring the in-process GC + max-keys pattern used in api/rateLimit.go.
type rateLimiter struct {
	mu      sync.Mutex
	entries map[int64]rlEntry
	max     int
	window  int64 // seconds
	maxKeys int
}

type rlEntry struct {
	windowStart int64
	count       int
}

func newRateLimiter(max int, windowSeconds int64) *rateLimiter {
	return &rateLimiter{
		entries: make(map[int64]rlEntry),
		max:     max,
		window:  windowSeconds,
		maxKeys: 8192,
	}
}

// allow reports whether the key may proceed at time now (unix seconds).
func (r *rateLimiter) allow(id int64, now int64) bool {
	return r.allowWithMax(id, now, r.max)
}

// allowWithMax is allow with a caller-supplied per-window cap (used when the cap
// comes from a runtime setting).
func (r *rateLimiter) allowWithMax(id int64, now int64, max int) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.entries) > r.maxKeys {
		r.gcLocked(now)
	}
	e, exists := r.entries[id]
	if !exists && len(r.entries) >= r.maxKeys {
		// Map is saturated with in-window keys (GC freed nothing); refuse new
		// keys to bound memory under a burst of many distinct ids.
		return false
	}
	if now-e.windowStart >= r.window {
		e = rlEntry{windowStart: now}
	}
	if max <= 0 || e.count >= max {
		r.entries[id] = e
		return false
	}
	e.count++
	r.entries[id] = e
	return true
}

func (r *rateLimiter) gcLocked(now int64) {
	for id, e := range r.entries {
		if now-e.windowStart >= r.window {
			delete(r.entries, id)
		}
	}
}
