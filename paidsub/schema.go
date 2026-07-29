package paidsub

import (
	"strings"

	"github.com/deposist/s-ui-x-extended/database/model"

	"gorm.io/gorm"
)

// EnsureSchema creates the module's tables and indexes idempotently. It is
// called from app wiring at startup so the module owns its schema without
// touching the central migration chain (cmd/migration) or database/db.go.
// Removing the module leaves these tables orphaned but harmless.
func EnsureSchema(db *gorm.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS paidsub_bindings (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			client_id INTEGER NOT NULL,
			tg_user_id INTEGER NOT NULL,
			created_at INTEGER NOT NULL DEFAULT 0,
			updated_at INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS tariffs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT,
			description TEXT,
			price INTEGER NOT NULL DEFAULT 0,
			currency TEXT NOT NULL DEFAULT 'RUB',
			stars_amount INTEGER NOT NULL DEFAULT 0,
			add_days INTEGER NOT NULL DEFAULT 0,
			add_traffic_bytes INTEGER NOT NULL DEFAULT 0,
			max_awg_devices INTEGER NOT NULL DEFAULT 0,
			sort INTEGER NOT NULL DEFAULT 0,
			enabled INTEGER NOT NULL DEFAULT 1,
			created_at INTEGER NOT NULL DEFAULT 0,
			updated_at INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS payment_orders (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			client_id INTEGER NOT NULL,
			tariff_id INTEGER NOT NULL,
			provider TEXT NOT NULL,
			amount INTEGER NOT NULL DEFAULT 0,
			currency TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			telegram_user_id INTEGER NOT NULL DEFAULT 0,
			idempotency_key TEXT NOT NULL,
			provider_ref TEXT NOT NULL DEFAULT '',
			provider_charge_id TEXT,
			provider_payload BLOB,
			external_url TEXT,
			created_at INTEGER NOT NULL DEFAULT 0,
			paid_at INTEGER NOT NULL DEFAULT 0,
			expires_at INTEGER NOT NULL DEFAULT 0,
			granted_up INTEGER NOT NULL DEFAULT 0,
			granted_down INTEGER NOT NULL DEFAULT 0,
			granted_days INTEGER NOT NULL DEFAULT 0,
			granted_traffic_bytes INTEGER NOT NULL DEFAULT 0,
			granted_awg_devices INTEGER NOT NULL DEFAULT 0,
			snapshot_version INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS client_endpoint_access (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			client_id INTEGER NOT NULL,
			endpoint_id INTEGER NOT NULL,
			device_limit INTEGER NOT NULL DEFAULT 0,
			source TEXT NOT NULL DEFAULT 'manual',
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS awg_devices (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			client_id INTEGER NOT NULL,
			endpoint_id INTEGER NOT NULL DEFAULT 0,
			name TEXT NOT NULL,
			create_request_key TEXT NOT NULL DEFAULT '',
			rotate_request_key TEXT NOT NULL DEFAULT '',
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
			ip_reusable_after INTEGER NOT NULL DEFAULT 0,
			expires_at INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_paidsub_bindings_client ON paidsub_bindings(client_id)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_paidsub_bindings_tg ON paidsub_bindings(tg_user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_tariffs_enabled_sort ON tariffs(enabled, sort)`,
		`CREATE INDEX IF NOT EXISTS idx_payment_orders_client ON payment_orders(client_id, status)`,
		`CREATE INDEX IF NOT EXISTS idx_payment_orders_pending_poll ON payment_orders(provider, status, created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_payment_orders_telegram ON payment_orders(telegram_user_id)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_payment_orders_idem ON payment_orders(idempotency_key)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_client_endpoint_access ON client_endpoint_access(client_id, endpoint_id)`,
		`CREATE INDEX IF NOT EXISTS idx_client_endpoint_access_endpoint ON client_endpoint_access(endpoint_id)`,
	}
	for _, stmt := range stmts {
		if err := db.Exec(stmt).Error; err != nil {
			return err
		}
	}
	// Additive columns for upgraded installs (the CREATE above only covers fresh
	// installs). SQLite lacks ADD COLUMN IF NOT EXISTS, so guard with HasColumn.
	// These MUST run before the awg_devices index statements below: on an
	// upgraded database the table already exists without the new columns, so an
	// index referencing endpoint_id fails with "no such column: endpoint_id"
	// until the guarded ALTER has added it (the 1.0.2-beta1 -> 1.0.2 crash).
	mig := db.Migrator()
	for _, migration := range []struct {
		model  any
		column string
		ddl    string
	}{
		{&PaymentOrder{}, "granted_up", `ALTER TABLE payment_orders ADD COLUMN granted_up INTEGER NOT NULL DEFAULT 0`},
		{&PaymentOrder{}, "granted_down", `ALTER TABLE payment_orders ADD COLUMN granted_down INTEGER NOT NULL DEFAULT 0`},
		{&PaymentOrder{}, "provider_ref", `ALTER TABLE payment_orders ADD COLUMN provider_ref TEXT NOT NULL DEFAULT ''`},
		{&PaymentOrder{}, "granted_days", `ALTER TABLE payment_orders ADD COLUMN granted_days INTEGER NOT NULL DEFAULT 0`},
		{&PaymentOrder{}, "granted_traffic_bytes", `ALTER TABLE payment_orders ADD COLUMN granted_traffic_bytes INTEGER NOT NULL DEFAULT 0`},
		{&PaymentOrder{}, "granted_awg_devices", `ALTER TABLE payment_orders ADD COLUMN granted_awg_devices INTEGER NOT NULL DEFAULT 0`},
		{&PaymentOrder{}, "snapshot_version", `ALTER TABLE payment_orders ADD COLUMN snapshot_version INTEGER NOT NULL DEFAULT 0`},
		{&Tariff{}, "max_awg_devices", `ALTER TABLE tariffs ADD COLUMN max_awg_devices INTEGER NOT NULL DEFAULT 0`},
		{&model.AWGDevice{}, "create_request_key", `ALTER TABLE awg_devices ADD COLUMN create_request_key TEXT NOT NULL DEFAULT ''`},
		{&model.AWGDevice{}, "rotate_request_key", `ALTER TABLE awg_devices ADD COLUMN rotate_request_key TEXT NOT NULL DEFAULT ''`},
		{&model.AWGDevice{}, "endpoint_id", `ALTER TABLE awg_devices ADD COLUMN endpoint_id INTEGER NOT NULL DEFAULT 0`},
		{&model.AWGDevice{}, "expires_at", `ALTER TABLE awg_devices ADD COLUMN expires_at INTEGER NOT NULL DEFAULT 0`},
	} {
		if mig.HasColumn(migration.model, migration.column) {
			continue
		}
		if err := db.Exec(migration.ddl).Error; err != nil {
			return err
		}
	}
	if err := migratePaymentOrderSnapshots(db); err != nil {
		return err
	}
	if err := migratePaymentProviderRefs(db); err != nil {
		return err
	}
	for _, stmt := range []string{
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_payment_orders_ref ON payment_orders(provider, provider_ref) WHERE provider_ref != ''`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_payment_orders_charge ON payment_orders(provider, provider_charge_id) WHERE provider_charge_id != ''`,
	} {
		if err := db.Exec(stmt).Error; err != nil {
			return err
		}
	}
	// awg_devices indexes run after the column migrations above so they can
	// reference columns added to upgraded installs.
	awgIndexes := []string{
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_awg_devices_public_key ON awg_devices(public_key)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_awg_devices_endpoint_ipv4 ON awg_devices(endpoint_id, ipv4_address) WHERE desired_enabled = 1`,
		`CREATE INDEX IF NOT EXISTS idx_awg_devices_client_enabled ON awg_devices(client_id, desired_enabled)`,
		`CREATE INDEX IF NOT EXISTS idx_awg_devices_endpoint_enabled ON awg_devices(endpoint_id, desired_enabled)`,
		`CREATE INDEX IF NOT EXISTS idx_awg_devices_sync_state ON awg_devices(sync_state)`,
		`CREATE INDEX IF NOT EXISTS idx_awg_devices_previous_public_key ON awg_devices(previous_public_key)`,
		`CREATE INDEX IF NOT EXISTS idx_awg_devices_ip_reusable_after ON awg_devices(ip_reusable_after)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_awg_devices_client_create_request
			ON awg_devices(client_id, create_request_key) WHERE create_request_key != ''`,
	}
	for _, stmt := range awgIndexes {
		if err := db.Exec(stmt).Error; err != nil {
			return err
		}
	}
	return nil
}

func migratePaymentProviderRefs(db *gorm.DB) error {
	var orders []PaymentOrder
	if err := db.Where("provider = ? AND provider_ref = ''", string(ProviderCryptoBot)).Find(&orders).Error; err != nil {
		return err
	}
	return db.Transaction(func(tx *gorm.DB) error {
		for _, order := range orders {
			ref := extractProviderRef(order.ProviderPayload)
			if ref == "" {
				ref = strings.TrimPrefix(order.ProviderChargeID, "cryptobot:")
			}
			if ref == "" {
				continue
			}
			if err := tx.Model(&PaymentOrder{}).Where("id = ? AND provider_ref = ''", order.Id).
				Update("provider_ref", ref).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func migratePaymentOrderSnapshots(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		// The original purchase-time grants cannot be reconstructed for pending
		// legacy orders, so cancel them instead of granting a mutable tariff.
		if err := tx.Model(&PaymentOrder{}).
			Where("status = ? AND snapshot_version = 0", StatusPending).
			Update("status", StatusFailed).Error; err != nil {
			return err
		}

		// Paid legacy orders need a frozen baseline for future refunds. Capture
		// the current tariff once; subsequent tariff edits/deletion cannot alter it.
		return tx.Exec(`UPDATE payment_orders
			SET granted_days = COALESCE((SELECT add_days FROM tariffs WHERE tariffs.id = payment_orders.tariff_id), 0),
				granted_traffic_bytes = COALESCE((SELECT add_traffic_bytes FROM tariffs WHERE tariffs.id = payment_orders.tariff_id), 0),
				granted_awg_devices = CASE
					WHEN granted_awg_devices > 0 THEN granted_awg_devices
					ELSE COALESCE((SELECT max_awg_devices FROM tariffs WHERE tariffs.id = payment_orders.tariff_id), 0)
				END,
				snapshot_version = ?
			WHERE status = ? AND snapshot_version = 0
				AND EXISTS (SELECT 1 FROM tariffs WHERE tariffs.id = payment_orders.tariff_id)`,
			paymentOrderSnapshotVersion, StatusPaid).Error
	})
}
