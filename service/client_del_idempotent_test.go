package service

import (
	"encoding/json"
	"testing"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
)

func newTestClient(t *testing.T, name string) model.Client {
	t.Helper()
	client := model.Client{
		Enable:      true,
		Name:        name,
		Config:      json.RawMessage(`{"mixed":{"username":"` + name + `","password":"pw"}}`),
		Inbounds:    json.RawMessage(`[]`),
		Links:       json.RawMessage(`[]`),
		IPLimitMode: "monitor",
	}
	if err := database.GetDB().Create(&client).Error; err != nil {
		t.Fatal(err)
	}
	return client
}

func clientExists(t *testing.T, id uint) bool {
	t.Helper()
	var count int64
	if err := database.GetDB().Model(model.Client{}).Where("id = ?", id).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	return count > 0
}

// Deleting a client that is already gone (stale UI row, concurrent delete from
// another session, a resubmitted request) must be an idempotent success, not a
// "record not found" failure surfaced to the operator.
func TestConfigSaveClientsDelIsIdempotent(t *testing.T) {
	initSettingTestDB(t)

	client := newTestClient(t, "todelete")
	cs := NewConfigServiceWithRuntime(NewRuntimeWithCoreProvider(nil))
	payload, err := json.Marshal(client.Id)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := cs.Save("clients", "del", payload, "", "admin", "example.com"); err != nil {
		t.Fatalf("deleting an existing client should succeed: %v", err)
	}
	if clientExists(t, client.Id) {
		t.Fatal("client should be gone after delete")
	}

	// Second delete of the same (now absent) id must not error.
	if _, err := cs.Save("clients", "del", payload, "", "admin", "example.com"); err != nil {
		t.Fatalf("deleting an already-gone client should be a no-op success, got: %v", err)
	}
}

// delbulk with a mix of present and already-gone ids deletes the present ones
// and ignores the missing ones instead of failing the whole batch.
func TestConfigSaveClientsDelBulkSkipsMissingIds(t *testing.T) {
	initSettingTestDB(t)

	present := newTestClient(t, "present")
	cs := NewConfigServiceWithRuntime(NewRuntimeWithCoreProvider(nil))

	// 999999 is an id that was never created.
	payload, err := json.Marshal([]uint{present.Id, 999999})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := cs.Save("clients", "delbulk", payload, "", "admin", "example.com"); err != nil {
		t.Fatalf("delbulk with a missing id should still succeed: %v", err)
	}
	if clientExists(t, present.Id) {
		t.Fatal("present client should be deleted by delbulk")
	}
}
