package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

func TestXUIRateLimitUniqueIPFloodBoundedIssue36(t *testing.T) {
	resetXUIRateLimitCache()
	t.Cleanup(resetXUIRateLimitCache)

	apiService := &ApiService{}
	router := newXUIRateLimitRouterIssue36(apiService)
	for i := 0; i < xuiRateMaxEntries+256; i++ {
		if status := performXUIRateLimitRequestIssue36(router, issue36RemoteAddr(i)); status != http.StatusNoContent {
			t.Fatalf("first request for unique remote addr %d returned status %d", i, status)
		}
	}

	xuiRateMu.Lock()
	defer xuiRateMu.Unlock()
	if len(xuiRates) > xuiRateMaxEntries {
		t.Fatalf("xui rate-limit cache length = %d, want <= %d", len(xuiRates), xuiRateMaxEntries)
	}
}

func TestXUIRateLimitPrunesExpiredBucketsIssue36(t *testing.T) {
	resetXUIRateLimitCache()
	t.Cleanup(resetXUIRateLimitCache)

	staleAt := time.Now().Add(-2 * xuiRequestWindow)
	xuiRateMu.Lock()
	for i := 0; i < xuiRateMaxEntries; i++ {
		xuiRates[fmt.Sprintf("stale-%d", i)] = xuiAttempt{
			Count:    xuiRequestMax,
			WindowAt: staleAt,
		}
	}
	xuiRateMu.Unlock()

	router := newXUIRateLimitRouterIssue36(&ApiService{})
	if status := performXUIRateLimitRequestIssue36(router, "10.250.0.1:1234"); status != http.StatusNoContent {
		t.Fatalf("new request after stale cache seed returned status %d", status)
	}

	xuiRateMu.Lock()
	defer xuiRateMu.Unlock()
	if len(xuiRates) != 1 {
		t.Fatalf("xui rate-limit cache length = %d, want only the fresh bucket", len(xuiRates))
	}
	if _, ok := xuiRates["10.250.0.1"]; !ok {
		t.Fatalf("fresh bucket missing after stale prune: %#v", xuiRates)
	}
}

func newXUIRateLimitRouterIssue36(apiService *ApiService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(sessions.Sessions("s-ui", cookie.NewStore([]byte("issue36-test-secret"))))
	router.GET("/api/import-xui/reports", func(c *gin.Context) {
		if !apiService.enforceXUIRateLimit(c) {
			return
		}
		c.Status(http.StatusNoContent)
	})
	return router
}

func performXUIRateLimitRequestIssue36(router *gin.Engine, remoteAddr string) int {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/import-xui/reports", nil)
	req.RemoteAddr = remoteAddr
	router.ServeHTTP(recorder, req)
	return recorder.Code
}

func issue36RemoteAddr(i int) string {
	return fmt.Sprintf("10.%d.%d.%d:1234", (i>>16)&255, (i>>8)&255, i&255)
}
