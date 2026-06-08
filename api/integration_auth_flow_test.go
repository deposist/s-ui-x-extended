package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
	"github.com/deposist/s-ui-x/realtime"
	"github.com/deposist/s-ui-x/service"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

func TestIntegrationAuthFlowLoginCSRFSaveSettingsPublishesRealtime(t *testing.T) {
	resetRateLimitState()
	settingService := initSessionTestDB(t)
	if _, err := settingService.GetAllSetting(); err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Model(model.Setting{}).Where("key = ?", "webPath").Update("value", "/").Error; err != nil {
		t.Fatal(err)
	}
	if err := (&service.UserService{}).UpdateFirstUser("admin", "phase3-password"); err != nil {
		t.Fatal(err)
	}

	realtime.CloseAll("phase3_auth_reset")
	t.Cleanup(func() { realtime.CloseAll("phase3_auth_done") })
	events := make(chan realtime.Event, 4)
	unregister := realtime.Register(&realtime.ClientHandle{
		User:   "admin",
		Scope:  realtime.ScopeAdmin,
		SendCh: events,
	})
	defer unregister()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(sessions.Sessions("s-ui", cookie.NewStore([]byte("test-secret"))))
	NewAPIHandler(router.Group("/api"), nil)
	jar := integrationCookieJar{}

	loginForm := url.Values{}
	loginForm.Set("user", "admin")
	loginForm.Set("pass", "phase3-password")
	loginReq := httptest.NewRequest(http.MethodPost, "/api/login", strings.NewReader(loginForm.Encode()))
	loginReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	loginRecorder := performIntegrationRequest(router, loginReq, &jar)
	if loginRecorder.Code != http.StatusOK {
		t.Fatalf("login returned %d body=%s", loginRecorder.Code, loginRecorder.Body.String())
	}
	assertIntegrationMsgSuccess(t, loginRecorder)

	csrfReq := httptest.NewRequest(http.MethodGet, "/api/csrf", nil)
	csrfRecorder := performIntegrationRequest(router, csrfReq, &jar)
	if csrfRecorder.Code != http.StatusOK {
		t.Fatalf("csrf returned %d body=%s", csrfRecorder.Code, csrfRecorder.Body.String())
	}
	csrfToken := integrationCSRFToken(t, csrfRecorder)

	payload, err := json.Marshal(map[string]string{
		"subPath": "/phase3-sub/",
	})
	if err != nil {
		t.Fatal(err)
	}
	saveForm := url.Values{}
	saveForm.Set("object", "settings")
	saveForm.Set("action", "set")
	saveForm.Set("data", string(payload))
	saveReq := httptest.NewRequest(http.MethodPost, "/api/save", strings.NewReader(saveForm.Encode()))
	saveReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	saveReq.Header.Set(csrfHeader, csrfToken)
	saveRecorder := performIntegrationRequest(router, saveReq, &jar)
	if saveRecorder.Code != http.StatusOK {
		t.Fatalf("save returned %d body=%s", saveRecorder.Code, saveRecorder.Body.String())
	}
	assertIntegrationMsgSuccess(t, saveRecorder)
	expectIntegrationRealtimeTopic(t, events, realtime.TopicConfigInvalidated)

	rejectedPayload, err := json.Marshal(map[string]string{
		"unexpectedKey": "value",
	})
	if err != nil {
		t.Fatal(err)
	}
	rejectForm := url.Values{}
	rejectForm.Set("object", "settings")
	rejectForm.Set("action", "set")
	rejectForm.Set("data", string(rejectedPayload))
	rejectReq := httptest.NewRequest(http.MethodPost, "/api/save", strings.NewReader(rejectForm.Encode()))
	rejectReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rejectReq.Header.Set(csrfHeader, csrfToken)
	rejectRecorder := performIntegrationRequest(router, rejectReq, &jar)
	if rejectRecorder.Code != http.StatusBadRequest {
		t.Fatalf("rejected settings save returned %d body=%s", rejectRecorder.Code, rejectRecorder.Body.String())
	}

	assertIntegrationAuditEvent(t, "login_success", "auth")
	assertIntegrationAuditEvent(t, "sub_path_changed", "subscription")
	assertIntegrationAuditEvent(t, "settings_save_rejected_key", "settings")

	var change model.Changes
	if err := database.GetDB().Where("actor = ? AND key = ? AND action = ?", "admin", "settings", "set").First(&change).Error; err != nil {
		t.Fatal(err)
	}
}

func TestIntegrationAuthFlowSettingsSaveSuccessAudit(t *testing.T) {
	settingService := initSessionTestDB(t)
	if _, err := settingService.GetAllSetting(); err != nil {
		t.Fatal(err)
	}
	router, cookies := newAuthenticatedTestRouter(t, settingService, func(router *gin.Engine) {
		router.POST("/api/save", func(c *gin.Context) {
			(&ApiService{}).Save(c, "admin")
		})
	})

	payload, err := json.Marshal(map[string]string{"subJsonPath": "/json-success/"})
	if err != nil {
		t.Fatal(err)
	}
	form := url.Values{}
	form.Set("object", "settings")
	form.Set("action", "set")
	form.Set("data", string(payload))
	req := httptest.NewRequest(http.MethodPost, "/api/save", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	recorder := performAuthenticatedTestRequest(router, req, cookies...)
	if recorder.Code != http.StatusOK {
		t.Fatalf("settings save returned %d body=%s", recorder.Code, recorder.Body.String())
	}

	var event model.AuditEvent
	if err := database.GetDB().Where("event = ?", "settings_save_succeeded").Order("id desc").First(&event).Error; err != nil {
		t.Fatal(err)
	}
	if event.Actor != "admin" || event.Resource != "settings" || event.Severity != service.AuditSeverityInfo {
		t.Fatalf("unexpected settings_save_succeeded audit: %#v", event)
	}
}

type integrationCookieJar struct {
	cookies []*http.Cookie
}

func (j *integrationCookieJar) addTo(req *http.Request) {
	if j == nil {
		return
	}
	for _, c := range j.cookies {
		req.AddCookie(c)
	}
}

func (j *integrationCookieJar) store(cookies []*http.Cookie) {
	if j == nil || len(cookies) == 0 {
		return
	}
	byName := map[string]*http.Cookie{}
	for _, c := range j.cookies {
		byName[c.Name] = c
	}
	for _, c := range cookies {
		byName[c.Name] = c
	}
	j.cookies = j.cookies[:0]
	for _, c := range byName {
		j.cookies = append(j.cookies, c)
	}
}

func performIntegrationRequest(router *gin.Engine, req *http.Request, jar *integrationCookieJar) *httptest.ResponseRecorder {
	if jar != nil {
		jar.addTo(req)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if jar != nil {
		jar.store(recorder.Result().Cookies())
	}
	return recorder
}

func integrationCSRFToken(t *testing.T, recorder *httptest.ResponseRecorder) string {
	t.Helper()
	var msg struct {
		Success bool   `json:"success"`
		Msg     string `json:"msg"`
		Obj     struct {
			Token string `json:"token"`
		} `json:"obj"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &msg); err != nil {
		t.Fatal(err)
	}
	if !msg.Success || msg.Obj.Token == "" {
		t.Fatalf("unexpected csrf response: %#v body=%s", msg, recorder.Body.String())
	}
	return msg.Obj.Token
}

func assertIntegrationMsgSuccess(t *testing.T, recorder *httptest.ResponseRecorder) {
	t.Helper()
	var msg Msg
	if err := json.Unmarshal(recorder.Body.Bytes(), &msg); err != nil {
		t.Fatal(err)
	}
	if !msg.Success {
		t.Fatalf("expected success response, got %#v body=%s", msg, recorder.Body.String())
	}
}

func expectIntegrationRealtimeTopic(t *testing.T, events <-chan realtime.Event, topic realtime.Topic) realtime.Event {
	t.Helper()
	select {
	case event := <-events:
		if event.Type != topic {
			t.Fatalf("expected realtime topic %s, got %s", topic, event.Type)
		}
		return event
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for realtime topic %s", topic)
		return realtime.Event{}
	}
}

func assertIntegrationAuditEvent(t *testing.T, eventName string, resource string) model.AuditEvent {
	t.Helper()
	var event model.AuditEvent
	if err := database.GetDB().Where("event = ?", eventName).Order("id desc").First(&event).Error; err != nil {
		t.Fatal(err)
	}
	if event.Actor != "admin" || event.Resource != resource {
		t.Fatalf("unexpected audit event for %s: %#v", eventName, event)
	}
	return event
}
