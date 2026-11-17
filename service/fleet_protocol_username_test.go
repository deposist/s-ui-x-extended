package service

import (
	"encoding/json"
	"testing"

	"github.com/deposist/s-ui-x-extended/database/model"
	CM "github.com/sagernet/sing-box/service/manager/constant"
)

func TestProjectClientUsesProtocolAuthUsername(t *testing.T) {
	state := fleetState{
		nodes:  []fleetNode{{row: model.Node{UUID: "node-a", Enabled: true}, squadID: 3, tags: map[string]string{"mixed-in": "mixed", "vless-in": "vless"}, roles: model.NodeLimiterTags{SpeedShared: "bw", Connections: "conn"}}},
		assign: map[uint][]string{7: {"node-a"}},
		limits: map[uint]model.ClientLimits{7: {ClientId: 7, SpeedShared: true, Speed: 1000, Connections: 5}},
	}
	client := model.Client{Id: 7, Name: "display-label", Config: json.RawMessage(`{"mixed":{"username":"auth-user","password":"pw"},"vless":{"uuid":"11111111-1111-4111-8111-111111111111"}}`)}
	users := map[string]CM.UserCreate{}
	bandwidth := map[string]CM.BandwidthLimiterCreate{}
	connections := map[string]CM.ConnectionLimiterCreate{}
	if err := state.projectClient(client, users, bandwidth, connections); err != nil {
		t.Fatal(err)
	}
	mixed, ok := users[userKey("auth-user", "mixed-in")]
	if !ok || mixed.Username != "auth-user" {
		t.Fatalf("mixed manager identity = %+v, present=%v", mixed, ok)
	}
	vless, ok := users[userKey("display-label", "vless-in")]
	if !ok || vless.UUID != "11111111-1111-4111-8111-111111111111" {
		t.Fatalf("VLESS identity = %+v, present=%v", vless, ok)
	}
	for _, username := range []string{"auth-user", "display-label"} {
		bandwidthLimiter, ok := bandwidth[limiterKey(username, "bw")]
		if !ok || bandwidthLimiter.Username != username {
			t.Errorf("bandwidth limiter identity = %+v, present=%v; want %q", bandwidthLimiter, ok, username)
		}
		connectionLimiter, ok := connections[limiterKey(username, "conn")]
		if !ok || connectionLimiter.Username != username {
			t.Errorf("connection limiter identity = %+v, present=%v; want %q", connectionLimiter, ok, username)
		}
	}
}
