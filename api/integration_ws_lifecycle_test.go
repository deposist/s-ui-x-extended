package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
	"github.com/deposist/s-ui-x/realtime"
	"github.com/deposist/s-ui-x/util/common"

	"github.com/coder/websocket"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

func TestIntegrationRealtimeWSIssueConnectPublishCloseReconnect(t *testing.T) {
	resetRateLimitState()
	resetRealtimeForTest()
	router := newIntegrationWSRouter(t)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	cookies := loginIntegrationWSUser(t, router, "admin")
	token := issueIntegrationWSToken(t, server, cookies)
	conn := dialIntegrationWS(t, server, cookies, token)
	connected := readIntegrationWSEvent(t, conn)
	if connected.Type != realtime.Topic("connected") {
		t.Fatalf("expected connected event, got %s", connected.Type)
	}

	realtime.Publish(realtime.TopicNotification, map[string]any{"phase": "phase3"})
	event := readIntegrationWSEvent(t, conn)
	if event.Type != realtime.TopicNotification {
		t.Fatalf("expected notification event, got %s", event.Type)
	}
	_ = conn.Close(websocket.StatusNormalClosure, "phase3 reconnect")

	token = issueIntegrationWSToken(t, server, cookies)
	reconnected := dialIntegrationWS(t, server, cookies, token)
	t.Cleanup(func() { _ = reconnected.CloseNow() })
	if event := readIntegrationWSEvent(t, reconnected); event.Type != realtime.Topic("connected") {
		t.Fatalf("expected connected event after reconnect, got %s", event.Type)
	}
}

func TestIntegrationRealtimeWSMultipleClientsReceivePublish(t *testing.T) {
	resetRateLimitState()
	resetRealtimeForTest()
	router := newIntegrationWSRouter(t)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	cookies := loginIntegrationWSUser(t, router, "admin")

	conns := make([]*websocket.Conn, 0, 2)
	for i := 0; i < 2; i++ {
		token := issueIntegrationWSToken(t, server, cookies)
		conn := dialIntegrationWS(t, server, cookies, token)
		conns = append(conns, conn)
		t.Cleanup(func() { _ = conn.CloseNow() })
		if event := readIntegrationWSEvent(t, conn); event.Type != realtime.Topic("connected") {
			t.Fatalf("client %d expected connected event, got %s", i, event.Type)
		}
	}

	realtime.Publish(realtime.TopicConfigInvalidated, nil)
	for i, conn := range conns {
		if event := readIntegrationWSEvent(t, conn); event.Type != realtime.TopicConfigInvalidated {
			t.Fatalf("client %d expected config invalidated event, got %s", i, event.Type)
		}
	}
}

func TestIntegrationRealtimeWSMaxPerUserCapacity(t *testing.T) {
	resetRateLimitState()
	resetRealtimeForTest()
	router := newIntegrationWSRouterWithOptions(t, WithPingInterval(time.Hour), WithPingTimeout(time.Second))
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	cookies := loginIntegrationWSUser(t, router, "admin")

	for i := 0; i < maxWSPerUser; i++ {
		token := issueIntegrationWSToken(t, server, cookies)
		conn := dialIntegrationWS(t, server, cookies, token)
		t.Cleanup(func() { _ = conn.CloseNow() })
		if event := readIntegrationWSEvent(t, conn); event.Type != realtime.Topic("connected") {
			t.Fatalf("client %d expected connected event, got %s", i, event.Type)
		}
	}

	token := issueIntegrationWSToken(t, server, cookies)
	overflow, resp, err := dialIntegrationWSRaw(t, server, cookies, token)
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	if err == nil {
		_ = overflow.CloseNow()
		t.Fatal("expected maxWSPerUser overflow to reject websocket")
	}
	if resp == nil || resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("unexpected overflow response: resp=%v err=%v", resp, err)
	}
}

func TestIntegrationRealtimeWSMaxPerIPCapacity(t *testing.T) {
	resetRateLimitState()
	resetRealtimeForTest()
	router := newIntegrationWSRouterWithOptions(t, WithPingInterval(time.Hour), WithPingTimeout(time.Second))
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	for i := 0; i < maxWSPerIP; i++ {
		cookies := loginIntegrationWSUser(t, router, fmt.Sprintf("user-%02d", i))
		token := issueIntegrationWSToken(t, server, cookies)
		conn := dialIntegrationWS(t, server, cookies, token)
		t.Cleanup(func() { _ = conn.CloseNow() })
		if event := readIntegrationWSEvent(t, conn); event.Type != realtime.Topic("connected") {
			t.Fatalf("client %d expected connected event, got %s", i, event.Type)
		}
	}

	overflowCookies := loginIntegrationWSUser(t, router, "user-overflow")
	token := issueIntegrationWSToken(t, server, overflowCookies)
	overflow, resp, err := dialIntegrationWSRaw(t, server, overflowCookies, token)
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	if err == nil {
		_ = overflow.CloseNow()
		t.Fatal("expected maxWSPerIP overflow to reject websocket")
	}
	if resp == nil || resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("unexpected IP overflow response: resp=%v err=%v", resp, err)
	}
}

func TestIntegrationRealtimeWSSlowClientDrop_XFAILPhase3(t *testing.T) {
	t.Skip("XFAIL Phase3: требуется hook для детерминированной блокировки websocket writer / заполнения ws send queue; связано с реестром п. 32 по WS reliability")
}

func newIntegrationWSRouter(t *testing.T) *gin.Engine {
	t.Helper()
	return newIntegrationWSRouterWithOptions(t, WithPingInterval(time.Second), WithPingTimeout(time.Second))
}

func newIntegrationWSRouterWithOptions(t *testing.T, options ...realtimeOption) *gin.Engine {
	t.Helper()
	settingService := initSessionTestDB(t)
	if _, err := settingService.GetAllSetting(); err != nil {
		t.Fatal(err)
	}
	gin.SetMode(gin.TestMode)
	router := gin.New()
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
	router.GET("/api/realtime/ws", (&ApiService{}).RealtimeWSWithOptions(options...))
	return router
}

func ensureIntegrationWSSessionUser(t *testing.T, username string) bool {
	t.Helper()
	var count int64
	if err := database.GetDB().Model(model.User{}).Where("username = ?", username).Count(&count).Error; err != nil {
		t.Logf("count session user %q failed: %v", username, err)
		return false
	}
	if count > 0 {
		return true
	}
	passwordHash, err := common.HashPassword("integration-ws-password")
	if err != nil {
		t.Logf("hash session user password failed: %v", err)
		return false
	}
	if err := database.GetDB().Create(&model.User{Username: username, Password: passwordHash}).Error; err != nil {
		t.Logf("create session user %q failed: %v", username, err)
		return false
	}
	return true
}

func loginIntegrationWSUser(t *testing.T, router *gin.Engine, user string) []*http.Cookie {
	t.Helper()
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/login/"+url.PathEscape(user), nil)
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("login %q returned %d", user, recorder.Code)
	}
	cookies := recorder.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatalf("login %q did not set session cookie", user)
	}
	return cookies
}

func issueIntegrationWSToken(t *testing.T, server *httptest.Server, cookies []*http.Cookie) string {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL+"/api/realtime/ws-token", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Cookie", cookieHeader(cookies))
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
		t.Fatalf("ws-token returned %d body=%s", resp.StatusCode, string(body))
	}
	var msg struct {
		Success bool   `json:"success"`
		Msg     string `json:"msg"`
		Obj     struct {
			Token string `json:"token"`
		} `json:"obj"`
	}
	if err := json.Unmarshal(body, &msg); err != nil {
		t.Fatal(err)
	}
	if !msg.Success || msg.Obj.Token == "" {
		t.Fatalf("unexpected ws-token response: %#v body=%s", msg, string(body))
	}
	return msg.Obj.Token
}

func dialIntegrationWS(t *testing.T, server *httptest.Server, cookies []*http.Cookie, token string) *websocket.Conn {
	t.Helper()
	conn, resp, err := dialIntegrationWSRaw(t, server, cookies, token)
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		t.Fatalf("dial websocket failed: err=%v", err)
	}
	return conn
}

func dialIntegrationWSRaw(t *testing.T, server *httptest.Server, cookies []*http.Cookie, token string) (*websocket.Conn, *http.Response, error) {
	t.Helper()
	header := http.Header{}
	header.Set("Origin", server.URL)
	header.Set("Cookie", cookieHeader(cookies))
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/api/realtime/ws?token=" + url.QueryEscape(token)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	conn, resp, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{
		HTTPHeader: header,
		OnPingReceived: func(context.Context, []byte) bool {
			return true
		},
	})
	return conn, resp, err
}

func readIntegrationWSEvent(t *testing.T, conn *websocket.Conn) realtime.Event {
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
	var event realtime.Event
	if err := json.Unmarshal(body, &event); err != nil {
		t.Fatal(err)
	}
	return event
}
