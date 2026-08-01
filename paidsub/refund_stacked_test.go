package paidsub

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/deposist/s-ui-x-extended/database/model"
)

// TestRefundOldOrderDoesNotClobberCurrentWindow pins M-08: refunding an older
// stacked purchase changes only purchased capacity. Current-window and
// cumulative usage remain untouched.
func TestRefundOldOrderDoesNotClobberCurrentWindow(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}
	client := model.Client{Enable: true, Name: "stk", Inbounds: json.RawMessage("[]"),
		Volume: 5 << 30, Up: 100, Down: 200, TotalUp: 1000, TotalDown: 2000}
	db.Create(&client)
	tariff := Tariff{Name: "T", Price: 10000, Currency: "RUB", AddTrafficBytes: 1 << 30, Enabled: true}
	db.Create(&tariff)

	ps := NewPaymentService()
	apply := func(key string) uint {
		o := PaymentOrder{ClientId: client.Id, TariffId: tariff.Id, Provider: "yookassa",
			Amount: 10000, Currency: "RUB", Status: StatusPending, TelegramUserId: 7, IdempotencyKey: key,
			GrantedDays: tariff.AddDays, GrantedTrafficBytes: tariff.AddTrafficBytes, SnapshotVersion: paymentOrderSnapshotVersion}
		db.Create(&o)
		if applied, _, err := ps.ApplyPaidOrder(o.Id, "ch:"+key, nil); err != nil || !applied {
			t.Fatalf("ApplyPaidOrder(%s) = (%v,%v)", key, applied, err)
		}
		return o.Id
	}
	accrue := func(up, down int64) {
		if err := db.Model(&model.Client{}).Where("id = ?", client.Id).
			Updates(map[string]any{"up": up, "down": down}).Error; err != nil {
			t.Fatal(err)
		}
	}

	orderA := apply("A") // granted_up/down = 100/200; total_up -> 1100
	accrue(50, 60)
	apply("B") // granted_up/down = 50/60; total_up -> 1150
	accrue(30, 40)

	// Refund the OLDER order A. A newer paid traffic order (B) exists, so up/down
	// must be left at the current window (30/40), not overwritten with A's 100/200.
	if _, err := ps.RefundOrder(context.Background(), orderA, true); err != nil {
		t.Fatalf("RefundOrder(A): %v", err)
	}
	var after model.Client
	db.Where("id = ?", client.Id).First(&after)

	if after.Up != 30 || after.Down != 40 {
		t.Fatalf("current-window up/down were clobbered: up=%d down=%d (want 30/40)", after.Up, after.Down)
	}
	if after.TotalUp != 1150 || after.TotalDown != 2260 {
		t.Fatalf("cumulative totals changed: total_up=%d total_down=%d (want 1150/2260)", after.TotalUp, after.TotalDown)
	}
	if after.Volume != 6<<30 {
		t.Fatalf("volume not rolled back once: %d (want %d)", after.Volume, int64(6<<30))
	}
}
