package api

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/deposist/s-ui-x-extended/service"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
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

func TestValidateRestoredCoreConfigValidatesFullGeneratedConfig(t *testing.T) {
	initSessionTestDB(t)
	const configSecret = "restore-config-secret"
	inbound := model.Inbound{
		Type:    "mixed",
		Tag:     "restore-poisoned-mixed",
		Addrs:   json.RawMessage(`[]`),
		Options: json.RawMessage(`{"listen":"127.0.0.1","listen_port":0,"bogus_top_level":"` + configSecret + `"}`),
	}
	if err := database.GetDB().Create(&inbound).Error; err != nil {
		t.Fatal(err)
	}

	a := NewApiService()
	err := a.validateRestoredCoreConfig()
	if err == nil {
		t.Fatal("poisoned restored config should be rejected")
	}
	if !strings.Contains(err.Error(), "unknown field") || !strings.Contains(err.Error(), "bogus_top_level") {
		t.Fatalf("validator error = %v, want unknown bogus_top_level field", err)
	}
	if strings.Contains(err.Error(), configSecret) {
		t.Fatalf("validator error leaked config secret: %v", err)
	}

	if err := database.GetDB().Model(&model.Inbound{}).Where("id = ?", inbound.Id).Update("options", json.RawMessage(`{"listen":"127.0.0.1","listen_port":0}`)).Error; err != nil {
		t.Fatal(err)
	}
	if err := a.validateRestoredCoreConfig(); err != nil {
		t.Fatalf("minimal generated config should pass validation: %v", err)
	}
}

func TestImportDbRejectsInvalidTrafficAgeBeforeSighup(t *testing.T) {
	settingService := initSessionTestDB(t)
	if err := setRestoreMarker("live-before-import"); err != nil {
		t.Fatal(err)
	}
	backup := newRestoreValidationBackup(t, func(db *gorm.DB) error {
		if err := setBackupSetting(db, "restore_marker", "restored-database"); err != nil {
			return err
		}
		return setBackupSetting(db, "trafficAge", "not-a-number")
	})
	assertImportDbRestoreRejected(t, settingService, backup)
}

func TestImportDbRejectsOverlappingListenersBeforeSighup(t *testing.T) {
	settingService := initSessionTestDB(t)
	if err := setRestoreMarker("live-before-import"); err != nil {
		t.Fatal(err)
	}
	backup := newRestoreValidationBackup(t, func(db *gorm.DB) error {
		for key, value := range map[string]string{
			"restore_marker": "restored-database",
			"webListen":      "",
			"webPort":        "2095",
			"subListen":      "127.0.0.1",
			"subPort":        "2095",
		} {
			if err := setBackupSetting(db, key, value); err != nil {
				return err
			}
		}
		return nil
	})
	assertImportDbRestoreRejected(t, settingService, backup)
}

func TestImportDbRejectsUnsafeLogOutputBeforeSighup(t *testing.T) {
	for _, tc := range []struct {
		name   string
		output string
	}{
		{name: "absolute", output: "/etc/s-ui-restore.log"},
		{name: "traversal", output: "../../etc/passwd"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			settingService := initSessionTestDB(t)
			if err := setRestoreMarker("live-before-import"); err != nil {
				t.Fatal(err)
			}
			backup := newRestoreValidationBackup(t, func(db *gorm.DB) error {
				if err := setBackupSetting(db, "restore_marker", "restored-database"); err != nil {
					return err
				}
				configValue, err := json.Marshal(map[string]any{
					"log": map[string]string{"level": "info", "output": tc.output},
				})
				if err != nil {
					return err
				}
				return setBackupSetting(db, "config", string(configValue))
			})
			assertImportDbRestoreRejected(t, settingService, backup)
		})
	}
}

func setBackupSetting(db *gorm.DB, key string, value string) error {
	setting := model.Setting{Key: key}
	return db.Where("key = ?", key).Assign(model.Setting{Value: value}).FirstOrCreate(&setting).Error
}

func newRestoreValidationBackup(t *testing.T, mutate func(*gorm.DB) error) []byte {
	t.Helper()
	backup, err := database.GetDb("")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "restore-validation.db")
	if err := os.WriteFile(path, backup, 0o600); err != nil {
		t.Fatal(err)
	}
	backupDB, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := backupDB.DB()
	if err != nil {
		t.Fatal(err)
	}
	if err := mutate(backupDB); err != nil {
		_ = sqlDB.Close()
		t.Fatal(err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatal(err)
	}
	mutated, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return mutated
}

func assertImportDbRestoreRejected(t *testing.T, settingService *service.SettingService, backup []byte) {
	t.Helper()
	var sighupCalls atomic.Int32
	withNoopSighup(t)
	database.SetSendSighupHook(func() error {
		sighupCalls.Add(1)
		return nil
	})

	apiService := NewApiService()
	router, cookies := newAuthenticatedTestRouter(t, settingService, func(router *gin.Engine) {
		router.POST("/api/importdb", withTestTokenScope("admin", "admin", apiService.ImportDb))
	})
	recorder := performAuthenticatedTestRequest(router, newDatabaseImportRequest(t, backup), cookies...)
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected restore status: %d body=%s", recorder.Code, recorder.Body.String())
	}
	var msg Msg
	if err := json.Unmarshal(recorder.Body.Bytes(), &msg); err != nil {
		t.Fatal(err)
	}
	if msg.Success {
		t.Fatalf("invalid restored database was accepted: %s", recorder.Body.String())
	}
	if sighupCalls.Load() != 0 {
		t.Fatalf("rejected restore triggered SIGHUP %d times", sighupCalls.Load())
	}
	if got := restoreMarkerValue(t); got != "live-before-import" {
		t.Fatalf("rollback marker = %q, want live-before-import", got)
	}
}
