package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
	"github.com/deposist/s-ui-x/service"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

func TestAPIV2TelegramBackupRequiresTelegramOrAdminScope(t *testing.T) {
	initSessionTestDB(t)
	readToken, err := (&service.UserService{}).AddToken("admin", 0, "read backup", "read")
	if err != nil {
		t.Fatal(err)
	}
	telegramToken, err := (&service.UserService{}).AddToken("admin", 0, "telegram backup", "telegram")
	if err != nil {
		t.Fatal(err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewAPIv2Handler(router.Group("/apiv2"))

	recorder := performTelegramBackupRequest(router, "/apiv2/telegram/backup", readToken)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("read token should be forbidden, got %d", recorder.Code)
	}

	var event model.AuditEvent
	if err := database.GetDB().Where("event = ?", "scope_denied").First(&event).Error; err != nil {
		t.Fatal(err)
	}
	if event.Actor != "admin" || event.Resource != "telegram" {
		t.Fatalf("unexpected audit event: %#v", event)
	}

	recorder = performTelegramBackupRequest(router, "/apiv2/telegram/backup/run", telegramToken)
	if recorder.Code == http.StatusForbidden {
		t.Fatalf("telegram scope should reach handler, got %d", recorder.Code)
	}
}

func TestAPIV2TelegramBackupDisabledFailureAuditsWithoutKey(t *testing.T) {
	initSessionTestDB(t)
	adminToken, err := (&service.UserService{}).AddToken("admin", 0, "admin backup", "admin")
	if err != nil {
		t.Fatal(err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewAPIv2Handler(router.Group("/apiv2"))

	recorder := performTelegramBackupRequest(router, "/apiv2/telegram/backup", adminToken)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("unexpected status: %d", recorder.Code)
	}
	assertTelegramBackupFailure(t, recorder, "disabled")
	if strings.Contains(recorder.Body.String(), "backupKey") {
		t.Fatalf("failed backup response leaked a backup key: %s", recorder.Body.String())
	}

	var event model.AuditEvent
	if err := database.GetDB().Where("event = ?", "tg_backup_failed").First(&event).Error; err != nil {
		t.Fatal(err)
	}
	if event.Actor != "admin" || event.Resource != "database" {
		t.Fatalf("unexpected audit event: %#v", event)
	}
	details := string(event.Details)
	if !strings.Contains(details, `"channel":"telegram"`) || !strings.Contains(details, `"errorClass":"disabled"`) {
		t.Fatalf("unexpected audit details: %s", details)
	}
	if strings.Contains(details, "backupKey") || strings.Contains(details, "123456:test-token") {
		t.Fatalf("audit details leaked secret material: %s", details)
	}
}

func TestTelegramBackupManualRoutesShareRateLimitBucket(t *testing.T) {
	resetRateLimitState()
	t.Cleanup(resetRateLimitState)
	settingService := initSessionTestDB(t)
	adminToken, err := (&service.UserService{}).AddToken("admin", 0, "admin backup", "admin")
	if err != nil {
		t.Fatal(err)
	}

	router, cookies, csrf := newTelegramBackupFullRouter(t, settingService)
	requests := []struct {
		method string
		path   string
		token  string
		csrf   string
	}{
		{method: http.MethodPost, path: "/api/telegram/backup", csrf: csrf},
		{method: http.MethodPost, path: "/apiv2/telegram/backup", token: adminToken},
		{method: http.MethodPost, path: "/api/telegram/backup/run", csrf: csrf},
		{method: http.MethodPost, path: "/apiv2/telegram/backup/run", token: adminToken},
	}
	for i, req := range requests {
		recorder := performTelegramBackupFullRequest(router, req.method, req.path, req.token, req.csrf, cookies...)
		if i < telegramBackupManualRateLimitMax {
			if recorder.Code != http.StatusServiceUnavailable {
				t.Fatalf("request %d should reach disabled handler, got %d body=%s", i+1, recorder.Code, recorder.Body.String())
			}
			assertTelegramBackupFailure(t, recorder, "disabled")
			continue
		}
		if recorder.Code != http.StatusTooManyRequests {
			t.Fatalf("request %d should be rate-limited, got %d body=%s", i+1, recorder.Code, recorder.Body.String())
		}
		retryAfter, err := strconv.Atoi(recorder.Header().Get("Retry-After"))
		if err != nil || retryAfter < 1 {
			t.Fatalf("invalid Retry-After header %q", recorder.Header().Get("Retry-After"))
		}
		assertTelegramBackupFailure(t, recorder, "rate_limited")
		if strings.Contains(recorder.Body.String(), "backupKey") {
			t.Fatalf("rate-limited response leaked backupKey: %s", recorder.Body.String())
		}
	}
}

func TestTelegramBackupBrowserRouteRequiresCSRF(t *testing.T) {
	settingService := initSessionTestDB(t)
	router, cookies, _ := newTelegramBackupFullRouter(t, settingService)
	recorder := performTelegramBackupFullRequest(router, http.MethodPost, "/api/telegram/backup/run", "", "", cookies...)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("missing csrf token should return 403, got %d", recorder.Code)
	}
}

func TestTelegramBackupManualRouteErrorMapping(t *testing.T) {
	tests := []struct {
		errorClass string
		want       int
	}{
		{"concurrent_run", http.StatusConflict},
		{"rate_limited", http.StatusTooManyRequests},
		{"disabled", http.StatusServiceUnavailable},
		{"missing_token", http.StatusServiceUnavailable},
		{"missing_chat", http.StatusServiceUnavailable},
		{"missing_passphrase", http.StatusServiceUnavailable},
		{"oversize", http.StatusServiceUnavailable},
		{"network", http.StatusServiceUnavailable},
		{"proxy", http.StatusServiceUnavailable},
		{"unauthorized", http.StatusServiceUnavailable},
		{"chat_not_found", http.StatusServiceUnavailable},
		{"db_snapshot_failed", http.StatusInternalServerError},
		{"encryption_failed", http.StatusInternalServerError},
		{"settings", http.StatusInternalServerError},
		{"payload", http.StatusInternalServerError},
		{"request", http.StatusInternalServerError},
		{"unknown", http.StatusInternalServerError},
		{"internal", http.StatusInternalServerError},
		{"new_telegram_class", http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.errorClass, func(t *testing.T) {
			if got := telegramBackupHTTPStatus(tt.errorClass); got != tt.want {
				t.Fatalf("telegramBackupHTTPStatus(%q)=%d, want %d", tt.errorClass, got, tt.want)
			}
		})
	}
}

func newTelegramBackupFullRouter(t *testing.T, settingService *service.SettingService) (*gin.Engine, []*http.Cookie, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(sessions.Sessions("s-ui", cookie.NewStore([]byte("test-secret"))))
	router.GET("/login", func(c *gin.Context) {
		generation, err := settingService.GetSessionGeneration()
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		if err := SetLoginUser(c, "admin", 0, generation); err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		c.Status(http.StatusNoContent)
	})
	apiv2 := NewAPIv2Handler(router.Group("/apiv2"))
	NewAPIHandler(router.Group("/api"), apiv2)

	loginRecorder := httptest.NewRecorder()
	router.ServeHTTP(loginRecorder, httptest.NewRequest(http.MethodGet, "/login", nil))
	if loginRecorder.Code != http.StatusNoContent {
		t.Fatalf("login returned %d", loginRecorder.Code)
	}
	cookies := loginRecorder.Result().Cookies()
	csrfRecorder := performTelegramBackupFullRequest(router, http.MethodGet, "/api/csrf", "", "", cookies...)
	if csrfRecorder.Code != http.StatusOK {
		t.Fatalf("csrf endpoint returned %d", csrfRecorder.Code)
	}
	var msg Msg
	if err := json.Unmarshal(csrfRecorder.Body.Bytes(), &msg); err != nil {
		t.Fatal(err)
	}
	obj, ok := msg.Obj.(map[string]any)
	if !ok {
		t.Fatalf("unexpected csrf payload: %#v", msg.Obj)
	}
	csrf, ok := obj["token"].(string)
	if !ok || csrf == "" {
		t.Fatalf("missing csrf token in payload: %#v", obj)
	}
	cookies = appendUpdatedCookies(cookies, csrfRecorder.Result().Cookies())
	return router, cookies, csrf
}

func appendUpdatedCookies(base []*http.Cookie, updates []*http.Cookie) []*http.Cookie {
	if len(updates) == 0 {
		return base
	}
	merged := make([]*http.Cookie, 0, len(base)+len(updates))
	for _, existing := range base {
		replaced := false
		for _, update := range updates {
			if existing.Name == update.Name {
				replaced = true
				break
			}
		}
		if !replaced {
			merged = append(merged, existing)
		}
	}
	return append(merged, updates...)
}

func performTelegramBackupFullRequest(router *gin.Engine, method string, path string, token string, csrf string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, strings.NewReader(""))
	req.RemoteAddr = "192.0.2.1:12345"
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if csrf != "" {
		req.Header.Set(csrfHeader, csrf)
	}
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	router.ServeHTTP(recorder, req)
	return recorder
}

func performTelegramBackupRequest(router *gin.Engine, path string, token string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, path, nil)
	req.RemoteAddr = "192.0.2.1:12345"
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(recorder, req)
	return recorder
}

func assertTelegramBackupFailure(t *testing.T, recorder *httptest.ResponseRecorder, wantClass string) {
	t.Helper()
	var msg Msg
	if err := json.Unmarshal(recorder.Body.Bytes(), &msg); err != nil {
		t.Fatal(err)
	}
	if msg.Success {
		t.Fatalf("expected failure response, got %#v", msg)
	}
	obj, ok := msg.Obj.(map[string]any)
	if !ok {
		t.Fatalf("unexpected response obj: %#v", msg.Obj)
	}
	if obj["errorClass"] != wantClass {
		t.Fatalf("unexpected errorClass: %#v body=%s", obj["errorClass"], recorder.Body.String())
	}
	if obj["trigger"] != service.TelegramBackupTriggerManual {
		t.Fatalf("unexpected trigger: %#v", obj["trigger"])
	}
}
