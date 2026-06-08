package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
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

func TestAdminCreateDeleteFlowRequiresCurrentPasswordAndAudits(t *testing.T) {
	resetRateLimitState()
	settingService := initSessionTestDB(t)
	if _, err := settingService.GetAllSetting(); err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Model(model.Setting{}).Where("key = ?", "webPath").Update("value", "/").Error; err != nil {
		t.Fatal(err)
	}
	if err := (&service.UserService{}).UpdateFirstUser("admin", "current-password"); err != nil {
		t.Fatal(err)
	}
	target, err := (&service.UserService{}).AddUser("admin", "current-password", "delete-me", "target-password")
	if err != nil {
		t.Fatal(err)
	}
	targetToken, err := (&service.UserService{}).AddToken("delete-me", 0, "target token", "admin")
	if err != nil {
		t.Fatal(err)
	}

	router, _ := newAdminFlowRouter(t)
	targetJar := &integrationCookieJar{}
	adminJar := &integrationCookieJar{}
	loginAdminFlowUser(t, router, targetJar, "delete-me", "target-password")
	loginAdminFlowUser(t, router, adminJar, "admin", "current-password")
	csrf := adminFlowCSRFToken(t, router, adminJar)

	deniedCreate := adminFlowPost(t, router, adminJar, csrf, "/api/addAdmin", url.Values{
		"currentPass": {"wrong-password"},
		"username":    {"denied-admin"},
		"password":    {"denied-password"},
	})
	assertAdminFlowMsgSuccess(t, deniedCreate, false)
	if exists, err := (&service.UserService{}).UserExists("denied-admin"); err != nil {
		t.Fatal(err)
	} else if exists {
		t.Fatal("wrong current password should not create an admin")
	}

	create := adminFlowPost(t, router, adminJar, csrf, "/api/addAdmin", url.Values{
		"currentPass": {"current-password"},
		"username":    {"new-admin"},
		"password":    {"new-secret"},
	})
	assertAdminFlowMsgSuccess(t, create, true)
	if strings.Contains(create.Body.String(), "new-secret") || strings.Contains(create.Body.String(), "current-password") {
		t.Fatalf("create response leaked credentials: %s", create.Body.String())
	}
	var created model.User
	if err := database.GetDB().Where("username = ?", "new-admin").First(&created).Error; err != nil {
		t.Fatal(err)
	}

	usersReq := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	usersReq.Header.Set("X-Requested-With", "XMLHttpRequest")
	users := performIntegrationRequest(router, usersReq, adminJar)
	assertAdminFlowUsersCurrentFlags(t, users)

	var admin model.User
	if err := database.GetDB().Where("username = ?", "admin").First(&admin).Error; err != nil {
		t.Fatal(err)
	}
	selfDelete := adminFlowPost(t, router, adminJar, csrf, "/api/deleteAdmin", url.Values{
		"currentPass": {"current-password"},
		"id":          {strconv.FormatUint(uint64(admin.Id), 10)},
	})
	assertAdminFlowMsgSuccess(t, selfDelete, false)
	if exists, err := (&service.UserService{}).UserExists("admin"); err != nil {
		t.Fatal(err)
	} else if !exists {
		t.Fatal("self delete must not remove the current admin")
	}

	deleteTarget := adminFlowPost(t, router, adminJar, csrf, "/api/deleteAdmin", url.Values{
		"currentPass": {"current-password"},
		"id":          {strconv.FormatUint(uint64(target.Id), 10)},
	})
	assertAdminFlowMsgSuccess(t, deleteTarget, true)
	if exists, err := (&service.UserService{}).UserExists("delete-me"); err != nil {
		t.Fatal(err)
	} else if exists {
		t.Fatal("target admin should be deleted")
	}
	var tokenCount int64
	if err := database.GetDB().Model(model.Tokens{}).Where("user_id = ?", target.Id).Count(&tokenCount).Error; err != nil {
		t.Fatal(err)
	}
	if tokenCount != 0 {
		t.Fatalf("target tokens should be deleted, got %d", tokenCount)
	}

	targetUsersReq := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	targetUsersReq.Header.Set("X-Requested-With", "XMLHttpRequest")
	targetUsers := performIntegrationRequest(router, targetUsersReq, targetJar)
	assertAdminFlowInvalidLogin(t, targetUsers)

	apiReq := httptest.NewRequest(http.MethodGet, "/apiv2/load", nil)
	apiReq.Header.Set("Authorization", "Bearer "+targetToken)
	apiRecorder := httptest.NewRecorder()
	router.ServeHTTP(apiRecorder, apiReq)
	assertAdminFlowMsgSuccess(t, apiRecorder, false)

	createdAudit := assertIntegrationAuditEvent(t, "admin_created", "admin")
	deletedAudit := assertIntegrationAuditEvent(t, "admin_deleted", "admin")
	for _, event := range []model.AuditEvent{createdAudit, deletedAudit} {
		details := string(event.Details)
		for _, secret := range []string{"current-password", "target-password", "new-secret", targetToken} {
			if strings.Contains(details, secret) {
				t.Fatalf("audit details leaked secret %q: %s", secret, details)
			}
		}
	}
}

func newAdminFlowRouter(t *testing.T) (*gin.Engine, *APIv2Handler) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(sessions.Sessions("s-ui", cookie.NewStore([]byte("test-secret"))))
	apiv2 := NewAPIv2Handler(router.Group("/apiv2"))
	NewAPIHandler(router.Group("/api"), apiv2)
	return router, apiv2
}

func loginAdminFlowUser(t *testing.T, router *gin.Engine, jar *integrationCookieJar, username string, password string) {
	t.Helper()
	loginForm := url.Values{}
	loginForm.Set("user", username)
	loginForm.Set("pass", password)
	req := httptest.NewRequest(http.MethodPost, "/api/login", strings.NewReader(loginForm.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	recorder := performIntegrationRequest(router, req, jar)
	assertAdminFlowMsgSuccess(t, recorder, true)
}

func adminFlowCSRFToken(t *testing.T, router *gin.Engine, jar *integrationCookieJar) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/csrf", nil)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	recorder := performIntegrationRequest(router, req, jar)
	return integrationCSRFToken(t, recorder)
}

func adminFlowPost(t *testing.T, router *gin.Engine, jar *integrationCookieJar, csrf string, path string, form url.Values) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set(csrfHeader, csrf)
	return performIntegrationRequest(router, req, jar)
}

func assertAdminFlowMsgSuccess(t *testing.T, recorder *httptest.ResponseRecorder, want bool) Msg {
	t.Helper()
	var msg Msg
	if err := json.Unmarshal(recorder.Body.Bytes(), &msg); err != nil {
		t.Fatalf("invalid msg body=%s: %v", recorder.Body.String(), err)
	}
	if msg.Success != want {
		t.Fatalf("success=%v, want %v, msg=%#v body=%s", msg.Success, want, msg, recorder.Body.String())
	}
	return msg
}

func assertAdminFlowInvalidLogin(t *testing.T, recorder *httptest.ResponseRecorder) {
	t.Helper()
	msg := assertAdminFlowMsgSuccess(t, recorder, false)
	if msg.Msg != "Invalid login" {
		t.Fatalf("expected Invalid login after delete, got %#v", msg)
	}
}

func assertAdminFlowUsersCurrentFlags(t *testing.T, recorder *httptest.ResponseRecorder) {
	t.Helper()
	msg := assertAdminFlowMsgSuccess(t, recorder, true)
	rawUsers, ok := msg.Obj.([]any)
	if !ok {
		t.Fatalf("unexpected users payload: %#v", msg.Obj)
	}
	seenAdmin := false
	seenNewAdmin := false
	for _, raw := range rawUsers {
		user, ok := raw.(map[string]any)
		if !ok {
			t.Fatalf("unexpected user payload: %#v", raw)
		}
		switch user["username"] {
		case "admin":
			seenAdmin = true
			if user["isCurrent"] != true {
				t.Fatalf("current admin flag missing: %#v", user)
			}
		case "new-admin":
			seenNewAdmin = true
			if user["isCurrent"] != false {
				t.Fatalf("new admin should not be current: %#v", user)
			}
		}
	}
	if !seenAdmin || !seenNewAdmin {
		t.Fatalf("expected admin and new-admin in users payload: %#v", rawUsers)
	}
}
