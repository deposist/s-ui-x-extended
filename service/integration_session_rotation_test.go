package service_test

import (
	"context"
	"encoding/json"
	"io"
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

	"github.com/coder/websocket"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

func TestIntegrationSessionRotationClosesWSInvalidatesTokensAndAudits(t *testing.T) {
	settingService := initSessionRotationIntegrationDB(t)
	router := newSessionRotationIntegrationRouter(t, settingService)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	cookies := loginSessionRotationUser(t, router, "admin")
	connectedToken := issueSessionRotationWSToken(t, server, cookies)
	unusedToken := issueSessionRotationWSToken(t, server, cookies)
	if connectedToken == unusedToken {
		t.Fatal("ws-token endpoint returned duplicate tokens")
	}
	conn := dialSessionRotationWS(t, server, cookies, connectedToken)
	t.Cleanup(func() { _ = conn.CloseNow() })
	if event := readSessionRotationWSEvent(t, conn); event.Type != "connected" {
		t.Fatalf("expected connected event, got %s", event.Type)
	}

	if _, err := settingService.RotateSessionGeneration(); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, _, err := conn.Reader(ctx)
	if err == nil {
		t.Fatal("websocket stayed open after session rotation")
	}
	if got := websocket.CloseStatus(err); got != websocket.StatusCode(4401) {
		t.Fatalf("expected close code 4401, got %v err=%v", got, err)
	}
	if !strings.Contains(err.Error(), "session_rotated") {
		t.Fatalf("expected close reason session_rotated, got %v", err)
	}

	var audit model.AuditEvent
	if err := database.GetDB().Where("event = ?", "ws_tokens_invalidated").Order("id desc").First(&audit).Error; err != nil {
		t.Fatal(err)
	}
	if audit.Actor != "system" || audit.Resource != "realtime" || audit.Severity != service.AuditSeverityInfo {
		t.Fatalf("unexpected audit event: %#v", audit)
	}
	if !strings.Contains(string(audit.Details), `"count":1`) {
		t.Fatalf("unused websocket token should be invalidated, details=%s", audit.Details)
	}
}

func initSessionRotationIntegrationDB(t *testing.T) *service.SettingService {
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
	return &service.SettingService{}
}

func newSessionRotationIntegrationRouter(t *testing.T, settingService *service.SettingService) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(sessions.Sessions("s-ui", cookie.NewStore([]byte("test-secret"))))
	router.GET("/login/:user", func(c *gin.Context) {
		generation, err := settingService.GetSessionGeneration()
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		if err := api.SetLoginUser(c, c.Param("user"), 0, generation); err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		c.Status(http.StatusNoContent)
	})
	router.GET("/api/realtime/ws-token", (&api.ApiService{}).IssueWSToken)
	router.GET("/api/realtime/ws", (&api.ApiService{}).RealtimeWS)
	return router
}

func loginSessionRotationUser(t *testing.T, router *gin.Engine, user string) []*http.Cookie {
	t.Helper()
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/login/"+user, nil))
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("login returned %d", recorder.Code)
	}
	return recorder.Result().Cookies()
}

func issueSessionRotationWSToken(t *testing.T, server *httptest.Server, cookies []*http.Cookie) string {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL+"/api/realtime/ws-token", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Cookie", sessionRotationCookieHeader(cookies))
	resp, err := server.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("ws-token status=%d body=%s", resp.StatusCode, string(body))
	}
	var msg struct {
		Success bool `json:"success"`
		Obj     struct {
			Token string `json:"token"`
		} `json:"obj"`
	}
	if err := json.Unmarshal(body, &msg); err != nil {
		t.Fatal(err)
	}
	if !msg.Success || msg.Obj.Token == "" {
		t.Fatalf("unexpected ws-token response: %s", string(body))
	}
	return msg.Obj.Token
}

func dialSessionRotationWS(t *testing.T, server *httptest.Server, cookies []*http.Cookie, token string) *websocket.Conn {
	t.Helper()
	header := http.Header{}
	header.Set("Origin", server.URL)
	header.Set("Cookie", sessionRotationCookieHeader(cookies))
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/api/realtime/ws?token=" + token
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	conn, resp, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{HTTPHeader: header})
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		t.Fatal(err)
	}
	return conn
}

func readSessionRotationWSEvent(t *testing.T, conn *websocket.Conn) struct {
	Type string `json:"type"`
} {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, reader, err := conn.Reader(ctx)
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	var event struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(body, &event); err != nil {
		t.Fatal(err)
	}
	return event
}

func sessionRotationCookieHeader(cookies []*http.Cookie) string {
	parts := make([]string, 0, len(cookies))
	for _, c := range cookies {
		parts = append(parts, c.String())
	}
	return strings.Join(parts, "; ")
}
