package migration

import (
	"bytes"
	"encoding/json"

	"gorm.io/gorm"
)

// AWG 3.0 default timing values written by the migration for endpoints that
// carried an AWG 2.0 amnezia profile. They mirror the built-in defaults of
// wireguard-go (device/constants.go: RekeyAfterTime 120, RekeyTimeout 5,
// RejectAfterTime 180, KeepaliveTimeout 10; MaxTimerHandshakes = 90/5 = 18)
// as single-value ranges, so a migrated endpoint keeps exactly the timing
// behavior it had before the upgrade. ContentPaddingAddition 0 means "no
// extra padding", the same as the field being absent. HeaderProtectionKey is
// deliberately NOT defaulted: it is a server-side key that must match the
// client and requires S1-S4 >= 12, so the migration leaves it unset (header
// protection stays off, vanilla behavior).
var awg30MigrationDefaults = map[string]any{
	"content_padding_addition": "0",
	"rekey_after_time":         "120",
	"rekey_timeout":            "5",
	"reject_after_time":        "180",
	"keepalive_timeout":        "10",
	"max_handshake_attempts":   "18",
}

// to1_8 migrates AWG 2.0 amnezia options in endpoints.options to the AWG 3.0
// schema and clears fields the 2.6.x kernel no longer understands:
//
//   - wireguard endpoints: the obsolete junk/init fields j1/j2/j3/itime are
//     removed (they no longer exist in wireguard-go v0.0.4 UAPI);
//   - warp endpoints: s1..s4, h1..h4 and header_protection_key are removed —
//     WARPAmnezia in 2.6.x carries only jc/jmin/jmax/i1..i5 plus the timing
//     fields, and the kernel silently ignores the old shared-schema fields;
//   - both types get the new 3.0 timing fields defaulted, so existing
//     endpoints keep working after the kernel switch.
//
// Rows whose options do not change are left untouched (byte-identical rewrite
// avoided so the migration is idempotent and does not invalidate
// optimistic-concurrency guards elsewhere).
func to1_8(db *gorm.DB) error {
	if !db.Migrator().HasTable("endpoints") {
		return nil
	}
	type endpointRow struct {
		Id      uint
		Type    string
		Options []byte
	}
	var rows []endpointRow
	if err := db.Raw("SELECT id, type, options FROM endpoints").Scan(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		if (row.Type != "wireguard" && row.Type != "warp") || len(row.Options) == 0 {
			continue
		}
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(row.Options, &raw); err != nil {
			// Leave malformed rows alone: failing the whole upgrade on a
			// hand-edited row would brick the panel.
			continue
		}
		amneziaRaw, ok := raw["amnezia"]
		if !ok || string(amneziaRaw) == "null" {
			continue
		}
		var amnezia map[string]json.RawMessage
		if err := json.Unmarshal(amneziaRaw, &amnezia); err != nil {
			continue
		}
		legacyFields := []string{"j1", "j2", "j3", "itime"}
		if row.Type == "warp" {
			legacyFields = []string{"s1", "s2", "s3", "s4", "h1", "h2", "h3", "h4", "header_protection_key"}
		}
		changed := false
		for _, legacy := range legacyFields {
			if _, present := amnezia[legacy]; present {
				delete(amnezia, legacy)
				changed = true
			}
		}
		for key, value := range awg30MigrationDefaults {
			if _, present := amnezia[key]; !present {
				encoded, err := json.Marshal(value)
				if err != nil {
					return err
				}
				amnezia[key] = encoded
				changed = true
			}
		}
		if !changed {
			continue
		}
		encodedAmnezia, err := json.Marshal(amnezia)
		if err != nil {
			return err
		}
		raw["amnezia"] = encodedAmnezia
		next, err := json.MarshalIndent(raw, "", "  ")
		if err != nil {
			return err
		}
		if bytes.Equal(bytes.TrimSpace(row.Options), bytes.TrimSpace(next)) {
			continue
		}
		if err := db.Exec("UPDATE endpoints SET options = ? WHERE id = ?", json.RawMessage(next), row.Id).Error; err != nil {
			return err
		}
	}
	return nil
}
