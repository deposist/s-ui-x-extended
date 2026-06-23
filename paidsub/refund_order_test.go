package paidsub

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/deposist/s-ui-x-extended/database/model"
)

func TestRefundOrderNonPaidIsNotRefundable(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}
	order := PaymentOrder{ClientId: 1, TariffId: 1, Provider: "yookassa", Amount: 10000, Currency: "RUB", Status: StatusPending, TelegramUserId: 7, IdempotencyKey: "np"}
	db.Create(&order)

	ps := NewPaymentService()
	status, err := ps.RefundOrder(context.Background(), order.Id, true)
	if !errors.Is(err, errRefundNotApplicable) {
		t.Fatalf("RefundOrder on pending = (%q,%v), want errRefundNotApplicable", status, err)
	}
	var o PaymentOrder
	db.Where("id = ?", order.Id).First(&o)
	if o.Status != StatusPending {
		t.Errorf("pending order must be unchanged, got %s", o.Status)
	}
}

func TestRefundOrderNonStarsMarksManualAndRevokes(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}
	now := time.Now().Unix()
	client := model.Client{Enable: true, Name: "ref", Inbounds: json.RawMessage("[]"), Volume: 5 << 30, Expiry: now + 40*86400}
	db.Create(&client)
	tariff := Tariff{Name: "M", Price: 10000, Currency: "RUB", AddDays: 30, AddTrafficBytes: 1 << 30, Enabled: true}
	db.Create(&tariff)
	order := PaymentOrder{ClientId: client.Id, TariffId: tariff.Id, Provider: "yookassa", Amount: 10000, Currency: "RUB", Status: StatusPaid, TelegramUserId: 7, IdempotencyKey: "man"}
	db.Create(&order)

	ps := NewPaymentService()
	status, err := ps.RefundOrder(context.Background(), order.Id, true)
	if err != nil {
		t.Fatalf("RefundOrder: %v", err)
	}
	if status != "refunded_manual" {
		t.Fatalf("status = %q, want refunded_manual", status)
	}
	var o PaymentOrder
	db.Where("id = ?", order.Id).First(&o)
	if o.Status != StatusRefunded {
		t.Errorf("order status = %s, want refunded", o.Status)
	}
	var c model.Client
	db.Where("id = ?", client.Id).First(&c)
	if c.Volume != (5<<30)-(1<<30) {
		t.Errorf("volume not rolled back: %d", c.Volume)
	}
	if !c.Enable {
		t.Error("refund must never disable the client")
	}
}

func TestRefundOrderNonStarsNoRevokeKeepsClient(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}
	now := time.Now().Unix()
	client := model.Client{Enable: true, Name: "ref2", Inbounds: json.RawMessage("[]"), Volume: 2 << 30, Expiry: now + 10*86400}
	db.Create(&client)
	tariff := Tariff{Name: "M", Price: 10000, Currency: "RUB", AddDays: 30, AddTrafficBytes: 1 << 30, Enabled: true}
	db.Create(&tariff)
	order := PaymentOrder{ClientId: client.Id, TariffId: tariff.Id, Provider: "stripe", Amount: 10000, Currency: "RUB", Status: StatusPaid, TelegramUserId: 7, IdempotencyKey: "man2"}
	db.Create(&order)

	ps := NewPaymentService()
	status, err := ps.RefundOrder(context.Background(), order.Id, false)
	if err != nil {
		t.Fatalf("RefundOrder: %v", err)
	}
	if status != "refunded_manual" {
		t.Fatalf("status = %q, want refunded_manual", status)
	}
	var c model.Client
	db.Where("id = ?", client.Id).First(&c)
	if c.Volume != 2<<30 || c.Expiry != now+10*86400 {
		t.Errorf("client changed despite revoke=false: volume=%d expiry=%d", c.Volume, c.Expiry)
	}
}

func TestRefundOrderDoubleRefundIsNotApplicable(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}
	order := PaymentOrder{ClientId: 1, TariffId: 1, Provider: "yookassa", Amount: 10000, Currency: "RUB", Status: StatusPaid, TelegramUserId: 7, IdempotencyKey: "dbl"}
	db.Create(&order)

	ps := NewPaymentService()
	if _, err := ps.RefundOrder(context.Background(), order.Id, false); err != nil {
		t.Fatalf("first refund: %v", err)
	}
	// The order is now refunded; a second refund must be rejected by the
	// status==paid gate, not double-processed.
	status, err := ps.RefundOrder(context.Background(), order.Id, false)
	if !errors.Is(err, errRefundNotApplicable) {
		t.Fatalf("second refund = (%q,%v), want errRefundNotApplicable", status, err)
	}
}

// TestRefundOrderRejectsNonPositiveAmount pins H-14: a paid order with a
// non-positive amount is a corrupted row; the refund path must refuse it
// (defense in depth, since CreateOrder rejects zero-priced orders up front).
func TestRefundOrderRejectsNonPositiveAmount(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}
	order := PaymentOrder{ClientId: 1, TariffId: 1, Provider: "yookassa", Amount: 0, Currency: "RUB", Status: StatusPaid, TelegramUserId: 7, IdempotencyKey: "zero"}
	db.Create(&order)

	ps := NewPaymentService()
	status, err := ps.RefundOrder(context.Background(), order.Id, true)
	if !errors.Is(err, errRefundNotApplicable) {
		t.Fatalf("RefundOrder on zero-amount = (%q,%v), want errRefundNotApplicable", status, err)
	}
	var o PaymentOrder
	db.Where("id = ?", order.Id).First(&o)
	if o.Status != StatusPaid {
		t.Errorf("corrupted order must be untouched, got %s", o.Status)
	}
}

// TestRefundOrderStarsRequiresBotToken asserts the Stars-refund branch refuses
// to mark an order refunded when the bot is not configured (newSenderBot fails),
// so the money path is never skipped silently. The order stays paid.
func TestRefundOrderStarsRequiresBotToken(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}
	order := PaymentOrder{ClientId: 1, TariffId: 1, Provider: string(ProviderStars), Amount: 100, Currency: "XTR", Status: StatusPaid, TelegramUserId: 7, ProviderChargeID: "tg:charge", IdempotencyKey: "st"}
	db.Create(&order)

	ps := NewPaymentService()
	status, err := ps.RefundOrder(context.Background(), order.Id, false)
	if err == nil {
		t.Fatalf("expected Stars refund to fail without a bot token, got status %q", status)
	}
	var o PaymentOrder
	db.Where("id = ?", order.Id).First(&o)
	if o.Status != StatusPaid {
		t.Errorf("order must remain paid when Stars refund fails, got %s", o.Status)
	}
}

// TestRefundRestoresUsageCounters pins M-2: a traffic-refilling renewal resets
// up/down and folds them into total_up/total_down; a refund with revoke must
// restore the pre-purchase accounting state symmetrically (volume AND the usage
// counters), using the granted_up/granted_down snapshot taken at apply time.
func TestRefundRestoresUsageCounters(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}
	client := model.Client{Enable: true, Name: "ctr", Inbounds: json.RawMessage("[]"),
		Volume: 5 << 30, Up: 100, Down: 200, TotalUp: 1000, TotalDown: 2000}
	db.Create(&client)
	tariff := Tariff{Name: "M", Price: 10000, Currency: "RUB", AddTrafficBytes: 1 << 30, Enabled: true}
	db.Create(&tariff)
	order := PaymentOrder{ClientId: client.Id, TariffId: tariff.Id, Provider: "yookassa", Amount: 10000, Currency: "RUB", Status: StatusPending, TelegramUserId: 7, IdempotencyKey: "ctr"}
	db.Create(&order)

	ps := NewPaymentService()
	if applied, _, err := ps.ApplyPaidOrder(order.Id, "ch:1", nil); err != nil || !applied {
		t.Fatalf("ApplyPaidOrder = (%v,%v), want applied", applied, err)
	}
	var afterApply model.Client
	db.Where("id = ?", client.Id).First(&afterApply)
	if afterApply.Up != 0 || afterApply.Down != 0 {
		t.Fatalf("apply must reset up/down, got up=%d down=%d", afterApply.Up, afterApply.Down)
	}
	if afterApply.TotalUp != 1100 || afterApply.TotalDown != 2200 {
		t.Fatalf("apply must fold usage into totals, got total_up=%d total_down=%d", afterApply.TotalUp, afterApply.TotalDown)
	}
	if afterApply.Volume != (5<<30)+(1<<30) {
		t.Fatalf("apply must refill volume, got %d", afterApply.Volume)
	}
	var snap PaymentOrder
	db.Where("id = ?", order.Id).First(&snap)
	if snap.GrantedUp != 100 || snap.GrantedDown != 200 {
		t.Fatalf("apply must snapshot granted up/down onto the order, got %d/%d", snap.GrantedUp, snap.GrantedDown)
	}

	if _, err := ps.RefundOrder(context.Background(), order.Id, true); err != nil {
		t.Fatalf("RefundOrder: %v", err)
	}
	var afterRefund model.Client
	db.Where("id = ?", client.Id).First(&afterRefund)
	if afterRefund.Up != 100 || afterRefund.Down != 200 {
		t.Errorf("refund must restore up/down, got up=%d down=%d", afterRefund.Up, afterRefund.Down)
	}
	if afterRefund.TotalUp != 1000 || afterRefund.TotalDown != 2000 {
		t.Errorf("refund must restore totals, got total_up=%d total_down=%d", afterRefund.TotalUp, afterRefund.TotalDown)
	}
	if afterRefund.Volume != 5<<30 {
		t.Errorf("refund must roll back volume, got %d", afterRefund.Volume)
	}
}
