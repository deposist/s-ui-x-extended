package service_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/deposist/s-ui-x/api"
	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
	"github.com/deposist/s-ui-x/realtime"
	"github.com/deposist/s-ui-x/service"

	"github.com/gin-gonic/gin"
)

func TestIntegrationSubSecretRotateReloadsClientAndPublishesRealtime(t *testing.T) {
	initSubSecretIntegrationDB(t)
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
	token, err := (&service.UserService{}).AddToken("admin", 0, "phase3-write", "write")
	if err != nil {
		t.Fatal(err)
	}

	realtime.CloseAll("phase3_subsecret_reset")
	t.Cleanup(func() { realtime.CloseAll("phase3_subsecret_done") })
	events := make(chan realtime.Event, 1)
	unregister := realtime.Register(&realtime.ClientHandle{
		User:   "admin",
		Scope:  realtime.ScopeAdmin,
		SendCh: events,
	})
	defer unregister()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	api.NewAPIv2Handler(router.Group("/apiv2"))

	rotateRecorder := httptest.NewRecorder()
	rotateReq := httptest.NewRequest(http.MethodPost, "/apiv2/rotateSubSecret?id="+strconv.FormatUint(uint64(client.Id), 10), nil)
	rotateReq.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(rotateRecorder, rotateReq)
	if rotateRecorder.Code != http.StatusOK {
		t.Fatalf("rotate status=%d body=%s", rotateRecorder.Code, rotateRecorder.Body.String())
	}
	var rotateMsg api.Msg
	if err := json.Unmarshal(rotateRecorder.Body.Bytes(), &rotateMsg); err != nil {
		t.Fatal(err)
	}
	if !rotateMsg.Success {
		t.Fatalf("rotate failed: %#v body=%s", rotateMsg, rotateRecorder.Body.String())
	}

	select {
	case event := <-events:
		if event.Type != realtime.TopicConfigInvalidated {
			t.Fatalf("expected config invalidated event, got %s", event.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("config invalidated event was not published")
	}

	loadRecorder := httptest.NewRecorder()
	loadReq := httptest.NewRequest(http.MethodGet, "/apiv2/clients?id="+strconv.FormatUint(uint64(client.Id), 10), nil)
	loadReq.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(loadRecorder, loadReq)
	if loadRecorder.Code != http.StatusOK {
		t.Fatalf("load clients status=%d body=%s", loadRecorder.Code, loadRecorder.Body.String())
	}
	var loadMsg struct {
		Success bool                       `json:"success"`
		Msg     string                     `json:"msg"`
		Obj     map[string]json.RawMessage `json:"obj"`
	}
	if err := json.Unmarshal(loadRecorder.Body.Bytes(), &loadMsg); err != nil {
		t.Fatal(err)
	}
	if !loadMsg.Success {
		t.Fatalf("load clients failed: %#v body=%s", loadMsg, loadRecorder.Body.String())
	}
	var clients []model.Client
	if err := json.Unmarshal(loadMsg.Obj["clients"], &clients); err != nil {
		t.Fatal(err)
	}
	if len(clients) != 1 {
		t.Fatalf("expected one loaded client, got %#v", clients)
	}
	if clients[0].SubSecret == "" || clients[0].SubSecret == "old-secret" {
		t.Fatalf("rotated sub_secret not visible through LoadPartialData: %#v", clients[0])
	}
	if strings.Contains(rotateRecorder.Body.String(), clients[0].SubSecret) {
		t.Fatal("rotate response leaked new sub_secret")
	}
}

func initSubSecretIntegrationDB(t *testing.T) {
	t.Helper()
	prevAuditSync := service.AuditSyncForTest
	service.AuditSyncForTest = true
	t.Cleanup(func() { service.AuditSyncForTest = prevAuditSync })
	tempDir := t.TempDir()
	t.Setenv("SUI_DB_FOLDER", tempDir)
	if db := database.GetDB(); db != nil {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	}
	if err := database.InitDB(filepath.Join(tempDir, "s-ui.db")); err != nil {
		if strings.Contains(err.Error(), "go-sqlite3 requires cgo") {
			t.Skip(err)
		}
		t.Fatal(err)
	}
	if _, err := (&service.SettingService{}).GetAllSetting(); err != nil {
		t.Fatal(err)
	}
	testDB := database.GetDB()
	t.Cleanup(func() {
		if testDB != nil {
			if sqlDB, err := testDB.DB(); err == nil {
				_ = sqlDB.Close()
				time.Sleep(25 * time.Millisecond)
			}
		}
	})
}
