package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewServerInitializesEmbeddedAssets(t *testing.T) {
	server, err := NewServer()
	if err != nil {
		t.Fatal(err)
	}
	if server == nil || server.assetsFS == nil {
		t.Fatal("expected server with embedded assets filesystem")
	}
}

func TestLoginPageSendsClearSiteDataCacheHeader(t *testing.T) {
	initSQLiteSessionTestDB(t)

	server, err := NewServer()
	if err != nil {
		t.Fatal(err)
	}
	engine, err := server.initRouter()
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/app/login", nil)
	if err != nil {
		t.Fatal(err)
	}
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for /app/login, got %d", w.Code)
	}
	clearHeader := w.Header().Get("Clear-Site-Data")
	if !strings.Contains(clearHeader, "cache") {
		t.Fatalf("expected Clear-Site-Data header to contain 'cache', got %q", clearHeader)
	}
}
