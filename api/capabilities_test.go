package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestGetCapabilitiesReturnsSafeAdminView verifies the /api/capabilities handler
// returns the build-tag flags and per-inbound availability, and — critically for
// the security requirement — leaks no paths, builder internals or secrets.
func TestGetCapabilitiesReturnsSafeAdminView(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/capabilities", nil)

	(&ApiService{}).GetCapabilities(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status %d body=%s", w.Code, w.Body.String())
	}

	var env struct {
		Success bool `json:"success"`
		Obj     struct {
			BuildTags map[string]bool `json:"buildTags"`
			Inbounds  []struct {
				Type      string `json:"type"`
				Available bool   `json:"available"`
				BuildTag  string `json:"buildTag"`
			} `json:"inbounds"`
		} `json:"obj"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal: %v body=%s", err, w.Body.String())
	}
	if !env.Success {
		t.Fatal("expected success=true")
	}
	if len(env.Obj.BuildTags) == 0 {
		t.Fatal("expected build tags")
	}
	if len(env.Obj.Inbounds) == 0 {
		t.Fatal("expected inbounds")
	}

	// Security: the response must expose ONLY bool flags + capability metadata —
	// never builder internals, the SQL user-field name, paths or key material.
	body := w.Body.String()
	for _, forbidden := range []string{
		"private_key", "host_key", "key_path", "secret",
		"outJsonBuilder", "credentialMap", "userField", "/etc/", "linkScheme",
	} {
		if strings.Contains(body, forbidden) {
			t.Errorf("capabilities response leaked %q: %s", forbidden, body)
		}
	}
}
