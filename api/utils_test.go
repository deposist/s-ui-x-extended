package api

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func makeContext(t *testing.T, remoteAddr, xff string) *gin.Context {
	t.Helper()
	gin.SetMode(gin.TestMode)
	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	req.RemoteAddr = remoteAddr
	if xff != "" {
		req.Header.Set("X-Forwarded-For", xff)
	}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = req
	return c
}

func TestCanonicalClientIPNormalizesMappedAndRejectsZones(t *testing.T) {
	cases := map[string]string{
		"::ffff:192.0.2.10":   "192.0.2.10",
		"[::ffff:192.0.2.10]": "192.0.2.10",
		"2001:db8::1":         "2001:db8::1",
		"fe80::1%eth0":        "",
		"not-ip":              "",
		"":                    "",
	}
	for input, want := range cases {
		if got := canonicalClientIP(input); got != want {
			t.Fatalf("canonicalClientIP(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestGetRemoteIpCanonicalizesMappedTransportPeer(t *testing.T) {
	t.Setenv("SUI_TRUSTED_PROXIES", "")
	c := makeContext(t, "[::ffff:203.0.113.5]:1234", "10.0.0.1")
	if got := getRemoteIp(c); got != "203.0.113.5" {
		t.Fatalf("expected canonical transport peer, got %s", got)
	}
}

func TestGetRemoteIpRejectsZoneIdentifiers(t *testing.T) {
	t.Setenv("SUI_TRUSTED_PROXIES", "fe80::/10")
	c := makeContext(t, "[fe80::1%25eth0]:1234", "203.0.113.5")
	if got := getRemoteIp(c); got != "" {
		t.Fatalf("expected invalid zoned transport peer to be rejected, got %q", got)
	}
}

func TestGetRemoteIpIgnoresXFFWhenProxiesUntrusted(t *testing.T) {
	t.Setenv("SUI_TRUSTED_PROXIES", "")
	c := makeContext(t, "203.0.113.5:1234", "10.0.0.1")
	if got := getRemoteIp(c); got != "203.0.113.5" {
		t.Fatalf("expected transport peer, got %s", got)
	}
}

func TestGetRemoteIpUsesRightmostUntrustedHop(t *testing.T) {
	t.Setenv("SUI_TRUSTED_PROXIES", "10.0.0.0/8")
	c := makeContext(t, "10.0.0.7:1234", "203.0.113.9, 198.51.100.5, 10.0.0.10")
	if got := getRemoteIp(c); got != "198.51.100.5" {
		t.Fatalf("expected rightmost untrusted hop, got %s", got)
	}
}

func TestGetRemoteIpAllUntrustedFallsBackToTransport(t *testing.T) {
	t.Setenv("SUI_TRUSTED_PROXIES", "10.0.0.0/8")
	c := makeContext(t, "10.0.0.7:1234", "10.0.0.1, 10.0.0.2")
	if got := getRemoteIp(c); got != "10.0.0.7" {
		t.Fatalf("expected transport peer fallback, got %s", got)
	}
}

func TestGetRemoteIpRejectsSpoofedXFFFromUntrustedClient(t *testing.T) {
	t.Setenv("SUI_TRUSTED_PROXIES", "10.0.0.0/8")
	c := makeContext(t, "203.0.113.5:1234", "1.2.3.4, 5.6.7.8")
	if got := getRemoteIp(c); got != "203.0.113.5" {
		t.Fatalf("expected transport peer, got %s", got)
	}
}

// TestRequestIsHTTPSGatesForwardedProtoByTrustedProxy locks the A6 gate that the
// HSTS middleware now reuses: X-Forwarded-Proto is honored only from a trusted
// proxy, so a spoofed header from an untrusted client cannot mark the request
// HTTPS (and thus cannot trigger HSTS).
func TestRequestIsHTTPSGatesForwardedProtoByTrustedProxy(t *testing.T) {
	t.Run("untrusted client spoofing https", func(t *testing.T) {
		t.Setenv("SUI_TRUSTED_PROXIES", "")
		c := makeContext(t, "203.0.113.5:1234", "")
		c.Request.Header.Set("X-Forwarded-Proto", "https")
		if RequestIsHTTPS(c) {
			t.Fatal("X-Forwarded-Proto from an untrusted client must not be treated as HTTPS")
		}
	})
	t.Run("trusted proxy forwarding https", func(t *testing.T) {
		t.Setenv("SUI_TRUSTED_PROXIES", "203.0.113.0/24")
		c := makeContext(t, "203.0.113.5:1234", "")
		c.Request.Header.Set("X-Forwarded-Proto", "https")
		if !RequestIsHTTPS(c) {
			t.Fatal("X-Forwarded-Proto: https from a trusted proxy should be treated as HTTPS")
		}
	})
	t.Run("trusted proxy forwarding http", func(t *testing.T) {
		t.Setenv("SUI_TRUSTED_PROXIES", "203.0.113.0/24")
		c := makeContext(t, "203.0.113.5:1234", "")
		c.Request.Header.Set("X-Forwarded-Proto", "http")
		if RequestIsHTTPS(c) {
			t.Fatal("X-Forwarded-Proto: http must not be treated as HTTPS")
		}
	})
}

func TestTelegramRequestFieldsUseOnlyAllowedMetadata(t *testing.T) {
	c := makeContext(t, "203.0.113.5:1234", "")
	userAgent := "Mozilla/5.0 test agent"
	c.Request.Header.Set("User-Agent", userAgent)

	fields := telegramRequestFields(c)
	if len(fields) != 3 {
		t.Fatalf("expected exactly 3 fields, got %#v", fields)
	}
	if fields["ip"] != "203.0.113.5" {
		t.Fatalf("unexpected ip field: %q", fields["ip"])
	}
	sum := sha256.Sum256([]byte(userAgent))
	if fields["ua_hash"] != hex.EncodeToString(sum[:]) {
		t.Fatalf("unexpected ua_hash: %q", fields["ua_hash"])
	}
	if _, err := time.Parse(time.RFC3339, fields["ts"]); err != nil {
		t.Fatalf("ts is not RFC3339: %q", fields["ts"])
	}
	for _, forbidden := range []string{"user", "username", "reason", "error"} {
		if _, ok := fields[forbidden]; ok {
			t.Fatalf("forbidden field %q leaked into Telegram payload: %#v", forbidden, fields)
		}
	}
}

func TestCoreRestartFailedTelegramFieldsDoNotExposeRawError(t *testing.T) {
	c := makeContext(t, "203.0.113.5:1234", "")
	rawErr := "config parse failed: Authorization: Bearer core-secret-token"

	fields := coreRestartFailedTelegramFields(c, errors.New(rawErr))
	if fields["errorClass"] != "config" {
		t.Fatalf("unexpected errorClass: %q", fields["errorClass"])
	}
	for _, forbiddenKey := range []string{"reason", "error"} {
		if _, ok := fields[forbiddenKey]; ok {
			t.Fatalf("forbidden field %q leaked into Telegram payload: %#v", forbiddenKey, fields)
		}
	}

	var values []string
	for _, value := range fields {
		values = append(values, value)
	}
	joined := strings.Join(values, "\n")
	for _, forbidden := range []string{rawErr, "core-secret-token", "Authorization: Bearer"} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("raw restart error leaked into Telegram payload: %#v", fields)
		}
	}
}

func TestLoginRedirectPathUsesConfiguredWebPath(t *testing.T) {
	settingService := initSessionTestDB(t)
	if _, err := settingService.GetAllSetting(); err != nil {
		t.Fatal(err)
	}
	if err := settingService.SetWebPath("/panel/"); err != nil {
		t.Fatal(err)
	}

	if got := loginRedirectPath(); got != "/panel/login" {
		t.Fatalf("unexpected login redirect path: %q", got)
	}
}
