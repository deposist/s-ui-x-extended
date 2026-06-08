package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
	"github.com/deposist/s-ui-x/realtime"
	"github.com/deposist/s-ui-x/service"
	"github.com/deposist/s-ui-x/util/common"

	"github.com/coder/websocket"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	gormlogger "gorm.io/gorm/logger"
)

func BenchmarkRealtimeWSConnectDisconnect(b *testing.B) {
	for _, clients := range []int{10, 20} {
		clients := clients
		b.Run(fmt.Sprintf("clients_%d", clients), func(b *testing.B) {
			router, cookiesByUser := newRealtimePerfRouter(b, clients)
			server := httptest.NewServer(router)
			defer server.Close()
			users := make([]string, 0, clients)
			for i := 0; i < clients; i++ {
				users = append(users, fmt.Sprintf("phase5-ws-%03d", i))
			}
			for _, user := range users {
				cookiesByUser[user] = loginRealtimePerfUser(b, server, user)
			}
			b.ReportMetric(float64(clients), "clients/op")
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				resetRateLimitState()
				resetRealtimeForTest()
				conns := make([]*websocket.Conn, 0, clients)
				for idx, user := range users {
					token := fmt.Sprintf("phase5-token-%d-%d", i, idx)
					setWSTokenForTest(token, user)
					conn := dialRealtimeWSForBench(b, server, cookiesByUser[user], token)
					readRealtimeEventForBench(b, conn)
					conns = append(conns, conn)
				}
				for _, conn := range conns {
					_ = conn.CloseNow()
				}
			}
		})
	}
}

func TestRealtimeWSCapacityAnchorPhase5(t *testing.T) {
	t.Run("max per user", func(t *testing.T) {
		router, cookiesByUser := newRealtimePerfRouter(t, 1)
		server := httptest.NewServer(router)
		t.Cleanup(server.Close)
		user := "phase5-same-user"
		cookiesByUser[user] = loginRealtimePerfUser(t, server, user)
		resetRateLimitState()
		resetRealtimeForTest()
		conns := make([]*websocket.Conn, 0, maxWSPerUser)
		for i := 0; i < maxWSPerUser; i++ {
			token := fmt.Sprintf("same-user-%d", i)
			setWSTokenForTest(token, user)
			conn := dialRealtimeWSForBench(t, server, cookiesByUser[user], token)
			readRealtimeEventForBench(t, conn)
			conns = append(conns, conn)
		}
		t.Cleanup(func() {
			for _, conn := range conns {
				_ = conn.CloseNow()
			}
		})
		setWSTokenForTest("same-user-over", user)
		_, resp, err := dialRealtimeWSRaw(server, cookiesByUser[user], "same-user-over")
		if resp != nil && resp.Body != nil {
			defer resp.Body.Close()
		}
		if err == nil {
			t.Fatal("expected max per user rejection")
		}
		if resp == nil || resp.StatusCode != http.StatusTooManyRequests {
			t.Fatalf("max per user status=%v err=%v", statusCode(resp), err)
		}
		t.Logf("phase5 ws capacity anchor: maxWSPerUser=%d status=%d", maxWSPerUser, resp.StatusCode)
	})

	t.Run("max per ip", func(t *testing.T) {
		router, cookiesByUser := newRealtimePerfRouter(t, maxWSPerIP+1)
		server := httptest.NewServer(router)
		t.Cleanup(server.Close)
		resetRateLimitState()
		resetRealtimeForTest()
		conns := make([]*websocket.Conn, 0, maxWSPerIP)
		for i := 0; i < maxWSPerIP; i++ {
			user := fmt.Sprintf("phase5-ip-%03d", i)
			cookiesByUser[user] = loginRealtimePerfUser(t, server, user)
			token := fmt.Sprintf("ip-token-%d", i)
			setWSTokenForTest(token, user)
			conn := dialRealtimeWSForBench(t, server, cookiesByUser[user], token)
			readRealtimeEventForBench(t, conn)
			conns = append(conns, conn)
		}
		t.Cleanup(func() {
			for _, conn := range conns {
				_ = conn.CloseNow()
			}
		})
		user := "phase5-ip-over"
		cookiesByUser[user] = loginRealtimePerfUser(t, server, user)
		setWSTokenForTest("ip-token-over", user)
		_, resp, err := dialRealtimeWSRaw(server, cookiesByUser[user], "ip-token-over")
		if resp != nil && resp.Body != nil {
			defer resp.Body.Close()
		}
		if err == nil {
			t.Fatal("expected max per ip rejection")
		}
		if resp == nil || resp.StatusCode != http.StatusTooManyRequests {
			t.Fatalf("max per ip status=%v err=%v", statusCode(resp), err)
		}
		t.Logf("phase5 ws capacity anchor: maxWSPerIP=%d status=%d", maxWSPerIP, resp.StatusCode)
	})
}

func newRealtimePerfRouter(tb testing.TB, users int) (*gin.Engine, map[string][]*http.Cookie) {
	tb.Helper()
	initAPIRealtimePerfDB(tb)
	resetRateLimitState()
	resetRealtimeForTest()
	settingService := &service.SettingService{}
	if _, err := settingService.GetAllSetting(); err != nil {
		tb.Fatal(err)
	}
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(sessions.Sessions("s-ui", cookie.NewStore([]byte("test-secret"))))
	router.GET("/login/:user", func(c *gin.Context) {
		if !ensureRealtimePerfSessionUser(tb, c.Param("user")) {
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
	router.GET("/api/realtime/ws", (&ApiService{}).RealtimeWSWithOptions(WithPingInterval(time.Hour), WithPingTimeout(time.Second)))
	return router, make(map[string][]*http.Cookie, users)
}

func ensureRealtimePerfSessionUser(tb testing.TB, username string) bool {
	tb.Helper()
	var count int64
	if err := database.GetDB().Model(model.User{}).Where("username = ?", username).Count(&count).Error; err != nil {
		tb.Logf("count session user %q failed: %v", username, err)
		return false
	}
	if count > 0 {
		return true
	}
	passwordHash, err := common.HashPassword("realtime-perf-password")
	if err != nil {
		tb.Logf("hash session user password failed: %v", err)
		return false
	}
	if err := database.GetDB().Create(&model.User{Username: username, Password: passwordHash}).Error; err != nil {
		tb.Logf("create session user %q failed: %v", username, err)
		return false
	}
	return true
}

func initAPIRealtimePerfDB(tb testing.TB) {
	tb.Helper()
	stopTokenUseDebouncerBeforeAPITestDBInit(tb)
	if db := database.GetDB(); db != nil {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
			time.Sleep(25 * time.Millisecond)
		}
	}
	prevAuditSync := service.AuditSyncForTest
	service.AuditSyncForTest = true
	tb.Cleanup(func() { service.AuditSyncForTest = prevAuditSync })
	dir := tb.TempDir()
	tb.Setenv("SUI_DB_FOLDER", dir)
	initAPITestDB(tb, filepath.Join(dir, "s-ui.db"))
	database.GetDB().Config.Logger = gormlogger.Discard
	tb.Cleanup(func() {
		stopTokenUseDebouncerBeforeAPITestDBInit(tb)
		if db := database.GetDB(); db != nil {
			if sqlDB, err := db.DB(); err == nil {
				_ = sqlDB.Close()
			}
		}
		realtime.CloseAll("phase5_done")
	})
}

func loginRealtimePerfUser(tb testing.TB, server *httptest.Server, user string) []*http.Cookie {
	tb.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL+"/login/"+user, nil)
	if err != nil {
		tb.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		tb.Fatal(err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != http.StatusNoContent {
		tb.Fatalf("login status=%d", resp.StatusCode)
	}
	return resp.Cookies()
}

func dialRealtimeWSForBench(tb testing.TB, server *httptest.Server, cookies []*http.Cookie, token string) *websocket.Conn {
	tb.Helper()
	conn, resp, err := dialRealtimeWSRaw(server, cookies, token)
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		tb.Fatalf("websocket dial status=%v err=%v", statusCode(resp), err)
	}
	return conn
}

func dialRealtimeWSRaw(server *httptest.Server, cookies []*http.Cookie, token string) (*websocket.Conn, *http.Response, error) {
	header := http.Header{}
	header.Set("Origin", server.URL)
	header.Set("Cookie", cookieHeader(cookies))
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/api/realtime/ws?token=" + token
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	conn, resp, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{
		HTTPHeader: header,
		OnPingReceived: func(context.Context, []byte) bool {
			return true
		},
	})
	return conn, resp, err
}

func readRealtimeEventForBench(tb testing.TB, conn *websocket.Conn) {
	tb.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, reader, err := conn.Reader(ctx)
	if err != nil {
		tb.Fatal(err)
	}
	body, err := io.ReadAll(reader)
	if err != nil {
		tb.Fatal(err)
	}
	var event realtime.Event
	if err := json.Unmarshal(body, &event); err != nil {
		tb.Fatal(err)
	}
}

func statusCode(resp *http.Response) int {
	if resp == nil {
		return 0
	}
	return resp.StatusCode
}
