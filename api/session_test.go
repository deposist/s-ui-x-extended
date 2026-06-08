package api

import (
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
	"github.com/deposist/s-ui-x/service"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

func initSessionTestDB(t *testing.T) *service.SettingService {
	t.Helper()
	saveDedup.reset() // isolate the global create-dedup cache between tests
	prevAuditSync := service.AuditSyncForTest
	service.AuditSyncForTest = true
	t.Cleanup(func() { service.AuditSyncForTest = prevAuditSync })
	t.Setenv("SUI_DB_FOLDER", t.TempDir())
	initAPITestDB(t, filepath.Join(t.TempDir(), "s-ui.db"))
	testDB := database.GetDB()
	t.Cleanup(func() {
		stopTokenUseDebouncerBeforeAPITestDBInit(t)
		if testDB != nil {
			if sqlDB, err := testDB.DB(); err == nil {
				_ = sqlDB.Close()
				time.Sleep(25 * time.Millisecond)
			}
		}
	})
	return &service.SettingService{}
}

func newSessionTestRouter(t *testing.T, settingService *service.SettingService) *gin.Engine {
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
	router.GET("/protected", func(c *gin.Context) {
		if GetLoginUser(c) != "admin" {
			c.Status(http.StatusUnauthorized)
			return
		}
		c.Status(http.StatusNoContent)
	})
	router.GET("/logout", func(c *gin.Context) {
		ClearSession(c)
		c.Status(http.StatusNoContent)
	})
	return router
}

func performSessionRequest(router *gin.Engine, path string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	router.ServeHTTP(recorder, req)
	return recorder
}

func findCookieByName(cookies []*http.Cookie) *http.Cookie {
	for _, c := range cookies {
		if c.Name == "s-ui" {
			return c
		}
	}
	return nil
}

func TestSessionCookieSecureForcedByEnv(t *testing.T) {
	settingService := initSessionTestDB(t)
	t.Setenv("SUI_FORCE_COOKIE_SECURE", "true")
	router := newSessionTestRouter(t, settingService)

	login := performSessionRequest(router, "/login")
	if login.Code != http.StatusNoContent {
		t.Fatalf("login returned %d", login.Code)
	}
	cookie := findCookieByName(login.Result().Cookies())
	if cookie == nil {
		t.Fatal("login did not set s-ui cookie")
	}
	if !cookie.Secure {
		t.Fatal("session cookie must be Secure when SUI_FORCE_COOKIE_SECURE=true")
	}
}

func TestSessionCookieSecureAutoFromWebURI(t *testing.T) {
	settingService := initSessionTestDB(t)
	payload, err := json.Marshal(map[string]string{"webURI": "https://panel.example.com/app/"})
	if err != nil {
		t.Fatal(err)
	}
	if err := settingService.Save(database.GetDB(), payload); err != nil {
		t.Fatal(err)
	}
	router := newSessionTestRouter(t, settingService)

	login := performSessionRequest(router, "/login")
	if login.Code != http.StatusNoContent {
		t.Fatalf("login returned %d", login.Code)
	}
	cookie := findCookieByName(login.Result().Cookies())
	if cookie == nil {
		t.Fatal("login did not set s-ui cookie")
	}
	if !cookie.Secure {
		t.Fatal("session cookie must be Secure when webURI starts with https://")
	}
}

func TestClearSessionCookieSecureForcedByEnv(t *testing.T) {
	settingService := initSessionTestDB(t)
	t.Setenv("SUI_FORCE_COOKIE_SECURE", "true")
	router := newSessionTestRouter(t, settingService)

	login := performSessionRequest(router, "/login")
	if login.Code != http.StatusNoContent {
		t.Fatalf("login returned %d", login.Code)
	}
	logout := performSessionRequest(router, "/logout", login.Result().Cookies()...)
	if logout.Code != http.StatusNoContent {
		t.Fatalf("logout returned %d", logout.Code)
	}
	cookie := findCookieByName(logout.Result().Cookies())
	if cookie == nil {
		t.Fatal("logout did not set s-ui cookie")
	}
	if !cookie.Secure {
		t.Fatal("logout cookie must be Secure when SUI_FORCE_COOKIE_SECURE=true")
	}
}

func TestResolveCookieSecureMatrix(t *testing.T) {
	tests := []struct {
		name    string
		env     string
		remote  string
		proxies string
		proto   string
		tls     bool
		setup   func(t *testing.T, settingService *service.SettingService)
		want    bool
	}{
		{
			name:   "plain http default",
			remote: "198.51.100.10:1234",
		},
		{
			name:   "env forced",
			env:    "true",
			remote: "198.51.100.10:1234",
			want:   true,
		},
		{
			name:   "request tls",
			remote: "198.51.100.10:1234",
			tls:    true,
			want:   true,
		},
		{
			name:    "trusted proxy https",
			remote:  "192.0.2.1:1234",
			proxies: "192.0.2.1",
			proto:   "https",
			want:    true,
		},
		{
			name:   "webURI https",
			remote: "198.51.100.10:1234",
			setup: func(t *testing.T, settingService *service.SettingService) {
				t.Helper()
				payload, err := json.Marshal(map[string]string{"webURI": "https://panel.example.com/app/"})
				if err != nil {
					t.Fatal(err)
				}
				if err := settingService.Save(database.GetDB(), payload); err != nil {
					t.Fatal(err)
				}
			},
			want: true,
		},
		{
			name:   "webDomain https",
			remote: "198.51.100.10:1234",
			setup: func(t *testing.T, _ *service.SettingService) {
				t.Helper()
				if err := database.GetDB().Model(model.Setting{}).Where("key = ?", "webDomain").Update("value", "https://panel.example.com").Error; err != nil {
					t.Fatal(err)
				}
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settingService := initSessionTestDB(t)
			if _, err := settingService.GetAllSetting(); err != nil {
				t.Fatal(err)
			}
			if tt.env != "" {
				t.Setenv("SUI_FORCE_COOKIE_SECURE", tt.env)
			}
			if tt.proxies != "" {
				t.Setenv("SUI_TRUSTED_PROXIES", tt.proxies)
			}
			if tt.setup != nil {
				tt.setup(t, settingService)
			}
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tt.remote
			if tt.proto != "" {
				req.Header.Set("X-Forwarded-Proto", tt.proto)
			}
			if tt.tls {
				req.TLS = &tls.ConnectionState{}
			}
			c.Request = req
			if got := resolveCookieSecure(c, settingService); got != tt.want {
				t.Fatalf("resolveCookieSecure=%v, want %v", got, tt.want)
			}
		})
	}
}

func TestRotateSessionGenerationInvalidatesExistingSessions(t *testing.T) {
	settingService := initSessionTestDB(t)
	router := newSessionTestRouter(t, settingService)

	login := performSessionRequest(router, "/login")
	if login.Code != http.StatusNoContent {
		t.Fatalf("login returned %d", login.Code)
	}
	cookies := login.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("login did not set a session cookie")
	}

	beforeRotation := performSessionRequest(router, "/protected", cookies...)
	if beforeRotation.Code != http.StatusNoContent {
		t.Fatalf("session should be valid before rotation, got %d", beforeRotation.Code)
	}

	if _, err := settingService.RotateSessionGeneration(); err != nil {
		t.Fatal(err)
	}

	afterRotation := performSessionRequest(router, "/protected", cookies...)
	if afterRotation.Code != http.StatusUnauthorized {
		t.Fatalf("old session should be invalid after rotation, got %d", afterRotation.Code)
	}

	newLogin := performSessionRequest(router, "/login")
	if newLogin.Code != http.StatusNoContent {
		t.Fatalf("new login returned %d", newLogin.Code)
	}
	newSession := performSessionRequest(router, "/protected", newLogin.Result().Cookies()...)
	if newSession.Code != http.StatusNoContent {
		t.Fatalf("new session should be valid after rotation, got %d", newSession.Code)
	}
}
