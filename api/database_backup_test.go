package api

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
	"github.com/deposist/s-ui-x/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func TestImportDbRequiresAdminScopeAndAuditsFailure(t *testing.T) {
	settingService := initSessionTestDB(t)

	readRouter, readCookies := newAuthenticatedTestRouter(t, settingService, func(router *gin.Engine) {
		router.POST("/api/importdb", withTestTokenScope("reader", "read", (&ApiService{}).ImportDb))
	})
	readRecorder := performAuthenticatedTestRequest(readRouter, newDatabaseImportRequest(t, []byte("not sqlite")), readCookies...)
	if readRecorder.Code != http.StatusForbidden {
		t.Fatalf("read scope should be forbidden, got %d", readRecorder.Code)
	}

	adminRouter, adminCookies := newAuthenticatedTestRouter(t, settingService, func(router *gin.Engine) {
		router.POST("/api/importdb", withTestTokenScope("admin", "admin", (&ApiService{}).ImportDb))
	})
	adminRecorder := performAuthenticatedTestRequest(adminRouter, newDatabaseImportRequest(t, []byte("not sqlite")), adminCookies...)
	if adminRecorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", adminRecorder.Code)
	}
	var msg Msg
	if err := json.Unmarshal(adminRecorder.Body.Bytes(), &msg); err != nil {
		t.Fatal(err)
	}
	if msg.Success {
		t.Fatal("invalid db import should fail")
	}

	var event model.AuditEvent
	if err := database.GetDB().Where("event = ?", "db_import_failed").First(&event).Error; err != nil {
		t.Fatal(err)
	}
	if event.Actor != "admin" || event.Resource != "database" || !strings.Contains(string(event.Details), `"reason":"invalid_db"`) {
		t.Fatalf("unexpected audit event: %#v details=%s", event, string(event.Details))
	}
}

func TestGetDbAuditsExport(t *testing.T) {
	settingService := initSessionTestDB(t)
	router, cookies := newAuthenticatedTestRouter(t, settingService, func(router *gin.Engine) {
		router.GET("/api/getdb", withTestTokenScope("admin", "admin", (&ApiService{}).GetDb))
	})
	recorder := performAuthenticatedTestRequest(router, httptest.NewRequest(http.MethodGet, "/api/getdb", nil), cookies...)
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", recorder.Code)
	}
	if recorder.Body.Len() == 0 {
		t.Fatal("empty database export")
	}
	var event model.AuditEvent
	if err := database.GetDB().Where("event = ?", "db_exported").First(&event).Error; err != nil {
		t.Fatal(err)
	}
	if event.Actor != "admin" || event.Resource != "database" || !strings.Contains(string(event.Details), `"channel":"download"`) {
		t.Fatalf("unexpected audit event: %#v details=%s", event, string(event.Details))
	}
}

func TestGetDbEncryptedWithTelegramBackupPassphrase(t *testing.T) {
	settingService := initSessionTestDB(t)
	passphrase := "correct horse battery staple"
	saveTelegramBackupPassphrase(t, settingService, passphrase)
	router, cookies := newAuthenticatedTestRouter(t, settingService, func(router *gin.Engine) {
		router.GET("/api/getdb", withTestTokenScope("admin", "admin", (&ApiService{}).GetDb))
	})
	recorder := performAuthenticatedTestRequest(router, httptest.NewRequest(http.MethodGet, "/api/getdb?encryptTelegramBackup=true&exclude=stats,audit,unknown", nil), cookies...)
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", recorder.Code, recorder.Body.String())
	}
	envelope := recorder.Body.Bytes()
	if !service.IsTelegramBackupEnvelope(envelope) {
		t.Fatalf("encrypted backup did not return Telegram backup envelope")
	}
	plaintext, err := service.OpenTelegramBackupEnvelope(envelope, []byte(passphrase))
	if err != nil {
		t.Fatal(err)
	}
	isDB, err := database.IsSQLiteDB(bytes.NewReader(plaintext))
	if err != nil {
		t.Fatal(err)
	}
	if !isDB {
		t.Fatal("decrypted encrypted backup is not SQLite")
	}

	var event model.AuditEvent
	if err := database.GetDB().Where("event = ?", "tg_backup_manual_encrypted").First(&event).Error; err != nil {
		t.Fatal(err)
	}
	var details map[string]any
	if err := json.Unmarshal(event.Details, &details); err != nil {
		t.Fatal(err)
	}
	if details["channel"] != "local_download" {
		t.Fatalf("unexpected audit details: %#v", details)
	}
	excluded, ok := details["excludedTables"].([]any)
	if !ok || len(excluded) != 2 || excluded[0] != "stats" || excluded[1] != "audit_events" {
		t.Fatalf("unexpected excludedTables audit details: %#v", details)
	}
	if strings.Contains(string(event.Details), passphrase) {
		t.Fatalf("audit details leaked passphrase: %s", string(event.Details))
	}
}

func TestGetDbEncryptedRejectsMissingTelegramBackupPassphrase(t *testing.T) {
	settingService := initSessionTestDB(t)
	router, cookies := newAuthenticatedTestRouter(t, settingService, func(router *gin.Engine) {
		router.GET("/api/getdb", withTestTokenScope("admin", "admin", (&ApiService{}).GetDb))
	})
	recorder := performAuthenticatedTestRequest(router, httptest.NewRequest(http.MethodGet, "/api/getdb?encryptTelegramBackup=true", nil), cookies...)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: %d body=%s", recorder.Code, recorder.Body.String())
	}
	var msg Msg
	if err := json.Unmarshal(recorder.Body.Bytes(), &msg); err != nil {
		t.Fatal(err)
	}
	obj, ok := msg.Obj.(map[string]any)
	if !ok || obj["errorClass"] != "missing_passphrase" {
		t.Fatalf("unexpected missing-passphrase response: %#v", msg.Obj)
	}
	var count int64
	if err := database.GetDB().Model(&model.AuditEvent{}).Where("event = ?", "db_exported").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("missing passphrase should not export db, got %d export audit events", count)
	}
}

func saveTelegramBackupPassphrase(t *testing.T, settingService *service.SettingService, passphrase string) {
	t.Helper()
	payload, err := json.Marshal(map[string]string{"telegramBackupPassphrase": passphrase})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Transaction(func(tx *gorm.DB) error {
		return settingService.Save(tx, payload)
	}); err != nil {
		t.Fatal(err)
	}
}

func newDatabaseImportRequest(t *testing.T, content []byte) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("db", "backup.db")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/importdb", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}
