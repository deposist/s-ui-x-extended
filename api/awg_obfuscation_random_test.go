package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

func newAWGObfuscationRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	apiService := &ApiService{}
	router := gin.New()
	router.Use(sessions.Sessions("s-ui", cookie.NewStore([]byte("test-secret"))))
	router.GET("/api/awg/obfuscation/random", apiService.GetAWGObfuscationRandom)
	return router
}

type awgObfuscationParams struct {
	H1   string `json:"h1"`
	H2   string `json:"h2"`
	H3   string `json:"h3"`
	H4   string `json:"h4"`
	JC   int    `json:"jc"`
	JMin int    `json:"jmin"`
	JMax int    `json:"jmax"`
}

func decodeAWGObfuscationMsg(t *testing.T, recorder *httptest.ResponseRecorder) awgObfuscationParams {
	t.Helper()
	var msg struct {
		Success bool                 `json:"success"`
		Obj     awgObfuscationParams `json:"obj"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &msg); err != nil {
		t.Fatalf("invalid body %s: %v", recorder.Body.String(), err)
	}
	if !msg.Success {
		t.Fatalf("request failed: %s", recorder.Body.String())
	}
	return msg.Obj
}

func TestGetAWGObfuscationRandomGeneratesHeadersOnly(t *testing.T) {
	router := newAWGObfuscationRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/api/awg/obfuscation/random", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	params := decodeAWGObfuscationMsg(t, recorder)
	for name, h := range map[string]string{"h1": params.H1, "h2": params.H2, "h3": params.H3, "h4": params.H4} {
		if h == "" {
			t.Fatalf("%s is empty: %+v", name, params)
		}
	}
	if params.JC != 0 || params.JMin != 0 || params.JMax != 0 {
		t.Fatalf("junk params must be omitted without preset=balanced: %+v", params)
	}
}

func TestGetAWGObfuscationRandomBalancedIncludesJunk(t *testing.T) {
	router := newAWGObfuscationRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/api/awg/obfuscation/random?preset=balanced", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	params := decodeAWGObfuscationMsg(t, recorder)
	if params.JC < 3 || params.JC > 6 {
		t.Fatalf("Jc=%d outside 3-6", params.JC)
	}
	if params.JMin < 40 || params.JMin > 89 {
		t.Fatalf("Jmin=%d outside 40-89", params.JMin)
	}
	if params.JMax < params.JMin+50 || params.JMax > params.JMin+250 {
		t.Fatalf("Jmax=%d outside Jmin+50..Jmin+250", params.JMax)
	}
}
