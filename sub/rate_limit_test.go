package sub

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
	"github.com/deposist/s-ui-x/service"
	"github.com/gin-gonic/gin"
)

func TestRateLimitMiddlewareCanonicalizesMappedClientIP(t *testing.T) {
	initSubTestDB(t)
	resetRateLimitBucketsForTest()
	if _, err := (&service.SettingService{}).GetAllSetting(); err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Model(model.Setting{}).Where("key = ?", "subRateLimitPerIP").Update("value", "2").Error; err != nil {
		t.Fatal(err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(rateLimitMiddleware())
	router.GET("/sub/:subid", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	for _, remoteAddr := range []string{"198.51.100.10:12345", "[::ffff:198.51.100.10]:12345"} {
		recorder := performRateLimitRequestWithRemoteAddr(router, remoteAddr)
		if recorder.Code != http.StatusNoContent {
			t.Fatalf("request from %s should pass, got %d", remoteAddr, recorder.Code)
		}
	}
	recorder := performRateLimitRequestWithRemoteAddr(router, "198.51.100.10:54321")
	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("canonical mapped client should share bucket, got %d", recorder.Code)
	}
}

func TestRateLimitMiddlewareUsesConfiguredLimitAndRetryAfter(t *testing.T) {
	initSubTestDB(t)
	resetRateLimitBucketsForTest()
	if _, err := (&service.SettingService{}).GetAllSetting(); err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Model(model.Setting{}).Where("key = ?", "subRateLimitPerIP").Update("value", "2").Error; err != nil {
		t.Fatal(err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(rateLimitMiddleware())
	router.GET("/sub/:subid", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	for i := 0; i < 2; i++ {
		recorder := performRateLimitRequest(router)
		if recorder.Code != http.StatusNoContent {
			t.Fatalf("request %d should pass, got %d", i, recorder.Code)
		}
	}
	recorder := performRateLimitRequest(router)
	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("third request should be rate-limited, got %d", recorder.Code)
	}
	if recorder.Header().Get("Retry-After") == "" {
		t.Fatal("missing Retry-After header")
	}
}

func TestRateLimitGCSweepsExpiredBuckets(t *testing.T) {
	resetRateLimitBucketsForTest()
	now := time.Now()

	rateLimitMu.Lock()
	rateLimitBuckets["expired"] = rateBucket{windowStart: now.Add(-rateLimitWindow - time.Second), count: 1}
	rateLimitBuckets["active"] = rateBucket{windowStart: now.Add(-10 * time.Second), count: 1}
	gcRateLimitBucketsLocked(now)
	_, expiredOK := rateLimitBuckets["expired"]
	_, activeOK := rateLimitBuckets["active"]
	gcAt := rateLimitGC
	rateLimitMu.Unlock()

	if expiredOK {
		t.Fatal("expired rate-limit bucket was not swept")
	}
	if !activeOK {
		t.Fatal("active rate-limit bucket was swept")
	}
	if !gcAt.Equal(now) {
		t.Fatalf("rateLimitGC=%s, want %s", gcAt, now)
	}
}

func TestRateLimitBucketCapEvictsOverflow(t *testing.T) {
	resetRateLimitBucketsForTest()
	now := time.Now()

	rateLimitMu.Lock()
	for i := 0; i < rateLimitMaxKeys+17; i++ {
		rateLimitBuckets[fmt.Sprintf("198.51.100.%d", i)] = rateBucket{windowStart: now, count: 1}
	}
	gcRateLimitBucketsLocked(now)
	count := len(rateLimitBuckets)
	rateLimitMu.Unlock()

	if count != rateLimitMaxKeys {
		t.Fatalf("rate-limit bucket count=%d, want %d", count, rateLimitMaxKeys)
	}
}

func performRateLimitRequest(router *gin.Engine) *httptest.ResponseRecorder {
	return performRateLimitRequestWithRemoteAddr(router, "198.51.100.10:12345")
}

func performRateLimitRequestWithRemoteAddr(router *gin.Engine, remoteAddr string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/sub/alice", nil)
	req.RemoteAddr = remoteAddr
	router.ServeHTTP(recorder, req)
	return recorder
}
