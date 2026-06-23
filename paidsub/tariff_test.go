package paidsub

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestTariffCRUD(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}
	ts := NewTariffService()

	if err := ts.Save("new", json.RawMessage(`{"name":"Month","price":10000,"currency":"RUB","addDays":30,"addTrafficBytes":0,"starsAmount":100,"enabled":true}`)); err != nil {
		t.Fatalf("new: %v", err)
	}
	all, err := ts.GetAll()
	if err != nil || len(all) != 1 {
		t.Fatalf("GetAll after new: len=%d err=%v", len(all), err)
	}
	id := all[0].Id
	if all[0].Name != "Month" || all[0].Price != 10000 || all[0].AddDays != 30 || all[0].StarsAmount != 100 {
		t.Fatalf("unexpected tariff: %+v", all[0])
	}
	if all[0].CreatedAt == 0 {
		t.Fatal("CreatedAt not set")
	}

	// Edit: raise price and disable. Zero-valued enabled=false must persist.
	if err := ts.Save("edit", json.RawMessage(fmt.Sprintf(`{"id":%d,"name":"Month","price":15000,"currency":"RUB","addDays":30,"enabled":false}`, id))); err != nil {
		t.Fatalf("edit: %v", err)
	}
	got, err := ts.Get(id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Price != 15000 || got.Enabled {
		t.Fatalf("edit not applied: %+v", got)
	}
	if en, _ := ts.GetEnabled(); len(en) != 0 {
		t.Fatalf("disabled tariff should not appear in GetEnabled")
	}

	if err := ts.Save("del", json.RawMessage(fmt.Sprintf(`%d`, id))); err != nil {
		t.Fatalf("del: %v", err)
	}
	if all, _ := ts.GetAll(); len(all) != 0 {
		t.Fatalf("tariff not deleted")
	}
}

// TestTariffRejectsNegativeValues pins F-14: negative money/duration/traffic/sort
// must be refused on create and edit (not silently stored and ignored at apply).
func TestTariffRejectsNegativeValues(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}
	ts := NewTariffService()

	if err := ts.Save("new", json.RawMessage(`{"name":"Bad","price":10000,"addDays":-5,"enabled":true}`)); err == nil {
		t.Fatal("new with negative addDays must be rejected")
	}
	if all, _ := ts.GetAll(); len(all) != 0 {
		t.Fatalf("rejected tariff must not be persisted, got %d", len(all))
	}

	// A valid tariff, then an edit that introduces a negative value must fail.
	if err := ts.Save("new", json.RawMessage(`{"name":"OK","price":10000,"addDays":30,"enabled":true}`)); err != nil {
		t.Fatalf("valid new: %v", err)
	}
	all, _ := ts.GetAll()
	if len(all) != 1 {
		t.Fatalf("expected 1 tariff, got %d", len(all))
	}
	if err := ts.Save("edit", json.RawMessage(fmt.Sprintf(`{"id":%d,"name":"OK","price":-1,"addDays":30,"enabled":true}`, all[0].Id))); err == nil {
		t.Fatal("edit with negative price must be rejected")
	}
	got, _ := ts.Get(all[0].Id)
	if got.Price != 10000 {
		t.Fatalf("rejected edit must not change price, got %d", got.Price)
	}
}
