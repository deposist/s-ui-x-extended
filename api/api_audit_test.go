package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
	"github.com/deposist/s-ui-x/service"
	"github.com/gin-gonic/gin"
)

func TestGetSecurityAuditDoesNotPruneOnRead(t *testing.T) {
	resetRateLimitState()
	settingService := initSessionTestDB(t)
	oldEvent := model.AuditEvent{
		DateTime: time.Now().Add(-31 * 24 * time.Hour).Unix(),
		Actor:    "admin",
		Event:    "old",
	}
	if err := database.GetDB().Create(&oldEvent).Error; err != nil {
		t.Fatal(err)
	}

	router, cookies := newAuthenticatedTestRouter(t, settingService, func(router *gin.Engine) {
		router.GET("/api/security/audit", (&ApiService{}).GetSecurityAudit)
	})
	recorder := performAuthenticatedTestRequest(router, httptest.NewRequest(http.MethodGet, "/api/security/audit?limit=10", nil), cookies...)
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", recorder.Code)
	}
	var count int64
	if err := database.GetDB().Model(model.AuditEvent{}).Where("event = ?", "old").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("audit read pruned old events, count=%d", count)
	}
}

func TestGetSecurityAuditPaginatesByCursorAndCapsLimit(t *testing.T) {
	resetRateLimitState()
	settingService := initSessionTestDB(t)
	now := time.Now().Unix()
	for i := 0; i < 3; i++ {
		if err := database.GetDB().Create(&model.AuditEvent{
			DateTime: now + int64(i),
			Actor:    "admin",
			Event:    "event",
		}).Error; err != nil {
			t.Fatal(err)
		}
	}

	router, cookies := newAuthenticatedTestRouter(t, settingService, func(router *gin.Engine) {
		router.GET("/api/security/audit", (&ApiService{}).GetSecurityAudit)
	})
	recorder := performAuthenticatedTestRequest(router, httptest.NewRequest(http.MethodGet, "/api/security/audit?limit=2", nil), cookies...)
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", recorder.Code)
	}
	var msg Msg
	if err := json.Unmarshal(recorder.Body.Bytes(), &msg); err != nil {
		t.Fatal(err)
	}
	payload, ok := msg.Obj.(map[string]any)
	if !ok {
		t.Fatalf("unexpected payload: %#v", msg.Obj)
	}
	events, ok := payload["events"].([]any)
	if !ok || len(events) != 2 {
		t.Fatalf("expected two events, got %#v", payload["events"])
	}
	nextCursor, ok := payload["nextCursor"].(float64)
	if !ok || nextCursor == 0 {
		t.Fatalf("expected next cursor, got %#v", payload["nextCursor"])
	}

	recorder = performAuthenticatedTestRequest(router, httptest.NewRequest(http.MethodGet, "/api/security/audit?limit=500&cursor="+strconv.FormatUint(uint64(nextCursor), 10), nil), cookies...)
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", recorder.Code)
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &msg); err != nil {
		t.Fatal(err)
	}
	payload, ok = msg.Obj.(map[string]any)
	if !ok {
		t.Fatalf("unexpected payload: %#v", msg.Obj)
	}
	if payload["limit"].(float64) != 200 {
		t.Fatalf("limit was not capped: %#v", payload["limit"])
	}
	events, ok = payload["events"].([]any)
	if !ok || len(events) != 1 {
		t.Fatalf("expected one event after cursor, got %#v", payload["events"])
	}
}

func TestGetSecurityAuditFiltersEventAndSeverity(t *testing.T) {
	resetRateLimitState()
	settingService := initSessionTestDB(t)
	now := time.Now().Unix()
	events := []model.AuditEvent{
		{DateTime: now, Actor: "admin", Event: "telegram_test", Severity: "warn"},
		{DateTime: now + 1, Actor: "admin", Event: "telegram_test", Severity: "info"},
		{DateTime: now + 2, Actor: "admin", Event: "login_success", Severity: "info"},
	}
	if err := database.GetDB().Create(&events).Error; err != nil {
		t.Fatal(err)
	}

	router, cookies := newAuthenticatedTestRouter(t, settingService, func(router *gin.Engine) {
		router.GET("/api/security/audit", (&ApiService{}).GetSecurityAudit)
	})
	recorder := performAuthenticatedTestRequest(router, httptest.NewRequest(http.MethodGet, "/api/security/audit?event=telegram_test&severity=warn", nil), cookies...)
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", recorder.Code)
	}
	var msg Msg
	if err := json.Unmarshal(recorder.Body.Bytes(), &msg); err != nil {
		t.Fatal(err)
	}
	payload, ok := msg.Obj.(map[string]any)
	if !ok {
		t.Fatalf("unexpected payload: %#v", msg.Obj)
	}
	gotEvents, ok := payload["events"].([]any)
	if !ok || len(gotEvents) != 1 {
		t.Fatalf("expected one filtered event, got %#v", payload["events"])
	}
	got := gotEvents[0].(map[string]any)
	if got["event"] != "telegram_test" || got["severity"] != "warn" {
		t.Fatalf("unexpected filtered event: %#v", got)
	}

	recorder = performAuthenticatedTestRequest(router, httptest.NewRequest(http.MethodGet, "/api/security/audit?event=telegram%20test", nil), cookies...)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("invalid event filter should fail, got %d", recorder.Code)
	}
}

func TestGetSecurityAuditFiltersDateRange(t *testing.T) {
	resetRateLimitState()
	settingService := initSessionTestDB(t)
	now := time.Now().Unix()
	events := []model.AuditEvent{
		{DateTime: now - 10, Actor: "admin", Event: "before", Severity: "info"},
		{DateTime: now, Actor: "admin", Event: "inside", Severity: "info"},
		{DateTime: now + 10, Actor: "admin", Event: "after", Severity: "info"},
	}
	if err := database.GetDB().Create(&events).Error; err != nil {
		t.Fatal(err)
	}

	router, cookies := newAuthenticatedTestRouter(t, settingService, func(router *gin.Engine) {
		router.GET("/api/security/audit", (&ApiService{}).GetSecurityAudit)
	})
	recorder := performAuthenticatedTestRequest(router, httptest.NewRequest(http.MethodGet, "/api/security/audit?since="+strconv.FormatInt(now, 10)+"&until="+strconv.FormatInt(now+5, 10), nil), cookies...)
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", recorder.Code)
	}
	var msg Msg
	if err := json.Unmarshal(recorder.Body.Bytes(), &msg); err != nil {
		t.Fatal(err)
	}
	payload, ok := msg.Obj.(map[string]any)
	if !ok {
		t.Fatalf("unexpected payload: %#v", msg.Obj)
	}
	gotEvents, ok := payload["events"].([]any)
	if !ok || len(gotEvents) != 1 {
		t.Fatalf("expected one date-filtered event, got %#v", payload["events"])
	}
	got := gotEvents[0].(map[string]any)
	if got["event"] != "inside" {
		t.Fatalf("unexpected date-filtered event: %#v", got)
	}

	recorder = performAuthenticatedTestRequest(router, httptest.NewRequest(http.MethodGet, "/api/security/audit?since=17000000000", nil), cookies...)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("oversized since filter should fail, got %d", recorder.Code)
	}
	recorder = performAuthenticatedTestRequest(router, httptest.NewRequest(http.MethodGet, "/api/security/audit?until=170000000a", nil), cookies...)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("non-digit until filter should fail, got %d", recorder.Code)
	}
}

func TestParseAuditUnixSecondsFilter(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    int64
		wantErr bool
	}{
		{name: "empty", raw: "", want: 0},
		{name: "zero", raw: "0", want: 0},
		{name: "seconds", raw: "1700000000", want: 1700000000},
		{name: "too long", raw: "17000000000", wantErr: true},
		{name: "alpha", raw: "170000000a", wantErr: true},
		{name: "negative", raw: "-1", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseAuditUnixSecondsFilter("since", tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("expected %d, got %d", tt.want, got)
			}
		})
	}
}

func TestGetSecurityAuditScopeMatrix(t *testing.T) {
	tests := []struct {
		name       string
		scope      string
		hasScope   bool
		wantStatus int
	}{
		{name: "cookie session without scope", wantStatus: http.StatusOK},
		{name: "admin bearer scope", scope: "admin", hasScope: true, wantStatus: http.StatusOK},
		{name: "read bearer scope", scope: "read", hasScope: true, wantStatus: http.StatusForbidden},
		{name: "write bearer scope", scope: "write", hasScope: true, wantStatus: http.StatusForbidden},
		{name: "observability bearer scope", scope: "observability", hasScope: true, wantStatus: http.StatusForbidden},
		{name: "unknown bearer scope", scope: "unknown", hasScope: true, wantStatus: http.StatusForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetRateLimitState()
			settingService := initSessionTestDB(t)
			handler := (&ApiService{}).GetSecurityAudit
			if tt.hasScope {
				handler = withTestTokenScope("api-user", tt.scope, handler)
			}
			router, cookies := newAuthenticatedTestRouter(t, settingService, func(router *gin.Engine) {
				router.GET("/api/security/audit", handler)
			})
			recorder := performAuthenticatedTestRequest(router, httptest.NewRequest(http.MethodGet, "/api/security/audit", nil), cookies...)
			if recorder.Code != tt.wantStatus {
				t.Fatalf("unexpected status: %d", recorder.Code)
			}
			if tt.wantStatus != http.StatusForbidden {
				return
			}
			var event model.AuditEvent
			if err := database.GetDB().Where("event = ?", "audit_scope_denied").First(&event).Error; err != nil {
				t.Fatal(err)
			}
			if event.Actor != "api-user" {
				t.Fatalf("unexpected actor: %q", event.Actor)
			}
		})
	}
}

func TestAuditAdminScopeAllowed(t *testing.T) {
	tests := []struct {
		name     string
		scope    string
		hasScope bool
		want     bool
	}{
		{name: "cookie session", want: true},
		{name: "admin bearer", scope: "admin", hasScope: true, want: true},
		{name: "read bearer", scope: "read", hasScope: true, want: false},
		{name: "write bearer", scope: "write", hasScope: true, want: false},
		{name: "observability bearer", scope: "observability", hasScope: true, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := auditAdminScopeAllowed(tt.scope, tt.hasScope); got != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetSecurityAuditRateLimitReturns429AndAudits(t *testing.T) {
	resetRateLimitState()
	settingService := initSessionTestDB(t)
	const ip = "198.51.100.10"
	for i := 0; i < auditEndpointRateLimitMax; i++ {
		if err := checkAuditEndpointRateLimit(auditEndpointRateLimitKey("admin", ip)); err != nil {
			t.Fatalf("unexpected prefill error: %v", err)
		}
	}
	router, cookies := newAuthenticatedTestRouter(t, settingService, func(router *gin.Engine) {
		router.GET("/api/security/audit", withTestTokenScope("admin", "admin", (&ApiService{}).GetSecurityAudit))
	})
	req := httptest.NewRequest(http.MethodGet, "/api/security/audit", nil)
	req.RemoteAddr = ip + ":1234"
	recorder := performAuthenticatedTestRequest(router, req, cookies...)
	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("unexpected status: %d", recorder.Code)
	}
	if recorder.Header().Get("Retry-After") == "" {
		t.Fatal("missing retry-after header")
	}
	var event model.AuditEvent
	if err := database.GetDB().Where("event = ?", "audit_rate_limited").First(&event).Error; err != nil {
		t.Fatal(err)
	}
	if event.Actor != "admin" {
		t.Fatalf("unexpected actor: %q", event.Actor)
	}
	var details map[string]any
	if err := json.Unmarshal(event.Details, &details); err != nil {
		t.Fatal(err)
	}
	if details["ip"] != ip {
		t.Fatalf("unexpected audit ip detail: %#v", details)
	}
}

func TestGetSecurityAuditRateLimitKeyUsesActorAndCanonicalIP(t *testing.T) {
	resetRateLimitState()
	settingService := initSessionTestDB(t)
	for i := 0; i < auditEndpointRateLimitMax; i++ {
		if err := checkAuditEndpointRateLimit(auditEndpointRateLimitKey("admin", "198.51.100.10")); err != nil {
			t.Fatalf("unexpected prefill error: %v", err)
		}
	}
	router, cookies := newAuthenticatedTestRouter(t, settingService, func(router *gin.Engine) {
		router.GET("/api/security/audit", withTestTokenScope("admin", "admin", (&ApiService{}).GetSecurityAudit))
	})

	blockedReq := httptest.NewRequest(http.MethodGet, "/api/security/audit", nil)
	blockedReq.RemoteAddr = "[::ffff:198.51.100.10]:1234"
	blocked := performAuthenticatedTestRequest(router, blockedReq, cookies...)
	if blocked.Code != http.StatusTooManyRequests {
		t.Fatalf("expected mapped-ip request to be blocked, got %d", blocked.Code)
	}

	allowedReq := httptest.NewRequest(http.MethodGet, "/api/security/audit", nil)
	allowedReq.RemoteAddr = "198.51.100.11:1234"
	allowed := performAuthenticatedTestRequest(router, allowedReq, cookies...)
	if allowed.Code != http.StatusOK {
		t.Fatalf("expected different ip bucket to pass, got %d", allowed.Code)
	}

	otherActorRouter, otherActorCookies := newAuthenticatedTestRouter(t, settingService, func(router *gin.Engine) {
		router.GET("/api/security/audit", withTestTokenScope("other-admin", "admin", (&ApiService{}).GetSecurityAudit))
	})
	otherActorReq := httptest.NewRequest(http.MethodGet, "/api/security/audit", nil)
	otherActorReq.RemoteAddr = "198.51.100.10:1234"
	otherActor := performAuthenticatedTestRequest(otherActorRouter, otherActorReq, otherActorCookies...)
	if otherActor.Code != http.StatusOK {
		t.Fatalf("expected different actor bucket to pass, got %d", otherActor.Code)
	}
}

func TestAPIV2SecurityAuditRequiresAdminScope(t *testing.T) {
	resetRateLimitState()
	initSessionTestDB(t)
	readToken, err := (&service.UserService{}).AddToken("admin", 0, "read audit", "read")
	if err != nil {
		t.Fatal(err)
	}
	adminToken, err := (&service.UserService{}).AddToken("admin", 0, "admin audit", "admin")
	if err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Create(&model.AuditEvent{
		DateTime: time.Now().Unix(),
		Actor:    "admin",
		Event:    "login_success",
	}).Error; err != nil {
		t.Fatal(err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewAPIv2Handler(router.Group("/apiv2"))

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/apiv2/security/audit", nil)
	req.Header.Set("Authorization", "Bearer "+readToken)
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("read token should be forbidden, got %d", recorder.Code)
	}

	recorder = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/apiv2/security/audit?limit=1", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("admin token should be allowed, got %d", recorder.Code)
	}
	var msg Msg
	if err := json.Unmarshal(recorder.Body.Bytes(), &msg); err != nil {
		t.Fatal(err)
	}
	if !msg.Success {
		t.Fatalf("admin audit request failed: %s", msg.Msg)
	}
}
