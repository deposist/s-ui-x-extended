package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x-extended/service"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

// fakeAWGEndpointDevices records the CreateDevice call so the test can assert
// the handler bound the request body correctly.
type fakeAWGEndpointDevices struct {
	createdClientID   uint
	createdEndpointID uint
	createdName       string
	createdExpiresAt  int64
}

func (f *fakeAWGEndpointDevices) CreateDevice(_ context.Context, clientID uint, endpointID uint, _ string, name string, expiresAt int64) (service.AWGDeviceInfo, error) {
	f.createdClientID, f.createdEndpointID, f.createdName, f.createdExpiresAt = clientID, endpointID, name, expiresAt
	return service.AWGDeviceInfo{ID: 7, Name: name, ExpiresAt: expiresAt}, nil
}

func (f *fakeAWGEndpointDevices) ListDevices(uint, uint) ([]service.AWGDeviceInfo, error) {
	return nil, nil
}

func (f *fakeAWGEndpointDevices) GetOwnedDevice(uint, uint, uint) (service.AWGDeviceInfo, error) {
	return service.AWGDeviceInfo{}, nil
}

func (f *fakeAWGEndpointDevices) RenderOwnedConfig(context.Context, uint, uint, uint) ([]byte, error) {
	return nil, nil
}

func (f *fakeAWGEndpointDevices) RotateOwnedDevice(context.Context, uint, uint, uint, string) (service.AWGDeviceInfo, error) {
	return service.AWGDeviceInfo{}, nil
}

func (f *fakeAWGEndpointDevices) RevokeOwnedDevice(context.Context, uint, uint, uint) error {
	return nil
}

func newAWGCreateDeviceRouter(t *testing.T) (*gin.Engine, *fakeAWGEndpointDevices) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	fake := &fakeAWGEndpointDevices{}
	runtime := service.NewRuntime(nil)
	runtime.SetAWGEndpointDeviceService(fake)
	apiService := &ApiService{Runtime: runtime}
	router := gin.New()
	router.Use(sessions.Sessions("s-ui", cookie.NewStore([]byte("test-secret"))))
	router.POST("/api/awg/clients/:clientId/devices", apiService.CreateClientAWGDevice)
	return router, fake
}

func decodeAWGCreateMsg(t *testing.T, recorder *httptest.ResponseRecorder) Msg {
	t.Helper()
	var msg Msg
	if err := json.Unmarshal(recorder.Body.Bytes(), &msg); err != nil {
		t.Fatalf("invalid msg body=%s: %v", recorder.Body.String(), err)
	}
	return msg
}

// The panel frontend posts application/x-www-form-urlencoded (the axios
// default in plugins/api.ts). A JSON-only binding rejects that body with
// "awg: invalid request", which is the exact regression this test pins down.
func TestCreateClientAWGDeviceBindsFormEncodedBody(t *testing.T) {
	router, fake := newAWGCreateDeviceRouter(t)

	form := url.Values{}
	form.Set("endpointId", "3")
	form.Set("name", "phone")
	req := httptest.NewRequest(http.MethodPost, "/api/awg/clients/5/devices", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	msg := decodeAWGCreateMsg(t, recorder)
	if !msg.Success {
		t.Fatalf("form-encoded create failed: %s", msg.Msg)
	}
	if fake.createdClientID != 5 || fake.createdEndpointID != 3 || fake.createdName != "phone" {
		t.Fatalf("bound values mismatch: clientID=%d endpointID=%d name=%q",
			fake.createdClientID, fake.createdEndpointID, fake.createdName)
	}
}

// JSON bodies must keep working for API consumers.
func TestCreateClientAWGDeviceBindsJSONBody(t *testing.T) {
	router, fake := newAWGCreateDeviceRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/awg/clients/5/devices",
		strings.NewReader(`{"endpointId":3,"name":"phone"}`))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	msg := decodeAWGCreateMsg(t, recorder)
	if !msg.Success {
		t.Fatalf("json create failed: %s", msg.Msg)
	}
	if fake.createdEndpointID != 3 || fake.createdName != "phone" {
		t.Fatalf("bound values mismatch: endpointID=%d name=%q", fake.createdEndpointID, fake.createdName)
	}
}

// The optional expiresAt field must reach the device service from both
// encodings; absence binds as 0 (never expires).
func TestCreateClientAWGDeviceBindsExpiresAt(t *testing.T) {
	router, fake := newAWGCreateDeviceRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/awg/clients/5/devices",
		strings.NewReader(`{"endpointId":3,"name":"phone","expiresAt":1900000000}`))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	msg := decodeAWGCreateMsg(t, recorder)
	if !msg.Success {
		t.Fatalf("json create with expiresAt failed: %s", msg.Msg)
	}
	if fake.createdExpiresAt != 1900000000 {
		t.Fatalf("bound expiresAt = %d, want 1900000000", fake.createdExpiresAt)
	}

	form := url.Values{}
	form.Set("endpointId", "3")
	form.Set("name", "phone")
	form.Set("expiresAt", "1900000001")
	req = httptest.NewRequest(http.MethodPost, "/api/awg/clients/5/devices", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	msg = decodeAWGCreateMsg(t, recorder)
	if !msg.Success {
		t.Fatalf("form create with expiresAt failed: %s", msg.Msg)
	}
	if fake.createdExpiresAt != 1900000001 {
		t.Fatalf("bound form expiresAt = %d, want 1900000001", fake.createdExpiresAt)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/awg/clients/5/devices",
		strings.NewReader(`{"endpointId":3,"name":"phone"}`))
	req.Header.Set("Content-Type", "application/json")
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	msg = decodeAWGCreateMsg(t, recorder)
	if !msg.Success {
		t.Fatalf("json create without expiresAt failed: %s", msg.Msg)
	}
	if fake.createdExpiresAt != 0 {
		t.Fatalf("absent expiresAt bound as %d, want 0", fake.createdExpiresAt)
	}
}

// A missing/zero endpointId must still be rejected regardless of encoding.
func TestCreateClientAWGDeviceRejectsMissingEndpoint(t *testing.T) {
	router, _ := newAWGCreateDeviceRouter(t)

	form := url.Values{}
	form.Set("name", "phone")
	req := httptest.NewRequest(http.MethodPost, "/api/awg/clients/5/devices", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	msg := decodeAWGCreateMsg(t, recorder)
	if msg.Success {
		t.Fatal("expected zero endpointId to be rejected")
	}
}
