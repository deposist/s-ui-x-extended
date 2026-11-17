package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
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

	createBody := strings.NewReader("name=" + url.QueryEscape("Германия-1") + "&linkHost=de1.example.com")
	createRequest := httptest.NewRequest(http.MethodPost, "/api/nodes", createBody)
	createRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := performAuthenticatedTestRequest(router, createRequest, cookies...)
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
	assignPayload, err := json.Marshal(ids)
	if err != nil {
		t.Fatal(err)
	}
	assignRequest := httptest.NewRequest(http.MethodPost, "/api/nodes/"+nodeUUID+"/clients",
		strings.NewReader("ids="+url.QueryEscape(string(assignPayload))))
	assignRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder = performAuthenticatedTestRequest(router, assignRequest, cookies...)
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

	nodesPayload, err := json.Marshal([]string{node.UUID})
	if err != nil {
		t.Fatal(err)
	}
	patchRequest := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/clients/%d/nodes", client.Id),
		strings.NewReader("nodes="+url.QueryEscape(string(nodesPayload))))
	patchRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := performAuthenticatedTestRequest(router, patchRequest, cookies...)
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
func TestNodePatchPreservesRolesWhenFieldsAreOmitted(t *testing.T) {
	router, cookies := nodeTestRouter(t)
	db := database.GetDB()
	inventory := json.RawMessage(`{"limiters":[{"tag":"bw","type":"bandwidth-limiter"},{"tag":"conn","type":"connection-limiter"}]}`)
	node := model.Node{UUID:"33333333-3333-4333-8333-333333333333",Name:"patch",Enabled:true,Inventory:inventory,LimiterRoles:json.RawMessage(`{"speedShared":"bw","connections":"conn"}`)}
	if err:=db.Create(&node).Error;err!=nil{t.Fatal(err)}
	recorder:=performAuthenticatedTestRequest(router,httptest.NewRequest(http.MethodPatch,"/api/nodes/"+node.UUID,strings.NewReader("name=renamed")),cookies...)
	if recorder.Code!=http.StatusOK{t.Fatalf("status=%d body=%s",recorder.Code,recorder.Body.String())}
	var stored model.Node
	if err:=db.Where("uuid = ?",node.UUID).First(&stored).Error;err!=nil{t.Fatal(err)}
	var roles model.NodeLimiterTags
	if err:=json.Unmarshal(stored.LimiterRoles,&roles);err!=nil{t.Fatal(err)}
	if roles.SpeedShared!="bw"||roles.Connections!="conn"{t.Fatalf("PATCH without role fields cleared mapping: %+v",roles)}
}

func TestNodePatchAcceptsAndRejectsLimiterRoles(t *testing.T) {
	router,cookies:=nodeTestRouter(t)
	db:=database.GetDB()
	node:=model.Node{UUID:"44444444-4444-4444-8444-444444444444",Name:"limits",Enabled:true,Inventory:json.RawMessage(`{"limiters":[{"tag":"bw","type":"bandwidth-limiter"},{"tag":"conn","type":"connection-limiter"}]}`)}
	if err:=db.Create(&node).Error;err!=nil{t.Fatal(err)}
	patch := func(body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPatch, "/api/nodes/"+node.UUID, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		return performAuthenticatedTestRequest(router, req, cookies...)
	}
	r:=patch("name=limits&speedShared=bw&connections=conn")
	if r.Code!=http.StatusOK{t.Fatalf("valid roles patch failed: %d %s",r.Code,r.Body.String())}
	r=patch("name=limits&connections=bw")
	var rejected struct{ Success bool `json:"success"` }
	if err:=json.Unmarshal(r.Body.Bytes(),&rejected);err!=nil{t.Fatalf("decode: %v",err)}
	if rejected.Success{t.Fatal("bandwidth outbound accepted for connections role")}
}

func TestNodeRoutesRejectBadInput(t *testing.T) {
	router, cookies := nodeTestRouter(t)

	recorder := performAuthenticatedTestRequest(router,
		httptest.NewRequest(http.MethodPost, "/api/nodes", strings.NewReader("enabled=maybe")), cookies...)
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
		httptest.NewRequest(http.MethodPatch, "/api/clients/notanumber/nodes", strings.NewReader("")), cookies...)
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Success {
		t.Fatal("a non-numeric client id must not succeed")
	}
}