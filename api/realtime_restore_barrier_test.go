package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/realtime"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

// WS-03: an open realtime dashboard socket must not stall a database restore.
// The engine-wide request lease used to cover the whole WebSocket session, and
// ImportDB drains readers before swapping SQLite, so the restore deadlocked
// behind the first open socket: the panel froze, cron jobs logged "cron: skip"
// on every tick behind the drain barrier, and only a manual service restart
// (which closed the socket) recovered. The WS handler now holds the lease only
// for its DB-touching preamble.
func TestOpenRealtimeWebSocketDoesNotBlockDatabaseRestore(t *testing.T) {
	resetRateLimitState()
	resetRealtimeForTest()

	settingService := initSessionTestDB(t)
	if _, err := settingService.GetAllSetting(); err != nil {
		t.Fatal(err)
	}
	database.SetSendSighupHook(func() error { return nil })
	t.Cleanup(func() { database.SetSendSighupHook(nil) })
	// ImportDB installs a new live handle; close that replacement before the
	// temporary DB directory cleanup on Windows releases its file lock.
	t.Cleanup(func() {
		if live := database.GetDB(); live != nil {
			if sqlDB, err := live.DB(); err == nil {
				_ = sqlDB.Close()
			}
		}
		time.Sleep(50 * time.Millisecond)
	})

	backup, err := database.GetDb("")
	if err != nil {
		t.Fatal(err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(databaseOperationMiddleware())
	router.Use(sessions.Sessions("s-ui", cookie.NewStore([]byte("test-secret"))))
	router.GET("/login/:user", func(c *gin.Context) {
		if !ensureIntegrationWSSessionUser(t, c.Param("user")) {
			c.Status(http.StatusInternalServerError)
			return
		}
		generation, err := settingService.GetSessionGeneration()
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		if err := SetLoginUser(c, c.Param("user"), 0, generation); err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		c.Status(http.StatusNoContent)
	})
	router.GET("/api/realtime/ws-token", (&ApiService{}).IssueWSToken)
	router.GET("/api/realtime/ws", (&ApiService{}).RealtimeWS)

	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	cookies := loginIntegrationWSUser(t, router, "admin")
	token := issueIntegrationWSToken(t, server, cookies)
	conn := dialIntegrationWS(t, server, cookies, token)
	t.Cleanup(func() { _ = conn.CloseNow() })
	if event := readIntegrationWSEvent(t, conn); event.Type != realtime.Topic("connected") {
		t.Fatalf("expected connected event, got %s", event.Type)
	}

	restoreDone := make(chan error, 1)
	go func() {
		restoreDone <- database.ImportDB(maintenanceBarrierFile{Reader: bytes.NewReader(backup)}, func() error { return nil })
	}()
	select {
	case err := <-restoreDone:
		if err != nil {
			t.Fatalf("restore failed with socket open: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("restore deadlocked behind open realtime websocket")
	}

	// The socket must survive the restore, proving the connection loop itself
	// never held the maintenance lease.
	realtime.Publish(realtime.TopicNotification, map[string]any{"phase": "restore-complete"})
	if event := readIntegrationWSEvent(t, conn); event.Type != realtime.TopicNotification {
		t.Fatalf("expected notification event after restore, got %s", event.Type)
	}
}

func TestIsLongLivedStreamPath(t *testing.T) {
	for _, path := range []string{
		"/api/realtime/ws",
		"/base/api/realtime/ws",
	} {
		if !IsLongLivedStreamPath(path) {
			t.Errorf("IsLongLivedStreamPath(%q) = false, want true", path)
		}
	}
	for _, path := range []string{
		"/api/realtime/ws-token",
		"/api/realtime/ws/",
		"/api/realtime",
		"/api/importdb",
		"/api/import-xui/rollback",
		"/api/observability/history",
	} {
		if IsLongLivedStreamPath(path) {
			t.Errorf("IsLongLivedStreamPath(%q) = true, want false", path)
		}
	}
}
