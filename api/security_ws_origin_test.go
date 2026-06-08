package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"

	"github.com/gin-gonic/gin"
)

func TestSecurityWSOriginAllowedMatrix(t *testing.T) {
	tests := []struct {
		name       string
		origin     string
		host       string
		webDomain  string
		want       bool
		wantReason string
	}{
		{name: "host mismatch", origin: "https://evil.example", host: "panel.example", wantReason: "host_mismatch"},
		{name: "invalid scheme", origin: "file://panel.example", host: "panel.example", wantReason: "invalid_scheme"},
		{name: "invalid raw query", origin: "https://panel.example?x=1", host: "panel.example", wantReason: "invalid_origin"},
		{name: "invalid fragment", origin: "https://panel.example/#token", host: "panel.example", wantReason: "invalid_origin"},
		{name: "request host match", origin: "https://panel.example", host: "panel.example", want: true, wantReason: "request_host"},
		{name: "web domain host port match", origin: "https://panel.example:8443", host: "other.example", webDomain: "https://panel.example:8443", want: true, wantReason: "web_domain"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, reason := wsOriginAllowed(tt.origin, tt.host, tt.webDomain)
			if got != tt.want || reason != tt.wantReason {
				t.Fatalf("wsOriginAllowed()=(%v,%q), want (%v,%q)", got, reason, tt.want, tt.wantReason)
			}
		})
	}
}

func TestSecurityValidateWSOriginRejectsAndAudits(t *testing.T) {
	initSessionTestDB(t)
	tests := []struct {
		name   string
		origin string
		host   string
		reason string
	}{
		{name: "host mismatch", origin: "https://evil.example", host: "panel.example", reason: "host_mismatch"},
		{name: "invalid scheme", origin: "file://panel.example", host: "panel.example", reason: "invalid_scheme"},
		{name: "invalid origin query", origin: "https://panel.example?x=1", host: "panel.example", reason: "invalid_origin"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			req := httptest.NewRequest(http.MethodGet, "http://"+tt.host+"/api/realtime/ws", nil)
			req.Host = tt.host
			req.Header.Set("Origin", tt.origin)
			c.Request = req

			if (&ApiService{}).validateWSOrigin(c, "admin") {
				t.Fatal("origin should have been rejected")
			}
			if c.Writer.Status() != http.StatusForbidden {
				t.Fatalf("unexpected status %d", c.Writer.Status())
			}
			var event model.AuditEvent
			if err := database.GetDB().Where("event = ?", "ws_origin_rejected").Order("id desc").First(&event).Error; err != nil {
				t.Fatal(err)
			}
			var details map[string]any
			if err := json.Unmarshal(event.Details, &details); err != nil {
				t.Fatal(err)
			}
			if details["reason"] != tt.reason {
				t.Fatalf("unexpected audit details: %#v", details)
			}
		})
	}
}
