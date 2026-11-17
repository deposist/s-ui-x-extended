package web

import (
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/deposist/s-ui-x-extended/database/model"
)

func TestServerStopReleasesUnservedListener(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	address := listener.Addr().String()
	server := &Server{listener: listener, httpServer: &http.Server{}}
	if err := server.Stop(); err != nil {
		t.Fatal(err)
	}
	restarted, err := net.Listen("tcp", address)
	if err != nil {
		t.Fatalf("listener remains bound after Stop before Serve: %v", err)
	}
	_ = restarted.Close()
}

func TestServerRapidRestart(t *testing.T) {
	db := initSQLiteSessionTestDB(t)
	for key, value := range map[string]string{"webListen": "127.0.0.1", "webPort": "0"} {
		if err := db.Save(&model.Setting{Key: key, Value: value}).Error; err != nil {
			t.Fatal(err)
		}
	}
	server, err := NewServer()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := server.Stop(); err != nil {
			t.Error(err)
		}
	})
	for range 20 {
		if err := server.Start(); err != nil {
			t.Fatal(err)
		}
		if err := server.Stop(); err != nil {
			t.Fatal(err)
		}
	}
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Timeout: 5 * time.Second}
	response, err := client.Get("http://" + server.listener.Addr().String() + "/app/login")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("restarted server returned %d", response.StatusCode)
	}
}
