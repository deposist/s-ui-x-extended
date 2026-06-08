package api

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// recordingConn captures the deadlines set on it. It embeds net.Conn (nil) so
// it satisfies the interface; only the deadline methods are exercised.
type recordingConn struct {
	net.Conn
	readDeadline  time.Time
	writeDeadline time.Time
}

func (r *recordingConn) SetReadDeadline(t time.Time) error  { r.readDeadline = t; return nil }
func (r *recordingConn) SetWriteDeadline(t time.Time) error { r.writeDeadline = t; return nil }
func (r *recordingConn) SetDeadline(t time.Time) error {
	r.readDeadline = t
	r.writeDeadline = t
	return nil
}

// noDeadlineWriter wraps gin.ResponseWriter but does NOT implement Unwrap or the
// deadline methods, mirroring the production gzip writer that breaks
// http.NewResponseController.
type noDeadlineWriter struct {
	gin.ResponseWriter
}

func (noDeadlineWriter) Unwrap() http.ResponseWriter { return nil }

// TestExtendSlowRequestDeadlinesUsesConn is the regression guard for the
// "Import failed / Network Error" bug: when the response writer cannot carry a
// deadline (the gzip case), the deadline must still be lifted via the raw
// net.Conn from the request context, not silently dropped.
func TestExtendSlowRequestDeadlinesUsesConn(t *testing.T) {
	gin.SetMode(gin.TestMode)
	conn := &recordingConn{}
	ctx := SaveConnContext(context.Background(), conn)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/import-xui/apply", nil).WithContext(ctx)
	// Simulate the gzip writer: no Unwrap to a deadline-capable writer.
	c.Writer = noDeadlineWriter{c.Writer}

	floor := time.Now().Add(xuiRequestTimeout)
	extendSlowRequestDeadlines(c)

	if !conn.writeDeadline.After(floor) {
		t.Fatalf("write deadline not extended: got %v, want after %v", conn.writeDeadline, floor)
	}
	if !conn.readDeadline.After(floor) {
		t.Fatalf("read deadline not extended: got %v, want after %v", conn.readDeadline, floor)
	}
}

// TestSaveConnContextRoundTrip ensures the ConnContext hook value is retrievable
// exactly as stored.
func TestSaveConnContextRoundTrip(t *testing.T) {
	conn := &recordingConn{}
	got, ok := connFromContext(SaveConnContext(context.Background(), conn))
	if !ok || got != conn {
		t.Fatalf("connFromContext = (%v, %v), want (%v, true)", got, ok, conn)
	}
	if _, ok := connFromContext(context.Background()); ok {
		t.Fatal("connFromContext returned ok for a context without a conn")
	}
}
