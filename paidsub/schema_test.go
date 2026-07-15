package paidsub

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/deposist/s-ui-x-extended/service"

	"gorm.io/gorm"
)

var testDBSeq atomic.Int64

func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	if err := service.StopAuditWriter(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SUI_DB_FOLDER", t.TempDir())
	prevAuditSync := service.AuditSyncForTest
	service.AuditSyncForTest = true
	t.Cleanup(func() { service.AuditSyncForTest = prevAuditSync })
	// A uniquely named shared-cache in-memory DB per test isolates each test
	// without touching disk (avoiding Windows temp-file lock flakiness). The
	// previous unnamed `:memory:?cache=shared` form was process-global: rows
	// leaked across tests and concurrent access raced with "database table is
	// locked".
	dsn := fmt.Sprintf("file:paidsub_test_%d?mode=memory&cache=shared", testDBSeq.Add(1))
	if err := database.InitDB(dsn); err != nil {
		if strings.Contains(err.Error(), "go-sqlite3 requires cgo") {
			t.Skip(err)
		}
		t.Fatal(err)
	}
	// For this in-memory DSN the first-run routine writes initial-admin.txt next
	// to the (virtual) db name, i.e. the working dir; remove that side file so it
	// never lingers in the package directory.
	t.Cleanup(func() { _ = os.Remove("initial-admin.txt") })
	db := database.GetDB()
	t.Cleanup(func() {
		if err := service.StopAuditWriter(context.Background()); err != nil {
			t.Errorf("stop audit writer: %v", err)
		}
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

func TestEnsureSchemaIdempotent(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema first run: %v", err)
	}
	// Second run must be a no-op (all statements use IF NOT EXISTS).
	if err := EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema second run: %v", err)
	}
	for _, table := range []string{"paidsub_bindings", "tariffs", "payment_orders", "awg_devices"} {
		if !db.Migrator().HasTable(table) {
			t.Fatalf("table %q missing after EnsureSchema", table)
		}
	}
}

// TestPaymentOrdersTelegramIndex pins O-1: order history is queried by
// telegram_user_id (OrdersForTgUser / RefundableOrdersForTgUser), so the column
// must be indexed to avoid a full-table scan.
func TestEnsureSchemaAddsAWGColumnsToLegacyTables(t *testing.T) {
	db := openTestDB(t)
	for _, statement := range []string{
		`CREATE TABLE tariffs (
			id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT, description TEXT,
			price INTEGER NOT NULL DEFAULT 0, currency TEXT NOT NULL DEFAULT 'RUB',
			stars_amount INTEGER NOT NULL DEFAULT 0, add_days INTEGER NOT NULL DEFAULT 0,
			add_traffic_bytes INTEGER NOT NULL DEFAULT 0, sort INTEGER NOT NULL DEFAULT 0,
			enabled INTEGER NOT NULL DEFAULT 1, created_at INTEGER NOT NULL DEFAULT 0,
			updated_at INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE payment_orders (
			id INTEGER PRIMARY KEY AUTOINCREMENT, client_id INTEGER NOT NULL, tariff_id INTEGER NOT NULL,
			provider TEXT NOT NULL, amount INTEGER NOT NULL DEFAULT 0, currency TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending', telegram_user_id INTEGER NOT NULL DEFAULT 0,
			idempotency_key TEXT NOT NULL, provider_charge_id TEXT, provider_payload BLOB,
			external_url TEXT, created_at INTEGER NOT NULL DEFAULT 0, paid_at INTEGER NOT NULL DEFAULT 0,
			expires_at INTEGER NOT NULL DEFAULT 0, granted_up INTEGER NOT NULL DEFAULT 0,
			granted_down INTEGER NOT NULL DEFAULT 0
		)`,
		`INSERT INTO tariffs(name) VALUES ('legacy')`,
		`INSERT INTO payment_orders(client_id, tariff_id, provider, currency, idempotency_key) VALUES (1, 1, 'test', 'RUB', 'legacy-order')`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema legacy migration: %v", err)
	}
	var tariffName string
	if err := db.Raw("SELECT name FROM tariffs WHERE id = 1").Scan(&tariffName).Error; err != nil {
		t.Fatal(err)
	}
	if tariffName != "legacy" {
		t.Fatalf("legacy tariff was not preserved: %q", tariffName)
	}
	for _, check := range []struct {
		model  any
		column string
	}{
		{&Tariff{}, "max_awg_devices"},
		{&PaymentOrder{}, "granted_awg_devices"},
	} {
		if !db.Migrator().HasColumn(check.model, check.column) {
			t.Fatalf("legacy migration did not add %q", check.column)
		}
	}
}

func TestAWGSchemaConstraints(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}
	for _, column := range []struct {
		model  any
		column string
	}{
		{&Tariff{}, "max_awg_devices"},
		{&PaymentOrder{}, "granted_awg_devices"},
	} {
		if !db.Migrator().HasColumn(column.model, column.column) {
			t.Fatalf("column %q is missing", column.column)
		}
	}
	for _, index := range []string{
		"idx_awg_devices_public_key",
		"idx_awg_devices_endpoint_ipv4",
		"idx_awg_devices_client_enabled",
		"idx_awg_devices_client_create_request",
	} {
		var count int64
		if err := db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name=?", index).Scan(&count).Error; err != nil {
			t.Fatalf("query index %s: %v", index, err)
		}
		if count != 1 {
			t.Fatalf("index %q missing", index)
		}
	}

	first := model.AWGDevice{ClientId: 1, Name: "first", CryptoContext: []byte{1}, PublicKey: "first", PrivateKeyEnc: []byte{1}, PSKEnc: []byte{1}, IPv4Address: "10.77.0.2", DesiredEnabled: true, SyncState: "in_sync", CreatedAt: 1, UpdatedAt: 1}
	if err := db.Create(&first).Error; err != nil {
		t.Fatal(err)
	}
	duplicate := first
	duplicate.Id = 0
	duplicate.PublicKey = "second"
	if err := db.Create(&duplicate).Error; err == nil {
		t.Fatal("active IPv4 partial unique index accepted a duplicate")
	}
	if err := db.Model(&first).Update("desired_enabled", false).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&duplicate).Error; err != nil {
		t.Fatalf("revoked IPv4 address should not block an active row: %v", err)
	}
}

func TestAWGCreateRequestKeyUniqueness(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}
	device := func(clientID uint, publicKey, address, requestKey string) model.AWGDevice {
		return model.AWGDevice{
			ClientId: clientID, Name: "phone", CreateRequestKey: requestKey,
			CryptoContext: []byte{1}, PublicKey: publicKey, PrivateKeyEnc: []byte{1}, PSKEnc: []byte{1},
			IPv4Address: address, DesiredEnabled: true, SyncState: "in_sync", CreatedAt: 1, UpdatedAt: 1,
		}
	}

	first := device(1, "first", "10.77.0.2", "telegram-update-100")
	if err := db.Create(&first).Error; err != nil {
		t.Fatalf("first request key: %v", err)
	}
	duplicateForClient := device(1, "second", "10.77.0.3", "telegram-update-100")
	if err := db.Create(&duplicateForClient).Error; err == nil {
		t.Fatal("same client accepted a duplicate nonempty create request key")
	}
	sameKeyOtherClient := device(2, "third", "10.77.0.4", "telegram-update-100")
	if err := db.Create(&sameKeyOtherClient).Error; err != nil {
		t.Fatalf("same request key must be allowed for another client: %v", err)
	}
	legacyEmptyOne := device(1, "fourth", "10.77.0.5", "")
	legacyEmptyTwo := device(1, "fifth", "10.77.0.6", "")
	if err := db.Create(&legacyEmptyOne).Error; err != nil {
		t.Fatalf("first legacy empty request key: %v", err)
	}
	if err := db.Create(&legacyEmptyTwo).Error; err != nil {
		t.Fatalf("second legacy empty request key must be allowed: %v", err)
	}
}

func TestEnsureSchemaAddsRequestKeysToLegacyAWGTable(t *testing.T) {
	db := openTestDB(t)
	// InitDB creates awg_devices on fresh databases; replace it with the
	// legacy layout (no request-key or endpoint_id columns) to model an
	// upgraded install.
	if err := db.Exec(`DROP TABLE IF EXISTS awg_devices`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE awg_devices (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		client_id INTEGER NOT NULL,
		name TEXT NOT NULL,
		crypto_context BLOB NOT NULL,
		public_key TEXT NOT NULL,
		previous_public_key TEXT,
		private_key_enc BLOB NOT NULL,
		psk_enc BLOB NOT NULL,
		ipv4_address TEXT NOT NULL,
		desired_enabled INTEGER NOT NULL DEFAULT 1,
		sync_state TEXT NOT NULL,
		provisioned INTEGER NOT NULL DEFAULT 0,
		last_error TEXT,
		rx_baseline INTEGER NOT NULL DEFAULT 0,
		tx_baseline INTEGER NOT NULL DEFAULT 0,
		total_rx INTEGER NOT NULL DEFAULT 0,
		total_tx INTEGER NOT NULL DEFAULT 0,
		last_handshake INTEGER NOT NULL DEFAULT 0,
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL,
		revoked_at INTEGER NOT NULL DEFAULT 0,
		ip_reusable_after INTEGER NOT NULL DEFAULT 0
	)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO awg_devices(
		client_id, name, crypto_context, public_key, private_key_enc, psk_enc,
		ipv4_address, desired_enabled, sync_state, created_at, updated_at
	) VALUES (1, 'legacy', x'01', 'legacy-public', x'01', x'01', '10.77.0.2', 1, 'in_sync', 1, 1)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema legacy AWG migration: %v", err)
	}
	if !db.Migrator().HasColumn(&model.AWGDevice{}, "create_request_key") {
		t.Fatal("legacy AWG migration did not add create_request_key")
	}
	if !db.Migrator().HasColumn(&model.AWGDevice{}, "rotate_request_key") {
		t.Fatal("legacy AWG migration did not add rotate_request_key")
	}
	var keys struct {
		CreateRequestKey string
		RotateRequestKey string
	}
	if err := db.Raw("SELECT create_request_key, rotate_request_key FROM awg_devices WHERE public_key = ?", "legacy-public").Scan(&keys).Error; err != nil {
		t.Fatal(err)
	}
	if keys.CreateRequestKey != "" || keys.RotateRequestKey != "" {
		t.Fatalf("legacy row request keys = create %q, rotate %q; want empty", keys.CreateRequestKey, keys.RotateRequestKey)
	}
}

func TestPaymentOrdersTelegramIndex(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}
	var count int64
	if err := db.Raw(
		"SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_payment_orders_telegram'",
	).Scan(&count).Error; err != nil {
		t.Fatalf("query index: %v", err)
	}
	if count != 1 {
		t.Fatalf("idx_payment_orders_telegram missing (count=%d)", count)
	}
}

func TestBindingUniqueness(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}
	if err := db.Create(&Binding{ClientId: 1, TgUserId: 1000}).Error; err != nil {
		t.Fatalf("first binding: %v", err)
	}
	// Same tg id, different client → must violate the UNIQUE(tg_user_id) index.
	if err := db.Create(&Binding{ClientId: 2, TgUserId: 1000}).Error; err == nil {
		t.Fatal("expected duplicate tg_user_id to be rejected")
	}
	// Same client, different tg id → must violate the UNIQUE(client_id) index.
	if err := db.Create(&Binding{ClientId: 1, TgUserId: 2000}).Error; err == nil {
		t.Fatal("expected duplicate client_id to be rejected")
	}
}

func TestSetBindingReleasesPrevious(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}
	svc := NewService()
	if err := svc.SetBinding(1, 1000); err != nil {
		t.Fatalf("SetBinding 1->1000: %v", err)
	}
	// Rebind the same tg id to a different client: old row must be released.
	if err := svc.SetBinding(2, 1000); err != nil {
		t.Fatalf("SetBinding 2->1000: %v", err)
	}
	var count int64
	db.Model(&Binding{}).Where("tg_user_id = ?", 1000).Count(&count)
	if count != 1 {
		t.Fatalf("expected exactly 1 binding for tg 1000, got %d", count)
	}
	if _, err := svc.BindingForClient(1); err == nil {
		t.Fatal("expected client 1 binding to be released")
	}
	b, err := svc.BindingForClient(2)
	if err != nil {
		t.Fatalf("BindingForClient(2): %v", err)
	}
	if b.TgUserId != 1000 {
		t.Fatalf("expected tg 1000 bound to client 2, got %d", b.TgUserId)
	}
}
