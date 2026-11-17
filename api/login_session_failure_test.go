package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/deposist/s-ui-x-extended/service"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

func TestLoginReportsSessionPersistenceFailure(t *testing.T) {
	for _, forceReset := range []bool{false, true} {
		name := "regular"
		if forceReset {
			name = "password_reset"
		}
		t.Run(name, func(t *testing.T) {
			resetRateLimitState()
			initSessionTestDB(t)
			if err := (&service.UserService{}).UpdateFirstUser("admin", "session-test-password"); err != nil {
				t.Fatal(err)
			}
			if err := database.GetDB().Model(&model.User{}).Where("username = ?", "admin").Update("force_password_reset", forceReset).Error; err != nil {
				t.Fatal(err)
			}
			gin.SetMode(gin.TestMode)
			router := gin.New()
			// An invalid AES key makes the real cookie store fail to save.
			store := cookie.NewStore([]byte("session-test-hash-key"), []byte("invalid-aes-key"))
			router.Use(sessions.Sessions("s-ui", store))
			api := NewApiService()
			router.POST("/login", api.Login)
			form := url.Values{"user": {"admin"}, "pass": {"session-test-password"}}
			req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, req)
			var msg Msg
			if err := json.Unmarshal(response.Body.Bytes(), &msg); err != nil {
				t.Fatal(err)
			}
			if msg.Success || msg.Msg == "" {
				t.Fatalf("session save failed but login reported success: %s", response.Body.String())
			}
			if len(response.Result().Cookies()) != 0 {
				t.Fatal("failed session save issued a cookie")
			}
		})
	}
}
