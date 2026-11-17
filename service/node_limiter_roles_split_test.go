package service

import (
	"github.com/deposist/s-ui-x-extended/database/model"
	"testing"
)

func TestNodeLimiterRolesRejectSameSplitOutbound(t *testing.T) {
	inventory := NodeInventory{Limiters: []NodeInventoryLimiter{{Tag: "bw", Type: "bandwidth-limiter"}}}
	if err := validateNodeLimiterRoles(inventory, model.NodeLimiterTags{SpeedDown: "bw", SpeedUp: "bw"}); err == nil {
		t.Fatal("same outbound cannot enforce both split directions")
	}
}
