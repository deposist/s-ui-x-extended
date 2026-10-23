package service

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/deposist/s-ui-x-extended/util/common"

	"gorm.io/gorm"
)

// initNodeTestDB stands up a throwaway on-disk database. It follows the
// Windows-safe discipline used by the other service tests: a manually created
// temp dir, the handle closed before removal, and retry-deletion, because
// t.TempDir's automatic RemoveAll races SQLite's WAL/SHM files.
func initNodeTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	tempDir, err := os.MkdirTemp("", "s-ui-nodes-")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("SUI_DB_FOLDER", tempDir)
	if err := database.InitDB(filepath.Join(tempDir, "s-ui.db")); err != nil {
		_ = os.RemoveAll(tempDir)
		if contains(err.Error(), "go-sqlite3 requires cgo") {
			t.Skip(err)
		}
		t.Fatal(err)
	}
	db := database.GetDB()
	t.Cleanup(func() {
		if sqlDB, closeErr := db.DB(); closeErr == nil {
			_ = sqlDB.Close()
		}
		_ = os.RemoveAll(tempDir)
	})
	return db
}

func nodeTestClient(t *testing.T, db *gorm.DB, name string) model.Client {
	t.Helper()
	client := model.Client{
		Enable:   true,
		Name:     name,
		Inbounds: json.RawMessage(`[]`),
		Config:   json.RawMessage(`{}`),
		Links:    json.RawMessage(`[]`),
	}
	if err := db.Create(&client).Error; err != nil {
		t.Fatalf("create client %q: %v", name, err)
	}
	return client
}

// TestNodeServiceCreateListUpdateDelete covers the lifecycle the Nodes section
// drives: a node is created with a generated uuid, edited, then removed.
func TestNodeServiceCreateListUpdateDelete(t *testing.T) {
	initNodeTestDB(t)
	nodes := &NodeService{}

	created, err := nodes.Create(model.Node{Name: "Германия-1", LinkHost: "de1.example.com", Enabled: true})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.UUID == "" {
		t.Fatal("created node has no uuid")
	}
	if _, err := nodes.Create(model.Node{Name: "  "}); err == nil {
		t.Fatal("a blank node name must be rejected")
	}

	list, err := nodes.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 || list[0].UUID != created.UUID {
		t.Fatalf("list = %+v", list)
	}

	updated, err := nodes.Update(created.UUID, "Германия-2", "de2.example.com", false)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Name != "Германия-2" || updated.LinkHost != "de2.example.com" || updated.Enabled {
		t.Fatalf("updated = %+v", updated)
	}
	if _, err := nodes.Update("11111111-1111-4111-8111-111111111111", "x", "", true); err == nil {
		t.Fatal("updating a missing node must fail")
	}

	if err := nodes.Delete(created.UUID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if list, _ := nodes.List(); len(list) != 0 {
		t.Fatalf("node survived delete: %+v", list)
	}
}

// TestNodeServiceClientsAreServerPaginated is the requirement behind the pager:
// the server returns one page plus the total under the current filter, so the
// browser never receives the whole client list.
func TestNodeServiceClientsAreServerPaginated(t *testing.T) {
	db := initNodeTestDB(t)
	nodes := &NodeService{}

	node, err := nodes.Create(model.Node{Name: "США-1", LinkHost: "us1.example.com", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	other, err := nodes.Create(model.Node{Name: "Финляндия-1", LinkHost: "fi1.example.com", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}

	// Twelve assigned clients plus one on another node, so the total proves the
	// query filters by node rather than counting every assignment.
	ids := make([]uint, 0, 12)
	for i := 0; i < 12; i++ {
		client := nodeTestClient(t, db, fmt.Sprintf("client-%02d", i))
		ids = append(ids, client.Id)
	}
	outsider := nodeTestClient(t, db, "client-outsider")
	if changed, err := nodes.Assign(node.UUID, ids); err != nil || changed != len(ids) {
		t.Fatalf("assign: changed=%d err=%v", changed, err)
	}
	if _, err := nodes.Assign(other.UUID, []uint{outsider.Id}); err != nil {
		t.Fatal(err)
	}

	page, err := nodes.Clients(node.UUID, 10, 0, "", true)
	if err != nil {
		t.Fatalf("clients page 1: %v", err)
	}
	if page.Total != 12 {
		t.Fatalf("total = %d, want 12 (the other node's client must not count)", page.Total)
	}
	if len(page.Items) != 10 {
		t.Fatalf("page 1 returned %d items, want 10", len(page.Items))
	}
	page2, err := nodes.Clients(node.UUID, 10, 10, "", true)
	if err != nil {
		t.Fatalf("clients page 2: %v", err)
	}
	if page2.Total != 12 || len(page2.Items) != 2 {
		t.Fatalf("page 2 = %+v, want 2 of 12", page2)
	}
	if page.Items[0].Id == page2.Items[0].Id {
		t.Fatal("pages overlap; offset is not applied")
	}

	filtered, err := nodes.Clients(node.UUID, 10, 0, "client-0", true)
	if err != nil {
		t.Fatalf("filtered: %v", err)
	}
	if filtered.Total != 10 || len(filtered.Items) != 10 {
		t.Fatalf("filtered = %+v, want the ten client-0N rows", filtered)
	}

	// A LIKE wildcard in the query must match literally, not act as "all".
	if wildcard, err := nodes.Clients(node.UUID, 10, 0, "%", true); err != nil || wildcard.Total != 0 {
		t.Fatalf("a bare %% query matched %d rows; wildcards must be escaped (%v)", wildcard.Total, err)
	}

	// A limit of zero falls back to the default page size, which is larger than
	// this fixture, so every row comes back; an oversized limit is capped
	// instead of letting a caller ask for everything.
	defaulted, err := nodes.Clients(node.UUID, 0, 0, "", true)
	if err != nil {
		t.Fatalf("default page size: %v", err)
	}
	if defaulted.Total != 12 || len(defaulted.Items) != 12 {
		t.Fatalf("default page = %d of %d, want all 12", len(defaulted.Items), defaulted.Total)
	}
	if _, err := nodes.Clients(node.UUID, 100000, 0, "", true); err != nil {
		t.Fatalf("oversized limit: %v", err)
	}
}

// TestNodeServiceAssignIsIdempotent keeps a bulk call that overlaps an earlier
// one from failing or double-counting.
func TestNodeServiceAssignIsIdempotent(t *testing.T) {
	db := initNodeTestDB(t)
	nodes := &NodeService{}
	node, err := nodes.Create(model.Node{Name: "NL", LinkHost: "nl.example.com", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	client := nodeTestClient(t, db, "dup-client")

	if changed, err := nodes.Assign(node.UUID, []uint{client.Id}); err != nil || changed != 1 {
		t.Fatalf("first assign: changed=%d err=%v", changed, err)
	}
	if changed, err := nodes.Assign(node.UUID, []uint{client.Id}); err != nil || changed != 0 {
		t.Fatalf("second assign must be a no-op: changed=%d err=%v", changed, err)
	}
	if _, err := nodes.Assign("11111111-1111-4111-8111-111111111111", []uint{client.Id}); err == nil {
		t.Fatal("assigning to a missing node must fail")
	}

	if changed, err := nodes.Unassign(node.UUID, []uint{client.Id}); err != nil || changed != 1 {
		t.Fatalf("unassign: changed=%d err=%v", changed, err)
	}
	if changed, err := nodes.Unassign(node.UUID, []uint{client.Id}); err != nil || changed != 0 {
		t.Fatalf("second unassign must be a no-op: changed=%d err=%v", changed, err)
	}
}

// TestSetClientNodesReplacesSelection is the client-card path: the checkboxes
// are saved as a set, so an unchecked box really removes the assignment.
func TestSetClientNodesReplacesSelection(t *testing.T) {
	db := initNodeTestDB(t)
	nodes := &NodeService{}
	first, err := nodes.Create(model.Node{Name: "A", LinkHost: "a.example.com", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	second, err := nodes.Create(model.Node{Name: "B", LinkHost: "b.example.com", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	client := nodeTestClient(t, db, "checkbox-client")

	if err := nodes.SetClientNodes(client.Id, []string{first.UUID, second.UUID}); err != nil {
		t.Fatalf("set both: %v", err)
	}
	got, err := nodes.ClientNodeUUIDs(client.Id)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("assigned nodes = %v, want both", got)
	}

	if err := nodes.SetClientNodes(client.Id, []string{second.UUID}); err != nil {
		t.Fatalf("keep one: %v", err)
	}
	got, _ = nodes.ClientNodeUUIDs(client.Id)
	if len(got) != 1 || got[0] != second.UUID {
		t.Fatalf("after unchecking one, assigned = %v, want just %s", got, second.UUID)
	}

	if err := nodes.SetClientNodes(client.Id, nil); err != nil {
		t.Fatalf("clear: %v", err)
	}
	if got, _ := nodes.ClientNodeUUIDs(client.Id); len(got) != 0 {
		t.Fatalf("after clearing, assigned = %v, want none", got)
	}
}

// TestDeleteNodeDropsAssignments keeps a removed server from leaving dangling
// rows that would still count as assignments.
func TestDeleteNodeDropsAssignments(t *testing.T) {
	db := initNodeTestDB(t)
	nodes := &NodeService{}
	node, err := nodes.Create(model.Node{Name: "SE", LinkHost: "se.example.com", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	client := nodeTestClient(t, db, "orphan-client")
	if _, err := nodes.Assign(node.UUID, []uint{client.Id}); err != nil {
		t.Fatal(err)
	}

	if err := nodes.Delete(node.UUID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	var count int64
	if err := db.Model(model.ClientNode{}).Where("node_uuid = ?", node.UUID).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("%d assignments survived the node delete", count)
	}
}

// TestClientDeleteClearsNodeAssignments is the mirror rule: a client that no
// longer exists must not linger on any server.
func TestClientDeleteClearsNodeAssignments(t *testing.T) {
	db := initNodeTestDB(t)
	nodes := &NodeService{}
	node, err := nodes.Create(model.Node{Name: "DE", LinkHost: "de.example.com", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	client := nodeTestClient(t, db, "deleted-client")
	if _, err := nodes.Assign(node.UUID, []uint{client.Id}); err != nil {
		t.Fatal(err)
	}

	clients := &ClientService{}
	payload, err := json.Marshal(client.Id)
	if err != nil {
		t.Fatal(err)
	}
	err = db.Transaction(func(tx *gorm.DB) error {
		_, err := clients.Save(tx, "del", payload, "")
		return err
	})
	if err != nil {
		t.Fatalf("delete client: %v", err)
	}
	var count int64
	if err := db.Model(model.ClientNode{}).Where("client_id = ?", client.Id).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("%d assignments survived the client delete", count)
	}
}

// TestNodeServiceRejectsBadUUID keeps a hand-written uuid from becoming a node
// identity that no node will ever announce.
func TestNodeServiceRejectsBadUUID(t *testing.T) {
	initNodeTestDB(t)
	if _, err := (&NodeService{}).Create(model.Node{Name: "bad", UUID: "not-a-uuid", Enabled: true}); err == nil {
		t.Fatal("a non-uuid node identity must be rejected")
	} else if !contains(err.Error(), "not a uuid") {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = common.NewError
}
