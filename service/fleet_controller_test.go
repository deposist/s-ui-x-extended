//go:build with_manager

package service

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x-extended/database/model"
)

// TestInjectFleetControllerAddsManager is the controller side of the fleet: once
// a server exists the panel's own core must run a manager, otherwise no node can
// ever connect.
func TestInjectFleetControllerAddsManager(t *testing.T) {
	db := initNodeTestDB(t)
	if _, err := (&NodeService{}).Create(model.Node{Name: "DE", LinkHost: "de.example.com", Enabled: true}); err != nil {
		t.Fatal(err)
	}

	services := InjectFleetController(db, nil)
	if len(services) != 1 {
		t.Fatalf("expected the manager to be added, got %d services", len(services))
	}
	var parsed struct {
		Type     string `json:"type"`
		Tag      string `json:"tag"`
		Database struct {
			Driver string `json:"driver"`
			DSN    string `json:"dsn"`
		} `json:"database"`
	}
	if err := json.Unmarshal(services[0], &parsed); err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if parsed.Type != "manager" || parsed.Tag != FleetControllerServiceTag {
		t.Fatalf("service = %+v", parsed)
	}
	if parsed.Database.Driver != "sqlite" {
		t.Fatalf("driver = %q, want sqlite", parsed.Database.Driver)
	}
	if !strings.HasSuffix(parsed.Database.DSN, "fleet.db") {
		t.Fatalf("dsn = %q, want the fleet db next to s-ui.db", parsed.Database.DSN)
	}
}

// TestInjectFleetControllerWithoutNodesIsANoOp keeps an installation that never
// uses Nodes from gaining a manager, a database and a listener it has no use for.
func TestInjectFleetControllerWithoutNodesIsANoOp(t *testing.T) {
	db := initNodeTestDB(t)
	existing := json.RawMessage(`{"type":"resolved","tag":"dns"}`)
	injected := InjectFleetController(db, []json.RawMessage{existing})
	if len(injected) != 1 || string(injected[0]) != string(existing) {
		t.Fatalf("a controller with no servers must keep its config, got %v", injected)
	}
}

// TestInjectFleetControllerKeepsExistingManager keeps a hand-built manager
// authoritative: the panel must not append a second one.
func TestInjectFleetControllerKeepsExistingManager(t *testing.T) {
	db := initNodeTestDB(t)
	if _, err := (&NodeService{}).Create(model.Node{Name: "DE", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	existing := json.RawMessage(`{"type":"manager","tag":"my-fleet","database":{"driver":"postgresql","dsn":"host=db"}}`)
	injected := InjectFleetController(db, []json.RawMessage{existing})
	if len(injected) != 1 {
		t.Fatalf("a configured manager must not be duplicated, got %d services", len(injected))
	}
	if string(injected[0]) != string(existing) {
		t.Fatalf("the existing manager was rewritten: %s", injected[0])
	}
}

// TestInjectFleetControllerKeepsOtherServices proves the injection is additive
// and leaves every other service untouched.
func TestInjectFleetControllerKeepsOtherServices(t *testing.T) {
	db := initNodeTestDB(t)
	if _, err := (&NodeService{}).Create(model.Node{Name: "DE", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	existing := json.RawMessage(`{"type":"resolved","tag":"dns"}`)
	injected := InjectFleetController(db, []json.RawMessage{existing})
	if len(injected) != 2 {
		t.Fatalf("expected the manager to be appended, got %d services", len(injected))
	}
	if string(injected[0]) != string(existing) {
		t.Fatalf("an existing service was rewritten: %s", injected[0])
	}
}
