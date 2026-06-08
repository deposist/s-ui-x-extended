package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
	"github.com/deposist/s-ui-x/realtime"
	"github.com/deposist/s-ui-x/service"

	"github.com/coder/websocket"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

func resetRealtimeForTest() {
	_ = sweepAllWSTokens()
	realtime.CloseAll("test_reset")
}

func setWSTokenForTest(token string, user string) {
	wsTokens.Lock()
	wsTokens.tokens[wsTokenDigest(token)] = realtimeToken{user: user, expiresAt: time.Now().Add(time.Minute)}
	wsTokens.Unlock()
}

func hasWSTokenForTest(token string) bool {
	wsTokens.Lock()
	defer wsTokens.Unlock()
	_, ok := wsTokens.tokens[wsTokenDigest(token)]
	return ok
}

func wsTokenCountForTest() int {
	wsTokens.Lock()
	defer wsTokens.Unlock()
	return len(wsTokens.tokens)
}

func TestConsumeWSTokenIsOneTime(t *testing.T) {
	resetRealtimeForTest()
	setWSTokenForTest("token", "admin")

	user, ok := consumeWSToken("token")
	if !ok || user != "admin" {
		t.Fatalf("expected first consume to work, got user=%q ok=%v", user, ok)
	}
	if _, ok := consumeWSToken("token"); ok {
		t.Fatal("expected second consume to fail")
	}
}

func TestWSTokenSweepRemovesExpiredUnusedTokens(t *testing.T) {
	resetRealtimeForTest()
	now := time.Now()

	expiredKey := wsTokenDigest("expired")
	activeKey := wsTokenDigest("active")
	wsTokens.Lock()
	wsTokens.tokens[expiredKey] = realtimeToken{user: "admin", expiresAt: now.Add(-time.Second)}
	wsTokens.tokens[activeKey] = realtimeToken{user: "admin", expiresAt: now.Add(time.Minute)}
	sweepWSTokensLocked(now)
	_, expiredOK := wsTokens.tokens[expiredKey]
	_, activeOK := wsTokens.tokens[activeKey]
	lastSweep := wsTokens.lastSweep
	wsTokens.Unlock()

	if expiredOK {
		t.Fatal("expired websocket token was not swept")
	}
	if !activeOK {
		t.Fatal("active websocket token was swept")
	}
	if !lastSweep.Equal(now) {
		t.Fatalf("lastSweep=%s, want %s", lastSweep, now)
	}
}

func TestWSTokenCapDropsOldestByExpiry(t *testing.T) {
	resetRealtimeForTest()
	base := time.Now()

	wsTokens.Lock()
	for i := 0; i < maxWSTokens+2; i++ {
		token := fmt.Sprintf("token-%04d", i)
		wsTokens.tokens[wsTokenDigest(token)] = realtimeToken{user: "admin", expiresAt: base.Add(time.Duration(i) * time.Millisecond)}
	}
	enforceWSTokenCapLocked()
	count := len(wsTokens.tokens)
	_, firstOK := wsTokens.tokens[wsTokenDigest("token-0000")]
	_, secondOK := wsTokens.tokens[wsTokenDigest("token-0001")]
	_, thirdOK := wsTokens.tokens[wsTokenDigest("token-0002")]
	wsTokens.Unlock()

	if count != maxWSTokens {
		t.Fatalf("token count=%d, want %d", count, maxWSTokens)
	}
	if firstOK || secondOK {
		t.Fatal("oldest websocket tokens were not dropped")
	}
	if !thirdOK {
		t.Fatal("newer websocket token was dropped")
	}
}

func TestSweepAllWSTokensClearsIssuedTokens(t *testing.T) {
	resetRealtimeForTest()
	setWSTokenForTest("token-a", "admin")
	setWSTokenForTest("token-b", "admin")

	if got := sweepAllWSTokens(); got != 2 {
		t.Fatalf("swept token count=%d, want 2", got)
	}
	if wsTokenCountForTest() != 0 {
		t.Fatal("websocket tokens remained after sweepAll")
	}
	if _, ok := consumeWSToken("token-a"); ok {
		t.Fatal("swept token was still accepted")
	}
}

func TestRotateSessionGenerationInvalidatesWSTokens(t *testing.T) {
	settingService := initSessionTestDB(t)
	resetRealtimeForTest()
	setWSTokenForTest("rotated-token", "admin")
	prevAuditSync := service.AuditSyncForTest
	service.AuditSyncForTest = true
	t.Cleanup(func() { service.AuditSyncForTest = prevAuditSync })

	if _, err := settingService.RotateSessionGeneration(); err != nil {
		t.Fatal(err)
	}
	if _, ok := consumeWSToken("rotated-token"); ok {
		t.Fatal("websocket token survived session generation rotation")
	}

	var event model.AuditEvent
	if err := database.GetDB().Where("event = ?", "ws_tokens_invalidated").First(&event).Error; err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(event.Details), `"count":1`) {
		t.Fatalf("unexpected invalidation audit details: %s", event.Details)
	}
}

func TestWSTokenFromRequestAcceptsPrefixedSubprotocolAnyOrder(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/realtime/ws", nil)
	ctx.Request.Header.Set("Sec-WebSocket-Protocol", "sui.realtime, extra.v1, sui.token.prefixed-token")

	token, legacy := wsTokenFromRequest(ctx)
	if token != "prefixed-token" || legacy {
		t.Fatalf("token=%q legacy=%v, want prefixed token without legacy flag", token, legacy)
	}
}

func TestWSTokenFromRequestAcceptsDeprecatedLegacySubprotocol(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/realtime/ws", nil)
	ctx.Request.Header.Set("Sec-WebSocket-Protocol", "legacy-token, sui.realtime")

	token, legacy := wsTokenFromRequest(ctx)
	if token != "legacy-token" || !legacy {
		t.Fatalf("token=%q legacy=%v, want legacy token with flag", token, legacy)
	}
}

func TestWSOriginAllowedAcceptsRequestHostAndWebDomain(t *testing.T) {
	tests := []struct {
		name        string
		origin      string
		requestHost string
		webDomain   string
		wantAllowed bool
	}{
		{
			name:        "request host",
			origin:      "https://panel.example:2095",
			requestHost: "panel.example:2095",
			wantAllowed: true,
		},
		{
			name:        "configured web domain",
			origin:      "https://admin.example",
			requestHost: "127.0.0.1:2095",
			webDomain:   "admin.example",
			wantAllowed: true,
		},
		{
			name:        "foreign host",
			origin:      "https://evil.example",
			requestHost: "panel.example",
			webDomain:   "admin.example",
			wantAllowed: false,
		},
		{
			name:        "invalid scheme",
			origin:      "file://panel.example",
			requestHost: "panel.example",
			wantAllowed: false,
		},
		{
			name:        "origin with query",
			origin:      "https://panel.example?token=secret",
			requestHost: "panel.example",
			wantAllowed: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed, _ := wsOriginAllowed(tt.origin, tt.requestHost, tt.webDomain)
			if allowed != tt.wantAllowed {
				t.Fatalf("allowed=%v, want %v", allowed, tt.wantAllowed)
			}
		})
	}
}

func TestIssueWSTokenRejectsForeignOriginAndAudits(t *testing.T) {
	resetRateLimitState()
	settingService := initSessionTestDB(t)
	if _, err := settingService.GetAllSetting(); err != nil {
		t.Fatal(err)
	}
	resetRealtimeForTest()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(sessions.Sessions("s-ui", cookie.NewStore([]byte("test-secret"))))
	router.GET("/login", func(c *gin.Context) {
		generation, err := settingService.GetSessionGeneration()
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		if err := SetLoginUser(c, "admin", 0, generation); err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		c.Status(http.StatusNoContent)
	})
	router.GET("/api/realtime/ws-token", (&ApiService{}).IssueWSToken)

	loginRecorder := httptest.NewRecorder()
	loginReq := httptest.NewRequest(http.MethodGet, "/login", nil)
	router.ServeHTTP(loginRecorder, loginReq)
	if loginRecorder.Code != http.StatusNoContent {
		t.Fatalf("login returned %d", loginRecorder.Code)
	}

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://panel.example/api/realtime/ws-token", nil)
	req.Host = "panel.example"
	req.Header.Set("Origin", "https://evil.example")
	for _, c := range loginRecorder.Result().Cookies() {
		req.AddCookie(c)
	}
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("unexpected status %d", recorder.Code)
	}
	if wsTokenCountForTest() != 0 {
		t.Fatal("foreign Origin request issued a websocket token")
	}

	var event model.AuditEvent
	if err := database.GetDB().Where("event = ?", "ws_origin_rejected").First(&event).Error; err != nil {
		t.Fatal(err)
	}
	if event.Actor != "admin" {
		t.Fatalf("unexpected audit actor: %q", event.Actor)
	}
	if strings.Contains(string(event.Details), "ws-token") || strings.Contains(string(event.Details), "token") {
		t.Fatalf("websocket token leaked to audit details: %s", event.Details)
	}
}

func TestRealtimeWSRejectsForeignOriginAuditsAndKeepsToken(t *testing.T) {
	settingService := initSessionTestDB(t)
	if _, err := settingService.GetAllSetting(); err != nil {
		t.Fatal(err)
	}
	resetRealtimeForTest()
	setWSTokenForTest("ws-token", "admin")

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(sessions.Sessions("s-ui", cookie.NewStore([]byte("test-secret"))))
	router.GET("/login", func(c *gin.Context) {
		generation, err := settingService.GetSessionGeneration()
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		if err := SetLoginUser(c, "admin", 0, generation); err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		c.Status(http.StatusNoContent)
	})
	router.GET("/api/realtime/ws", (&ApiService{}).RealtimeWS)

	loginRecorder := httptest.NewRecorder()
	loginReq := httptest.NewRequest(http.MethodGet, "/login", nil)
	router.ServeHTTP(loginRecorder, loginReq)
	if loginRecorder.Code != http.StatusNoContent {
		t.Fatalf("login returned %d", loginRecorder.Code)
	}

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://panel.example/api/realtime/ws?token=ws-token", nil)
	req.Host = "panel.example"
	req.Header.Set("Origin", "https://evil.example")
	for _, c := range loginRecorder.Result().Cookies() {
		req.AddCookie(c)
	}
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("unexpected status %d", recorder.Code)
	}

	if !hasWSTokenForTest("ws-token") {
		t.Fatal("foreign Origin consumed the websocket token")
	}

	var event model.AuditEvent
	if err := database.GetDB().Where("event = ?", "ws_origin_rejected").First(&event).Error; err != nil {
		t.Fatal(err)
	}
	var details map[string]any
	if err := json.Unmarshal(event.Details, &details); err != nil {
		t.Fatal(err)
	}
	if details["originHost"] != "evil.example" || details["requestHost"] != "panel.example" {
		t.Fatalf("unexpected audit details: %#v", details)
	}
	if _, ok := details["token"]; ok {
		t.Fatalf("websocket token leaked to audit details: %#v", details)
	}
}

func TestRealtimeWSSendsHeartbeatPing(t *testing.T) {
	router, cookies := newRealtimeWSTestRouterWithOptions(t, WithPingInterval(10*time.Millisecond), WithPingTimeout(100*time.Millisecond))
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	resetRealtimeForTest()
	setWSTokenForTest("ws-token", "admin")

	var pings atomic.Int32
	conn := dialRealtimeWSForTest(t, server, cookies, func(context.Context, []byte) bool {
		pings.Add(1)
		return true
	})
	t.Cleanup(func() { _ = conn.CloseNow() })
	errCh := startRealtimeReadLoop(conn)

	deadline := time.After(time.Second)
	tick := time.NewTicker(5 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case err := <-errCh:
			t.Fatalf("websocket closed before heartbeat ping: %v", err)
		case <-tick.C:
			if pings.Load() > 0 {
				return
			}
		case <-deadline:
			t.Fatal("heartbeat ping was not observed")
		}
	}
}

func TestRealtimeWSSendsPublishedEvents(t *testing.T) {
	router, cookies := newRealtimeWSTestRouter(t)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	resetRealtimeForTest()
	setWSTokenForTest("ws-token", "admin")

	conn := dialRealtimeWSForTest(t, server, cookies, func(context.Context, []byte) bool {
		return true
	})
	t.Cleanup(func() { _ = conn.CloseNow() })

	connected := readRealtimeEventForTest(t, conn)
	if connected.Type != realtime.Topic("connected") {
		t.Fatalf("expected connected event, got %s", connected.Type)
	}

	realtime.Publish(realtime.TopicNotification, map[string]any{"kind": "test"})
	event := readRealtimeEventForTest(t, conn)
	if event.Type != realtime.TopicNotification {
		t.Fatalf("expected published notification, got %s", event.Type)
	}
}

func TestRealtimeWSUsesTokenScopeForSecurityEvents(t *testing.T) {
	tests := []struct {
		name         string
		scope        string
		wantSecurity bool
	}{
		{name: "admin scope receives security event", scope: "admin", wantSecurity: true},
		{name: "read scope filters security event", scope: "read", wantSecurity: false},
		{name: "write scope filters security event", scope: "write", wantSecurity: false},
		{name: "observability scope filters security event", scope: "observability", wantSecurity: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router, cookies := newRealtimeWSTestRouterWithScope(t, tt.scope)
			server := httptest.NewServer(router)
			t.Cleanup(server.Close)

			resetRealtimeForTest()
			setWSTokenForTest("ws-token", "admin")

			conn := dialRealtimeWSForTest(t, server, cookies, func(context.Context, []byte) bool {
				return true
			})
			t.Cleanup(func() { _ = conn.CloseNow() })

			connected := readRealtimeEventForTest(t, conn)
			if connected.Type != realtime.Topic("connected") {
				t.Fatalf("expected connected event, got %s", connected.Type)
			}

			realtime.Publish(realtime.TopicSecurityEvent, map[string]any{"kind": "scope"})
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			event, err := readRealtimeEvent(ctx, conn)
			if tt.wantSecurity {
				if err != nil {
					t.Fatalf("expected security event for scope %q: %v", tt.scope, err)
				}
				if event.Type != realtime.TopicSecurityEvent {
					t.Fatalf("expected security event, got %s", event.Type)
				}
				return
			}
			if err == nil {
				t.Fatalf("scope %q received filtered event: %#v", tt.scope, event)
			}
		})
	}
}

func TestRealtimeScopeFromContextDefaultsSessionToAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	if got := realtimeScopeFromContext(ctx); got != realtime.ScopeAdmin {
		t.Fatalf("session flow should default to admin scope, got %q", got)
	}
	ctx.Set(apiTokenScopeKey, "observability")
	if got := realtimeScopeFromContext(ctx); got != realtime.ScopeObservability {
		t.Fatalf("unexpected bearer scope: %q", got)
	}
	ctx.Set(apiTokenScopeKey, "unknown")
	if got := realtimeScopeFromContext(ctx); got != realtime.ScopeRead {
		t.Fatalf("unknown scope should fall back to least privilege, got %q", got)
	}
}

func TestRealtimeWSDeliversEventsWhileHeartbeatWaitsForPong(t *testing.T) {
	router, cookies := newRealtimeWSTestRouterWithOptions(t, WithPingInterval(10*time.Millisecond), WithPingTimeout(200*time.Millisecond))
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	resetRealtimeForTest()
	setWSTokenForTest("ws-token", "admin")

	var pings atomic.Int32
	conn := dialRealtimeWSForTest(t, server, cookies, func(context.Context, []byte) bool {
		pings.Add(1)
		return false
	})
	t.Cleanup(func() { _ = conn.CloseNow() })

	connected := readRealtimeEventForTest(t, conn)
	if connected.Type != realtime.Topic("connected") {
		t.Fatalf("expected connected event, got %s", connected.Type)
	}

	eventCh := make(chan realtime.Event, 1)
	errCh := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		event, err := readRealtimeEvent(ctx, conn)
		if err != nil {
			errCh <- err
			return
		}
		eventCh <- event
	}()

	deadline := time.After(time.Second)
	tick := time.NewTicker(5 * time.Millisecond)
	defer tick.Stop()
	for pings.Load() == 0 {
		select {
		case <-tick.C:
		case err := <-errCh:
			t.Fatalf("websocket closed before test event publish: %v", err)
		case <-deadline:
			t.Fatal("heartbeat ping was not observed")
		}
	}

	realtime.Publish(realtime.TopicNotification, map[string]any{"kind": "during-ping"})
	select {
	case event := <-eventCh:
		if event.Type != realtime.TopicNotification {
			t.Fatalf("expected published notification, got %s", event.Type)
		}
	case err := <-errCh:
		t.Fatalf("websocket closed before event delivery: %v", err)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("event delivery was blocked by heartbeat ping")
	}
}

func TestRealtimeWSRejectsReplayToken(t *testing.T) {
	router, cookies := newRealtimeWSTestRouter(t)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	resetRealtimeForTest()
	setWSTokenForTest("ws-token", "admin")

	conn := dialRealtimeWSForTest(t, server, cookies, func(context.Context, []byte) bool {
		return true
	})
	t.Cleanup(func() { _ = conn.CloseNow() })

	header := http.Header{}
	header.Set("Origin", server.URL)
	header.Set("Cookie", cookieHeader(cookies))
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	replayedConn, resp, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(server.URL, "http")+"/api/realtime/ws?token=ws-token", &websocket.DialOptions{
		HTTPHeader: header,
	})
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	if err == nil {
		_ = replayedConn.CloseNow()
		t.Fatal("replayed websocket token was accepted")
	}
	if resp == nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unexpected replay response: resp=%v err=%v", resp, err)
	}
}

func TestRealtimeWSClosesWhenPongMissing(t *testing.T) {
	router, cookies := newRealtimeWSTestRouterWithOptions(t, WithPingInterval(10*time.Millisecond), WithPingTimeout(30*time.Millisecond))
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	resetRealtimeForTest()
	setWSTokenForTest("ws-token", "admin")

	conn := dialRealtimeWSForTest(t, server, cookies, func(context.Context, []byte) bool {
		return false
	})
	t.Cleanup(func() { _ = conn.CloseNow() })
	errCh := startRealtimeReadLoop(conn)

	select {
	case err := <-errCh:
		if websocket.CloseStatus(err) != websocket.StatusInternalError {
			t.Fatalf("expected internal error close, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("websocket did not close after missing pong")
	}
}

func newRealtimeWSTestRouter(t *testing.T) (*gin.Engine, []*http.Cookie) {
	return newRealtimeWSTestRouterWithScope(t, "")
}

func newRealtimeWSTestRouterWithOptions(t *testing.T, options ...realtimeOption) (*gin.Engine, []*http.Cookie) {
	return newRealtimeWSTestRouterWithScopeAndOptions(t, "", options...)
}

func newRealtimeWSTestRouterWithScope(t *testing.T, scope string) (*gin.Engine, []*http.Cookie) {
	return newRealtimeWSTestRouterWithScopeAndOptions(t, scope)
}

func newRealtimeWSTestRouterWithScopeAndOptions(t *testing.T, scope string, options ...realtimeOption) (*gin.Engine, []*http.Cookie) {
	t.Helper()
	settingService := initSessionTestDB(t)
	if _, err := settingService.GetAllSetting(); err != nil {
		t.Fatal(err)
	}
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(sessions.Sessions("s-ui", cookie.NewStore([]byte("test-secret"))))
	if scope != "" {
		router.Use(func(c *gin.Context) {
			c.Set(apiTokenScopeKey, scope)
			c.Next()
		})
	}
	router.GET("/login", func(c *gin.Context) {
		generation, err := settingService.GetSessionGeneration()
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		if err := SetLoginUser(c, "admin", 0, generation); err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		c.Status(http.StatusNoContent)
	})
	router.GET("/api/realtime/ws", (&ApiService{}).RealtimeWSWithOptions(options...))

	loginRecorder := httptest.NewRecorder()
	loginReq := httptest.NewRequest(http.MethodGet, "/login", nil)
	router.ServeHTTP(loginRecorder, loginReq)
	if loginRecorder.Code != http.StatusNoContent {
		t.Fatalf("login returned %d", loginRecorder.Code)
	}
	return router, loginRecorder.Result().Cookies()
}

func dialRealtimeWSForTest(t *testing.T, server *httptest.Server, cookies []*http.Cookie, onPing func(context.Context, []byte) bool) *websocket.Conn {
	t.Helper()
	header := http.Header{}
	header.Set("Origin", server.URL)
	header.Set("Cookie", cookieHeader(cookies))
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/api/realtime/ws?token=ws-token"
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	conn, resp, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{
		HTTPHeader:     header,
		OnPingReceived: onPing,
	})
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		t.Fatal(err)
	}
	return conn
}

func startRealtimeReadLoop(conn *websocket.Conn) <-chan error {
	errCh := make(chan error, 1)
	go func() {
		for {
			_, reader, err := conn.Reader(context.Background())
			if err != nil {
				errCh <- err
				return
			}
			if _, err := io.Copy(io.Discard, reader); err != nil {
				errCh <- err
				return
			}
		}
	}()
	return errCh
}

func readRealtimeEventForTest(t *testing.T, conn *websocket.Conn) realtime.Event {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	event, err := readRealtimeEvent(ctx, conn)
	if err != nil {
		t.Fatal(err)
	}
	return event
}

func readRealtimeEvent(ctx context.Context, conn *websocket.Conn) (realtime.Event, error) {
	_, reader, err := conn.Reader(ctx)
	if err != nil {
		return realtime.Event{}, err
	}
	body, err := io.ReadAll(reader)
	if err != nil {
		return realtime.Event{}, err
	}
	var event realtime.Event
	if err := json.Unmarshal(body, &event); err != nil {
		return realtime.Event{}, err
	}
	return event, nil
}

func cookieHeader(cookies []*http.Cookie) string {
	parts := make([]string, 0, len(cookies))
	for _, c := range cookies {
		parts = append(parts, c.String())
	}
	return strings.Join(parts, "; ")
}
