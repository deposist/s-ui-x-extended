package paidsub

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/deposist/s-ui-x-extended/service"
)

type paymentAWGHook struct {
	resumed   []uint
	suspended [][]uint
}

func (h *paymentAWGHook) ResumeClient(_ context.Context, id uint) error {
	h.resumed = append(h.resumed, id)
	return nil
}

func (h *paymentAWGHook) SuspendClients(_ context.Context, ids []uint) error {
	h.suspended = append(h.suspended, append([]uint(nil), ids...))
	return nil
}

func TestNewPaymentOrderSnapshotsAWGEntitlement(t *testing.T) {
	client := &model.Client{Id: 7}
	tariff := &Tariff{Id: 9, AddDays: 30, AddTrafficBytes: 1 << 30, MaxAWGDevices: 4}
	order := newPaymentOrder(client, tariff, ProviderStars, 11, 100, "XTR", 1_000, 15)

	tariff.AddDays = 1
	tariff.AddTrafficBytes = 2
	tariff.MaxAWGDevices = 1
	if order.GrantedDays != 30 || order.GrantedTrafficBytes != 1<<30 || order.GrantedAWGDevices != 4 {
		t.Fatalf("grant snapshot = days %d, traffic %d, AWG %d", order.GrantedDays, order.GrantedTrafficBytes, order.GrantedAWGDevices)
	}
	if order.ClientId != 7 || order.TariffId != 9 || order.ExpiresAt != 1_900 || order.SnapshotVersion != paymentOrderSnapshotVersion {
		t.Fatalf("unexpected order snapshot: %+v", order)
	}
}

func TestApplyPaidOrderUsesSnapshotAfterTariffChanges(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSchema(db); err != nil {
		t.Fatal(err)
	}
	client := model.Client{Enable: false, Name: "snapshot-change", Inbounds: json.RawMessage("[]")}
	if err := db.Create(&client).Error; err != nil {
		t.Fatal(err)
	}
	tariff := Tariff{Name: "Original", Price: 100, Currency: "RUB", AddDays: 30, AddTrafficBytes: 1 << 30, MaxAWGDevices: 4, Enabled: true}
	if err := db.Create(&tariff).Error; err != nil {
		t.Fatal(err)
	}
	order := newPaymentOrder(&client, &tariff, ProviderYooKassa, 42, tariff.Price, tariff.Currency, time.Now().Unix(), 15)
	if err := db.Create(order).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&tariff).Updates(map[string]any{"add_days": 1, "add_traffic_bytes": 2, "max_awg_devices": 1}).Error; err != nil {
		t.Fatal(err)
	}

	before := time.Now().Unix()
	applied, _, err := NewPaymentService().ApplyPaidOrder(order.Id, "charge-snapshot", nil)
	if err != nil || !applied {
		t.Fatalf("ApplyPaidOrder = applied %v, err %v", applied, err)
	}
	var got model.Client
	if err := db.First(&got, client.Id).Error; err != nil {
		t.Fatal(err)
	}
	if got.Volume != 1<<30 || got.Expiry < before+30*86400 || got.Expiry > before+30*86400+5 {
		t.Fatalf("client received changed tariff: volume=%d expiry=%d", got.Volume, got.Expiry)
	}
	if limit, err := NewTariffService().EffectiveAWGDeviceLimit(client.Id, 2); err != nil || limit != 4 {
		t.Fatalf("AWG limit = %d, %v; want snapshotted 4", limit, err)
	}
}

func TestApplyPaidOrderUsesSnapshotAfterTariffDeletion(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSchema(db); err != nil {
		t.Fatal(err)
	}
	client := model.Client{Enable: false, Name: "snapshot-delete", Inbounds: json.RawMessage("[]")}
	if err := db.Create(&client).Error; err != nil {
		t.Fatal(err)
	}
	tariff := Tariff{Name: "Deleted", Price: 100, Currency: "RUB", AddDays: 7, AddTrafficBytes: 1024, Enabled: true}
	if err := db.Create(&tariff).Error; err != nil {
		t.Fatal(err)
	}
	order := newPaymentOrder(&client, &tariff, ProviderYooKassa, 42, tariff.Price, tariff.Currency, time.Now().Unix(), 15)
	if err := db.Create(order).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Delete(&tariff).Error; err != nil {
		t.Fatal(err)
	}

	applied, _, err := NewPaymentService().ApplyPaidOrder(order.Id, "charge-deleted", nil)
	if err != nil || !applied {
		t.Fatalf("ApplyPaidOrder after tariff deletion = applied %v, err %v", applied, err)
	}
	var got model.Client
	if err := db.First(&got, client.Id).Error; err != nil {
		t.Fatal(err)
	}
	if got.Volume != 1024 {
		t.Fatalf("volume = %d; want snapshotted 1024", got.Volume)
	}
}

func TestApplyPaidOrderIdempotentRenewal(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}

	// Disabled, expired-by-default client with usage, no inbounds (no restart).
	client := model.Client{
		Enable:    false,
		Name:      "tg42",
		Inbounds:  json.RawMessage("[]"),
		Volume:    0,
		Expiry:    0,
		Up:        100,
		Down:      200,
		TotalUp:   0,
		TotalDown: 0,
	}
	if err := db.Create(&client).Error; err != nil {
		t.Fatalf("create client: %v", err)
	}

	tariff := Tariff{Name: "Month", Price: 10000, Currency: "RUB", AddDays: 30, AddTrafficBytes: 1 << 30, Enabled: true}
	if err := db.Create(&tariff).Error; err != nil {
		t.Fatalf("create tariff: %v", err)
	}

	order := PaymentOrder{
		ClientId: client.Id, TariffId: tariff.Id, Provider: "yookassa",
		Amount: 10000, Currency: "RUB", Status: StatusPending,
		TelegramUserId: 42, IdempotencyKey: "key-1", CreatedAt: time.Now().Unix(),
		GrantedDays: tariff.AddDays, GrantedTrafficBytes: tariff.AddTrafficBytes, SnapshotVersion: paymentOrderSnapshotVersion,
	}
	if err := db.Create(&order).Error; err != nil {
		t.Fatalf("create order: %v", err)
	}

	ps := NewPaymentService()
	applied, tgID, err := ps.ApplyPaidOrder(order.Id, "charge-1", nil)
	if err != nil {
		t.Fatalf("ApplyPaidOrder: %v", err)
	}
	if !applied {
		t.Fatal("expected first apply to succeed")
	}
	if tgID != 42 {
		t.Fatalf("expected tgID 42, got %d", tgID)
	}

	var got model.Client
	db.Where("id = ?", client.Id).First(&got)
	if !got.Enable {
		t.Error("client should be re-enabled")
	}
	if got.Volume != 1<<30 {
		t.Errorf("volume = %d, want %d", got.Volume, int64(1<<30))
	}
	if got.Up != 0 || got.Down != 0 {
		t.Errorf("up/down should reset, got up=%d down=%d", got.Up, got.Down)
	}
	if got.TotalUp != 100 || got.TotalDown != 200 {
		t.Errorf("totals = %d/%d, want 100/200", got.TotalUp, got.TotalDown)
	}
	now := time.Now().Unix()
	if got.Expiry < now+29*86400 || got.Expiry > now+31*86400 {
		t.Errorf("expiry not extended ~30d: %d (now %d)", got.Expiry, now)
	}

	var paidOrder PaymentOrder
	db.Where("id = ?", order.Id).First(&paidOrder)
	if paidOrder.Status != StatusPaid || paidOrder.ProviderChargeID != "charge-1" {
		t.Errorf("order not marked paid: %+v", paidOrder)
	}

	// Second apply must be an idempotent no-op (no double renewal).
	applied2, _, err := ps.ApplyPaidOrder(order.Id, "charge-1", nil)
	if err != nil {
		t.Fatalf("second ApplyPaidOrder: %v", err)
	}
	if applied2 {
		t.Fatal("second apply must be a no-op")
	}
	var got2 model.Client
	db.Where("id = ?", client.Id).First(&got2)
	if got2.Volume != 1<<30 {
		t.Errorf("volume changed on replay: %d", got2.Volume)
	}
	if got2.Expiry != got.Expiry {
		t.Errorf("expiry changed on replay: %d != %d", got2.Expiry, got.Expiry)
	}
}

func TestApplyPaidOrderNotifiesAWGOnlyAfterSuccessfulCommit(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSchema(db); err != nil {
		t.Fatal(err)
	}
	hook := &paymentAWGHook{}
	runtime := service.NewRuntimeWithCoreProvider(nil)
	runtime.SetAWGClientStateHook(hook)
	restore := service.ReplaceDefaultRuntimeForTest(runtime)
	t.Cleanup(restore)
	client := model.Client{Enable: false, Name: "renew-awg", Inbounds: json.RawMessage(`[]`)}
	if err := db.Create(&client).Error; err != nil {
		t.Fatal(err)
	}
	tariff := Tariff{Name: "Month", Price: 100, Currency: "RUB", AddDays: 30, Enabled: true}
	if err := db.Create(&tariff).Error; err != nil {
		t.Fatal(err)
	}
	order := PaymentOrder{ClientId: client.Id, TariffId: tariff.Id, Provider: "manual", Amount: 100, Currency: "RUB", Status: StatusPending, IdempotencyKey: "awg-hook", GrantedDays: tariff.AddDays, SnapshotVersion: paymentOrderSnapshotVersion}
	if err := db.Create(&order).Error; err != nil {
		t.Fatal(err)
	}
	if applied, _, err := NewPaymentService().ApplyPaidOrder(order.Id, "charge", nil); err != nil || !applied {
		t.Fatalf("applied=%v err=%v", applied, err)
	}
	if len(hook.resumed) != 1 || hook.resumed[0] != client.Id {
		t.Fatalf("resumed=%v", hook.resumed)
	}
	if applied, _, err := NewPaymentService().ApplyPaidOrder(order.Id, "charge", nil); err != nil || applied {
		t.Fatalf("replay applied=%v err=%v", applied, err)
	}
	if len(hook.resumed) != 1 {
		t.Fatalf("replay notified AWG: %v", hook.resumed)
	}
}

func TestApplyPaidOrderRejectsZeroPriceTariff(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}
	client := model.Client{Enable: false, Name: "tg99", Inbounds: json.RawMessage("[]"), Expiry: 100}
	db.Create(&client)
	// Price 0 and StarsAmount 0 → must never grant a renewal.
	tariff := Tariff{Name: "Free", Price: 0, StarsAmount: 0, Currency: "RUB", AddDays: 30, Enabled: true}
	db.Create(&tariff)
	order := PaymentOrder{ClientId: client.Id, TariffId: tariff.Id, Provider: "yookassa", Amount: 0, Currency: "RUB", Status: StatusPending, IdempotencyKey: "zero"}
	db.Create(&order)

	ps := NewPaymentService()
	applied, _, err := ps.ApplyPaidOrder(order.Id, "c", nil)
	if err == nil {
		t.Fatal("expected error for zero-price tariff")
	}
	if applied {
		t.Fatal("zero-price tariff must not apply a renewal")
	}
	// Transaction rolled back: order stays pending, client not renewed.
	var o PaymentOrder
	db.Where("id = ?", order.Id).First(&o)
	if o.Status != StatusPending {
		t.Errorf("order should remain pending after rejected apply, got %s", o.Status)
	}
	var c model.Client
	db.Where("id = ?", client.Id).First(&c)
	if c.Enable || c.Expiry != 100 {
		t.Errorf("client must be unchanged, got enable=%v expiry=%d", c.Enable, c.Expiry)
	}
}

func TestExpireStaleOrders(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}
	now := time.Now().Unix()
	// Non-polled provider: short order-TTL expiry applies.
	stale := PaymentOrder{ClientId: 1, TariffId: 1, Provider: "stripe", Amount: 1, Currency: "RUB", Status: StatusPending, IdempotencyKey: "stale", ExpiresAt: now - 10}
	fresh := PaymentOrder{ClientId: 1, TariffId: 1, Provider: "stripe", Amount: 1, Currency: "RUB", Status: StatusPending, IdempotencyKey: "fresh", ExpiresAt: now + 3600}
	// Polled provider (cryptobot) past its short TTL must NOT be expired here:
	// it stays pending so a late out-of-band payment is still caught by polling.
	cbStale := PaymentOrder{ClientId: 1, TariffId: 1, Provider: "cryptobot", Amount: 1, Currency: "RUB", Status: StatusPending, IdempotencyKey: "cb-stale", ExpiresAt: now - 10}
	db.Create(&stale)
	db.Create(&fresh)
	db.Create(&cbStale)

	ps := NewPaymentService()
	if err := ps.ExpireStaleOrders(); err != nil {
		t.Fatalf("ExpireStaleOrders: %v", err)
	}
	var s, f, cb PaymentOrder
	db.Where("idempotency_key = ?", "stale").First(&s)
	db.Where("idempotency_key = ?", "fresh").First(&f)
	db.Where("idempotency_key = ?", "cb-stale").First(&cb)
	if s.Status != StatusExpired {
		t.Errorf("stale order not expired: %s", s.Status)
	}
	if f.Status != StatusPending {
		t.Errorf("fresh order should stay pending: %s", f.Status)
	}
	if cb.Status != StatusPending {
		t.Errorf("polled (cryptobot) order must NOT be short-TTL expired: %s", cb.Status)
	}
	_ = database.GetDB()
}

func TestExpireStalePolledOrders(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}
	now := time.Now().Unix()
	grace := int64(3600)
	// Created well before the grace window -> reaped as abandoned.
	old := PaymentOrder{ClientId: 1, TariffId: 1, Provider: "cryptobot", Amount: 1, Currency: "RUB", Status: StatusPending, IdempotencyKey: "cb-old", CreatedAt: now - grace - 10}
	// Recent cryptobot order within grace -> stays pending (poll keeps trying).
	recent := PaymentOrder{ClientId: 1, TariffId: 1, Provider: "cryptobot", Amount: 1, Currency: "RUB", Status: StatusPending, IdempotencyKey: "cb-recent", CreatedAt: now - 10}
	db.Create(&old)
	db.Create(&recent)

	ps := NewPaymentService()
	if err := ps.ExpireStalePolledOrders(grace); err != nil {
		t.Fatalf("ExpireStalePolledOrders: %v", err)
	}
	var o, r PaymentOrder
	db.Where("idempotency_key = ?", "cb-old").First(&o)
	db.Where("idempotency_key = ?", "cb-recent").First(&r)
	if o.Status != StatusExpired {
		t.Errorf("old polled order not reaped: %s", o.Status)
	}
	if r.Status != StatusPending {
		t.Errorf("recent polled order should stay pending: %s", r.Status)
	}
}

func TestOrdersForTgUserScoped(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}
	db.Create(&PaymentOrder{ClientId: 1, TariffId: 1, Provider: "stars", Amount: 5, Currency: "XTR", Status: StatusPaid, TelegramUserId: 100, IdempotencyKey: "a"})
	db.Create(&PaymentOrder{ClientId: 1, TariffId: 1, Provider: "stars", Amount: 6, Currency: "XTR", Status: StatusPending, TelegramUserId: 100, IdempotencyKey: "b"})
	db.Create(&PaymentOrder{ClientId: 2, TariffId: 1, Provider: "stars", Amount: 7, Currency: "XTR", Status: StatusPaid, TelegramUserId: 200, IdempotencyKey: "c"})

	ps := NewPaymentService()
	got, err := ps.OrdersForTgUser(100, 20)
	if err != nil {
		t.Fatalf("OrdersForTgUser: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("OrdersForTgUser(100) = %d orders, want 2", len(got))
	}
	for _, o := range got {
		if o.TelegramUserId != 100 {
			t.Errorf("leaked order belonging to tg %d", o.TelegramUserId)
		}
	}
	// Refundable = paid only.
	ref, err := ps.RefundableOrdersForTgUser(100, 20)
	if err != nil {
		t.Fatalf("RefundableOrdersForTgUser: %v", err)
	}
	if len(ref) != 1 || ref[0].Status != StatusPaid {
		t.Errorf("RefundableOrdersForTgUser(100) = %+v, want exactly 1 paid", ref)
	}
}

func TestFinalizeRefundRevokeRollsBackOnce(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}
	now := time.Now().Unix()
	client := model.Client{Enable: true, Name: "tg7", Inbounds: json.RawMessage("[]"), Volume: 5 << 30, Expiry: now + 40*86400}
	db.Create(&client)
	tariff := Tariff{Name: "M", Price: 10000, Currency: "RUB", AddDays: 30, AddTrafficBytes: 1 << 30, Enabled: true}
	db.Create(&tariff)
	order := PaymentOrder{ClientId: client.Id, TariffId: tariff.Id, Provider: "yookassa", Amount: 10000, Currency: "RUB", Status: StatusPaid, TelegramUserId: 7, IdempotencyKey: "r1", GrantedDays: tariff.AddDays, GrantedTrafficBytes: tariff.AddTrafficBytes, SnapshotVersion: paymentOrderSnapshotVersion}
	db.Create(&order)

	ps := NewPaymentService()
	if err := ps.finalizeRefund(order.Id, true); err != nil {
		t.Fatalf("finalizeRefund: %v", err)
	}
	var o PaymentOrder
	db.Where("id = ?", order.Id).First(&o)
	if o.Status != StatusRefunded {
		t.Errorf("status = %s, want refunded", o.Status)
	}
	var c model.Client
	db.Where("id = ?", client.Id).First(&c)
	wantExpiry := (now + 40*86400) - 30*86400
	if c.Expiry < wantExpiry-2 || c.Expiry > wantExpiry+2 {
		t.Errorf("expiry = %d, want ~%d", c.Expiry, wantExpiry)
	}
	if c.Volume != (5<<30)-(1<<30) {
		t.Errorf("volume = %d, want %d", c.Volume, int64((5<<30)-(1<<30)))
	}
	if !c.Enable {
		t.Error("client must not be disabled by a refund")
	}

	// Second call must be an idempotent no-op (no double roll-back).
	if err := ps.finalizeRefund(order.Id, true); !errors.Is(err, errAlreadyApplied) {
		t.Errorf("second finalizeRefund = %v, want errAlreadyApplied", err)
	}
	var c2 model.Client
	db.Where("id = ?", client.Id).First(&c2)
	if c2.Volume != c.Volume || c2.Expiry != c.Expiry {
		t.Error("second refund must not change the client again")
	}
}

func TestFinalizeRefundNoRevokeKeepsClient(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}
	now := time.Now().Unix()
	client := model.Client{Enable: true, Name: "tg8", Inbounds: json.RawMessage("[]"), Volume: 2 << 30, Expiry: now + 10*86400}
	db.Create(&client)
	tariff := Tariff{Name: "M", Price: 10000, Currency: "RUB", AddDays: 30, AddTrafficBytes: 1 << 30, Enabled: true}
	db.Create(&tariff)
	order := PaymentOrder{ClientId: client.Id, TariffId: tariff.Id, Provider: "yookassa", Amount: 10000, Currency: "RUB", Status: StatusPaid, TelegramUserId: 8, IdempotencyKey: "r2", GrantedDays: tariff.AddDays, GrantedTrafficBytes: tariff.AddTrafficBytes, SnapshotVersion: paymentOrderSnapshotVersion}
	db.Create(&order)

	ps := NewPaymentService()
	if err := ps.finalizeRefund(order.Id, false); err != nil {
		t.Fatalf("finalizeRefund: %v", err)
	}
	var c model.Client
	db.Where("id = ?", client.Id).First(&c)
	if c.Volume != 2<<30 || c.Expiry != now+10*86400 {
		t.Errorf("client changed despite revoke=false: volume=%d expiry=%d", c.Volume, c.Expiry)
	}
	var o PaymentOrder
	db.Where("id = ?", order.Id).First(&o)
	if o.Status != StatusRefunded {
		t.Errorf("status = %s, want refunded", o.Status)
	}
}

func TestFinalizeRefundFloorsExpiryAndVolume(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}
	now := time.Now().Unix()
	// addDays (365) exceeds remaining (5d) and addTraffic exceeds volume → floor.
	client := model.Client{Enable: true, Name: "tg9", Inbounds: json.RawMessage("[]"), Volume: 1 << 20, Expiry: now + 5*86400}
	db.Create(&client)
	tariff := Tariff{Name: "Y", Price: 1, Currency: "RUB", AddDays: 365, AddTrafficBytes: 1 << 30, Enabled: true}
	db.Create(&tariff)
	order := PaymentOrder{ClientId: client.Id, TariffId: tariff.Id, Provider: "yookassa", Amount: 1, Currency: "RUB", Status: StatusPaid, TelegramUserId: 9, IdempotencyKey: "r3", GrantedDays: tariff.AddDays, GrantedTrafficBytes: tariff.AddTrafficBytes, SnapshotVersion: paymentOrderSnapshotVersion}
	db.Create(&order)

	ps := NewPaymentService()
	if err := ps.finalizeRefund(order.Id, true); err != nil {
		t.Fatalf("finalizeRefund: %v", err)
	}
	var c model.Client
	db.Where("id = ?", client.Id).First(&c)
	if c.Expiry < now-2 || c.Expiry > now+2 {
		t.Errorf("expiry floor = %d, want ~now %d", c.Expiry, now)
	}
	if c.Volume != 0 {
		t.Errorf("volume floor = %d, want 0", c.Volume)
	}
}
