package web

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x/database"
	ginsessions "github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/securecookie"
	"gorm.io/gorm"
)

// TestNewSessionStoreDefaultsToSecureCookie pins H-2: the store's default cookie
// options are Secure (and HttpOnly) so any future session-creating path that
// forgets to call Options() still issues a hardened cookie.
func TestNewSessionStoreDefaultsToSecureCookie(t *testing.T) {
	db := initSQLiteSessionTestDB(t)
	store, err := NewSQLiteSessionStore(db, securecookie.GenerateRandomKey(32))
	if err != nil {
		t.Fatalf("NewSQLiteSessionStore: %v", err)
	}
	opts := store.currentOptions()
	if !opts.Secure {
		t.Fatal("default session cookie options must be Secure")
	}
	if !opts.HttpOnly {
		t.Fatal("default session cookie options must be HttpOnly")
	}
}

func TestSQLiteSessionStorePersistsServerSide(t *testing.T) {
	db := initSQLiteSessionTestDB(t)
	router := newSQLiteSessionTestRouter(t, db)

	login := performSQLiteSessionRequest(router, "/login")
	if login.Code != http.StatusNoContent {
		t.Fatalf("login returned %d", login.Code)
	}
	cookies := login.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected one session cookie, got %d", len(cookies))
	}
	if strings.Contains(cookies[0].Value, "admin") {
		t.Fatalf("session cookie leaked user data: %q", cookies[0].Value)
	}

	var row sqliteSessionRow
	if err := db.Table("sessions").First(&row).Error; err != nil {
		t.Fatal(err)
	}
	if row.ID == "" || len(row.Data) == 0 {
		t.Fatalf("session row was not persisted: %#v", row)
	}

	protected := performSQLiteSessionRequest(router, "/protected", cookies...)
	if protected.Code != http.StatusNoContent {
		t.Fatalf("protected route returned %d", protected.Code)
	}

	logout := performSQLiteSessionRequest(router, "/logout", cookies...)
	if logout.Code != http.StatusNoContent {
		t.Fatalf("logout returned %d", logout.Code)
	}
	var count int64
	if err := db.Table("sessions").Where("id = ?", row.ID).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("session row was not deleted on logout: %d", count)
	}
	afterLogout := performSQLiteSessionRequest(router, "/protected", cookies...)
	if afterLogout.Code != http.StatusUnauthorized {
		t.Fatalf("old cookie should be invalid after logout, got %d", afterLogout.Code)
	}
}

func TestSQLiteSessionStoreSupportsCookieKeyRollover(t *testing.T) {
	db := initSQLiteSessionTestDB(t)

	oldKey := []byte("0123456789abcdef0123456789abcdef")
	newKey := []byte("abcdef0123456789abcdef0123456789")

	oldRouter := newSQLiteSessionTestRouterWithKeys(t, db, oldKey)
	oldLogin := performSQLiteSessionRequest(oldRouter, "/login")
	if oldLogin.Code != http.StatusNoContent {
		t.Fatalf("old-key login returned %d", oldLogin.Code)
	}
	oldCookies := oldLogin.Result().Cookies()
	if len(oldCookies) != 1 {
		t.Fatalf("expected one old-key cookie, got %d", len(oldCookies))
	}

	var oldSessionID string
	if err := securecookie.DecodeMulti("s-ui", oldCookies[0].Value, &oldSessionID, codecsFromHashKeys(oldKey)...); err != nil {
		t.Fatalf("old cookie decode with old key failed: %v", err)
	}

	rotatedRouter := newSQLiteSessionTestRouterWithKeys(t, db, newKey, oldKey)
	protected := performSQLiteSessionRequest(rotatedRouter, "/protected", oldCookies...)
	if protected.Code != http.StatusNoContent {
		t.Fatalf("rotated store did not accept old cookie, got %d", protected.Code)
	}

	newLogin := performSQLiteSessionRequest(rotatedRouter, "/login")
	if newLogin.Code != http.StatusNoContent {
		t.Fatalf("rotated login returned %d", newLogin.Code)
	}
	newCookies := newLogin.Result().Cookies()
	if len(newCookies) != 1 {
		t.Fatalf("expected one new-key cookie, got %d", len(newCookies))
	}

	var newSessionID string
	if err := securecookie.DecodeMulti("s-ui", newCookies[0].Value, &newSessionID, codecsFromHashKeys(newKey, oldKey)...); err != nil {
		t.Fatalf("new cookie decode with rotated keyset failed: %v", err)
	}
	if err := securecookie.DecodeMulti("s-ui", newCookies[0].Value, &newSessionID, codecsFromHashKeys(oldKey)...); err == nil {
		t.Fatal("new cookie should not decode with old key only")
	}
}

func BenchmarkSQLiteSessionStoreProtectedRequest(b *testing.B) {
	db := initSQLiteSessionTestDB(b)
	router := newSQLiteSessionTestRouter(b, db)
	login := performSQLiteSessionRequest(router, "/login")
	if login.Code != http.StatusNoContent {
		b.Fatalf("login returned %d", login.Code)
	}
	cookies := login.Result().Cookies()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		protected := performSQLiteSessionRequest(router, "/protected", cookies...)
		if protected.Code != http.StatusNoContent {
			b.Fatalf("protected route returned %d", protected.Code)
		}
	}
}

func initSQLiteSessionTestDB(tb testing.TB) *gorm.DB {
	tb.Helper()
	tempDir := tb.TempDir()
	tb.Setenv("SUI_DB_FOLDER", tempDir)
	closeSQLiteSessionTestDB(database.GetDB())
	if err := database.InitDB(filepath.Join(tempDir, "s-ui.db")); err != nil {
		if strings.Contains(err.Error(), "go-sqlite3 requires cgo") {
			tb.Skip(err)
		}
		tb.Fatal(err)
	}
	db := database.GetDB()
	tb.Cleanup(func() { closeSQLiteSessionTestDB(db) })
	return db
}

func closeSQLiteSessionTestDB(db *gorm.DB) {
	if db == nil {
		return
	}
	if sqlDB, err := db.DB(); err == nil {
		_ = sqlDB.Close()
	}
}

func newSQLiteSessionTestRouter(tb testing.TB, db *gorm.DB) *gin.Engine {
	return newSQLiteSessionTestRouterWithKeys(tb, db, []byte("test-session-secret-32-bytes-long"))
}

func newSQLiteSessionTestRouterWithKeys(tb testing.TB, db *gorm.DB, keyPairs ...[]byte) *gin.Engine {
	tb.Helper()
	gin.SetMode(gin.TestMode)
	store, err := NewSQLiteSessionStore(db, keyPairs...)
	if err != nil {
		tb.Fatal(err)
	}
	router := gin.New()
	router.Use(ginsessions.Sessions("s-ui", store))
	router.GET("/login", func(c *gin.Context) {
		session := ginsessions.Default(c)
		session.Set("user", "admin")
		session.Options(ginsessions.Options{Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode})
		if err := session.Save(); err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		c.Status(http.StatusNoContent)
	})
	router.GET("/protected", func(c *gin.Context) {
		if ginsessions.Default(c).Get("user") != "admin" {
			c.Status(http.StatusUnauthorized)
			return
		}
		c.Status(http.StatusNoContent)
	})
	router.GET("/logout", func(c *gin.Context) {
		session := ginsessions.Default(c)
		session.Clear()
		session.Options(ginsessions.Options{Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteLaxMode})
		if err := session.Save(); err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		c.Status(http.StatusNoContent)
	})
	return router
}

func performSQLiteSessionRequest(router *gin.Engine, path string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	router.ServeHTTP(recorder, req)
	return recorder
}
