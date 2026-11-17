package service

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x-extended/database/model"

	CM "github.com/sagernet/sing-box/service/manager/constant"
)

func TestValidateNodeLimiterRolesUsesTypeSpecificInventory(t *testing.T) {
	inventory := NodeInventory{Limiters: []NodeInventoryLimiter{{Tag: "bw", Type: "bandwidth-limiter"}, {Tag: "conn", Type: "connection-limiter"}}}
	if err := validateNodeLimiterRoles(inventory, model.NodeLimiterTags{SpeedShared: "bw", Connections: "conn"}); err != nil {
		t.Fatal(err)
	}
	for _, roles := range []model.NodeLimiterTags{{SpeedDown: "conn"}, {Connections: "bw"}, {SpeedShared: "missing"}} {
		if err := validateNodeLimiterRoles(inventory, roles); err == nil {
			t.Errorf("invalid roles accepted: %+v", roles)
		}
	}
}

func TestEnsureSquadsUsesRecordedOwnership(t *testing.T) {
	db := &fleetProjectionManagerDouble{}
	state := fleetState{
		nodes:     []fleetNode{{row: model.Node{UUID: "11111111-1111-4111-8111-111111111111", Name: "new name", Enabled: true}}},
		ownership: map[string]int{"11111111-1111-4111-8111-111111111111": 7},
	}
	if err := state.ensureSquads(db); err != nil {
		t.Fatal(err)
	}
	if len(db.updatedNodes) != 1 || db.updatedNodes[0] != "11111111-1111-4111-8111-111111111111" {
		t.Fatalf("updated nodes = %v", db.updatedNodes)
	}
}

type fleetProjectionManagerDouble struct {
	CM.Manager
	updatedNodes []string
}

func (d *fleetProjectionManagerDouble) GetSquads(map[string][]string) ([]CM.Squad, error) {
	return []CM.Squad{{ID: 7}}, nil
}
func (d *fleetProjectionManagerDouble) GetNodes(map[string][]string) ([]CM.Node, error) {
	return []CM.Node{{UUID: "11111111-1111-4111-8111-111111111111", Name: "DE", SquadIDs: []int{7}}}, nil
}
func (d *fleetProjectionManagerDouble) CreateSquad(CM.SquadCreate) (CM.Squad, error) {
	return CM.Squad{}, nil
}
func (d *fleetProjectionManagerDouble) CreateNode(CM.NodeCreate) (CM.Node, error) {
	return CM.Node{}, nil
}
func (d *fleetProjectionManagerDouble) UpdateNode(uuid string, _ CM.NodeUpdate) (CM.Node, error) {
	d.updatedNodes = append(d.updatedNodes, uuid)
	return CM.Node{}, nil
}

func TestEnsureSquadsRefusesUnownedManagerNode(t *testing.T) {
	state := fleetState{nodes: []fleetNode{{row: model.Node{UUID: "11111111-1111-4111-8111-111111111111", Name: "DE", Enabled: true}}}}
	err := state.ensureSquads(&fleetProjectionManagerDouble{})
	if err == nil || !strings.Contains(err.Error(), "not panel-owned") {
		t.Fatalf("unowned manager node error=%v", err)
	}
}

func TestProjectLimitsAggregatesSharedOutboundSquads(t *testing.T) {
	config, err := json.Marshal(map[string]any{"vless": map[string]any{"uuid": "11111111-1111-4111-8111-111111111111"}})
	if err != nil {
		t.Fatal(err)
	}
	state := fleetState{
		nodes: []fleetNode{
			{row: model.Node{UUID: "a", Enabled: true}, squadID: 3, roles: model.NodeLimiterTags{SpeedShared: "bw"}},
			{row: model.Node{UUID: "b", Enabled: true}, squadID: 4, roles: model.NodeLimiterTags{SpeedShared: "bw"}},
		},
		assign: map[uint][]string{1: {"a", "b"}},
		limits: map[uint]model.ClientLimits{1: {ClientId: 1, SpeedShared: true, Speed: 1000}},
	}
	users := map[string]CM.UserCreate{}
	bandwidth := map[string]CM.BandwidthLimiterCreate{}
	connections := map[string]CM.ConnectionLimiterCreate{}
	if err := state.projectClient(model.Client{Id: 1, Name: "alice", Enable: true, Config: config}, users, bandwidth, connections); err != nil {
		t.Fatal(err)
	}
	limiter, ok := bandwidth[limiterKey("alice", "bw")]
	if !ok || !sameSquadSet(limiter.SquadIDs, []int{3, 4}) {
		t.Fatalf("aggregated limiter=%+v present=%v", limiter, ok)
	}
}

type fleetForeignMembershipDouble struct {
	fleetProjectionManagerDouble
	deleted bool
}

func (d *fleetForeignMembershipDouble) GetNodes(map[string][]string) ([]CM.Node, error) {
	return []CM.Node{{UUID: "11111111-1111-4111-8111-111111111111", SquadIDs: []int{7, 99}}}, nil
}

func (d *fleetForeignMembershipDouble) DeleteNode(string) (CM.Node, error) {
	d.deleted = true
	return CM.Node{}, nil
}

func TestEnsureSquadsDisabledNodePreservesForeignMembership(t *testing.T) {
	manager := &fleetForeignMembershipDouble{}
	state := fleetState{
		nodes:     []fleetNode{{row: model.Node{UUID: "11111111-1111-4111-8111-111111111111", Enabled: false}}},
		ownership: map[string]int{"11111111-1111-4111-8111-111111111111": 7},
	}
	err := state.ensureSquads(manager)
	if err == nil || manager.deleted {
		t.Fatalf("disabled node with unrelated membership must be preserved: err=%v deleted=%v", err, manager.deleted)
	}
}
