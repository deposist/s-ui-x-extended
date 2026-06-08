package service_test

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/deposist/s-ui-x/api"
	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
	"github.com/deposist/s-ui-x/service"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

func TestRestoreEndpointAcceptsPlaintextAndTelegramBackupEnvelope(t *testing.T) {
	t.Run("plaintext", func(t *testing.T) {
		initRestoreEndpointTestDB(t)
		if err := setRestoreEndpointMarker("plaintext-backup"); err != nil {
			t.Fatal(err)
		}
		backup, err := database.GetDb("")
		if err != nil {
			t.Fatal(err)
		}
		if err := setRestoreEndpointMarker("live-before-import"); err != nil {
			t.Fatal(err)
		}
		recorder := performRestoreEndpointRequest(t, backup, "")
		if recorder.Code != http.StatusOK {
			t.Fatalf("unexpected status %d body=%s", recorder.Code, recorder.Body.String())
		}
		assertRestoreEndpointSuccess(t, recorder)
		if got := restoreEndpointMarkerValue(t); got != "plaintext-backup" {
			t.Fatalf("plaintext restore marker=%q", got)
		}
	})

	t.Run("envelope", func(t *testing.T) {
		initRestoreEndpointTestDB(t)
		passphrase := "correct horse battery staple"
		if err := setRestoreEndpointMarker("encrypted-backup"); err != nil {
			t.Fatal(err)
		}
		backup, err := database.GetDb("")
		if err != nil {
			t.Fatal(err)
		}
		envelope, err := service.BuildTelegramBackupEnvelope(backup, []byte(passphrase))
		if err != nil {
			t.Fatal(err)
		}
		if err := setRestoreEndpointMarker("live-before-import"); err != nil {
			t.Fatal(err)
		}
		recorder := performRestoreEndpointRequest(t, envelope, passphrase)
		if recorder.Code != http.StatusOK {
			t.Fatalf("unexpected status %d body=%s", recorder.Code, recorder.Body.String())
		}
		assertRestoreEndpointSuccess(t, recorder)
		if got := restoreEndpointMarkerValue(t); got != "encrypted-backup" {
			t.Fatalf("encrypted restore marker=%q", got)
		}
	})

	t.Run("wrong passphrase", func(t *testing.T) {
		initRestoreEndpointTestDB(t)
		if err := setRestoreEndpointMarker("encrypted-backup"); err != nil {
			t.Fatal(err)
		}
		backup, err := database.GetDb("")
		if err != nil {
			t.Fatal(err)
		}
		envelope, err := service.BuildTelegramBackupEnvelope(backup, []byte("correct horse battery staple"))
		if err != nil {
			t.Fatal(err)
		}
		if err := setRestoreEndpointMarker("live-before-import"); err != nil {
			t.Fatal(err)
		}
		recorder := performRestoreEndpointRequest(t, envelope, "wrong horse battery staple")
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("unexpected status %d body=%s", recorder.Code, recorder.Body.String())
		}
		assertRestoreEndpointFailureClass(t, recorder, "decryption_failed")
		if got := restoreEndpointMarkerValue(t); got != "live-before-import" {
			t.Fatalf("failed decrypt touched live DB, marker=%q", got)
		}
		var event model.AuditEvent
		if err := database.GetDB().Where("event = ?", "tg_backup_restore_failed").First(&event).Error; err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(event.Details), "wrong horse") || strings.Contains(string(event.Details), "correct horse") {
			t.Fatalf("restore audit leaked passphrase: %s", string(event.Details))
		}
	})
}

func initRestoreEndpointTestDB(t *testing.T) {
	t.Helper()
	prevAuditSync := service.AuditSyncForTest
	service.AuditSyncForTest = true
	t.Cleanup(func() { service.AuditSyncForTest = prevAuditSync })
	t.Setenv("SUI_DB_FOLDER", t.TempDir())
	if err := database.InitDB(filepath.Join(t.TempDir(), "s-ui.db")); err != nil {
		if strings.Contains(err.Error(), "go-sqlite3 requires cgo") {
			t.Skip(err)
		}
		t.Fatal(err)
	}
	if _, err := (&service.SettingService{}).GetAllSetting(); err != nil {
		t.Fatal(err)
	}
	database.SetSendSighupHook(func() error { return nil })
	t.Cleanup(func() { database.SetSendSighupHook(nil) })
	t.Cleanup(func() {
		if db := database.GetDB(); db != nil {
			if sqlDB, err := db.DB(); err == nil {
				_ = sqlDB.Close()
				time.Sleep(25 * time.Millisecond)
			}
		}
	})
}

func performRestoreEndpointRequest(t *testing.T, content []byte, passphrase string) *httptest.ResponseRecorder {
	t.Helper()
	router := gin.New()
	router.Use(sessions.Sessions("s-ui", cookie.NewStore([]byte("test-secret"))))
	router.POST("/api/importdb", (&api.ApiService{}).ImportDb)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, newRestoreEndpointRequest(t, content, passphrase))
	return recorder
}

func newRestoreEndpointRequest(t *testing.T, content []byte, passphrase string) *http.Request {
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
	if passphrase != "" {
		if err := writer.WriteField("telegramBackupPassphrase", passphrase); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/importdb", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func setRestoreEndpointMarker(value string) error {
	db := database.GetDB()
	if err := db.Where("key = ?", "restore_marker").Delete(&model.Setting{}).Error; err != nil {
		return err
	}
	return db.Create(&model.Setting{Key: "restore_marker", Value: value}).Error
}

func restoreEndpointMarkerValue(t *testing.T) string {
	t.Helper()
	var setting model.Setting
	if err := database.GetDB().Where("key = ?", "restore_marker").Order("id desc").First(&setting).Error; err != nil {
		t.Fatal(err)
	}
	return setting.Value
}

func assertRestoreEndpointSuccess(t *testing.T, recorder *httptest.ResponseRecorder) {
	t.Helper()
	var msg api.Msg
	if err := json.Unmarshal(recorder.Body.Bytes(), &msg); err != nil {
		t.Fatal(err)
	}
	if !msg.Success {
		t.Fatalf("expected restore success, got %#v body=%s", msg, recorder.Body.String())
	}
}

func assertRestoreEndpointFailureClass(t *testing.T, recorder *httptest.ResponseRecorder, want string) {
	t.Helper()
	var msg api.Msg
	if err := json.Unmarshal(recorder.Body.Bytes(), &msg); err != nil {
		t.Fatal(err)
	}
	obj, ok := msg.Obj.(map[string]any)
	if msg.Success || !ok || obj["errorClass"] != want {
		t.Fatalf("unexpected restore failure: %#v body=%s", msg, recorder.Body.String())
	}
}
