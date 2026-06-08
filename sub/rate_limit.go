package sub

import (
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/deposist/s-ui-x/service"

	"github.com/gin-gonic/gin"
)

const (
	rateLimitWindow          = time.Minute
	defaultRateLimitRequests = 60
	rateLimitSettingTTL      = time.Minute
	rateLimitMaxKeys         = 4096
	rateLimitGCEvery         = time.Minute
)

type rateBucket struct {
	windowStart time.Time
	count       int
}

var (
	rateLimitMu      sync.Mutex
	rateLimitBuckets = map[string]rateBucket{}
	rateLimitGC      time.Time

	rateLimitSettingMu sync.Mutex
	rateLimitSetting   = struct {
		limit     int
		expiresAt time.Time
	}{}
)

func rateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := canonicalClientIP(c.ClientIP())
		if ip == "" {
			ip = c.ClientIP()
		}
		now := time.Now()
		rateLimitMu.Lock()
		gcRateLimitBucketsLocked(now)
		bucket := rateLimitBuckets[ip]
		if now.Sub(bucket.windowStart) >= rateLimitWindow {
			bucket = rateBucket{windowStart: now}
		}
		bucket.count++
		rateLimitBuckets[ip] = bucket
		if len(rateLimitBuckets) > rateLimitMaxKeys {
			enforceRateLimitBucketCapLocked()
		}
		limit := currentRateLimitRequests(now)
		allowed := bucket.count <= limit
		retryAfter := int(bucket.windowStart.Add(rateLimitWindow).Sub(now).Seconds())
		if retryAfter <= 0 {
			retryAfter = int(rateLimitWindow / time.Second)
		}
		rateLimitMu.Unlock()

		if !allowed {
			c.Header("Retry-After", strconv.Itoa(retryAfter))
			c.AbortWithStatus(http.StatusTooManyRequests)
			return
		}
		c.Next()
	}
}

func canonicalClientIP(value string) string {
	value = strings.TrimSpace(strings.Trim(value, "[]"))
	if value == "" || strings.Contains(value, "%") {
		return ""
	}
	addr, err := netip.ParseAddr(value)
	if err != nil || addr.Zone() != "" {
		return ""
	}
	return addr.Unmap().String()
}

func resetRateLimitBucketsForTest() {
	rateLimitMu.Lock()
	defer rateLimitMu.Unlock()
	rateLimitBuckets = map[string]rateBucket{}
	rateLimitGC = time.Time{}
	rateLimitSettingMu.Lock()
	defer rateLimitSettingMu.Unlock()
	rateLimitSetting.limit = 0
	rateLimitSetting.expiresAt = time.Time{}
}

func gcRateLimitBucketsLocked(now time.Time) {
	if now.Sub(rateLimitGC) < rateLimitGCEvery && len(rateLimitBuckets) < rateLimitMaxKeys {
		return
	}
	rateLimitGC = now
	for ip, bucket := range rateLimitBuckets {
		if now.Sub(bucket.windowStart) >= rateLimitWindow {
			delete(rateLimitBuckets, ip)
		}
	}
	enforceRateLimitBucketCapLocked()
}

func enforceRateLimitBucketCapLocked() {
	if len(rateLimitBuckets) <= rateLimitMaxKeys {
		return
	}
	for ip := range rateLimitBuckets {
		delete(rateLimitBuckets, ip)
		if len(rateLimitBuckets) <= rateLimitMaxKeys {
			return
		}
	}
}

func currentRateLimitRequests(now time.Time) int {
	rateLimitSettingMu.Lock()
	defer rateLimitSettingMu.Unlock()
	if rateLimitSetting.limit > 0 && now.Before(rateLimitSetting.expiresAt) {
		return rateLimitSetting.limit
	}
	limit, err := (&service.SettingService{}).GetSubRateLimitPerIP()
	if err != nil || limit <= 0 {
		limit = defaultRateLimitRequests
	}
	rateLimitSetting.limit = limit
	rateLimitSetting.expiresAt = now.Add(rateLimitSettingTTL)
	return limit
}
