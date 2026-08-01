package paidsub

import (
	"fmt"
	"strings"

	"github.com/deposist/s-ui-x-extended/database/model"

	"gorm.io/gorm"
)

// EnsureSchema creates the module's tables and indexes idempotently. It is
// called from app wiring at startup so the module owns its schema without
// touching the central migration chain (cmd/migration) or database/db.go.
// Removing the module leaves these tables orphaned but harmless.
func EnsureSchema(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		for _, stmt := range []string{
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
			`CREATE TABLE IF NOT EXISTS paidsub_payment_charges (
				provider TEXT NOT NULL,
				charge_id TEXT NOT NULL,
				order_id INTEGER NOT NULL,
				disposition TEXT NOT NULL,
				raw_payload BLOB,
				confirmed_at INTEGER NOT NULL,
				refunded_at INTEGER NOT NULL DEFAULT 0,
				PRIMARY KEY(provider, charge_id)
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
			`CREATE TABLE IF NOT EXISTS paidsub_poll_cursors (
				provider TEXT PRIMARY KEY,
				last_order_id INTEGER NOT NULL DEFAULT 0
			)`,
			`CREATE TABLE IF NOT EXISTS paidsub_invoice_cancellations (
				order_id INTEGER NOT NULL,
				provider TEXT NOT NULL,
				provider_ref TEXT NOT NULL,
				PRIMARY KEY(provider, provider_ref)
			)`,
		} {
			if err := tx.Exec(stmt).Error; err != nil {
				return fmt.Errorf("paidsub schema create table: %w", err)
			}
		}

		migrator := tx.Migrator()
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
			if migrator.HasColumn(migration.model, migration.column) {
				continue
			}
			if err := tx.Exec(migration.ddl).Error; err != nil {
				return fmt.Errorf("paidsub schema add column %s: %w", migration.column, err)
			}
		}
		if err := migratePaymentProviderRefs(tx); err != nil {
			return fmt.Errorf("paidsub schema migrate provider references: %w", err)
		}
		if err := migratePaymentOrderSnapshots(tx); err != nil {
			return fmt.Errorf("paidsub schema migrate payment snapshots: %w", err)
		}
		if err := resolveLegacySchemaDuplicates(tx); err != nil {
			return fmt.Errorf("paidsub schema resolve duplicates: %w", err)
		}

		for _, index := range paidSubSchemaIndexes() {
			if err := ensureVerifiedIndex(tx, index); err != nil {
				return err
			}
		}
		return nil
	})
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
		// legacy orders. CryptoBot invoices may already be payable, so preserve
		// them for provider reconciliation/cancellation; other providers fail.
		if err := tx.Model(&PaymentOrder{}).
			Where("status IN ? AND snapshot_version = 0 AND provider = ?",
				[]string{StatusPending, StatusFailed, StatusExpired}, string(ProviderCryptoBot)).
			Update("status", StatusRecoverable).Error; err != nil {
			return err
		}
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

type schemaIndex struct {
	name    string
	table   string
	unique  bool
	columns []string
	where   string
	ddl     string
}

func paidSubSchemaIndexes() []schemaIndex {
	return []schemaIndex{
		{name: "idx_paidsub_bindings_client", table: "paidsub_bindings", unique: true, columns: []string{"client_id"}, ddl: `CREATE UNIQUE INDEX idx_paidsub_bindings_client ON paidsub_bindings(client_id)`},
		{name: "idx_paidsub_bindings_tg", table: "paidsub_bindings", unique: true, columns: []string{"tg_user_id"}, ddl: `CREATE UNIQUE INDEX idx_paidsub_bindings_tg ON paidsub_bindings(tg_user_id)`},
		{name: "idx_tariffs_enabled_sort", table: "tariffs", columns: []string{"enabled", "sort"}, ddl: `CREATE INDEX idx_tariffs_enabled_sort ON tariffs(enabled, sort)`},
		{name: "idx_payment_orders_client", table: "payment_orders", columns: []string{"client_id", "status"}, ddl: `CREATE INDEX idx_payment_orders_client ON payment_orders(client_id, status)`},
		{name: "idx_payment_orders_pending_poll", table: "payment_orders", columns: []string{"provider", "status", "id"}, ddl: `CREATE INDEX idx_payment_orders_pending_poll ON payment_orders(provider, status, id)`},
		{name: "idx_payment_orders_telegram", table: "payment_orders", columns: []string{"telegram_user_id"}, ddl: `CREATE INDEX idx_payment_orders_telegram ON payment_orders(telegram_user_id)`},
		{name: "idx_payment_orders_idem", table: "payment_orders", unique: true, columns: []string{"idempotency_key"}, ddl: `CREATE UNIQUE INDEX idx_payment_orders_idem ON payment_orders(idempotency_key)`},
		{name: "idx_payment_orders_ref", table: "payment_orders", unique: true, columns: []string{"provider", "provider_ref"}, where: "provider_ref != ''", ddl: `CREATE UNIQUE INDEX idx_payment_orders_ref ON payment_orders(provider, provider_ref) WHERE provider_ref != ''`},
		{name: "idx_payment_orders_charge", table: "payment_orders", unique: true, columns: []string{"provider", "provider_charge_id"}, where: "provider_charge_id != ''", ddl: `CREATE UNIQUE INDEX idx_payment_orders_charge ON payment_orders(provider, provider_charge_id) WHERE provider_charge_id != ''`},
		{name: "idx_paidsub_payment_charges_order", table: "paidsub_payment_charges", columns: []string{"order_id"}, ddl: `CREATE INDEX idx_paidsub_payment_charges_order ON paidsub_payment_charges(order_id)`},
		{name: "idx_client_endpoint_access", table: "client_endpoint_access", unique: true, columns: []string{"client_id", "endpoint_id"}, ddl: `CREATE UNIQUE INDEX idx_client_endpoint_access ON client_endpoint_access(client_id, endpoint_id)`},
		{name: "idx_client_endpoint_access_endpoint", table: "client_endpoint_access", columns: []string{"endpoint_id"}, ddl: `CREATE INDEX idx_client_endpoint_access_endpoint ON client_endpoint_access(endpoint_id)`},
		{name: "idx_awg_devices_public_key", table: "awg_devices", unique: true, columns: []string{"public_key"}, ddl: `CREATE UNIQUE INDEX idx_awg_devices_public_key ON awg_devices(public_key)`},
		{name: "idx_awg_devices_endpoint_ipv4", table: "awg_devices", unique: true, columns: []string{"endpoint_id", "ipv4_address"}, where: "desired_enabled = 1", ddl: `CREATE UNIQUE INDEX idx_awg_devices_endpoint_ipv4 ON awg_devices(endpoint_id, ipv4_address) WHERE desired_enabled = 1`},
		{name: "idx_awg_devices_client_enabled", table: "awg_devices", columns: []string{"client_id", "desired_enabled"}, ddl: `CREATE INDEX idx_awg_devices_client_enabled ON awg_devices(client_id, desired_enabled)`},
		{name: "idx_awg_devices_endpoint_enabled", table: "awg_devices", columns: []string{"endpoint_id", "desired_enabled"}, ddl: `CREATE INDEX idx_awg_devices_endpoint_enabled ON awg_devices(endpoint_id, desired_enabled)`},
		{name: "idx_awg_devices_sync_state", table: "awg_devices", columns: []string{"sync_state"}, ddl: `CREATE INDEX idx_awg_devices_sync_state ON awg_devices(sync_state)`},
		{name: "idx_awg_devices_previous_public_key", table: "awg_devices", columns: []string{"previous_public_key"}, ddl: `CREATE INDEX idx_awg_devices_previous_public_key ON awg_devices(previous_public_key)`},
		{name: "idx_awg_devices_ip_reusable_after", table: "awg_devices", columns: []string{"ip_reusable_after"}, ddl: `CREATE INDEX idx_awg_devices_ip_reusable_after ON awg_devices(ip_reusable_after)`},
		{name: "idx_awg_devices_client_create_request", table: "awg_devices", unique: true, columns: []string{"client_id", "create_request_key"}, where: "create_request_key != ''", ddl: `CREATE UNIQUE INDEX idx_awg_devices_client_create_request ON awg_devices(client_id, create_request_key) WHERE create_request_key != ''`},
	}
}

func resolveLegacySchemaDuplicates(tx *gorm.DB) error {
	for _, stmt := range []string{
		`DELETE FROM paidsub_bindings WHERE id NOT IN (SELECT MAX(id) FROM paidsub_bindings GROUP BY client_id)`,
		`DELETE FROM paidsub_bindings WHERE id NOT IN (SELECT MAX(id) FROM paidsub_bindings GROUP BY tg_user_id)`,
		`UPDATE payment_orders SET idempotency_key = idempotency_key || ':legacy:' || id
			WHERE id NOT IN (SELECT MIN(id) FROM payment_orders GROUP BY idempotency_key)`,
		`UPDATE payment_orders SET provider_ref = '', provider_payload = NULL, external_url = '', status = CASE
			WHEN status IN ('pending', 'invoice_creating') THEN 'recoverable' ELSE status END
			WHERE provider_ref != '' AND id NOT IN (
				SELECT COALESCE(MIN(CASE WHEN status = 'paid' THEN id END), MIN(id))
				FROM payment_orders WHERE provider_ref != '' GROUP BY provider, provider_ref
			)`,
		`UPDATE payment_orders SET provider_charge_id = NULL
			WHERE provider_charge_id != '' AND id NOT IN (
				SELECT COALESCE(MIN(CASE WHEN status = 'paid' THEN id END), MIN(id))
				FROM payment_orders WHERE provider_charge_id != '' GROUP BY provider, provider_charge_id
			)`,
		`DELETE FROM client_endpoint_access WHERE id NOT IN (
			SELECT MAX(id) FROM client_endpoint_access GROUP BY client_id, endpoint_id
		)`,
	} {
		if err := tx.Exec(stmt).Error; err != nil {
			return err
		}
	}
	return nil
}

func ensureVerifiedIndex(tx *gorm.DB, want schemaIndex) error {
	var existing struct {
		SQL string `gorm:"column:sql"`
	}
	if err := tx.Raw(`SELECT sql FROM sqlite_master WHERE type = 'index' AND name = ?`, want.name).Scan(&existing).Error; err != nil {
		return fmt.Errorf("paidsub schema inspect index %s: %w", want.name, err)
	}
	valid := false
	if existing.SQL != "" {
		var err error
		valid, err = schemaIndexMatches(tx, want, existing.SQL)
		if err != nil {
			return fmt.Errorf("paidsub schema verify index %s: %w", want.name, err)
		}
		if !valid {
			if err := tx.Exec(`DROP INDEX ` + quoteSQLiteIdentifier(want.name)).Error; err != nil {
				return fmt.Errorf("paidsub schema replace index %s: %w", want.name, err)
			}
		}
	}
	if !valid {
		if err := tx.Exec(want.ddl).Error; err != nil {
			return fmt.Errorf("paidsub schema create index %s: %w", want.name, err)
		}
	}
	var sql string
	if err := tx.Raw(`SELECT sql FROM sqlite_master WHERE type = 'index' AND name = ?`, want.name).Scan(&sql).Error; err != nil {
		return fmt.Errorf("paidsub schema read index %s: %w", want.name, err)
	}
	verified, err := schemaIndexMatches(tx, want, sql)
	if err != nil || !verified {
		return fmt.Errorf("paidsub schema index %s failed verification", want.name)
	}
	return nil
}

func schemaIndexMatches(tx *gorm.DB, want schemaIndex, sql string) (bool, error) {
	var listed []struct {
		Name    string `gorm:"column:name"`
		Unique  int    `gorm:"column:unique"`
		Partial int    `gorm:"column:partial"`
	}
	if err := tx.Raw(`PRAGMA index_list(` + quoteSQLiteIdentifier(want.table) + `)`).Scan(&listed).Error; err != nil {
		return false, err
	}
	matched := false
	for _, item := range listed {
		if item.Name == want.name {
			matched = item.Unique == boolInt(want.unique) && item.Partial == boolInt(want.where != "")
			break
		}
	}
	if !matched {
		return false, nil
	}
	var columns []struct {
		Seqno int    `gorm:"column:seqno"`
		Name  string `gorm:"column:name"`
	}
	if err := tx.Raw(`PRAGMA index_info(` + quoteSQLiteIdentifier(want.name) + `)`).Scan(&columns).Error; err != nil {
		return false, err
	}
	if len(columns) != len(want.columns) {
		return false, nil
	}
	for i, column := range columns {
		if column.Seqno != i || column.Name != want.columns[i] {
			return false, nil
		}
	}
	if want.where != "" && !strings.Contains(normalizeSQLiteSQL(sql), normalizeSQLiteSQL(want.where)) {
		return false, nil
	}
	return true, nil
}

func normalizeSQLiteSQL(value string) string {
	return strings.Join(strings.Fields(strings.ToLower(value)), " ")
}

func quoteSQLiteIdentifier(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
