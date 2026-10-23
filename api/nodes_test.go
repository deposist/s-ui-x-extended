package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"

	"github.com/gin-gonic/gin"
)

// nodeTestRouter mounts just the node routes behind a logged-in session, which
// is exactly what the browser talks to once the operator is inside the panel.
func nodeTestRouter(t *testing.T) (*gin.Engine, []*http.Cookie) {
	t.Helper()
	settingService := initSessionTestDB(t)
	return newAuthenticatedTestRouter(t, settingService, func(router *gin.Engine) {
		handler := &APIHandler{ApiService: NewApiService()}
		handler.registerNodeRoutes(router.Group("/api"))
	})
}

func decodeNodePage(t *testing.T, recorder *httptest.ResponseRecorder) struct {
	Success bool `json:"success"`
	Obj     struct {
		Items []struct {
			Id   uint   `json:"id"`
			Name string `json:"name"`
		} `json:"items"`
		Total int64 `json:"total"`
	} `json:"obj"`
} {
	t.Helper()
	var payload struct {
		Success bool `json:"success"`
		Obj     struct {
			Items []struct {
				Id   uint   `json:"id"`
				Name string `json:"name"`
			} `json:"items"`
			Total int64 `json:"total"`
		} `json:"obj"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode %q: %v", recorder.Body.String(), err)
	}
	if !payload.Success {
		t.Fatalf("request failed: %s", recorder.Body.String())
	}
	return payload
}

// TestNodeRoutesServePaginatedClients exercises the whole browser contract: a
// node is created over HTTP, clients are assigned in one bulk call, and the
// page endpoint honours limit, offset and the name filter.
func TestNodeRoutesServePaginatedClients(t *testing.T) {
	router, cookies := nodeTestRouter(t)
	db := database.GetDB()

	createBody := strings.NewReader(`{"name":"Германия-1","linkHost":"de1.example.com"}`)
	recorder := performAuthenticatedTestRequest(router,
		httptest.NewRequest(http.MethodPost, "/api/nodes", createBody), cookies...)
	var created struct {
		Success bool `json:"success"`
		Obj     struct {
			UUID string `json:"uuid"`
		} `json:"obj"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &created); err != nil || !created.Success {
		t.Fatalf("create node: %s (%v)", recorder.Body.String(), err)
	}
	if created.Obj.UUID == "" {
		t.Fatal("create node returned no uuid")
	}
	nodeUUID := created.Obj.UUID

	ids := make([]uint, 0, 7)
	for i := 0; i < 7; i++ {
		client := model.Client{
			Enable:   true,
			Name:     fmt.Sprintf("node-client-%d", i),
			Inbounds: json.RawMessage(`[]`),
			Config:   json.RawMessage(`{}`),
			Links:    json.RawMessage(`[]`),
		}
		if err := db.Create(&client).Error; err != nil {
			t.Fatal(err)
		}
		ids = append(ids, client.Id)
	}
	assignPayload, err := json.Marshal(map[string]any{"ids": ids})
	if err != nil {
		t.Fatal(err)
	}
	recorder = performAuthenticatedTestRequest(router,
		httptest.NewRequest(http.MethodPost, "/api/nodes/"+nodeUUID+"/clients", strings.NewReader(string(assignPayload))), cookies...)
	if !decodeNodePage(t, recorder).Success || recorder.Code != http.StatusOK {
		t.Fatalf("assign clients: %s", recorder.Body.String())
	}

	recorder = performAuthenticatedTestRequest(router,
		httptest.NewRequest(http.MethodGet, "/api/nodes/"+nodeUUID+"/clients?limit=5&offset=0", nil), cookies...)
	first := decodeNodePage(t, recorder)
	if first.Obj.Total != 7 || len(first.Obj.Items) != 5 {
		t.Fatalf("page 1 = %d of %d, want 5 of 7", len(first.Obj.Items), first.Obj.Total)
	}

	recorder = performAuthenticatedTestRequest(router,
		httptest.NewRequest(http.MethodGet, "/api/nodes/"+nodeUUID+"/clients?limit=5&offset=5", nil), cookies...)
	second := decodeNodePage(t, recorder)
	if second.Obj.Total != 7 || len(second.Obj.Items) != 2 {
		t.Fatalf("page 2 = %d of %d, want 2 of 7", len(second.Obj.Items), second.Obj.Total)
	}
	if first.Obj.Items[0].Id == second.Obj.Items[0].Id {
		t.Fatal("pages overlap; offset is not applied by the server")
	}

	recorder = performAuthenticatedTestRequest(router,
		httptest.NewRequest(http.MethodGet, "/api/nodes/"+nodeUUID+"/clients?limit=5&q=node-client-1", nil), cookies...)
	filtered := decodeNodePage(t, recorder)
	if filtered.Obj.Total != 1 || len(filtered.Obj.Items) != 1 {
		t.Fatalf("filtered = %d of %d, want the single match", len(filtered.Obj.Items), filtered.Obj.Total)
	}
}

// TestClientNodesRouteReplacesSelection is the client-card checkbox endpoint.
func TestClientNodesRouteReplacesSelection(t *testing.T) {
	router, cookies := nodeTestRouter(t)
	db := database.GetDB()

	client := model.Client{
		Enable:   true,
		Name:     "checkbox-client",
		Inbounds: json.RawMessage(`[]`),
		Config:   json.RawMessage(`{}`),
		Links:    json.RawMessage(`[]`),
	}
	if err := db.Create(&client).Error; err != nil {
		t.Fatal(err)
	}
	node := model.Node{UUID: "22222222-2222-4222-8222-222222222222", Name: "SE", LinkHost: "se.example.com", Enabled: true}
	if err := db.Create(&node).Error; err != nil {
		t.Fatal(err)
	}

	body := fmt.Sprintf(`{"nodes":[%q]}`, node.UUID)
	recorder := performAuthenticatedTestRequest(router,
		httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/clients/%d/nodes", client.Id), strings.NewReader(body)), cookies...)
	var result struct {
		Success bool `json:"success"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil || !result.Success {
		t.Fatalf("set client nodes: %s (%v)", recorder.Body.String(), err)
	}

	var assignments []model.ClientNode
	if err := db.Where("client_id = ?", client.Id).Find(&assignments).Error; err != nil {
		t.Fatal(err)
	}
	if len(assignments) != 1 || assignments[0].NodeUUID != node.UUID {
		t.Fatalf("assignments = %+v", assignments)
	}
}

// TestNodeRoutesRejectBadInput keeps a malformed body from half-applying.
func TestNodeRoutesRejectBadInput(t *testing.T) {
	router, cookies := nodeTestRouter(t)

	recorder := performAuthenticatedTestRequest(router,
		httptest.NewRequest(http.MethodPost, "/api/nodes", strings.NewReader("{not json")), cookies...)
	var result struct {
		Success bool `json:"success"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Success {
		t.Fatal("a malformed create body must not succeed")
	}

	recorder = performAuthenticatedTestRequest(router,
		httptest.NewRequest(http.MethodPatch, "/api/clients/notanumber/nodes", strings.NewReader(`{"nodes":[]}`)), cookies...)
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Success {
		t.Fatal("a non-numeric client id must not succeed")
	}
}
