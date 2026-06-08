package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x/service"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

func newCSRFTestRouter(t *testing.T, settingService *service.SettingService) *gin.Engine {
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
	handler := &APIHandler{}
	handler.initRouter(router.Group("/api"))
	return router
}

func performCSRFRequest(router *gin.Engine, method string, path string, token string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, strings.NewReader(""))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if token != "" {
		req.Header.Set(csrfHeader, token)
	}
	for _, c := range cookies {
		req.AddCookie(c)
	}
	router.ServeHTTP(recorder, req)
	return recorder
}

func TestCSRFMiddlewareRequiresTokenForMutatingBrowserAPI(t *testing.T) {
	settingService := initSessionTestDB(t)
	router := newCSRFTestRouter(t, settingService)

	login := performCSRFRequest(router, http.MethodGet, "/login", "")
	if login.Code != http.StatusNoContent {
		t.Fatalf("login returned %d", login.Code)
	}

	missing := performCSRFRequest(router, http.MethodPost, "/api/logoutAllAdmins", "", login.Result().Cookies()...)
	if missing.Code != http.StatusForbidden {
		t.Fatalf("missing csrf token should return 403, got %d", missing.Code)
	}

	csrf := performCSRFRequest(router, http.MethodGet, "/api/csrf", "", login.Result().Cookies()...)
	if csrf.Code != http.StatusOK {
		t.Fatalf("csrf endpoint returned %d", csrf.Code)
	}
	var msg Msg
	if err := json.Unmarshal(csrf.Body.Bytes(), &msg); err != nil {
		t.Fatal(err)
	}
	obj, ok := msg.Obj.(map[string]any)
	if !ok {
		t.Fatalf("unexpected csrf response obj: %#v", msg.Obj)
	}
	token, ok := obj["token"].(string)
	if !ok || token == "" {
		t.Fatalf("csrf token missing in response: %#v", obj)
	}

	accepted := performCSRFRequest(router, http.MethodPost, "/api/logoutAllAdmins", token, csrf.Result().Cookies()...)
	if accepted.Code != http.StatusOK {
		t.Fatalf("valid csrf token should allow request, got %d", accepted.Code)
	}
}

func TestCSRFCookieSecureForcedByEnv(t *testing.T) {
	settingService := initSessionTestDB(t)
	t.Setenv("SUI_FORCE_COOKIE_SECURE", "true")
	router := newCSRFTestRouter(t, settingService)

	login := performCSRFRequest(router, http.MethodGet, "/login", "")
	if login.Code != http.StatusNoContent {
		t.Fatalf("login returned %d", login.Code)
	}
	csrf := performCSRFRequest(router, http.MethodGet, "/api/csrf", "", login.Result().Cookies()...)
	if csrf.Code != http.StatusOK {
		t.Fatalf("csrf endpoint returned %d", csrf.Code)
	}
	cookie := findCookieByName(csrf.Result().Cookies())
	if cookie == nil {
		t.Fatal("csrf did not set s-ui cookie")
	}
	if !cookie.Secure {
		t.Fatal("csrf session cookie must be Secure when SUI_FORCE_COOKIE_SECURE=true")
	}
}

func TestCSRFExemptPathOnlyAllowsAPILogin(t *testing.T) {
	tests := []struct {
		name string
		base string
		path string
		want bool
	}{
		{name: "exact base path", base: "/app/", path: "/app/api/login", want: true},
		{name: "suffix match rejected", base: "/app/", path: "/api/login", want: false},
		{name: "empty path rejected", base: "/app/", path: "", want: false},
		{name: "without base url", base: "/", path: "/api/login", want: true},
		{name: "similar suffix rejected", base: "/", path: "/api/sublogin", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := csrfExemptPath(tt.path, csrfLoginPathForBase(tt.base)); got != tt.want {
				t.Fatalf("csrfExemptPath(%q, %q)=%v, want %v", tt.path, csrfLoginPathForBase(tt.base), got, tt.want)
			}
		})
	}
}

func TestLoginResetsCSRFTokens(t *testing.T) {
	settingService := initSessionTestDB(t)
	router := newCSRFTestRouter(t, settingService)

	login := performCSRFRequest(router, http.MethodGet, "/login", "")
	if login.Code != http.StatusNoContent {
		t.Fatalf("login returned %d", login.Code)
	}
	cookies := login.Result().Cookies()

	csrfBefore := performCSRFRequest(router, http.MethodGet, "/api/csrf", "", cookies...)
	if csrfBefore.Code != http.StatusOK {
		t.Fatalf("csrf before relogin returned %d", csrfBefore.Code)
	}
	var before Msg
	if err := json.Unmarshal(csrfBefore.Body.Bytes(), &before); err != nil {
		t.Fatal(err)
	}
	beforeObj, ok := before.Obj.(map[string]any)
	if !ok {
		t.Fatalf("unexpected csrf before payload: %#v", before.Obj)
	}
	oldToken, ok := beforeObj["token"].(string)
	if !ok || oldToken == "" {
		t.Fatalf("missing csrf before token: %#v", beforeObj)
	}
	cookies = appendUpdatedCSRFCookies(cookies, csrfBefore.Result().Cookies())

	relogin := performCSRFRequest(router, http.MethodGet, "/login", "", cookies...)
	if relogin.Code != http.StatusNoContent {
		t.Fatalf("relogin returned %d", relogin.Code)
	}
	cookies = appendUpdatedCSRFCookies(cookies, relogin.Result().Cookies())

	stale := performCSRFRequest(router, http.MethodPost, "/api/logoutAllAdmins", oldToken, cookies...)
	if stale.Code != http.StatusForbidden {
		t.Fatalf("stale csrf token after relogin should be rejected, got %d", stale.Code)
	}

	csrfAfter := performCSRFRequest(router, http.MethodGet, "/api/csrf", "", cookies...)
	if csrfAfter.Code != http.StatusOK {
		t.Fatalf("csrf after relogin returned %d", csrfAfter.Code)
	}
	var after Msg
	if err := json.Unmarshal(csrfAfter.Body.Bytes(), &after); err != nil {
		t.Fatal(err)
	}
	afterObj, ok := after.Obj.(map[string]any)
	if !ok {
		t.Fatalf("unexpected csrf after payload: %#v", after.Obj)
	}
	newToken, ok := afterObj["token"].(string)
	if !ok || newToken == "" {
		t.Fatalf("missing csrf after token: %#v", afterObj)
	}
	if newToken == oldToken {
		t.Fatal("csrf token must rotate after relogin")
	}
	cookies = appendUpdatedCSRFCookies(cookies, csrfAfter.Result().Cookies())

	accepted := performCSRFRequest(router, http.MethodPost, "/api/logoutAllAdmins", newToken, cookies...)
	if accepted.Code != http.StatusOK {
		t.Fatalf("new csrf token should be accepted, got %d", accepted.Code)
	}
}

func TestRotateSessionGenerationRejectsOldCSRFToken(t *testing.T) {
	settingService := initSessionTestDB(t)
	router := newCSRFTestRouter(t, settingService)

	login := performCSRFRequest(router, http.MethodGet, "/login", "")
	if login.Code != http.StatusNoContent {
		t.Fatalf("login returned %d", login.Code)
	}
	cookies := login.Result().Cookies()

	csrf := performCSRFRequest(router, http.MethodGet, "/api/csrf", "", cookies...)
	if csrf.Code != http.StatusOK {
		t.Fatalf("csrf endpoint returned %d", csrf.Code)
	}
	var msg Msg
	if err := json.Unmarshal(csrf.Body.Bytes(), &msg); err != nil {
		t.Fatal(err)
	}
	obj, ok := msg.Obj.(map[string]any)
	if !ok {
		t.Fatalf("unexpected csrf payload: %#v", msg.Obj)
	}
	oldToken, ok := obj["token"].(string)
	if !ok || oldToken == "" {
		t.Fatalf("missing csrf token: %#v", obj)
	}
	cookies = appendUpdatedCSRFCookies(cookies, csrf.Result().Cookies())

	if _, err := settingService.RotateSessionGeneration(); err != nil {
		t.Fatal(err)
	}

	stale := performCSRFRequest(router, http.MethodPost, "/api/logoutAllAdmins", oldToken, cookies...)
	if stale.Code == http.StatusOK {
		t.Fatal("old csrf token from rotated session should not be accepted")
	}

	newLogin := performCSRFRequest(router, http.MethodGet, "/login", "")
	if newLogin.Code != http.StatusNoContent {
		t.Fatalf("new login returned %d", newLogin.Code)
	}
	newCSRF := performCSRFRequest(router, http.MethodGet, "/api/csrf", "", newLogin.Result().Cookies()...)
	if newCSRF.Code != http.StatusOK {
		t.Fatalf("new csrf endpoint returned %d", newCSRF.Code)
	}
	var newMsg Msg
	if err := json.Unmarshal(newCSRF.Body.Bytes(), &newMsg); err != nil {
		t.Fatal(err)
	}
	newObj, ok := newMsg.Obj.(map[string]any)
	if !ok {
		t.Fatalf("unexpected new csrf payload: %#v", newMsg.Obj)
	}
	newToken, ok := newObj["token"].(string)
	if !ok || newToken == "" {
		t.Fatalf("missing new csrf token: %#v", newObj)
	}
	newCookies := appendUpdatedCSRFCookies(newLogin.Result().Cookies(), newCSRF.Result().Cookies())
	accepted := performCSRFRequest(router, http.MethodPost, "/api/logoutAllAdmins", newToken, newCookies...)
	if accepted.Code != http.StatusOK {
		t.Fatalf("new csrf token should be accepted, got %d", accepted.Code)
	}
}

func appendUpdatedCSRFCookies(base []*http.Cookie, updates []*http.Cookie) []*http.Cookie {
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
