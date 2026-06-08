package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"golang.org/x/net/idna"
)

func runDomainValidator(t *testing.T, configDomain, hostHeader string) int {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(DomainValidator(configDomain))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = hostHeader
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w.Code
}

func TestDomainValidatorPlainASCII(t *testing.T) {
	if code := runDomainValidator(t, "panel.example.com", "panel.example.com:2095"); code != http.StatusOK {
		t.Fatalf("ascii host:port match: got %d, want 200", code)
	}
	if code := runDomainValidator(t, "panel.example.com", "PANEL.example.com"); code != http.StatusOK {
		t.Fatalf("case-insensitive match: got %d, want 200", code)
	}
	if code := runDomainValidator(t, "panel.example.com", "evil.example.com"); code != http.StatusForbidden {
		t.Fatalf("mismatch must be rejected: got %d, want 403", code)
	}
}

// TestDomainValidatorMatchesIDN pins H-13: a Unicode-configured domain matches
// the punycode Host header browsers actually send, while mismatches stay 403.
func TestDomainValidatorMatchesIDN(t *testing.T) {
	const unicode = "münchen.example"
	puny, err := idna.ToASCII(unicode)
	if err != nil {
		t.Fatalf("idna.ToASCII(%q): %v", unicode, err)
	}
	if puny == unicode {
		t.Skip("idna did not transform the test domain")
	}
	if code := runDomainValidator(t, unicode, puny); code != http.StatusOK {
		t.Fatalf("unicode config vs punycode host: got %d, want 200", code)
	}
	if code := runDomainValidator(t, puny, puny); code != http.StatusOK {
		t.Fatalf("punycode config vs punycode host: got %d, want 200", code)
	}
	if code := runDomainValidator(t, unicode, "evil.example"); code != http.StatusForbidden {
		t.Fatalf("IDN mismatch must be rejected: got %d, want 403", code)
	}
}
