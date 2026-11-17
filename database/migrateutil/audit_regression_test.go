package migrateutil

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"gorm.io/gorm"
)

func TestDNSMigrationReturnsConfigReadFailure(t *testing.T) {
	db := openLegacyInboundMigrationTestDB(t)
	createSettingTable(t, db)
	setConfigBlob(t, db, `{"dns":{"independent_cache":true}}`)
	wantErr := errors.New("injected config read failure")
	if err := db.Callback().Query().Before("gorm:query").Register("audit_config_read_failure", func(tx *gorm.DB) {
		if tx.Statement.Table == "settings" {
			_ = tx.AddError(wantErr)
		}
	}); err != nil {
		t.Fatal(err)
	}
	if err := MigrateDNS114(db); !errors.Is(err, wantErr) {
		t.Fatalf("expected config read failure, got %v", err)
	}
}

func TestDNSMigrationRejectsNestedLegacyAddressFilter(t *testing.T) {
	for _, rules := range []string{
		`[{"ip_cidr":["10.0.0.0/8"]}]`,
		`[{"type":"logical","mode":"or","rules":[{"ip_is_private":true}]}]`,
	} {
		raw := json.RawMessage(`{"type":"logical","mode":"and","rules":` + rules + `,"action":"route","server":"local"}`)
		if _, _, err := convertDNSAddressFilterRule(raw, 0); err == nil || !strings.Contains(err.Error(), "logical rule") {
			t.Fatalf("expected nested logical-rule rejection for %s, got %v", raw, err)
		}
	}
}

func TestDNSMigrationPreservesLogicalResponseFilter(t *testing.T) {
	raw := json.RawMessage(`{"type":"logical","mode":"and","match_response":true,"rules":[{"ip_cidr":["10.0.0.0/8"]}],"action":"route","server":"local"}`)
	converted, changed, err := convertDNSAddressFilterRule(raw, 0)
	if err != nil || changed || len(converted) != 1 || string(converted[0]) != string(raw) {
		t.Fatalf("new-form logical response filter changed: result=%s changed=%v err=%v", converted, changed, err)
	}
}
