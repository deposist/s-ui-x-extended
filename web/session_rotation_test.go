package web

import (
	"net/http"
	"testing"

	"github.com/deposist/s-ui-x/service"
	ginsessions "github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

// TestSQLiteSessionStoreRotatesIDOnLogin pins S-k: the login flow sets
// service.SessionRegenerateKey, which must make the store mint a fresh session
// ID and erase the old row, so a planted pre-auth (CSRF) session cannot survive
// authentication under the same ID (session-fixation defense). The marker must
// be consumed (a subsequent normal save must NOT rotate).
func TestSQLiteSessionStoreRotatesIDOnLogin(t *testing.T) {
	db := initSQLiteSessionTestDB(t)
	gin.SetMode(gin.TestMode)
	store, err := NewSQLiteSessionStore(db, []byte("test-session-secret-32-bytes-long"))
	if err != nil {
		t.Fatal(err)
	}
	opts := ginsessions.Options{Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode}
	router := gin.New()
	router.Use(ginsessions.Sessions("s-ui", store))
	router.GET("/preauth", func(c *gin.Context) {
		s := ginsessions.Default(c)
		s.Set("csrf", "x")
		s.Options(opts)
		_ = s.Save()
		c.Status(http.StatusNoContent)
	})
	router.GET("/login", func(c *gin.Context) {
		s := ginsessions.Default(c)
		s.Set("user", "admin")
		s.Set(service.SessionRegenerateKey, true)
		s.Options(opts)
		_ = s.Save()
		c.Status(http.StatusNoContent)
	})
	router.GET("/touch", func(c *gin.Context) {
		s := ginsessions.Default(c)
		s.Set("ping", "1")
		s.Options(opts)
		_ = s.Save()
		c.Status(http.StatusNoContent)
	})

	onlyRowID := func() string {
		t.Helper()
		var rows []sqliteSessionRow
		if err := db.Table("sessions").Find(&rows).Error; err != nil {
			t.Fatal(err)
		}
		if len(rows) != 1 {
			t.Fatalf("expected exactly one session row, got %d", len(rows))
		}
		return rows[0].ID
	}

	// Pre-auth session (mirrors CSRF issuance before login).
	pre := performSQLiteSessionRequest(router, "/preauth")
	preCookies := pre.Result().Cookies()
	if len(preCookies) != 1 {
		t.Fatalf("expected a pre-auth cookie, got %d", len(preCookies))
	}
	preID := onlyRowID()

	// Login carrying the pre-auth cookie must rotate the ID and drop the old row.
	login := performSQLiteSessionRequest(router, "/login", preCookies...)
	if login.Code != http.StatusNoContent {
		t.Fatalf("login returned %d", login.Code)
	}
	loginCookies := login.Result().Cookies()
	if len(loginCookies) != 1 {
		t.Fatalf("expected a session cookie after login, got %d", len(loginCookies))
	}
	loginID := onlyRowID()
	if loginID == preID {
		t.Fatal("session ID was not rotated on login (session fixation not prevented)")
	}

	// A normal save (no marker) must NOT rotate — proves the marker was consumed
	// and did not persist into the stored session.
	touch := performSQLiteSessionRequest(router, "/touch", loginCookies...)
	if touch.Code != http.StatusNoContent {
		t.Fatalf("touch returned %d", touch.Code)
	}
	if id := onlyRowID(); id != loginID {
		t.Fatalf("a normal save rotated the ID (regenerate marker persisted): %q -> %q", loginID, id)
	}

	// The post-login cookie still authenticates against the rotated session.
	if again := performSQLiteSessionRequest(router, "/touch", loginCookies...); again.Code != http.StatusNoContent {
		t.Fatalf("post-rotation cookie failed to authenticate: %d", again.Code)
	}
}
