package paidsub

import "gorm.io/gorm"

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
			provider_charge_id TEXT,
			provider_payload BLOB,
			external_url TEXT,
			created_at INTEGER NOT NULL DEFAULT 0,
			paid_at INTEGER NOT NULL DEFAULT 0,
			expires_at INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_paidsub_bindings_client ON paidsub_bindings(client_id)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_paidsub_bindings_tg ON paidsub_bindings(tg_user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_tariffs_enabled_sort ON tariffs(enabled, sort)`,
		`CREATE INDEX IF NOT EXISTS idx_payment_orders_client ON payment_orders(client_id, status)`,
		`CREATE INDEX IF NOT EXISTS idx_payment_orders_pending_poll ON payment_orders(provider, status, created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_payment_orders_telegram ON payment_orders(telegram_user_id)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_payment_orders_idem ON payment_orders(idempotency_key)`,
		// Partial unique index: many pending orders have an empty charge id, so
		// the uniqueness only applies once a provider charge id is recorded.
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_payment_orders_charge ON payment_orders(provider, provider_charge_id) WHERE provider_charge_id != ''`,
	}
	for _, stmt := range stmts {
		if err := db.Exec(stmt).Error; err != nil {
			return err
		}
	}
	return nil
}
