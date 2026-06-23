package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/deposist/s-ui-x-extended/service"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

// GET /api/failover-status returns one entry per failover group: the live active
// member plus per-member health sourced from the crash-safe failover_state table.
func TestFailoverStatusReturnsPerGroupActiveAndHealth(t *testing.T) {
	settingService := initSessionTestDB(t)
	db := database.GetDB()
	if err := service.EnsureFailoverSchema(db); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.Outbound{
		Type:    service.FailoverType,
		Tag:     "auto",
		Options: json.RawMessage(`{"outbounds":["m1","m2"],"failover":{"interval":"30s"}}`),
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := service.WriteFailoverMemberStates(db, []service.FailoverMemberState{
		{GroupTag: "auto", MemberTag: "m1", Healthy: false, LastProbeAt: 1},
		{GroupTag: "auto", MemberTag: "m2", Healthy: true, LastProbeAt: 1},
	}); err != nil {
		t.Fatal(err)
	}

	router, cookies := newAuthenticatedTestRouter(t, settingService, func(router *gin.Engine) {
		router.GET("/api/failover-status", (&ApiService{}).GetFailoverStatus)
	})
	recorder := performAuthenticatedTestRequest(router, httptest.NewRequest(http.MethodGet, "/api/failover-status", nil), cookies...)
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", recorder.Code)
	}
	var msg Msg
	if err := json.Unmarshal(recorder.Body.Bytes(), &msg); err != nil {
		t.Fatal(err)
	}
	if !msg.Success {
		t.Fatalf("failover-status failed: %s", msg.Msg)
	}
	list, ok := msg.Obj.([]interface{})
	if !ok || len(list) != 1 {
		t.Fatalf("obj = %#v, want exactly one group entry", msg.Obj)
	}
	entry := list[0].(map[string]interface{})
	if entry["tag"] != "auto" {
		t.Fatalf("entry tag = %v, want auto", entry["tag"])
	}
	if entry["allDown"] != false {
		t.Fatalf("allDown = %v, want false (m2 is healthy)", entry["allDown"])
	}
	members, ok := entry["members"].([]interface{})
	if !ok || len(members) != 2 {
		t.Fatalf("members = %#v, want 2 in priority order", entry["members"])
	}
	m2 := members[1].(map[string]interface{})
	if m2["tag"] != "m2" || m2["healthy"] != true || m2["priority"].(float64) != 1 {
		t.Fatalf("member m2 health/priority not surfaced: %#v", m2)
	}
}

// With no failover groups the endpoint returns an empty list (not an error).
func TestFailoverStatusEmptyWhenNoGroups(t *testing.T) {
	settingService := initSessionTestDB(t)
	router, cookies := newAuthenticatedTestRouter(t, settingService, func(router *gin.Engine) {
		router.GET("/api/failover-status", (&ApiService{}).GetFailoverStatus)
	})
	recorder := performAuthenticatedTestRequest(router, httptest.NewRequest(http.MethodGet, "/api/failover-status", nil), cookies...)
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", recorder.Code)
	}
	var msg Msg
	if err := json.Unmarshal(recorder.Body.Bytes(), &msg); err != nil {
		t.Fatal(err)
	}
	if !msg.Success {
		t.Fatalf("failover-status failed: %s", msg.Msg)
	}
	if list, ok := msg.Obj.([]interface{}); !ok || len(list) != 0 {
		t.Fatalf("obj = %#v, want empty list when no failover groups", msg.Obj)
	}
}

// The route lives on the browser-session group, so an unauthenticated request is
// redirected to login by checkLogin instead of returning status data.
func TestFailoverStatusRequiresLogin(t *testing.T) {
	initSessionTestDB(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(sessions.Sessions("s-ui", cookie.NewStore([]byte("test-secret"))))
	handler := &APIHandler{}
	handler.initRouter(router.Group("/api"))

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/failover-status", nil))
	if recorder.Code != http.StatusTemporaryRedirect {
		t.Fatalf("unauthenticated failover-status status = %d, want 307 redirect to login", recorder.Code)
	}
}
