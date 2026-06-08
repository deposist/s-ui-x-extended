package util

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestExternalHTTPClientDialerBlocksLoopback verifies the dialer hook rejects
// connections that resolve to private addresses, even when the URL passed
// validation. This guards against DNS-rebinding attacks.
func TestExternalHTTPClientDialerBlocksLoopback(t *testing.T) {
	t.Setenv("SUI_ALLOW_PRIVATE_SUB_URLS", "")

	srv := httptest.NewServer(nil)
	defer srv.Close()

	transport, ok := getExternalHTTPClient().Transport.(*http.Transport)
	if !ok || transport.DialContext == nil {
		t.Fatalf("expected configured *http.Transport with DialContext")
	}
	addr := strings.TrimPrefix(srv.URL, "http://")
	_, err := transport.DialContext(context.Background(), "tcp", addr)
	if err == nil {
		t.Fatal("expected dialer to reject loopback address")
	}
	if !errors.Is(err, errBlockedExternalAddress) {
		t.Fatalf("expected errBlockedExternalAddress, got %v", err)
	}
}

func TestExternalHTTPClientDialerAllowsWhenOptedIn(t *testing.T) {
	t.Setenv("SUI_ALLOW_PRIVATE_SUB_URLS", "true")

	srv := httptest.NewServer(nil)
	defer srv.Close()

	transport, ok := getExternalHTTPClient().Transport.(*http.Transport)
	if !ok || transport.DialContext == nil {
		t.Fatalf("expected configured *http.Transport with DialContext")
	}
	addr := strings.TrimPrefix(srv.URL, "http://")
	conn, err := transport.DialContext(context.Background(), "tcp", addr)
	if err != nil {
		t.Fatalf("expected dialer to allow loopback when opted-in, got %v", err)
	}
	conn.Close()
}
