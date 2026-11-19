package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/deposist/s-ui-x-extended/service"
	"github.com/gin-gonic/gin"
)

func TestSlaveTokenAPIBlocksClientWritesButAllowsLocalNetworkEdits(t *testing.T) {
	router := newAPIV2TokenTestRouter(t)
	db := database.GetDB()
	if err := db.Create(&model.NodePanel{ID: 1, Role: "slave"}).Error; err != nil {
		t.Fatal(err)
	}
	post := func(path string, values url.Values) (*httptest.ResponseRecorder, Msg) {
		t.Helper()
		request := httptest.NewRequest(http.MethodPost, "/apiv2"+path, strings.NewReader(values.Encode()))
		request.Header.Set("Authorization", "Bearer legacy-token")
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)
		var body Msg
		if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
			t.Fatalf("%s returned status %d without a JSON response: %v", path, recorder.Code, err)
		}
		return recorder, body
	}
	assertBlocked := func(path string, values url.Values) {
		t.Helper()
		response, body := post(path, values)
		if response.Code != http.StatusForbidden || body.Success || body.Msg != service.ErrSlaveClientWrite.Error() {
			t.Errorf("%s %s: expected Slave role denial, got %d %s", path, values.Get("action"), response.Code, body.Msg)
		}
	}
	for _, action := range []string{"new", "edit", "del", "addbulk", "editbulk", "delbulk"} {
		assertBlocked("/save", url.Values{"object": {"clients"}, "action": {action}})
	}
	for _, path := range []string{"/importdb", "/import-xui", "/import-xui/apply", "/import-xui/rollback", "/rotateSubSecret", "/regenerateClientLinks"} {
		assertBlocked(path, nil)
	}
	response, body := post("/save", url.Values{"object": {"outbounds"}, "action": {"new"}, "data": {`{"tag":"slave-local-direct","type":"direct"}`}})
	if response.Code != http.StatusOK || !body.Success {
		t.Fatalf("local outbound write: %d %s", response.Code, body.Msg)
	}
	var outbound model.Outbound
	if err := db.Where("tag = ?", "slave-local-direct").Take(&outbound).Error; err != nil {
		t.Fatal(err)
	}
}

func TestNodePeerRejectsPlainHTTP(t *testing.T) {
	router := gin.New()
	RegisterNodePeerRoutes(router.Group("/panel/node-peer"))
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/panel/node-peer/enroll", strings.NewReader(`{}`))
	router.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("expected HTTP rejection, got %d", response.Code)
	}
}

func TestNodeDeleteClientsReadsFormBody(t *testing.T) {
	request := httptest.NewRequest(http.MethodDelete, "/nodes/slave/clients", strings.NewReader("inboundId=23&ids=%5B1%5D"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	form, err := nodeClientsForm(httptest.NewRecorder(), request)
	if err != nil || form.Get("inboundId") != "23" || form.Get("ids") != "[1]" || request.Method != http.MethodDelete {
		t.Fatalf("could not read DELETE form: %v (%v)", form, err)
	}
}

func TestNodePeerTLSInventoryAndRevocation(t *testing.T) {
	initAPITestDB(t, filepath.Join(t.TempDir(), "nodes.db"))
	pool, err := database.GetDB().DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pool.Close() })
	master := &service.NodeService{}
	if err := master.SetMaster(); err != nil {
		t.Fatal(err)
	}
	code, err := master.Invite()
	if err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	RegisterNodePeerRoutes(router.Group("/panel/node-peer"))
	server := httptest.NewTLSServer(router)
	defer server.Close()
	response, err := server.Client().Post(server.URL+"/panel/node-peer/enroll", "application/json", strings.NewReader(`{"name":"test Slave","code":"`+code+`"}`))
	if err != nil {
		t.Fatal(err)
	}
	var credential struct{ UUID, Token string }
	decodeErr := json.NewDecoder(response.Body).Decode(&credential)
	_ = response.Body.Close()
	if response.StatusCode != http.StatusCreated || decodeErr != nil || credential.Token == "" {
		t.Fatalf("enrollment failed: status %d, decode error %v", response.StatusCode, decodeErr)
	}
	sync := func(body string, expected int) {
		t.Helper()
		request, err := http.NewRequest(http.MethodPost, server.URL+"/panel/node-peer/sync", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		request.Header.Set("Authorization", "Bearer "+credential.Token)
		request.Header.Set("X-Node-ID", credential.UUID)
		response, err := server.Client().Do(request)
		if err != nil {
			t.Fatal(err)
		}
		_ = response.Body.Close()
		if response.StatusCode != expected {
			t.Fatalf("sync: want %d, got %d", expected, response.StatusCode)
		}
	}
	sync(`{"inbounds":[{"id":1,"tag":"local","type":"vless","rawConfig":{}}]}`, http.StatusBadRequest)
	sync(`{"inbounds":[{"id":1,"tag":"local","type":"vless"}]}`, http.StatusOK)
	node, err := master.Get(credential.UUID)
	if err != nil || !node.Online || !node.Paired || len(node.Inbounds) != 1 || node.Inbounds[0] != "local" {
		t.Fatalf("unexpected paired node state: %v", err)
	}
	client := model.Client{Name: "assigned-client"}
	if err := database.GetDB().Create(&client).Error; err != nil {
		t.Fatal(err)
	}
	admin := gin.New()
	admin.Use(func(c *gin.Context) { c.Set(apiUsernameKey, "test-admin") })
	(&APIHandler{}).registerNodeRoutes(admin.Group("/api"))
	path := "/api/nodes/" + credential.UUID + "/clients"
	for _, method := range []string{http.MethodPost, http.MethodDelete, http.MethodPost} {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(method, path, strings.NewReader(fmt.Sprintf("inboundId=1&ids=%%5B%d%%5D", client.Id)))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		admin.ServeHTTP(response, request)
		var message struct {
			Success bool
			Obj     struct{ Changed int }
		}
		if err := json.Unmarshal(response.Body.Bytes(), &message); err != nil || !message.Success || message.Obj.Changed != 1 {
			t.Fatalf("%s assignment: status %d, body %s", method, response.Code, response.Body.String())
		}
	}
	if err := master.Delete(credential.UUID); err != nil {
		t.Fatal(err)
	}
	var remaining int64
	if err := database.GetDB().Model(&model.PeerAssignment{}).Where("node_uuid = ?", credential.UUID).Count(&remaining).Error; err != nil || remaining != 0 {
		t.Fatalf("node removal left %d assignments: %v", remaining, err)
	}
	sync(`{"inbounds":[]}`, http.StatusUnauthorized)
}
