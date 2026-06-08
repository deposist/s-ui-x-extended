package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
	"github.com/deposist/s-ui-x/service"
	"github.com/gin-gonic/gin"
)

func TestSaveSettingsAuditsSubscriptionPathChange(t *testing.T) {
	settingService := initSessionTestDB(t)
	if _, err := settingService.GetAllSetting(); err != nil {
		t.Fatal(err)
	}
	router, cookies := newAuthenticatedTestRouter(t, settingService, func(router *gin.Engine) {
		router.POST("/api/save", func(c *gin.Context) {
			(&ApiService{}).Save(c, "admin")
		})
	})

	payload, err := json.Marshal(map[string]string{
		"subJsonPath": "/json-alt/",
	})
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
		t.Fatalf("unexpected status: %d", recorder.Code)
	}

	var event model.AuditEvent
	if err := database.GetDB().Where("event = ?", "sub_path_changed").First(&event).Error; err != nil {
		t.Fatal(err)
	}
	if event.Actor != "admin" || event.Resource != "subscription" || event.Severity != service.AuditSeverityWarn {
		t.Fatalf("unexpected audit event: %#v", event)
	}
	details := string(event.Details)
	if !strings.Contains(details, `"subJsonPath"`) || !strings.Contains(details, `"restartRequired":true`) {
		t.Fatalf("audit details missing path change metadata: %s", details)
	}
}

func TestSaveSettingsRejectsUnknownKeyAndAudits(t *testing.T) {
	settingService := initSessionTestDB(t)
	if _, err := settingService.GetAllSetting(); err != nil {
		t.Fatal(err)
	}
	router, cookies := newAuthenticatedTestRouter(t, settingService, func(router *gin.Engine) {
		router.POST("/api/save", func(c *gin.Context) {
			(&ApiService{}).Save(c, "admin")
		})
	})

	payload, err := json.Marshal(map[string]string{
		"unexpectedKey": "value",
	})
	if err != nil {
		t.Fatal(err)
	}
	form := url.Values{}
	form.Set("object", "settings")
	form.Set("action", "set")
	form.Set("data", string(payload))
	body := strings.NewReader(form.Encode())
	req := httptest.NewRequest(http.MethodPost, "/api/save", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	recorder := performAuthenticatedTestRequest(router, req, cookies...)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: %d", recorder.Code)
	}

	var event model.AuditEvent
	if err := database.GetDB().Where("event = ?", "settings_save_rejected_key").Order("id desc").First(&event).Error; err != nil {
		t.Fatal(err)
	}
	if event.Actor != "admin" || event.Resource != "settings" || event.Severity != service.AuditSeverityWarn {
		t.Fatalf("unexpected audit event: %#v", event)
	}
	details := string(event.Details)
	if !strings.Contains(details, `"reason":"invalid setting key:`) || !strings.Contains(details, `unexpectedKey`) {
		t.Fatalf("audit details missing reject reason: %s", details)
	}

	var count int64
	if err := database.GetDB().Model(model.Setting{}).Where("key = ?", "unexpectedKey").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("unexpected setting key persisted, count=%d", count)
	}
}

func TestSaveSettingsRejectsProtectedKeyAndAudits(t *testing.T) {
	settingService := initSessionTestDB(t)
	if _, err := settingService.GetAllSetting(); err != nil {
		t.Fatal(err)
	}
	router, cookies := newAuthenticatedTestRouter(t, settingService, func(router *gin.Engine) {
		router.POST("/api/save", func(c *gin.Context) {
			(&ApiService{}).Save(c, "admin")
		})
	})

	payload, err := json.Marshal(map[string]string{
		"secret": "override-not-allowed",
	})
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
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: %d", recorder.Code)
	}

	var event model.AuditEvent
	if err := database.GetDB().Where("event = ?", "settings_save_rejected_key").Order("id desc").First(&event).Error; err != nil {
		t.Fatal(err)
	}
	if event.Actor != "admin" || event.Resource != "settings" || event.Severity != service.AuditSeverityWarn {
		t.Fatalf("unexpected audit event: %#v", event)
	}
	if !strings.Contains(string(event.Details), `secret`) {
		t.Fatalf("audit details missing protected key: %s", event.Details)
	}
}
