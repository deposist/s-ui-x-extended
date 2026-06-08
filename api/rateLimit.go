package api

import (
	"strings"
	"sync"
	"time"

	"github.com/deposist/s-ui-x/util/common"
)

const (
	loginRateLimitWindow  = 15 * time.Minute
	loginRateLimitBlock   = 15 * time.Minute
	loginRateLimitMax     = 5
	loginRateLimitMaxKeys = 4096
	loginRateLimitGCEvery = 1 * time.Minute

	wsHandshakeRateLimitWindow  = 1 * time.Minute
	wsHandshakeRateLimitMax     = 30
	wsHandshakeRateLimitMaxKeys = 4096
	wsHandshakeRateLimitGCEvery = 1 * time.Minute

	auditEndpointRateLimitWindow  = 1 * time.Minute
	auditEndpointRateLimitMax     = 60
	auditEndpointRateLimitMaxKeys = 4096
	auditEndpointRateLimitGCEvery = 1 * time.Minute

	telegramBackupManualRateLimitWindow  = 1 * time.Minute
	telegramBackupManualRateLimitMax     = 3
	telegramBackupManualRateLimitMaxKeys = 4096
	telegramBackupManualRateLimitGCEvery = 1 * time.Minute
)

type loginAttempt struct {
	failures     int
	firstFailAt  time.Time
	blockedUntil time.Time
}

type wsHandshakeAttempt struct {
	count     int
	windowAt  time.Time
	updatedAt time.Time
}

type auditEndpointAttempt struct {
	count     int
	windowAt  time.Time
	updatedAt time.Time
}

type telegramBackupManualAttempt struct {
	timestamps []time.Time
	updatedAt  time.Time
}

var (
	loginRateLimitMu sync.Mutex
	loginRateLimits  = map[string]loginAttempt{}
	loginRateLimitGC time.Time

	wsHandshakeRateLimitMu sync.Mutex
	wsHandshakeRateLimits  = map[string]wsHandshakeAttempt{}
	wsHandshakeRateLimitGC time.Time

	auditEndpointRateLimitMu sync.Mutex
	auditEndpointRateLimits  = map[string]auditEndpointAttempt{}
	auditEndpointRateLimitGC time.Time

	telegramBackupManualRateLimitMu sync.Mutex
	telegramBackupManualRateLimits  = map[string]telegramBackupManualAttempt{}
	telegramBackupManualRateLimitGC time.Time
)

// gcLoginRateLimitsLocked drops stale entries. Caller must hold loginRateLimitMu.
func gcLoginRateLimitsLocked(now time.Time) {
	if now.Sub(loginRateLimitGC) < loginRateLimitGCEvery && len(loginRateLimits) < loginRateLimitMaxKeys {
		return
	}
	loginRateLimitGC = now
	for key, attempt := range loginRateLimits {
		if !attempt.blockedUntil.IsZero() && now.Before(attempt.blockedUntil) {
			continue
		}
		if !attempt.firstFailAt.IsZero() && now.Sub(attempt.firstFailAt) < loginRateLimitWindow {
			continue
		}
		delete(loginRateLimits, key)
	}
	// Hard cap: if still over the limit, evict oldest unblocked entries.
	if len(loginRateLimits) > loginRateLimitMaxKeys {
		for key, attempt := range loginRateLimits {
			if !attempt.blockedUntil.IsZero() && now.Before(attempt.blockedUntil) {
				continue
			}
			delete(loginRateLimits, key)
			if len(loginRateLimits) <= loginRateLimitMaxKeys {
				break
			}
		}
	}
}

func checkLoginRateLimit(key string) error {
	loginRateLimitMu.Lock()
	defer loginRateLimitMu.Unlock()
	now := time.Now()
	gcLoginRateLimitsLocked(now)
	attempt := loginRateLimits[key]
	if !attempt.blockedUntil.IsZero() && now.Before(attempt.blockedUntil) {
		return common.NewError("too many login attempts")
	}
	if !attempt.firstFailAt.IsZero() && now.Sub(attempt.firstFailAt) > loginRateLimitWindow {
		delete(loginRateLimits, key)
	}
	return nil
}

func recordLoginFailure(key string) {
	loginRateLimitMu.Lock()
	defer loginRateLimitMu.Unlock()
	now := time.Now()
	gcLoginRateLimitsLocked(now)
	attempt := loginRateLimits[key]
	if attempt.firstFailAt.IsZero() || now.Sub(attempt.firstFailAt) > loginRateLimitWindow {
		attempt = loginAttempt{firstFailAt: now}
	}
	attempt.failures++
	if attempt.failures >= loginRateLimitMax {
		attempt.blockedUntil = now.Add(loginRateLimitBlock)
	}
	loginRateLimits[key] = attempt
}

func resetLoginFailures(key string) {
	loginRateLimitMu.Lock()
	defer loginRateLimitMu.Unlock()
	delete(loginRateLimits, key)
}

// loginRateLimitUserKey namespaces the per-username login throttle so it never
// collides with the per-IP key (raw IPs are used unprefixed). The per-username
// limit caps a DISTRIBUTED brute-force on one account that rotates source IPs,
// which the per-IP limit alone cannot stop.
func loginRateLimitUserKey(username string) string {
	u := strings.ToLower(strings.TrimSpace(username))
	if u == "" {
		u = "unknown"
	}
	return "user|" + u
}

func gcWSHandshakeRateLimitsLocked(now time.Time) {
	if now.Sub(wsHandshakeRateLimitGC) < wsHandshakeRateLimitGCEvery && len(wsHandshakeRateLimits) < wsHandshakeRateLimitMaxKeys {
		return
	}
	wsHandshakeRateLimitGC = now
	for key, attempt := range wsHandshakeRateLimits {
		if now.Sub(attempt.updatedAt) > wsHandshakeRateLimitWindow {
			delete(wsHandshakeRateLimits, key)
		}
	}
	if len(wsHandshakeRateLimits) > wsHandshakeRateLimitMaxKeys {
		for key := range wsHandshakeRateLimits {
			delete(wsHandshakeRateLimits, key)
			if len(wsHandshakeRateLimits) <= wsHandshakeRateLimitMaxKeys {
				break
			}
		}
	}
}

func checkWSHandshakeRateLimit(key string) error {
	wsHandshakeRateLimitMu.Lock()
	defer wsHandshakeRateLimitMu.Unlock()
	now := time.Now()
	gcWSHandshakeRateLimitsLocked(now)
	attempt := wsHandshakeRateLimits[key]
	if attempt.windowAt.IsZero() || now.Sub(attempt.windowAt) >= wsHandshakeRateLimitWindow {
		attempt = wsHandshakeAttempt{windowAt: now}
	}
	if attempt.count >= wsHandshakeRateLimitMax {
		attempt.updatedAt = now
		wsHandshakeRateLimits[key] = attempt
		return common.NewError("too many websocket handshake attempts")
	}
	attempt.count++
	attempt.updatedAt = now
	wsHandshakeRateLimits[key] = attempt
	return nil
}

func wsHandshakeRateLimitKey(endpoint string, ip string) string {
	return endpoint + "|" + ip
}

func auditEndpointRateLimitKey(actor string, ip string) string {
	if actor == "" {
		actor = "unknown"
	}
	if ip == "" {
		ip = "unknown"
	}
	return actor + "|" + ip
}

func gcAuditEndpointRateLimitsLocked(now time.Time) {
	if now.Sub(auditEndpointRateLimitGC) < auditEndpointRateLimitGCEvery && len(auditEndpointRateLimits) < auditEndpointRateLimitMaxKeys {
		return
	}
	auditEndpointRateLimitGC = now
	for key, attempt := range auditEndpointRateLimits {
		if now.Sub(attempt.updatedAt) > auditEndpointRateLimitWindow {
			delete(auditEndpointRateLimits, key)
		}
	}
	if len(auditEndpointRateLimits) > auditEndpointRateLimitMaxKeys {
		for key := range auditEndpointRateLimits {
			delete(auditEndpointRateLimits, key)
			if len(auditEndpointRateLimits) <= auditEndpointRateLimitMaxKeys {
				break
			}
		}
	}
}

func checkAuditEndpointRateLimit(key string) error {
	auditEndpointRateLimitMu.Lock()
	defer auditEndpointRateLimitMu.Unlock()
	now := time.Now()
	gcAuditEndpointRateLimitsLocked(now)
	attempt := auditEndpointRateLimits[key]
	if attempt.windowAt.IsZero() || now.Sub(attempt.windowAt) >= auditEndpointRateLimitWindow {
		attempt = auditEndpointAttempt{windowAt: now}
	}
	if attempt.count >= auditEndpointRateLimitMax {
		attempt.updatedAt = now
		auditEndpointRateLimits[key] = attempt
		return common.NewError("too many audit requests")
	}
	attempt.count++
	attempt.updatedAt = now
	auditEndpointRateLimits[key] = attempt
	return nil
}

func gcTelegramBackupManualRateLimitsLocked(now time.Time) {
	if now.Sub(telegramBackupManualRateLimitGC) < telegramBackupManualRateLimitGCEvery && len(telegramBackupManualRateLimits) < telegramBackupManualRateLimitMaxKeys {
		return
	}
	telegramBackupManualRateLimitGC = now
	for key, attempt := range telegramBackupManualRateLimits {
		filtered := pruneTelegramBackupManualTimestamps(attempt.timestamps, now)
		if len(filtered) == 0 || now.Sub(attempt.updatedAt) > telegramBackupManualRateLimitWindow {
			delete(telegramBackupManualRateLimits, key)
			continue
		}
		attempt.timestamps = filtered
		telegramBackupManualRateLimits[key] = attempt
	}
	if len(telegramBackupManualRateLimits) > telegramBackupManualRateLimitMaxKeys {
		for key := range telegramBackupManualRateLimits {
			delete(telegramBackupManualRateLimits, key)
			if len(telegramBackupManualRateLimits) <= telegramBackupManualRateLimitMaxKeys {
				break
			}
		}
	}
}

func checkTelegramBackupManualRateLimit(key string) (time.Duration, error) {
	telegramBackupManualRateLimitMu.Lock()
	defer telegramBackupManualRateLimitMu.Unlock()
	now := time.Now()
	gcTelegramBackupManualRateLimitsLocked(now)
	attempt := telegramBackupManualRateLimits[key]
	attempt.timestamps = pruneTelegramBackupManualTimestamps(attempt.timestamps, now)
	if len(attempt.timestamps) >= telegramBackupManualRateLimitMax {
		retryAfter := attempt.timestamps[0].Add(telegramBackupManualRateLimitWindow).Sub(now)
		if retryAfter < time.Second {
			retryAfter = time.Second
		}
		attempt.updatedAt = now
		telegramBackupManualRateLimits[key] = attempt
		return retryAfter, common.NewError("too many telegram backup requests")
	}
	attempt.timestamps = append(attempt.timestamps, now)
	attempt.updatedAt = now
	telegramBackupManualRateLimits[key] = attempt
	return 0, nil
}

func pruneTelegramBackupManualTimestamps(timestamps []time.Time, now time.Time) []time.Time {
	if len(timestamps) == 0 {
		return timestamps
	}
	cutoff := now.Add(-telegramBackupManualRateLimitWindow)
	first := 0
	for first < len(timestamps) && !timestamps[first].After(cutoff) {
		first++
	}
	if first == 0 {
		return timestamps
	}
	copy(timestamps, timestamps[first:])
	return timestamps[:len(timestamps)-first]
}
