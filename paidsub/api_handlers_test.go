package paidsub

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x/database/model"

	"github.com/gin-gonic/gin"
)

func newTestHandlers() *apiHandlers {
	return &apiHandlers{svc: NewService(), tariffs: NewTariffService(), payments: NewPaymentService(), deps: Deps{}}
}

// doHandler drives one admin handler with a JSON body and decodes the apiMsg
// envelope. A panicking handler fails the test (the goroutine is the test's).
func doHandler(t *testing.T, handler gin.HandlerFunc, body string) apiMsg {
	t.Helper()
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/paidsub/x", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	handler(c)
	var m apiMsg
	if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil {
		t.Fatalf("response is not an apiMsg envelope: %v (body %q)", err, rec.Body.String())
	}
	return m
}

func TestRefundHandlerValidationAndSuccess(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}
	h := newTestHandlers()

	if m := doHandler(t, h.refund, `{bad`); m.Success || m.Msg != "invalid request" {
		t.Fatalf("malformed JSON: %+v", m)
	}
	if m := doHandler(t, h.refund, `{"orderId":0}`); m.Success || m.Msg != "orderId is required" {
		t.Fatalf("missing orderId: %+v", m)
	}
	if m := doHandler(t, h.refund, `{"orderId":999}`); m.Success {
		t.Fatalf("non-existent order should fail: %+v", m)
	}

	order := PaymentOrder{ClientId: 1, TariffId: 1, Provider: "yookassa", Amount: 10000, Currency: "RUB", Status: StatusPaid, IdempotencyKey: "h-refund"}
	db.Create(&order)
	if m := doHandler(t, h.refund, fmt.Sprintf(`{"orderId":%d,"revoke":false}`, order.Id)); !m.Success {
		t.Fatalf("valid refund failed: %+v", m)
	}
}

func TestSetBindingHandlerValidation(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}
	h := newTestHandlers()

	if m := doHandler(t, h.setBinding, `{bad`); m.Success || m.Msg != "invalid request" {
		t.Fatalf("malformed JSON: %+v", m)
	}
	if m := doHandler(t, h.setBinding, `{"clientId":0}`); m.Success || m.Msg != "clientId is required" {
		t.Fatalf("missing clientId: %+v", m)
	}
	if m := doHandler(t, h.setBinding, `{"clientId":999,"tgUserId":5}`); m.Success || m.Msg != "client not found" {
		t.Fatalf("unknown client: %+v", m)
	}

	client := model.Client{Name: "bind", Inbounds: json.RawMessage("[]")}
	db.Create(&client)
	if m := doHandler(t, h.setBinding, fmt.Sprintf(`{"clientId":%d,"tgUserId":5}`, client.Id)); !m.Success {
		t.Fatalf("valid bind failed: %+v", m)
	}
	if m := doHandler(t, h.setBinding, fmt.Sprintf(`{"clientId":%d,"tgUserId":0}`, client.Id)); !m.Success {
		t.Fatalf("unbind failed: %+v", m)
	}
}

func TestSaveTariffHandlerValidation(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}
	h := newTestHandlers()

	if m := doHandler(t, h.saveTariff, `{bad`); m.Success || m.Msg != "invalid request" {
		t.Fatalf("malformed JSON: %+v", m)
	}
	if m := doHandler(t, h.saveTariff, `{"action":"bogus","data":{}}`); m.Success || m.Msg != "invalid action" {
		t.Fatalf("invalid action: %+v", m)
	}
	if m := doHandler(t, h.saveTariff, `{"action":"new","data":"notatariff"}`); m.Success {
		t.Fatalf("malformed tariff data should fail: %+v", m)
	}
	if m := doHandler(t, h.saveTariff, `{"action":"edit","data":{"name":"x"}}`); m.Success {
		t.Fatalf("edit without id should fail: %+v", m)
	}
	if m := doHandler(t, h.saveTariff, `{"action":"new","data":{"name":"Month","enabled":true}}`); !m.Success {
		t.Fatalf("valid new tariff failed: %+v", m)
	}
}
