//go:build with_manager

package service

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/deposist/s-ui-x-extended/database/model"
	CM "github.com/sagernet/sing-box/service/manager/constant"
)

func TestProjectFleetCreatesAndRemovesClientRecords(t *testing.T) {
	db := initNodeTestDB(t)
	coreInstance := startFleetTestCore(t)
	node := model.Node{
		UUID: "11111111-1111-4111-8111-111111111111", Name: "DE", Enabled: true,
		Inventory: mustInventory(t, NodeInventory{Inbounds: []NodeInventoryInbound{{Tag: "vless-in", Type: "vless"}}}),
	}
	if err := db.Create(&node).Error; err != nil {
		t.Fatal(err)
	}
	client := model.Client{
		Enable: true, Name: "alice",
		Config:   json.RawMessage(`{"vless":{"uuid":"22222222-2222-4222-8222-222222222222","flow":"xtls-rprx-vision"}}`),
		Inbounds: json.RawMessage("[]"), Links: json.RawMessage("[]"),
	}
	if err := db.Create(&client).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.ClientNode{ClientId: client.Id, NodeUUID: node.UUID}).Error; err != nil {
		t.Fatal(err)
	}

	nodes := &NodeService{Runtime: NewRuntime(coreInstance)}
	if err := nodes.ProjectFleet(); err != nil {
		t.Fatalf("initial projection: %v", err)
	}
	var users []CM.User
	if err := coreInstance.WithFleetManager(func(manager CM.Manager) error {
		var err error
		users, err = manager.GetUsers(nil)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if len(users) != 1 || users[0].Username != "alice" || users[0].Inbound != "vless-in" || users[0].UUID != "22222222-2222-4222-8222-222222222222" {
		t.Fatalf("projected users = %+v", users)
	}

	if err := db.Where("client_id = ? AND node_uuid = ?", client.Id, node.UUID).Delete(&model.ClientNode{}).Error; err != nil {
		t.Fatal(err)
	}
	if err := nodes.ProjectFleet(); err != nil {
		t.Fatalf("projection after unassign: %v", err)
	}
	if err := coreInstance.WithFleetManager(func(manager CM.Manager) error {
		var readErr error
		users, readErr = manager.GetUsers(nil)
		return readErr
	}); err != nil {
		t.Fatal(err)
	}
	if len(users) != 0 {
		t.Fatalf("stale users after unassign = %+v", users)
	}
}

func TestProjectFleetDoesNotAdoptNameCollidingSquadOrDeleteItsNode(t *testing.T) {
	db := initNodeTestDB(t)
	coreInstance := startFleetTestCore(t)
	localUUID := "11111111-1111-4111-8111-111111111111"
	foreignUUID := "22222222-2222-4222-8222-222222222222"
	var foreign CM.Node
	if err := coreInstance.WithFleetManager(func(manager CM.Manager) error {
		squad, err := manager.CreateSquad(CM.SquadCreate{Name: FleetSquadPrefix + localUUID})
		if err != nil {
			return err
		}
		foreign, err = manager.CreateNode(CM.NodeCreate{UUID: foreignUUID, Name: "unrelated", SquadIDs: []int{squad.ID}})
		return err
	}); err != nil {
		t.Fatalf("create foreign manager entries: %v", err)
	}
	if err := db.Create(&model.Node{UUID: localUUID, Name: "local", Enabled: true}).Error; err != nil {
		t.Fatal(err)
	}

	err := (&NodeService{Runtime: NewRuntime(coreInstance)}).ProjectFleet()
	if err == nil {
		t.Fatal("projection must reject a name-colliding squad that has no ownership record")
	}
	if err := coreInstance.WithFleetManager(func(manager CM.Manager) error {
		var err error
		nodes, err := manager.GetNodes(nil)
		if err != nil {
			return err
		}
		for _, node := range nodes {
			if node.UUID == foreign.UUID {
				return nil
			}
		}
		return fmt.Errorf("unrelated manager node %q was deleted", foreign.UUID)
	}); err != nil {
		t.Fatal(err)
	}
	var ownership model.FleetNodeOwnership
	if err := db.First(&ownership, "node_uuid = ?", localUUID).Error; err == nil {
		t.Fatal("projection claimed ownership of the foreign squad")
	}
}

func mustInventory(t *testing.T, inventory NodeInventory) json.RawMessage {
	t.Helper()
	encoded, err := json.Marshal(inventory)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func TestProjectFleetPersistsRecreatedSquadOwnership(t *testing.T) {
	db := initNodeTestDB(t)
	instance := startFleetTestCore(t)
	nodes := &NodeService{Runtime: NewRuntime(instance)}
	node, err := nodes.Create(model.Node{Name: "recreated", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	var previous model.FleetNodeOwnership
	if err := db.First(&previous, "node_uuid = ?", node.UUID).Error; err != nil {
		t.Fatal(err)
	}
	if err := instance.WithFleetManager(func(manager CM.Manager) error {
		_, err := manager.DeleteSquad(previous.SquadID)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if err := nodes.ProjectFleet(); err != nil {
		t.Fatalf("recreate deleted squad: %v", err)
	}
	var ownership model.FleetNodeOwnership
	if err := db.First(&ownership, "node_uuid = ?", node.UUID).Error; err != nil {
		t.Fatal(err)
	}
	if ownership.SquadID == previous.SquadID {
		t.Fatalf("ownership retained deleted squad %d", previous.SquadID)
	}
	if err := nodes.ProjectFleet(); err != nil {
		t.Fatalf("repeat projection after recreation: %v", err)
	}
}

func TestNodeDeleteRevokesProjectedManagerRecords(t *testing.T) {
	db := initNodeTestDB(t)
	instance := startFleetTestCore(t)
	nodes := &NodeService{Runtime: NewRuntime(instance)}
	node, err := nodes.Create(model.Node{Name: "DE", Enabled: true,
		Inventory: mustInventory(t, NodeInventory{Inbounds: []NodeInventoryInbound{{Tag: "vless-in", Type: "vless"}}})})
	if err != nil {
		t.Fatal(err)
	}
	client := nodeTestClient(t, db, "alice")
	if err := db.Model(&client).Update("config", json.RawMessage(`{"vless":{"uuid":"22222222-2222-4222-8222-222222222222"}}`)).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := nodes.Assign(node.UUID, []uint{client.Id}); err != nil {
		t.Fatal(err)
	}
	if err := nodes.Delete(node.UUID); err != nil {
		t.Fatal(err)
	}
	if err := instance.WithFleetManager(func(manager CM.Manager) error {
		remaining, err := manager.GetNodes(nil)
		if err != nil {
			return err
		}
		users, err := manager.GetUsers(nil)
		if err != nil {
			return err
		}
		squads, err := manager.GetSquads(nil)
		if err != nil {
			return err
		}
		if len(remaining) != 0 || len(users) != 0 || len(squads) != 0 {
			return fmt.Errorf("deleted node retains manager records: nodes=%d users=%d squads=%d", len(remaining), len(users), len(squads))
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	var ownershipCount int64
	if err := db.Model(&model.FleetNodeOwnership{}).Count(&ownershipCount).Error; err != nil {
		t.Fatal(err)
	}
	if ownershipCount != 0 {
		t.Fatalf("deleted node retains %d ownership records", ownershipCount)
	}
}

func TestNodeUpdateDisabledRevokesManagerAccess(t *testing.T) {
	initNodeTestDB(t)
	instance := startFleetTestCore(t)
	nodes := &NodeService{Runtime: NewRuntime(instance)}
	node, err := nodes.Create(model.Node{Name: "DE", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := nodes.Update(node.UUID, "DE", "", false); err != nil {
		t.Fatal(err)
	}
	if err := instance.WithFleetManager(func(manager CM.Manager) error {
		remaining, err := manager.GetNodes(nil)
		if err == nil && len(remaining) != 0 {
			return fmt.Errorf("disabled node retains %d manager registrations", len(remaining))
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
}
