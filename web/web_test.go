package web

import "testing"

func TestNewServerInitializesEmbeddedAssets(t *testing.T) {
	server, err := NewServer()
	if err != nil {
		t.Fatal(err)
	}
	if server == nil || server.assetsFS == nil {
		t.Fatal("expected server with embedded assets filesystem")
	}
}
