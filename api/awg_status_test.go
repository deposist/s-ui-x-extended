package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x-extended/database"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

func TestAWGStatusRouteDoesNotExposeSecretFields(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SUI_DB_FOLDER", dir)
	initAPITestDB(t, filepath.Join(dir, "s-ui.db"))
	testDB := database.GetDB()
	t.Cleanup(func() {
		if testDB != nil {
			if sqlDB, err := testDB.DB(); err == nil {
				_ = sqlDB.Close()
			}
		}
	})
	gin.SetMode(gin.TestMode)
	apiService := &ApiService{}
	router := gin.New()
	router.Use(sessions.Sessions("s-ui", cookie.NewStore([]byte("test-secret"))))
	router.GET("/api/awg/status", apiService.AWGStatus)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/awg/status", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status: %d, body: %s", recorder.Code, recorder.Body.String())
	}
	var msg struct {
		Success bool           `json:"success"`
		Obj     map[string]any `json:"obj"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &msg); err != nil {
		t.Fatal(err)
	}
	if !msg.Success {
		t.Fatalf("status failed: %s", recorder.Body.String())
	}
	for _, key := range []string{"managedEndpoints", "coreReachable", "encryptionKeyAvailable", "desired", "provisioned", "pending", "errors"} {
		if _, ok := msg.Obj[key]; !ok {
			t.Fatalf("status response is missing %q: %v", key, msg.Obj)
		}
	}
	encoded, err := json.Marshal(msg.Obj)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.ToLower(string(encoded))
	for _, forbidden := range []string{"private", "preshared", "psk", "cipher", "crypto_context"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("status exposed secret field %q: %s", forbidden, encoded)
		}
	}
}
