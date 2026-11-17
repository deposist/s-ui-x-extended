//go:build with_manager

package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/deposist/s-ui-x-extended/core"
	"github.com/deposist/s-ui-x-extended/database/model"

	CM "github.com/sagernet/sing-box/service/manager/constant"
)

// startFleetTestCore starts a real core with a manager on a throwaway fleet
// database. The manager types are only compiled under the tag, so the accessor
// has to be exercised against a live service rather than a fake one.
func startFleetTestCore(t *testing.T) *core.Core {
	t.Helper()
	dir, err := os.MkdirTemp("", "sui-inv-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	fleetDB := filepath.ToSlash(filepath.Join(dir, "fleet.db"))
	config := `{
		"log":{"disabled":true},
		"dns":{"servers":[{"type":"local","tag":"local"}]},
		"outbounds":[{"type":"direct","tag":"direct"}],
		"route":{"final":"direct"},
		"services":[{"type":"manager","tag":"fleet","database":{"driver":"sqlite","dsn":"` + fleetDB + `"}}]
	}`
	instance := core.NewCore()
	if err := instance.Start([]byte(config)); err != nil {
		t.Fatalf("start core: %v", err)
	}
	t.Cleanup(func() { _ = instance.Stop() })
	return instance
}

// TestSyncInventoryStoresAnnouncedInventory is the reason the panel needs the
// fork release: a node's inventory reaches the panel through the manager, and
// the row keeps it so the UI can show what the node accepts.
func TestSyncInventoryStoresAnnouncedInventory(t *testing.T) {
	db := initNodeTestDB(t)
	nodes := &NodeService{Runtime: NewRuntime(startFleetTestCore(t))}
	created, err := nodes.Create(model.Node{Name: "DE", LinkHost: "de.example.com", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}

	// Feed the manager the hello a node would send.
	stored := false
	instance := nodes.runtime().Core()
	if err := instance.WithFleetManager(func(manager CM.Manager) error {
		setter, ok := manager.(interface{ SetNodeHello(string, CM.NodeHello) })
		if !ok {
			t.Fatal("the running manager cannot record hellos")
		}
		setter.SetNodeHello(created.UUID, CM.NodeHello{
			ProtocolVersion: 1,
			Inbounds: []CM.NodeHelloInbound{{
				Tag: "vless-443", Type: "vless",
				Descriptor: `{"hostname":"de.example.com","inbound":{"type":"vless","tag":"vless-443","listen_port":443,"addrs":[],"out_json":{}}}`,
			}},
			Limiters: []CM.NodeHelloLimiter{{Tag: "bw", Type: "bandwidth-limiter", Variant: "bandwidth"}},
		})
		stored = true
		return nil
	}); err != nil {
		t.Fatalf("set hello: %v", err)
	}
	if !stored {
		t.Fatal("no manager ran")
	}

	updated, err := nodes.SyncInventory()
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	if updated != 1 {
		t.Fatalf("updated %d nodes, want 1", updated)
	}

	view, err := nodes.Get(created.UUID)
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Inbounds) != 1 || view.Inbounds[0] != "vless-443" {
		t.Fatalf("inbounds = %v, want the announced tag", view.Inbounds)
	}
	if len(view.Limiters) != 1 || view.Limiters[0].Tag != "bw" || view.Limiters[0].Type != "bandwidth-limiter" {
		t.Fatalf("limiters = %+v, want typed announced limiter", view.Limiters)
	}

	// The stored blob must be the shape the link builder reads.
	var node model.Node
	if err := db.Model(model.Node{}).Where("uuid = ?", created.UUID).First(&node).Error; err != nil {
		t.Fatal(err)
	}
	inventory := DecodeNodeInventory(node.Inventory)
	if len(inventory.Inbounds) != 1 {
		t.Fatalf("stored inventory = %+v", inventory)
	}
	if _, ok := DecodeNodeInboundDescriptor(inventory.Inbounds[0].Descriptor); !ok {
		t.Fatal("the stored descriptor is not readable by the link builder")
	}
}

// TestSyncInventoryWithoutCoreIsANoOp keeps a stopped core from clearing rows.
func TestSyncInventoryWithoutCoreIsANoOp(t *testing.T) {
	db := initNodeTestDB(t)
	nodes := &NodeService{}
	created, err := nodes.Create(model.Node{Name: "FI", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	inventory, err := json.Marshal(NodeInventory{Inbounds: []NodeInventoryInbound{{Tag: "kept", Type: "vless"}}})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Model(model.Node{}).Where("uuid = ?", created.UUID).Update("inventory", inventory).Error; err != nil {
		t.Fatal(err)
	}

	if _, err := nodes.SyncInventory(); err != nil {
		t.Fatalf("sync without a core: %v", err)
	}
	view, err := nodes.Get(created.UUID)
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Inbounds) != 1 || view.Inbounds[0] != "kept" {
		t.Fatalf("a stopped core must not clear the stored inventory, got %v", view.Inbounds)
	}
}
