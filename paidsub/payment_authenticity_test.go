package paidsub

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"path"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x/database/model"
)

// recordingTransport is a fake Telegram Bot API transport. It records each call
// (method = last URL path segment, plus the decoded JSON body) and returns a
// canned ok response, so handlers that send messages / answer pre-checkouts can
// be tested without network access.
type recordingTransport struct {
	calls []recordedCall
}

type recordedCall struct {
	method string
	body   map[string]any
}

func (rt *recordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var body map[string]any
	if req.Body != nil {
		raw, _ := io.ReadAll(req.Body)
		_ = json.Unmarshal(raw, &body)
	}
	rt.calls = append(rt.calls, recordedCall{method: path.Base(req.URL.Path), body: body})
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{"ok":true,"result":{}}`)),
		Header:     make(http.Header),
	}, nil
}

func (rt *recordingTransport) lastCall(method string) (recordedCall, bool) {
	for i := len(rt.calls) - 1; i >= 0; i-- {
		if rt.calls[i].method == method {
			return rt.calls[i], true
		}
	}
	return recordedCall{}, false
}

func newTestBot(rt http.RoundTripper) *Bot {
	b := newBot()
	b.token = "test-token"
	b.client = &http.Client{Transport: rt}
	return b
}

func TestHandlePreCheckoutApprovesValidOrder(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}
	db.Create(&PaymentOrder{ClientId: 1, TariffId: 1, Provider: string(ProviderStars), Amount: 100, Currency: "XTR", Status: StatusPending, TelegramUserId: 7, IdempotencyKey: "pc-ok"})

	rt := &recordingTransport{}
	b := newTestBot(rt)
	b.handlePreCheckout(context.Background(), &tgPreCheckoutQuery{ID: "q1", From: tgUser{ID: 7}, Currency: "XTR", TotalAmount: 100, InvoicePayload: "pc-ok"})

	call, ok := rt.lastCall("answerPreCheckoutQuery")
	if !ok {
		t.Fatal("expected answerPreCheckoutQuery to be called")
	}
	if call.body["ok"] != true {
		t.Fatalf("expected pre-checkout ok=true, got %v", call.body["ok"])
	}
}

// TestHandlePreCheckoutRejectsInvalid asserts the pre-checkout gate refuses any
// query whose amount/currency does not match the trusted order, whose order is
// no longer pending, or whose payload is unknown (T1).
func TestHandlePreCheckoutRejectsInvalid(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}
	db.Create(&PaymentOrder{ClientId: 1, TariffId: 1, Provider: string(ProviderStars), Amount: 100, Currency: "XTR", Status: StatusPending, TelegramUserId: 7, IdempotencyKey: "pc"})
	db.Create(&PaymentOrder{ClientId: 1, TariffId: 1, Provider: string(ProviderStars), Amount: 100, Currency: "XTR", Status: StatusPaid, TelegramUserId: 7, IdempotencyKey: "pc-paid"})

	cases := []struct {
		name string
		q    tgPreCheckoutQuery
	}{
		{"wrong amount", tgPreCheckoutQuery{ID: "q", From: tgUser{ID: 7}, Currency: "XTR", TotalAmount: 999, InvoicePayload: "pc"}},
		{"wrong currency", tgPreCheckoutQuery{ID: "q", From: tgUser{ID: 7}, Currency: "USD", TotalAmount: 100, InvoicePayload: "pc"}},
		{"non-pending order", tgPreCheckoutQuery{ID: "q", From: tgUser{ID: 7}, Currency: "XTR", TotalAmount: 100, InvoicePayload: "pc-paid"}},
		{"unknown payload", tgPreCheckoutQuery{ID: "q", From: tgUser{ID: 7}, Currency: "XTR", TotalAmount: 100, InvoicePayload: "nope"}},
		{"wrong telegram user", tgPreCheckoutQuery{ID: "q", From: tgUser{ID: 999}, Currency: "XTR", TotalAmount: 100, InvoicePayload: "pc"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rt := &recordingTransport{}
			b := newTestBot(rt)
			q := tc.q
			b.handlePreCheckout(context.Background(), &q)
			call, ok := rt.lastCall("answerPreCheckoutQuery")
			if !ok {
				t.Fatal("expected answerPreCheckoutQuery to be called")
			}
			if call.body["ok"] != false {
				t.Fatalf("expected pre-checkout ok=false for %s, got %v", tc.name, call.body["ok"])
			}
		})
	}
}

func TestHandleSuccessfulPaymentAppliesRenewalOnMatch(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}
	client := model.Client{Enable: false, Name: "tgpay", Inbounds: json.RawMessage("[]"), Expiry: 0}
	db.Create(&client)
	tariff := Tariff{Name: "Stars", StarsAmount: 100, Currency: "XTR", AddDays: 30, Enabled: true}
	db.Create(&tariff)
	order := PaymentOrder{ClientId: client.Id, TariffId: tariff.Id, Provider: string(ProviderStars), Amount: 100, Currency: "XTR", Status: StatusPending, TelegramUserId: 7, IdempotencyKey: "sp-ok"}
	db.Create(&order)

	rt := &recordingTransport{}
	b := newTestBot(rt)
	b.handleSuccessfulPayment(context.Background(), &tgMessage{
		From: &tgUser{ID: 7}, Chat: tgChat{ID: 7},
		SuccessfulPayment: &tgSuccessfulPayment{Currency: "XTR", TotalAmount: 100, InvoicePayload: "sp-ok", TelegramPaymentChargeID: "ch1"},
	})

	var o PaymentOrder
	db.Where("id = ?", order.Id).First(&o)
	if o.Status != StatusPaid {
		t.Fatalf("order status = %s, want paid", o.Status)
	}
	if o.ProviderChargeID != "tg:ch1" {
		t.Errorf("charge id = %q, want tg:ch1", o.ProviderChargeID)
	}
	var c model.Client
	db.Where("id = ?", client.Id).First(&c)
	if !c.Enable {
		t.Error("client should be renewed (enabled) after a matching payment")
	}
}

// TestHandleSuccessfulPaymentRefusesMismatch is the core payment-integrity
// regression (T1): an amount or currency that disagrees with the trusted order
// must mark the order failed and NEVER renew the client.
func TestHandleSuccessfulPaymentRefusesMismatch(t *testing.T) {
	cases := []struct {
		name string
		sp   tgSuccessfulPayment
	}{
		{"amount mismatch", tgSuccessfulPayment{Currency: "XTR", TotalAmount: 50, InvoicePayload: "sp", TelegramPaymentChargeID: "x"}},
		{"currency mismatch", tgSuccessfulPayment{Currency: "USD", TotalAmount: 100, InvoicePayload: "sp", TelegramPaymentChargeID: "x"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := openTestDB(t)
			if err := EnsureSchema(db); err != nil {
				t.Fatalf("EnsureSchema: %v", err)
			}
			client := model.Client{Enable: false, Name: "tgmm", Inbounds: json.RawMessage("[]"), Expiry: 0}
			db.Create(&client)
			order := PaymentOrder{ClientId: client.Id, TariffId: 1, Provider: string(ProviderStars), Amount: 100, Currency: "XTR", Status: StatusPending, TelegramUserId: 7, IdempotencyKey: "sp"}
			db.Create(&order)

			rt := &recordingTransport{}
			b := newTestBot(rt)
			sp := tc.sp
			b.handleSuccessfulPayment(context.Background(), &tgMessage{From: &tgUser{ID: 7}, Chat: tgChat{ID: 7}, SuccessfulPayment: &sp})

			var o PaymentOrder
			db.Where("id = ?", order.Id).First(&o)
			if o.Status != StatusFailed {
				t.Fatalf("order status = %s, want failed on %s", o.Status, tc.name)
			}
			var c model.Client
			db.Where("id = ?", client.Id).First(&c)
			if c.Enable {
				t.Fatalf("client must NOT be renewed on %s", tc.name)
			}
		})
	}
}

// TestHandleSuccessfulPaymentRefusesWrongTelegramUser covers A4: even with a
// matching amount/currency, a payment whose payer differs from the order's
// Telegram user must mark the order failed and never renew.
func TestHandleSuccessfulPaymentRefusesWrongTelegramUser(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}
	client := model.Client{Enable: false, Name: "tgwrong", Inbounds: json.RawMessage("[]"), Expiry: 0}
	db.Create(&client)
	order := PaymentOrder{ClientId: client.Id, TariffId: 1, Provider: string(ProviderStars), Amount: 100, Currency: "XTR", Status: StatusPending, TelegramUserId: 7, IdempotencyKey: "spw"}
	db.Create(&order)

	rt := &recordingTransport{}
	b := newTestBot(rt)
	b.handleSuccessfulPayment(context.Background(), &tgMessage{
		From: &tgUser{ID: 999}, Chat: tgChat{ID: 999},
		SuccessfulPayment: &tgSuccessfulPayment{Currency: "XTR", TotalAmount: 100, InvoicePayload: "spw", TelegramPaymentChargeID: "x"},
	})

	var o PaymentOrder
	db.Where("id = ?", order.Id).First(&o)
	if o.Status != StatusFailed {
		t.Fatalf("order status = %s, want failed for wrong tg user", o.Status)
	}
	var c model.Client
	db.Where("id = ?", client.Id).First(&c)
	if c.Enable {
		t.Fatal("client must NOT be renewed for a payment from the wrong tg user")
	}
}

func TestHandleSuccessfulPaymentUnknownPayloadIsNoop(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}
	order := PaymentOrder{ClientId: 1, TariffId: 1, Provider: string(ProviderStars), Amount: 100, Currency: "XTR", Status: StatusPending, TelegramUserId: 7, IdempotencyKey: "real"}
	db.Create(&order)

	rt := &recordingTransport{}
	b := newTestBot(rt)
	b.handleSuccessfulPayment(context.Background(), &tgMessage{
		From: &tgUser{ID: 7}, Chat: tgChat{ID: 7},
		SuccessfulPayment: &tgSuccessfulPayment{Currency: "XTR", TotalAmount: 100, InvoicePayload: "ghost", TelegramPaymentChargeID: "x"},
	})

	// The known order must be untouched (no order matched the unknown payload).
	var o PaymentOrder
	db.Where("id = ?", order.Id).First(&o)
	if o.Status != StatusPending {
		t.Fatalf("unrelated order changed to %s on unknown payload", o.Status)
	}
}
