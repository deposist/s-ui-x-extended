package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/service"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

func securityCSRFPostRoutes() []string {
	return []string{
		"/api/changePass",
		"/api/addAdmin",
		"/api/deleteAdmin",
		"/api/save",
		"/api/restartApp",
		"/api/restartSb",
		"/api/linkConvert",
		"/api/subConvert",
		"/api/importdb",
		"/api/import-xui",
		"/api/import-xui/plan",
		"/api/import-xui/apply",
		"/api/import-xui/rollback",
		"/api/addToken",
		"/api/deleteToken",
		"/api/setTokenEnabled",
		"/api/logoutAllAdmins",
		"/api/checkOutbounds",
		"/api/rotateSubSecret",
		"/api/telegram/test",
		"/api/telegram/backup",
		"/api/telegram/backup/run",
		"/api/ip-monitor/alice/clear",
		"/api/paidsub/bindings",
		"/api/paidsub/tariffs",
		"/api/paidsub/broadcast",
		"/api/paidsub/refund",
	}
}

func newSecurityCSRFTestRouter(t *testing.T, settingService *service.SettingService) *gin.Engine {
	t.Helper()
	if _, err := settingService.GetAllSetting(); err != nil {
		t.Fatal(err)
	}
	webPathPayload, err := json.Marshal(map[string]string{"webPath": "/"})
	if err != nil {
		t.Fatal(err)
	}
	if err := settingService.Save(database.GetDB(), webPathPayload); err != nil {
		t.Fatal(err)
	}
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
	router.POST("/test/expire-csrf", func(c *gin.Context) {
		session := sessions.Default(c)
		session.Set(csrfExpiresKey, time.Now().Add(-time.Minute).Unix())
		if err := session.Save(); err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		c.Status(http.StatusNoContent)
	})
	handler := &APIHandler{}
	handler.initRouter(router.Group("/api"))
	return router
}

func TestSecurityCSRFMatrixRejectsMissingExpiredAndRotatedTokens(t *testing.T) {
	settingService := initSessionTestDB(t)
	router := newSecurityCSRFTestRouter(t, settingService)
	login := performCSRFRequest(router, http.MethodGet, "/login", "")
	if login.Code != http.StatusNoContent {
		t.Fatalf("login returned %d", login.Code)
	}
	cookies := login.Result().Cookies()

	for _, path := range securityCSRFPostRoutes() {
		t.Run("missing "+path, func(t *testing.T) {
			recorder := performCSRFRequest(router, http.MethodPost, path, "", cookies...)
			if recorder.Code != http.StatusForbidden {
				t.Fatalf("missing csrf for %s returned %d body=%s", path, recorder.Code, recorder.Body.String())
			}
		})
	}

	for _, path := range securityCSRFPostRoutes() {
		t.Run("expired "+path, func(t *testing.T) {
			token, freshCookies := issueSecurityCSRFToken(t, router, cookies)
			expire := performCSRFRequest(router, http.MethodPost, "/test/expire-csrf", "", freshCookies...)
			if expire.Code != http.StatusNoContent {
				t.Fatalf("expire csrf helper returned %d", expire.Code)
			}
			freshCookies = appendUpdatedCSRFCookies(freshCookies, expire.Result().Cookies())
			recorder := performCSRFRequest(router, http.MethodPost, path, token, freshCookies...)
			if recorder.Code != http.StatusForbidden {
				t.Fatalf("expired csrf for %s returned %d body=%s", path, recorder.Code, recorder.Body.String())
			}
		})
	}

	token, csrfCookies := issueSecurityCSRFToken(t, router, cookies)
	if _, err := settingService.RotateSessionGeneration(); err != nil {
		t.Fatal(err)
	}
	for _, path := range securityCSRFPostRoutes() {
		t.Run("rotated "+path, func(t *testing.T) {
			recorder := performCSRFRequest(router, http.MethodPost, path, token, csrfCookies...)
			if recorder.Code == http.StatusOK {
				t.Fatalf("rotated session csrf for %s unexpectedly reached handler: body=%s", path, recorder.Body.String())
			}
		})
	}
}

func TestSecurityCSRFMatrixDocumentsExceptions(t *testing.T) {
	settingService := initSessionTestDB(t)
	router := newSecurityCSRFTestRouter(t, settingService)

	login := performCSRFRequest(router, http.MethodPost, "/api/login", "")
	if login.Code == http.StatusForbidden {
		t.Fatal("login must remain CSRF-exempt")
	}
	logout := performCSRFRequest(router, http.MethodGet, "/api/logout", "")
	if logout.Code == http.StatusForbidden {
		t.Fatal("GET logout must remain CSRF-exempt")
	}
	sessionLogin := performCSRFRequest(router, http.MethodGet, "/login", "")
	if sessionLogin.Code != http.StatusNoContent {
		t.Fatalf("session login returned %d", sessionLogin.Code)
	}
	csrf := performCSRFRequest(router, http.MethodGet, "/api/csrf", "", sessionLogin.Result().Cookies()...)
	if csrf.Code != http.StatusOK {
		t.Fatalf("csrf endpoint with session returned %d", csrf.Code)
	}
}

func issueSecurityCSRFToken(t *testing.T, router *gin.Engine, cookies []*http.Cookie) (string, []*http.Cookie) {
	t.Helper()
	recorder := performCSRFRequest(router, http.MethodGet, "/api/csrf", "", cookies...)
	if recorder.Code != http.StatusOK {
		t.Fatalf("csrf endpoint returned %d body=%s", recorder.Code, recorder.Body.String())
	}
	var msg Msg
	if err := json.Unmarshal(recorder.Body.Bytes(), &msg); err != nil {
		t.Fatal(err)
	}
	obj, ok := msg.Obj.(map[string]any)
	if !ok {
		t.Fatalf("unexpected csrf payload: %#v", msg.Obj)
	}
	token, ok := obj["token"].(string)
	if !ok || strings.TrimSpace(token) == "" {
		t.Fatalf("missing csrf token: %#v", obj)
	}
	return token, appendUpdatedCSRFCookies(cookies, recorder.Result().Cookies())
}
