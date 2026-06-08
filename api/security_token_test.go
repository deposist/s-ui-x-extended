package api

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestSecurityAPITokenFromRequestBearerAndLegacyHeader(t *testing.T) {
	withAPITokenNow(t, legacyTokenHeaderSunsetAt.Add(-time.Second))

	tests := []struct {
		name       string
		auth       string
		legacy     string
		wantToken  string
		wantLegacy bool
	}{
		{name: "bearer", auth: "Bearer bearer-token", wantToken: "bearer-token"},
		{name: "bearer takes precedence", auth: "Bearer bearer-token", legacy: "legacy-token", wantToken: "bearer-token"},
		{name: "legacy token header", legacy: "legacy-token", wantToken: "legacy-token", wantLegacy: true},
		{name: "missing", wantToken: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			req := httptest.NewRequest(http.MethodGet, "/apiv2/settings", nil)
			if tt.auth != "" {
				req.Header.Set("Authorization", tt.auth)
			}
			if tt.legacy != "" {
				req.Header.Set("Token", tt.legacy)
			}
			c.Request = req
			got, legacy := apiTokenFromRequest(c)
			if got != tt.wantToken || legacy != tt.wantLegacy {
				t.Fatalf("apiTokenFromRequest()=(%q,%v), want (%q,%v)", got, legacy, tt.wantToken, tt.wantLegacy)
			}
		})
	}
}

func TestSecurityAPITokenFromRequestLegacySunsetIssue34(t *testing.T) {
	withAPITokenNow(t, legacyTokenHeaderSunsetAt)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(http.MethodGet, "/apiv2/settings", nil)
	req.Header.Set("Token", "legacy-token")
	c.Request = req

	got, legacy := apiTokenFromRequest(c)
	if got != "" || !legacy {
		t.Fatalf("expired legacy apiTokenFromRequest()=(%q,%v), want empty token and legacy=true", got, legacy)
	}
	if !c.GetBool(legacyTokenHeaderExpiredKey) {
		t.Fatal("expired legacy token header did not set expired marker")
	}

	recorder = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(recorder)
	req = httptest.NewRequest(http.MethodGet, "/apiv2/settings", nil)
	req.Header.Set("Authorization", "Bearer bearer-token")
	req.Header.Set("Token", "legacy-token")
	c.Request = req

	got, legacy = apiTokenFromRequest(c)
	if got != "bearer-token" || legacy {
		t.Fatalf("bearer with expired legacy header apiTokenFromRequest()=(%q,%v), want bearer token and legacy=false", got, legacy)
	}
	if c.GetBool(legacyTokenHeaderExpiredKey) {
		t.Fatal("bearer token path should not set expired legacy marker")
	}
}

func withAPITokenNow(t *testing.T, now time.Time) {
	t.Helper()
	previous := apiTokenNow
	apiTokenNow = func() time.Time { return now }
	t.Cleanup(func() { apiTokenNow = previous })
}

func TestSecurityConsumeWSTokenDoubleSpendExpiredAndCapacity(t *testing.T) {
	_ = sweepAllWSTokens()
	t.Cleanup(func() { _ = sweepAllWSTokens() })

	wsTokens.Lock()
	wsTokens.tokens[wsTokenDigest("single-use")] = realtimeToken{user: "admin", expiresAt: time.Now().Add(time.Minute)}
	wsTokens.Unlock()
	if user, ok := consumeWSToken("single-use"); !ok || user != "admin" {
		t.Fatalf("first consume failed: user=%q ok=%v", user, ok)
	}
	if user, ok := consumeWSToken("single-use"); ok || user != "" {
		t.Fatalf("second consume should fail: user=%q ok=%v", user, ok)
	}

	wsTokens.Lock()
	expiredDigest := wsTokenDigest("expired")
	wsTokens.tokens[expiredDigest] = realtimeToken{user: "admin", expiresAt: time.Now().Add(-time.Second)}
	wsTokens.Unlock()
	if user, ok := consumeWSToken("expired"); ok || user != "" {
		t.Fatalf("expired consume should fail: user=%q ok=%v", user, ok)
	}
	wsTokens.Lock()
	_, stillPresent := wsTokens.tokens[expiredDigest]
	wsTokens.Unlock()
	if stillPresent {
		t.Fatal("expired matched token should be deleted after consume attempt")
	}

	oldDigest := wsTokenDigest("oldest")
	wsTokens.Lock()
	wsTokens.tokens[oldDigest] = realtimeToken{user: "admin", expiresAt: time.Now().Add(time.Minute)}
	for i := 0; i < maxWSTokens; i++ {
		token := "new-" + strconv.Itoa(i)
		wsTokens.tokens[wsTokenDigest(token)] = realtimeToken{user: "admin", expiresAt: time.Now().Add(time.Hour + time.Duration(i)*time.Second)}
	}
	enforceWSTokenCapLocked()
	_, oldStillPresent := wsTokens.tokens[oldDigest]
	size := len(wsTokens.tokens)
	wsTokens.Unlock()
	if oldStillPresent {
		t.Fatal("oldest websocket token was not evicted when capacity was exceeded")
	}
	if size != maxWSTokens {
		t.Fatalf("unexpected websocket token capacity size %d, want %d", size, maxWSTokens)
	}
}
