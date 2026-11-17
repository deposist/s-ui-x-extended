package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func keypairResponse(t *testing.T, privateKey string) struct {
	Success bool     `json:"success"`
	Msg     string   `json:"msg"`
	Obj     []string `json:"obj"`
} {
	t.Helper()
	router := gin.New()
	service := &ApiService{}
	router.GET("/api/keypairs", service.GetKeypairs)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/keypairs?k=wireguard&o="+url.QueryEscape(privateKey), nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("keypair API status: %d", recorder.Code)
	}
	var response struct {
		Success bool     `json:"success"`
		Msg     string   `json:"msg"`
		Obj     []string `json:"obj"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	return response
}

func TestWireGuardKeypairAPIRejectsInvalidPrivateKey(t *testing.T) {
	for _, invalid := range []string{"invalid-private-key", "YQ=="} {
		t.Run(invalid, func(t *testing.T) {
			response := keypairResponse(t, invalid)
			if response.Success || len(response.Obj) != 0 {
				t.Fatalf("invalid private key produced a successful public key: %+v", response)
			}
			if response.Msg == "" || strings.Contains(response.Msg, invalid) {
				t.Fatalf("invalid-key response must explain failure without echoing key: %q", response.Msg)
			}
		})
	}
}

func TestWireGuardKeypairAPIUsesProvidedPrivateKey(t *testing.T) {
	privateKey, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		t.Fatal(err)
	}
	response := keypairResponse(t, privateKey.String())
	if !response.Success || len(response.Obj) != 1 || response.Obj[0] != privateKey.PublicKey().String() {
		t.Fatalf("provided private key did not produce its public key: %+v", response)
	}
}

func TestWireGuardKeypairAPIGeneratesPairWithoutPrivateKey(t *testing.T) {
	response := keypairResponse(t, "")
	if !response.Success || len(response.Obj) != 2 {
		t.Fatalf("keypair generation failed: %+v", response)
	}
	privateKey, err := wgtypes.ParseKey(strings.TrimPrefix(response.Obj[0], "PrivateKey: "))
	if err != nil {
		t.Fatal(err)
	}
	if response.Obj[1] != "PublicKey: "+privateKey.PublicKey().String() {
		t.Fatal("generated private and public key do not match")
	}
}
