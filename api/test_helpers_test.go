package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/service"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

func newAuthenticatedTestRouter(t *testing.T, settingService *service.SettingService, register func(*gin.Engine)) (*gin.Engine, []*http.Cookie) {
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
	if register != nil {
		register(router)
	}

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/login", nil))
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("login returned %d", recorder.Code)
	}
	cookies := recorder.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("login did not set a session cookie")
	}
	return router, cookies
}

func performAuthenticatedTestRequest(router *gin.Engine, req *http.Request, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	for _, c := range cookies {
		req.AddCookie(c)
	}
	router.ServeHTTP(recorder, req)
	return recorder
}

func withTestTokenScope(username string, scope string, handler gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(apiUsernameKey, username)
		c.Set(apiTokenScopeKey, scope)
		handler(c)
	}
}

func stopTokenUseDebouncerBeforeAPITestDBInit(tb testing.TB) {
	tb.Helper()
	if err := service.StopTokenUseDebouncer(context.Background()); err != nil {
		tb.Logf("token use debouncer stop before test DB init failed: %v", err)
	}
}

func initAPITestDB(tb testing.TB, dbPath string) {
	tb.Helper()
	stopTokenUseDebouncerBeforeAPITestDBInit(tb)
	if err := database.InitDB(dbPath); err != nil {
		if strings.Contains(err.Error(), "go-sqlite3 requires cgo") {
			tb.Skip(err)
		}
		tb.Fatal(err)
	}
}
