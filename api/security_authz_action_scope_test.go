package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestSecurityAuthZAPIV2ActionScopeGate pins the M1 fix: the apiv2 action
// dispatchers must enforce per-action token scope. Before the fix, a token with
// any valid scope (read / observability / telegram / ...) could POST
// /apiv2/save, /apiv2/restartApp|restartSb and GET /apiv2/settings because the
// dispatchers called the handlers with no scope check at all.
func TestSecurityAuthZAPIV2ActionScopeGate(t *testing.T) {
	initSessionTestDB(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := NewAPIv2Handler(router.Group("/apiv2"))

	// allowed lists exclude "admin", which must always be permitted.
	cases := []struct {
		action  string
		allowed []string
	}{
		{"save", []string{"write"}},
		{"restartApp", []string{"write"}},
		{"restartSb", []string{"write"}},
		{"checkOutbound", []string{"write"}},
		{"linkConvert", []string{"read", "write"}},
		{"settings", []string{"read", "write"}},
		{"users", []string{"read", "write"}},
		{"keypairs", []string{"read", "write"}},
		{"clients", []string{"read", "write"}},
		{"stats", []string{"read", "write", "observability"}},
		{"status", []string{"read", "write", "observability"}},
		{"logs", []string{"read", "write", "observability"}},
	}
	allScopes := []string{"read", "write", "database", "telegram", "observability"}

	for _, tc := range cases {
		t.Run(tc.action, func(t *testing.T) {
			if code := runActionScopeGate(h, tc.action, "admin"); code != http.StatusOK {
				t.Fatalf("admin must be allowed on %q, got status %d", tc.action, code)
			}
			for _, sc := range tc.allowed {
				if code := runActionScopeGate(h, tc.action, sc); code != http.StatusOK {
					t.Fatalf("allowed scope %q must pass on %q, got status %d", sc, tc.action, code)
				}
			}
			for _, sc := range allScopes {
				if scopeSliceContains(tc.allowed, sc) {
					continue
				}
				if code := runActionScopeGate(h, tc.action, sc); code != http.StatusForbidden {
					t.Fatalf("disallowed scope %q on %q status=%d, want 403", sc, tc.action, code)
				}
			}
		})
	}

	// Self-gated actions (getdb/importdb/rotateSubSecret) must NOT be blocked by
	// enforceActionScope — they enforce their own scope inside the handler, so
	// the dispatcher gate has to pass them through regardless of scope.
	if code := runActionScopeGate(h, "getdb", "telegram"); code != http.StatusOK {
		t.Fatalf("self-gated action getdb must pass enforceActionScope, got %d", code)
	}
}

// TestSecurityAuthZAPIV2BrowserSessionBypassesActionScope pins the dependency the
// gate relies on: a browser session carries no token scope, so enforceActionScope
// must allow even the most gated actions (the gate only constrains API tokens).
func TestSecurityAuthZAPIV2BrowserSessionBypassesActionScope(t *testing.T) {
	initSessionTestDB(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := NewAPIv2Handler(router.Group("/apiv2"))

	for _, action := range []string{"save", "restartApp", "settings", "users", "stats"} {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = httptest.NewRequest(http.MethodGet, "/apiv2/"+action, nil)
		c.Set(apiUsernameKey, "session-admin")
		// Deliberately do NOT set apiTokenScopeKey (browser session).
		if !h.enforceActionScope(c, action) {
			t.Fatalf("browser session must bypass the action-scope gate for %q, got status %d", action, rec.Code)
		}
	}
}

func runActionScopeGate(h *APIv2Handler, action, scope string) int {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/apiv2/"+action, nil)
	c.Set(apiUsernameKey, "api-user")
	c.Set(apiTokenScopeKey, scope)
	if !h.enforceActionScope(c, action) {
		return rec.Code
	}
	c.Status(http.StatusOK)
	return rec.Code
}

func scopeSliceContains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
