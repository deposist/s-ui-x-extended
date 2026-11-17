package network

import (
	"bufio"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestAutoHttpsRedirectPreservesEscapedPath(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()
	if err := client.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		conn := NewAutoHttpsConn(server)
		_, _ = conn.Read(make([]byte, 1))
	}()
	if _, err := client.Write([]byte("GET /a%2Fb?q=%2F HTTP/1.1\r\nHost: example.com\r\n\r\n")); err != nil {
		t.Fatal(err)
	}
	response, err := http.ReadResponse(bufio.NewReader(client), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if got := response.Header.Get("Location"); got != "https://example.com/a%2Fb?q=%2F" {
		t.Fatalf("Location = %q, want unchanged escaped path", got)
	}
	<-done
}
