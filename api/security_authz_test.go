package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
	"github.com/deposist/s-ui-x/service"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

type securityAuthZRow struct {
	method     string
	path       string
	resource   string
	allowed    []string
	auditAdmin bool
}

func securityAuthZScopeRows() []securityAuthZRow {
	return []securityAuthZRow{
		{method: http.MethodGet, path: "/apiv2/security/audit", auditAdmin: true},
		{method: http.MethodPost, path: "/apiv2/rotateSubSecret", resource: "client", allowed: []string{"admin", "write"}},
		{method: http.MethodPost, path: "/apiv2/telegram/test", resource: "telegram", allowed: []string{"admin"}},
		{method: http.MethodPost, path: "/apiv2/telegram/backup", resource: "telegram", allowed: []string{"telegram", "admin"}},
		{method: http.MethodPost, path: "/apiv2/telegram/backup/run", resource: "telegram", allowed: []string{"telegram", "admin"}},
		{method: http.MethodPost, path: "/apiv2/import-xui/plan", resource: "database", allowed: []string{"admin", "database"}},
		{method: http.MethodPost, path: "/apiv2/import-xui/apply", resource: "database", allowed: []string{"admin", "database"}},
		{method: http.MethodPost, path: "/apiv2/import-xui/rollback", resource: "database", allowed: []string{"admin", "database"}},
		{method: http.MethodGet, path: "/apiv2/import-xui/reports", resource: "database", allowed: []string{"admin", "database"}},
		{method: http.MethodPost, path: "/apiv2/importdb", resource: "database", allowed: []string{"admin", "database"}},
		{method: http.MethodPost, path: "/apiv2/import-xui", resource: "database", allowed: []string{"admin", "database"}},
		{method: http.MethodGet, path: "/apiv2/getdb", resource: "database", allowed: []string{"admin", "database"}},
	}
}

func TestSecurityAuthZScopeMatrixRows(t *testing.T) {
	initSessionTestDB(t)
	serviceUnderTest := &ApiService{}
	for _, row := range securityAuthZScopeRows() {
		t.Run(row.method+" "+row.path, func(t *testing.T) {
			wrongScope := firstDisallowedScope(row.allowed, row.auditAdmin)
			if got := runSecurityScopeGate(serviceUnderTest, row, wrongScope); got != http.StatusForbidden {
				t.Fatalf("wrong scope %q status=%d, want 403", wrongScope, got)
			}
			correctScope := "admin"
			if len(row.allowed) > 0 {
				correctScope = row.allowed[0]
			}
			if got := runSecurityScopeGate(serviceUnderTest, row, correctScope); got != http.StatusOK {
				t.Fatalf("correct scope %q status=%d, want 200", correctScope, got)
			}
		})
	}
}

func runSecurityScopeGate(serviceUnderTest *ApiService, row securityAuthZRow, scope string) int {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(row.method, row.path, nil)
	c.Set(apiUsernameKey, "api-user")
	c.Set(apiTokenScopeKey, scope)
	if row.auditAdmin {
		if !serviceUnderTest.requireAuditAdminScope(c) {
			return recorder.Code
		}
	} else if !serviceUnderTest.requireTokenScopeAny(c, row.resource, row.allowed...) {
		return recorder.Code
	}
	c.Status(http.StatusOK)
	return recorder.Code
}

func firstDisallowedScope(allowed []string, auditAdmin bool) string {
	for _, candidate := range []string{"read", "write", "database", "telegram", "observability", "admin"} {
		if auditAdmin {
			if candidate != "admin" {
				return candidate
			}
			continue
		}
		found := false
		for _, allowedScope := range allowed {
			if candidate == allowedScope {
				found = true
				break
			}
		}
		if !found {
			return candidate
		}
	}
	return "read"
}

func TestSecurityAuthZAPIV2InvalidAndExpiredTokenCurrentStatus(t *testing.T) {
	initSessionTestDB(t)
	expiredPlain := "expired-security-token"
	expiredHash, err := (&service.UserService{}).HashAPIToken(expiredPlain)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Create(&model.Tokens{
		Desc:        "expired",
		TokenHash:   expiredHash,
		TokenPrefix: "expired-",
		Scope:       "admin",
		Enabled:     true,
		Expiry:      time.Now().Add(-time.Minute).Unix(),
		UserId:      1,
	}).Error; err != nil {
		t.Fatal(err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewAPIv2Handler(router.Group("/apiv2"))

	for _, req := range []*http.Request{
		httptest.NewRequest(http.MethodGet, "/apiv2/settings", nil),
		httptest.NewRequest(http.MethodGet, "/apiv2/settings", nil),
	} {
		if req.Header.Get("Authorization") == "" && strings.Contains(req.URL.RawQuery, "expired") {
			req.Header.Set("Authorization", "Bearer "+expiredPlain)
		}
	}
	missing := httptest.NewRequest(http.MethodGet, "/apiv2/settings", nil)
	assertAPIV2TokenFailureStatus(t, router, missing)
	expired := httptest.NewRequest(http.MethodGet, "/apiv2/settings?expired=1", nil)
	expired.Header.Set("Authorization", "Bearer "+expiredPlain)
	assertAPIV2TokenFailureStatus(t, router, expired)
}

func assertAPIV2TokenFailureStatus(t *testing.T, router *gin.Engine, req *http.Request) {
	t.Helper()
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("invalid-token status=%d body=%s, want 401", recorder.Code, recorder.Body.String())
	}
	var msg Msg
	if err := json.Unmarshal(recorder.Body.Bytes(), &msg); err != nil {
		t.Fatal(err)
	}
	if msg.Success {
		t.Fatalf("invalid token unexpectedly succeeded: %#v", msg)
	}
}

func TestSecurityAuthZImportXUISharedRegistryPreservesAuthSurfacesIssue35(t *testing.T) {
	initSessionTestDB(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(sessions.Sessions("s-ui", cookie.NewStore([]byte("test-secret"))))
	apiv2 := NewAPIv2Handler(router.Group("/apiv2"))
	NewAPIHandler(router.Group("/api"), apiv2)

	v1 := httptest.NewRecorder()
	router.ServeHTTP(v1, httptest.NewRequest(http.MethodPost, "/api/import-xui/plan", strings.NewReader("")))
	v2 := httptest.NewRecorder()
	router.ServeHTTP(v2, httptest.NewRequest(http.MethodPost, "/apiv2/import-xui/plan", strings.NewReader("")))

	if v1.Code != http.StatusTemporaryRedirect {
		t.Fatalf("unexpected v1 unauthenticated session surface: status=%d body=%s", v1.Code, v1.Body.String())
	}
	if v2.Code != http.StatusUnauthorized || !strings.Contains(v2.Body.String(), "invalid token") {
		t.Fatalf("unexpected v2 unauthenticated contract: status=%d body=%s", v2.Code, v2.Body.String())
	}
}
