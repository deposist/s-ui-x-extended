package sub

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/deposist/s-ui-x-extended/service"
)

// TestGetSubsIncludesNodeLinks is the fleet subscription requirement: a client
// of the main server must receive, in one document, every connection it can use
// on the main server and on every node it is assigned to.
func TestGetSubsIncludesNodeLinks(t *testing.T) {
	initSubTestDB(t)
	if _, err := (&service.SettingService{}).GetAllSetting(); err != nil {
		t.Fatal(err)
	}
	setSubSetting(t, "subEncode", "false")
	setSubSetting(t, "subShowInfo", "false")
	setSubSetting(t, "subNameInRemark", "false")

	inbound := model.Inbound{
		Type:    "vless",
		Tag:     "vless-443",
		Options: json.RawMessage(`{"listen":"::","listen_port":443}`),
		Addrs:   json.RawMessage(`[]`),
		OutJson: json.RawMessage(`{}`),
	}
	if err := database.GetDB().Create(&inbound).Error; err != nil {
		t.Fatal(err)
	}

	descriptor, err := json.Marshal(map[string]interface{}{
		"hostname": "de1.example.com",
		"inbound": map[string]interface{}{
			"type": "vless", "tag": "vless-443", "listen_port": 443,
			"addrs": []interface{}{}, "out_json": map[string]interface{}{},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	inventory, err := json.Marshal(service.NodeInventory{Inbounds: []service.NodeInventoryInbound{{
		Tag: "vless-443", Type: "vless", Descriptor: string(descriptor),
	}}})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Create(&model.Node{
		UUID: "11111111-1111-4111-8111-111111111111", Name: "Германия-1",
		LinkHost: "de1.example.com", Enabled: true, Inventory: inventory,
	}).Error; err != nil {
		t.Fatal(err)
	}

	inboundsJSON, err := json.Marshal([]uint{inbound.Id})
	if err != nil {
		t.Fatal(err)
	}
	client := model.Client{
		Enable:    true,
		Name:      "fleet-client",
		SubSecret: "fleet-secret-0001",
		Inbounds:  inboundsJSON,
		Config:    json.RawMessage(`{"vless":{"uuid":"11111111-2222-4333-8444-555555555555","flow":""}}`),
		Links:     []byte(`[]`),
	}
	if err := database.GetDB().Create(&client).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Create(&model.ClientNode{
		ClientId: client.Id, NodeUUID: "11111111-1111-4111-8111-111111111111",
	}).Error; err != nil {
		t.Fatal(err)
	}

	assertLinks := func(stage string, wantPeer bool) {
		t.Helper()
		body, _, err := (&SubService{}).GetSubs(client.SubSecret)
		if err != nil || body == nil {
			t.Fatalf("%s: could not read the subscription: %v", stage, err)
		}
		if !strings.Contains(*body, "de1.example.com:443") || strings.Contains(*body, "slave.example.com:8443") != wantPeer {
			t.Fatalf("%s: missing legacy link or unexpected Slave link state", stage)
		}
	}
	assertLinks("standalone", false)
	nodes := &service.NodeService{}
	if err := nodes.SetMaster(); err != nil {
		t.Fatal(err)
	}
	code, err := nodes.Invite()
	if err != nil {
		t.Fatal(err)
	}
	peer, err := nodes.RedeemInvite(code, "paired-slave")
	if err != nil {
		t.Fatal(err)
	}
	var request service.PeerSyncRequest
	if err := json.Unmarshal([]byte(`{"inbounds":[{"id":23,"tag":"slave-only","type":"vless","link":{"port":8443}}],"publicHost":"slave.example.com"}`), &request); err != nil {
		t.Fatal(err)
	}
	sync := func() service.PeerSyncResponse {
		t.Helper()
		response, err := nodes.AcceptPeerSync(peer.UUID, peer.Token, request)
		if err != nil {
			t.Fatal(err)
		}
		return response
	}
	sync()
	if _, err := nodes.ChangePeerClients(peer.UUID, 23, []uint{client.Id}, true); err != nil {
		t.Fatal(err)
	}
	desired := sync()
	if len(desired.Users) != 1 {
		t.Fatal("Slave assignment did not produce a desired user")
	}
	assertLinks("waiting for Slave acknowledgment", false)
	request.AppliedRevision = desired.Revision
	sync()
	assertLinks("both nodes assigned", true)
	if err := nodes.RequestPeerDisconnect(peer.UUID); err != nil {
		t.Fatal(err)
	}
	assertLinks("waiting for Slave cleanup", false)
	request.AppliedRevision = sync().Revision
	if !sync().Detached {
		t.Fatal("Slave cleanup was not acknowledged")
	}
	assertLinks("Slave detached", false)
}

// TestGetSubsWithoutNodeAssignmentIsUnchanged guards the pre-fleet behaviour: a
// client with no node rows must produce exactly its local links.
func TestGetSubsWithoutNodeAssignmentIsUnchanged(t *testing.T) {
	initSubTestDB(t)
	if _, err := (&service.SettingService{}).GetAllSetting(); err != nil {
		t.Fatal(err)
	}
	setSubSetting(t, "subEncode", "false")
	setSubSetting(t, "subShowInfo", "false")
	setSubSetting(t, "subNameInRemark", "false")
	seedGoldenSubClient(t, "plain-secret-0001")

	body, _, err := (&SubService{}).GetSubs("plain-secret-0001")
	if err != nil {
		t.Fatal(err)
	}
	if body == nil || *body != goldenSubBody {
		t.Fatalf("body = %v, want the golden local-only body", body)
	}
}
