package migration

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/deposist/s-ui-x/util/common"
	"github.com/gofrs/uuid/v5"
	"gorm.io/gorm"
)

func to1_5(db *gorm.DB) error {
	if err := addColumnIfMissing(db, "clients", "limit_ip", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	if err := addColumnIfMissing(db, "clients", "ip_limit_mode", "TEXT NOT NULL DEFAULT 'monitor'"); err != nil {
		return err
	}
	if err := addColumnIfMissing(db, "clients", "last_online", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	if err := addColumnIfMissing(db, "clients", "last_ip_count", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	if err := addColumnIfMissing(db, "clients", "sub_secret", "TEXT"); err != nil {
		return err
	}
	if err := createClientIPsTable(db); err != nil {
		return err
	}
	if err := backfillClientIPHashes(db); err != nil {
		return err
	}
	if err := backfillClientSubSecrets(db); err != nil {
		return err
	}
	return nil
}

func createClientIPsTable(db *gorm.DB) error {
	if err := db.Exec(`
CREATE TABLE IF NOT EXISTS client_ips (
	id integer PRIMARY KEY AUTOINCREMENT,
	client_name text,
	ip text,
	ip_hash text,
	ip_display text,
	first_seen integer,
	last_seen integer
)`).Error; err != nil {
		return err
	}
	if err := addColumnIfMissing(db, "client_ips", "ip_hash", "TEXT"); err != nil {
		return err
	}
	if err := addColumnIfMissing(db, "client_ips", "ip_display", "TEXT"); err != nil {
		return err
	}
	if err := db.Exec("DROP INDEX IF EXISTS idx_client_ips_client_ip").Error; err != nil {
		return err
	}
	if err := db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_client_ips_client_hash ON client_ips(client_name, ip_hash)").Error; err != nil {
		return err
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_client_ips_client_legacy_ip ON client_ips(client_name, ip) WHERE ip IS NOT NULL AND ip != ''").Error; err != nil {
		return err
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_client_ips_last_seen ON client_ips(last_seen)").Error; err != nil {
		return err
	}
	return nil
}

func backfillClientIPHashes(db *gorm.DB) error {
	rows, err := db.Raw("SELECT id, ip FROM client_ips WHERE (ip_hash IS NULL OR ip_hash = '') AND ip IS NOT NULL AND ip != ''").Rows()
	if err != nil {
		return err
	}
	defer rows.Close()

	type clientIPRow struct {
		id uint64
		ip string
	}
	clientIPs := make([]clientIPRow, 0)
	for rows.Next() {
		var row clientIPRow
		if err := rows.Scan(&row.id, &row.ip); err != nil {
			return err
		}
		clientIPs = append(clientIPs, row)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(clientIPs) == 0 {
		return nil
	}

	salt, err := migrationInstallSalt(db)
	if err != nil {
		return err
	}
	for _, row := range clientIPs {
		ipHash := row.ip
		if !migrationLooksLikeSHA256Hex(row.ip) {
			ipHash = migrationHashIP(salt, row.ip)
		}
		if err := db.Exec("UPDATE client_ips SET ip_hash = ? WHERE id = ?", ipHash, row.id).Error; err != nil {
			return err
		}
	}
	return nil
}

func migrationInstallSalt(db *gorm.DB) ([]byte, error) {
	var salt string
	err := db.Raw("SELECT value FROM settings WHERE key = ?", "installSalt").Scan(&salt).Error
	if err != nil {
		return nil, err
	}
	if salt != "" {
		return []byte(salt), nil
	}
	salt = common.Random(32)
	var count int64
	if err := db.Raw("SELECT COUNT(*) FROM settings WHERE key = ?", "installSalt").Scan(&count).Error; err != nil {
		return nil, err
	}
	if count == 0 {
		err = db.Exec("INSERT INTO settings(key, value) VALUES(?, ?)", "installSalt", salt).Error
	} else {
		err = db.Exec("UPDATE settings SET value = ? WHERE key = ?", salt, "installSalt").Error
	}
	if err != nil {
		return nil, err
	}
	return []byte(salt), nil
}

func migrationHashIP(salt []byte, ip string) string {
	h := sha256.New()
	_, _ = h.Write(salt)
	_, _ = h.Write([]byte(ip))
	return hex.EncodeToString(h.Sum(nil))
}

func migrationLooksLikeSHA256Hex(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func backfillClientSubSecrets(db *gorm.DB) error {
	rows, err := db.Raw("SELECT id FROM clients WHERE sub_secret IS NULL OR sub_secret = ''").Rows()
	if err != nil {
		return err
	}
	defer rows.Close()

	var clientIDs []uint
	for rows.Next() {
		var id uint
		if err := rows.Scan(&id); err != nil {
			return err
		}
		clientIDs = append(clientIDs, id)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, id := range clientIDs {
		secret, err := uuid.NewV4()
		if err != nil {
			return err
		}
		if err := db.Exec("UPDATE clients SET sub_secret = ? WHERE id = ?", secret.String(), id).Error; err != nil {
			return err
		}
	}
	return nil
}
