package service

import (
	"encoding/json"
	"testing"

	"github.com/deposist/s-ui-x-extended/database/model"
)

func TestUpdateWithLimiterRolesPersistsOnlyAnnouncedKinds(t *testing.T) {
	db := initNodeTestDB(t)
	inventory, err := json.Marshal(NodeInventory{Limiters: []NodeInventoryLimiter{
		{Tag: "bw-out", Type: "bandwidth-limiter", Variant: "bandwidth"},
		{Tag: "conn-out", Type: "connection-limiter", Variant: "connection"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	nodes := &NodeService{}
	node, err := nodes.Create(model.Node{Name: "DE", LinkHost: "de.example.com", Enabled: true, Inventory: inventory})
	if err != nil {
		t.Fatal(err)
	}
	roles := model.NodeLimiterTags{SpeedShared: "bw-out", Connections: "conn-out"}
	updated, err := nodes.UpdateWithLimiterRoles(node.UUID, "DE", "de.example.com", true, roles)
	if err != nil {
		t.Fatalf("save roles: %v", err)
	}
	if updated.LimiterRoles != roles {
		t.Fatalf("returned roles = %+v, want %+v", updated.LimiterRoles, roles)
	}

	if _, err := nodes.UpdateWithLimiterRoles(node.UUID, "DE", "de.example.com", true, model.NodeLimiterTags{Connections: "bw-out"}); err == nil {
		t.Fatal("a bandwidth outbound must not be accepted as a connection limiter")
	}
	var stored model.Node
	if err := db.Where("uuid = ?", node.UUID).First(&stored).Error; err != nil {
		t.Fatal(err)
	}
	var got model.NodeLimiterTags
	if err := json.Unmarshal(stored.LimiterRoles, &got); err != nil {
		t.Fatal(err)
	}
	if got != roles {
		t.Fatalf("invalid update changed stored roles to %+v; want %+v", got, roles)
	}
}
