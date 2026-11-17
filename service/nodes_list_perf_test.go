package service

import (
	"testing"

	"github.com/deposist/s-ui-x-extended/database/model"
)

func TestNodeViewsBatchCountsAndLimiterTypes(t *testing.T) {
	initNodeTestDB(t)
	nodes := &NodeService{}
	for _, name := range []string{"A", "B"} {
		if _, err := nodes.Create(model.Node{Name: name, Enabled: true}); err != nil {
			t.Fatal(err)
		}
	}
	views, err := nodes.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(views) != 2 {
		t.Fatalf("got %d nodes, want 2", len(views))
	}
}

func TestLimiterRoleValidationRejectsSameSplitOutbound(t *testing.T) {
	inventory := NodeInventory{Limiters: []NodeInventoryLimiter{{Tag: "bw", Type: "bandwidth-limiter"}, {Tag: "conn", Type: "connection-limiter"}}}
	roles := model.NodeLimiterTags{SpeedDown: "bw", SpeedUp: "bw"}
	if err := validateNodeLimiterRoles(inventory, roles); err == nil {
		t.Fatal("same outbound cannot enforce both split directions")
	}
	if err := validateNodeLimiterRoles(inventory, model.NodeLimiterTags{Connections: "bw"}); err == nil {
		t.Fatal("bandwidth outbound cannot be used for connection limits")
	}
}
