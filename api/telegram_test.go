package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
	"github.com/deposist/s-ui-x/service"
	"github.com/gin-gonic/gin"
)

func TestAPIV2TelegramTestRequiresAdminScope(t *testing.T) {
	initSessionTestDB(t)
	readToken, err := (&service.UserService{}).AddToken("admin", 0, "read telegram", "read")
	if err != nil {
		t.Fatal(err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewAPIv2Handler(router.Group("/apiv2"))

	recorder := performTelegramTestRequest(router, readToken)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("read token should be forbidden, got %d", recorder.Code)
	}

	var event model.AuditEvent
	if err := database.GetDB().Where("event = ?", "scope_denied").First(&event).Error; err != nil {
		t.Fatal(err)
	}
	if event.Actor != "admin" || event.Resource != "telegram" {
		t.Fatalf("unexpected audit event: %#v", event)
	}
}

func TestAPIV2TelegramTestAuditsWithoutSecrets(t *testing.T) {
	initSessionTestDB(t)
	adminToken, err := (&service.UserService{}).AddToken("admin", 0, "admin telegram", "admin")
	if err != nil {
		t.Fatal(err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewAPIv2Handler(router.Group("/apiv2"))

	recorder := performTelegramTestRequest(router, adminToken)
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", recorder.Code)
	}
	var msg Msg
	if err := json.Unmarshal(recorder.Body.Bytes(), &msg); err != nil {
		t.Fatal(err)
	}
	if !msg.Success {
		t.Fatalf("Telegram test request failed at API layer: %s", msg.Msg)
	}
	payload, ok := msg.Obj.(map[string]any)
	if !ok {
		t.Fatalf("unexpected response payload: %#v", msg.Obj)
	}
	if payload["success"] == true || payload["errorClass"] != "disabled" {
		t.Fatalf("disabled Telegram test should report provider failure, got %#v", payload)
	}

	var event model.AuditEvent
	if err := database.GetDB().Where("event = ?", "telegram_test").First(&event).Error; err != nil {
		t.Fatal(err)
	}
	if event.Actor != "admin" || event.Resource != "telegram" {
		t.Fatalf("unexpected audit event: %#v", event)
	}
	details := string(event.Details)
	if !strings.Contains(details, `"errorClass":"disabled"`) {
		t.Fatalf("unexpected audit details: %s", details)
	}
	if strings.Contains(details, "bot") || strings.Contains(details, "token") {
		t.Fatalf("audit details leaked secret material: %s", details)
	}
}

func performTelegramTestRequest(router *gin.Engine, token string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/apiv2/telegram/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(recorder, req)
	return recorder
}
