package paidsub

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x/database/model"
)

// TestListOrdersHandler closes the T3 gap for listOrders: it must succeed on an
// empty DB (no panic on the zero-row LEFT JOIN) and join the client name through.
func TestListOrdersHandler(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}
	h := newTestHandlers()

	if m := doHandler(t, h.listOrders, ""); !m.Success {
		t.Fatalf("listOrders on empty DB should succeed: %+v", m)
	}

	client := model.Client{Name: "ordered-客户", Inbounds: json.RawMessage("[]")}
	if err := db.Create(&client).Error; err != nil {
		t.Fatal(err)
	}
	order := PaymentOrder{ClientId: client.Id, TariffId: 1, Provider: "stars", Amount: 100, Currency: "XTR", Status: StatusPaid, IdempotencyKey: "h-list"}
	if err := db.Create(&order).Error; err != nil {
		t.Fatal(err)
	}

	m := doHandler(t, h.listOrders, "")
	if !m.Success {
		t.Fatalf("listOrders should succeed: %+v", m)
	}
	raw, _ := json.Marshal(m.Obj)
	if !strings.Contains(string(raw), "ordered-客户") {
		t.Fatalf("listOrders should join the client name through; obj=%s", raw)
	}
}

// TestBroadcastHandlerValidation closes the T3 gap for broadcast: malformed and
// empty/whitespace messages are rejected with the right envelope, and a valid
// message with no configured bot fails cleanly (no panic, proper apiMsg).
func TestBroadcastHandlerValidation(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}
	h := newTestHandlers()

	if m := doHandler(t, h.broadcast, `{bad`); m.Success || m.Msg != "invalid request" {
		t.Fatalf("malformed JSON: %+v", m)
	}
	if m := doHandler(t, h.broadcast, `{"text":""}`); m.Success || m.Msg != "message is empty" {
		t.Fatalf("empty text: %+v", m)
	}
	if m := doHandler(t, h.broadcast, `{"text":"   "}`); m.Success || m.Msg != "message is empty" {
		t.Fatalf("whitespace text: %+v", m)
	}
	// Valid text but no configured bot token: must fail cleanly via respFail.
	m := doHandler(t, h.broadcast, `{"text":"hello everyone"}`)
	if m.Success {
		t.Fatalf("broadcast without a configured bot must not report success: %+v", m)
	}
	if !strings.Contains(m.Msg, "bot token not configured") {
		t.Fatalf("expected bot-token error, got %+v", m)
	}
}
