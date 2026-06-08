package api

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
	"github.com/deposist/s-ui-x/realtime"
	"github.com/deposist/s-ui-x/service"

	"github.com/gin-gonic/gin"
)

func TestImportXuiRollbackPublishesConfigInvalidatedIssue39(t *testing.T) {
	settingService, src := setupXuiAPITestDB(t)
	database.SetSendSighupHook(func() error { return nil })
	t.Cleanup(func() { database.SetSendSighupHook(nil) })

	router, cookies := newAuthenticatedTestRouter(t, settingService, func(router *gin.Engine) {
		router.POST("/api/import-xui/plan", withTestTokenScope("admin", "admin", (&ApiService{}).ImportXuiPlan))
		router.POST("/api/import-xui/apply", withTestTokenScope("admin", "admin", (&ApiService{}).ImportXuiApply))
		router.POST("/api/import-xui/rollback", withTestTokenScope("admin", "admin", (&ApiService{}).ImportXuiRollback))
	})

	planRecorder := performAuthenticatedTestRequest(router, newXuiImportRequest(t, "/api/import-xui/plan", readFile(t, src), "1"), cookies...)
	if planRecorder.Code != http.StatusOK {
		t.Fatalf("plan status=%d body=%s", planRecorder.Code, planRecorder.Body.String())
	}
	plan := decodePlanResponse(t, planRecorder.Body.Bytes())
	applyRecorder := performAuthenticatedTestRequest(router, newXuiApplyRequest(t, readFile(t, src), plan), cookies...)
	if applyRecorder.Code != http.StatusOK {
		t.Fatalf("apply status=%d body=%s", applyRecorder.Code, applyRecorder.Body.String())
	}
	report := decodeReportResponse(t, applyRecorder.Body.Bytes())
	if report.BackupPath == "" {
		t.Fatal("apply response did not include backup path")
	}

	events, unregister := registerImportXuiRealtimeIssue39(t)
	defer unregister()

	body := strings.NewReader("backup=" + url.QueryEscape(report.BackupPath))
	req := httptest.NewRequest(http.MethodPost, "/api/import-xui/rollback", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rollbackRecorder := performAuthenticatedTestRequest(router, req, cookies...)
	if rollbackRecorder.Code != http.StatusOK {
		t.Fatalf("rollback status=%d body=%s", rollbackRecorder.Code, rollbackRecorder.Body.String())
	}

	expectImportXuiRealtimeTopicIssue39(t, events, realtime.TopicConfigInvalidated)

	var event model.AuditEvent
	if err := database.GetDB().Where("event = ?", "xui_import_rollback").Order("id desc").First(&event).Error; err != nil {
		t.Fatal(err)
	}
	if event.Actor != "admin" || event.Resource != "database" || event.Severity != service.AuditSeverityWarn {
		t.Fatalf("unexpected rollback audit: %#v", event)
	}
	if !strings.Contains(string(event.Details), filepath.Base(report.BackupPath)) {
		t.Fatalf("rollback audit did not include backup basename: %s", event.Details)
	}
}

func TestImportXuiRollbackInvalidBackupDoesNotPublishIssue39(t *testing.T) {
	settingService, _ := setupXuiAPITestDB(t)
	router, cookies := newAuthenticatedTestRouter(t, settingService, func(router *gin.Engine) {
		router.POST("/api/import-xui/rollback", withTestTokenScope("admin", "admin", (&ApiService{}).ImportXuiRollback))
	})
	events, unregister := registerImportXuiRealtimeIssue39(t)
	defer unregister()

	req := httptest.NewRequest(http.MethodPost, "/api/import-xui/rollback", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := performAuthenticatedTestRequest(router, req, cookies...)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("missing backup path should return 400, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	expectNoImportXuiRealtimeIssue39(t, events)
}

func registerImportXuiRealtimeIssue39(t *testing.T) (<-chan realtime.Event, func()) {
	t.Helper()
	realtime.CloseAll("issue39_reset")
	t.Cleanup(func() { realtime.CloseAll("issue39_done") })
	events := make(chan realtime.Event, 2)
	unregister := realtime.Register(&realtime.ClientHandle{
		User:   "admin",
		Scope:  realtime.ScopeAdmin,
		SendCh: events,
	})
	return events, unregister
}

func expectImportXuiRealtimeTopicIssue39(t *testing.T, events <-chan realtime.Event, topic realtime.Topic) {
	t.Helper()
	select {
	case event := <-events:
		if event.Type != topic {
			t.Fatalf("expected realtime topic %s, got %s", topic, event.Type)
		}
	case <-time.After(time.Second):
		t.Fatalf("realtime topic %s was not published", topic)
	}
}

func expectNoImportXuiRealtimeIssue39(t *testing.T, events <-chan realtime.Event) {
	t.Helper()
	select {
	case event := <-events:
		t.Fatalf("unexpected realtime event: %s", event.Type)
	case <-time.After(100 * time.Millisecond):
	}
}
