package api

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/deposist/s-ui-x/database"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

func TestSecuritySessionCookieFlagsAndMaxAge(t *testing.T) {
	settingService := initSessionTestDB(t)
	t.Setenv("SUI_FORCE_COOKIE_SECURE", "true")
	if _, err := settingService.GetAllSetting(); err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(map[string]string{"sessionMaxAge": "7"})
	if err != nil {
		t.Fatal(err)
	}
	if err := settingService.Save(database.GetDB(), payload); err != nil {
		t.Fatal(err)
	}
	router := newSecuritySessionMaxAgeRouter(t, settingService)

	login := performSessionRequest(router, "/login")
	if login.Code != http.StatusNoContent {
		t.Fatalf("login returned %d", login.Code)
	}
	cookie := findCookieByName(login.Result().Cookies())
	if cookie == nil {
		t.Fatal("login did not set s-ui cookie")
	}
	if !cookie.Secure {
		t.Fatal("session cookie must be Secure when forced")
	}
	if !cookie.HttpOnly {
		t.Fatal("session cookie must be HttpOnly")
	}
	if cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("session cookie SameSite=%v, want Lax", cookie.SameSite)
	}
	if cookie.MaxAge != 7*60 {
		t.Fatalf("session cookie MaxAge=%d, want %d", cookie.MaxAge, 7*60)
	}
}

func newSecuritySessionMaxAgeRouter(t *testing.T, settingService interface {
	GetSessionGeneration() (string, error)
	GetSessionMaxAge() (int, error)
}) *gin.Engine {
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
		maxAge, err := settingService.GetSessionMaxAge()
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		if err := SetLoginUser(c, "admin", maxAge, generation); err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		c.Status(http.StatusNoContent)
	})
	return router
}

func TestSecuritySessionRotationInvalidatesOldCookie(t *testing.T) {
	settingService := initSessionTestDB(t)
	router := newSessionTestRouter(t, settingService)

	login := performSessionRequest(router, "/login")
	if login.Code != http.StatusNoContent {
		t.Fatalf("login returned %d", login.Code)
	}
	if before := performSessionRequest(router, "/protected", login.Result().Cookies()...); before.Code != http.StatusNoContent {
		t.Fatalf("session should be valid before rotation, got %d", before.Code)
	}
	if _, err := settingService.RotateSessionGeneration(); err != nil {
		t.Fatal(err)
	}
	if after := performSessionRequest(router, "/protected", login.Result().Cookies()...); after.Code != http.StatusUnauthorized {
		t.Fatalf("old session should be unauthorized after rotation, got %d", after.Code)
	}
}

func TestSecuritySessionStrictSameSite(t *testing.T) {
	settingService := initSessionTestDB(t)
	if _, err := settingService.GetAllSetting(); err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(map[string]string{"sessionSameSiteStrict": "true"})
	if err != nil {
		t.Fatal(err)
	}
	if err := settingService.Save(database.GetDB(), payload); err != nil {
		t.Fatal(err)
	}
	router := newSecuritySessionMaxAgeRouter(t, settingService)

	login := performSessionRequest(router, "/login")
	if login.Code != http.StatusNoContent {
		t.Fatalf("login returned %d", login.Code)
	}
	cookie := findCookieByName(login.Result().Cookies())
	if cookie == nil {
		t.Fatal("login did not set s-ui cookie")
	}
	if cookie.SameSite != http.SameSiteStrictMode {
		t.Fatalf("session cookie SameSite=%v, want Strict", cookie.SameSite)
	}
}
