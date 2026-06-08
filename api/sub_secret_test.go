package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
	"github.com/deposist/s-ui-x/realtime"
	"github.com/deposist/s-ui-x/service"
	"github.com/gin-gonic/gin"
)

func TestAPIV2RotateSubSecretRequiresWriteScopeAndAudits(t *testing.T) {
	initSessionTestDB(t)
	client := model.Client{
		Enable:    true,
		Name:      "alice",
		SubSecret: "old-secret",
		Inbounds:  []byte("[]"),
		Links:     []byte("[]"),
	}
	if err := database.GetDB().Create(&client).Error; err != nil {
		t.Fatal(err)
	}
	readToken, err := (&service.UserService{}).AddToken("admin", 0, "read", "read")
	if err != nil {
		t.Fatal(err)
	}
	writeToken, err := (&service.UserService{}).AddToken("admin", 0, "write", "write")
	if err != nil {
		t.Fatal(err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewAPIv2Handler(router.Group("/apiv2"))

	readRecorder := performRotateSubSecretRequest(router, readToken, client.Id)
	if readRecorder.Code != http.StatusForbidden {
		t.Fatalf("read token should be forbidden, got %d", readRecorder.Code)
	}
	var stored model.Client
	if err := database.GetDB().Where("id = ?", client.Id).First(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if stored.SubSecret != "old-secret" {
		t.Fatal("read token rotated sub secret")
	}

	realtime.CloseAll("test_reset")
	t.Cleanup(func() { realtime.CloseAll("test_done") })
	ch := make(chan realtime.Event, 1)
	unregister := realtime.Register(&realtime.ClientHandle{
		User:   "admin",
		Scope:  realtime.ScopeAdmin,
		SendCh: ch,
	})
	defer unregister()

	writeRecorder := performRotateSubSecretRequest(router, writeToken, client.Id)
	if writeRecorder.Code != http.StatusOK {
		t.Fatalf("write token should be allowed, got %d", writeRecorder.Code)
	}
	var msg Msg
	if err := json.Unmarshal(writeRecorder.Body.Bytes(), &msg); err != nil {
		t.Fatal(err)
	}
	if !msg.Success {
		t.Fatalf("rotate request failed: %s", msg.Msg)
	}
	if err := database.GetDB().Where("id = ?", client.Id).First(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if stored.SubSecret == "" || stored.SubSecret == "old-secret" {
		t.Fatalf("sub secret was not rotated: %#v", stored)
	}
	if strings.Contains(writeRecorder.Body.String(), stored.SubSecret) {
		t.Fatal("response leaked rotated sub secret")
	}
	select {
	case event := <-ch:
		if event.Type != realtime.TopicConfigInvalidated {
			t.Fatalf("expected config_invalidated event, got %s", event.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("config_invalidated was not published after sub secret rotation")
	}

	var event model.AuditEvent
	if err := database.GetDB().Where("event = ?", "sub_secret_rotated").First(&event).Error; err != nil {
		t.Fatal(err)
	}
	if event.Actor != "admin" {
		t.Fatalf("unexpected audit actor: %s", event.Actor)
	}
	if strings.Contains(string(event.Details), stored.SubSecret) {
		t.Fatal("audit details leaked rotated sub secret")
	}
}

func performRotateSubSecretRequest(router *gin.Engine, token string, clientID uint) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/apiv2/rotateSubSecret?id="+strconv.FormatUint(uint64(clientID), 10), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(recorder, req)
	return recorder
}
