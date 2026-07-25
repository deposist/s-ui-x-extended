package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/gin-gonic/gin"
)

func newPlainHTTPContext(t *testing.T) *gin.Context {
	t.Helper()
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "http://panel.example/app/", nil)
	return c
}

func TestCookieSameSiteNoneOptIn(t *testing.T) {
	for _, value := range []string{"none", "None", "NONE", " none "} {
		t.Run("accepts "+value, func(t *testing.T) {
			t.Setenv("SUI_COOKIE_SAMESITE", value)
			if got := resolveCookieSameSite(nil); got != http.SameSiteNoneMode {
				t.Fatalf("SUI_COOKIE_SAMESITE=%q should select SameSite=None, got %v", value, got)
			}
		})
	}

	for _, value := range []string{"", "lax", "strict", "garbage"} {
		t.Run("ignores "+value, func(t *testing.T) {
			t.Setenv("SUI_COOKIE_SAMESITE", value)
			if got := resolveCookieSameSite(nil); got == http.SameSiteNoneMode {
				t.Fatalf("SUI_COOKIE_SAMESITE=%q must not select SameSite=None", value)
			}
		})
	}
}

func TestCookieSameSiteNoneForcesSecure(t *testing.T) {
	t.Setenv("SUI_COOKIE_SAMESITE", "none")
	settingService := initSessionTestDB(t)

	if !resolveCookieSecure(newPlainHTTPContext(t), settingService) {
		t.Fatal("SameSite=None must force Secure, otherwise browsers discard the cookie")
	}
}

func TestCookieSameSiteDefaultsToLax(t *testing.T) {
	t.Setenv("SUI_COOKIE_SAMESITE", "")
	settingService := initSessionTestDB(t)

	if got := resolveCookieSameSite(settingService); got != http.SameSiteLaxMode {
		t.Fatalf("default SameSite should be Lax, got %v", got)
	}
	if resolveCookieSecure(newPlainHTTPContext(t), settingService) {
		t.Fatal("plain HTTP without the opt-in should not force Secure")
	}
}

func TestLoginAndLogoutCookiesAgreeOnSameSiteNone(t *testing.T) {
	t.Setenv("SUI_COOKIE_SAMESITE", "none")
	settingService := initSessionTestDB(t)
	router := newSessionTestRouter(t, settingService)

	login := performSessionRequest(router, "/login")
	loginCookie := findCookieByName(login.Result().Cookies())
	if loginCookie == nil {
		t.Fatal("login did not set a session cookie")
	}
	if loginCookie.SameSite != http.SameSiteNoneMode {
		t.Fatalf("login cookie SameSite = %v, want None", loginCookie.SameSite)
	}
	if !loginCookie.Secure {
		t.Fatal("login cookie must be Secure alongside SameSite=None")
	}

	logout := performSessionRequest(router, "/logout", loginCookie)
	logoutCookie := findCookieByName(logout.Result().Cookies())
	if logoutCookie == nil {
		t.Fatal("logout did not emit a clearing cookie")
	}
	if logoutCookie.SameSite != loginCookie.SameSite || logoutCookie.Secure != loginCookie.Secure {
		t.Fatalf("logout cookie attributes (SameSite=%v Secure=%v) must match login (SameSite=%v Secure=%v)",
			logoutCookie.SameSite, logoutCookie.Secure, loginCookie.SameSite, loginCookie.Secure)
	}
	if logoutCookie.MaxAge >= 0 {
		t.Fatalf("logout cookie should expire the session, got MaxAge=%d", logoutCookie.MaxAge)
	}
}

func TestSessionSurvivesFollowUpRequestUnderSameSiteNone(t *testing.T) {
	t.Setenv("SUI_COOKIE_SAMESITE", "none")
	settingService := initSessionTestDB(t)
	router := newSessionTestRouter(t, settingService)

	login := performSessionRequest(router, "/login")
	loginCookie := findCookieByName(login.Result().Cookies())
	if loginCookie == nil {
		t.Fatal("login did not set a session cookie")
	}

	protected := performSessionRequest(router, "/protected", loginCookie)
	if protected.Code != http.StatusNoContent {
		t.Fatalf("follow-up request should stay authenticated, got %d", protected.Code)
	}
}

func TestCookieSameSiteNoneOverridesStrictSetting(t *testing.T) {
	t.Setenv("SUI_COOKIE_SAMESITE", "none")
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

	if got := resolveCookieSameSite(settingService); got != http.SameSiteNoneMode {
		t.Fatalf("SameSite=None must win over the Strict setting, got %v", got)
	}
}
