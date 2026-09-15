package migrateutil

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x-extended/database/model"
	"gorm.io/gorm"
)

func createACMECertificateProviderTable(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.Exec(`
CREATE TABLE tls (
	id integer PRIMARY KEY AUTOINCREMENT,
	name text,
	server blob,
	client blob
)`).Error; err != nil {
		t.Fatal(err)
	}
}

func insertTLS(t *testing.T, db *gorm.DB, name, server string) {
	t.Helper()
	if err := db.Exec("INSERT INTO tls(name, server) VALUES(?, ?)", name, []byte(server)).Error; err != nil {
		t.Fatal(err)
	}
}

func tlsServerForName(t *testing.T, db *gorm.DB, name string) map[string]any {
	t.Helper()
	var row model.Tls
	if err := db.Where("name = ?", name).First(&row).Error; err != nil {
		t.Fatal(err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(row.Server, &parsed); err != nil {
		t.Fatal(err)
	}
	return parsed
}

func TestMigrateACMEToCertificateProviderConverts(t *testing.T) {
	db := openLegacyInboundMigrationTestDB(t)
	createACMECertificateProviderTable(t, db)

	server := `{
		"enabled": true,
		"server_name": "example.com",
		"acme": {
			"domain": ["example.com"],
			"email": "admin@example.com",
			"provider": "letsencrypt",
			"data_directory": "/var/lib/acme",
			"disable_http_challenge": false,
			"alternative_http_port": 8080
		}
	}`
	insertTLS(t, db, "main", server)

	if err := MigrateACMEToCertificateProvider(db); err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	got := tlsServerForName(t, db, "main")
	if _, present := got["acme"]; present {
		t.Fatalf("acme key still present after migration")
	}
	cp, ok := got["certificate_provider"].(map[string]any)
	if !ok {
		t.Fatalf("certificate_provider missing: %v", got)
	}
	if cp["type"] != "acme" {
		t.Fatalf("certificate_provider.type=%v, want acme", cp["type"])
	}
	if cp["email"] != "admin@example.com" || cp["provider"] != "letsencrypt" {
		t.Fatalf("acme fields not carried: %v", cp)
	}
	if cp["data_directory"] != "/var/lib/acme" {
		t.Fatalf("data_directory not carried: %v", cp)
	}
	// enabled/server_name untouched
	if got["enabled"] != true || got["server_name"] != "example.com" {
		t.Fatalf("sibling TLS fields disturbed: %v", got)
	}
}

func TestMigrateACMEToCertificateProviderSkipsExistingProvider(t *testing.T) {
	db := openLegacyInboundMigrationTestDB(t)
	createACMECertificateProviderTable(t, db)

	// Both present: provider wins, acme remnant dropped, provider preserved.
	server := `{
		"enabled": true,
		"acme": {"domain": ["a.com"]},
		"certificate_provider": {"type": "acme", "domain": ["b.com"]}
	}`
	insertTLS(t, db, "both", server)

	if err := MigrateACMEToCertificateProvider(db); err != nil {
		t.Fatalf("migration failed: %v", err)
	}
	got := tlsServerForName(t, db, "both")
	if _, present := got["acme"]; present {
		t.Fatalf("acme remnant not dropped")
	}
	cp := got["certificate_provider"].(map[string]any)
	domains := cp["domain"].([]any)
	if len(domains) != 1 || domains[0] != "b.com" {
		t.Fatalf("existing certificate_provider overwritten: %v", cp)
	}
}

func TestMigrateACMEToCertificateProviderDropsEmptyACME(t *testing.T) {
	db := openLegacyInboundMigrationTestDB(t)
	createACMECertificateProviderTable(t, db)

	insertTLS(t, db, "empty", `{"enabled": true, "acme": null}`)
	if err := MigrateACMEToCertificateProvider(db); err != nil {
		t.Fatalf("migration failed: %v", err)
	}
	got := tlsServerForName(t, db, "empty")
	if _, present := got["acme"]; present {
		t.Fatalf("null acme not dropped")
	}
	if _, present := got["certificate_provider"]; present {
		t.Fatalf("certificate_provider created from empty acme")
	}
	if got["enabled"] != true {
		t.Fatalf("sibling fields disturbed")
	}
}

func TestMigrateACMEToCertificateProviderNoACMEIsNoOp(t *testing.T) {
	db := openLegacyInboundMigrationTestDB(t)
	createACMECertificateProviderTable(t, db)

	insertTLS(t, db, "plain", `{"enabled": true, "server_name": "x.com"}`)
	before := tlsServerForName(t, db, "plain")
	if err := MigrateACMEToCertificateProvider(db); err != nil {
		t.Fatalf("migration failed: %v", err)
	}
	after := tlsServerForName(t, db, "plain")
	if len(before) != len(after) {
		t.Fatalf("no-acme row modified")
	}
}

func TestMigrateACMEToCertificateProviderIdempotent(t *testing.T) {
	db := openLegacyInboundMigrationTestDB(t)
	createACMECertificateProviderTable(t, db)

	insertTLS(t, db, "once", `{"enabled": true, "acme": {"domain": ["x.com"], "email": "a@b.c"}}`)
	if err := MigrateACMEToCertificateProvider(db); err != nil {
		t.Fatalf("first pass failed: %v", err)
	}
	first := tlsServerForName(t, db, "once")
	if err := MigrateACMEToCertificateProvider(db); err != nil {
		t.Fatalf("second pass failed: %v", err)
	}
	second := tlsServerForName(t, db, "once")
	firstJSON, _ := json.Marshal(first)
	secondJSON, _ := json.Marshal(second)
	if string(firstJSON) != string(secondJSON) {
		t.Fatalf("not idempotent:\nfirst=%s\nsecond=%s", firstJSON, secondJSON)
	}
}

func TestMigrateACMEToCertificateProviderRejectsUnknownField(t *testing.T) {
	db := openLegacyInboundMigrationTestDB(t)
	createACMECertificateProviderTable(t, db)

	insertTLS(t, db, "unknown", `{"acme": {"domain": ["x.com"], "some_future_acme_field": 1}}`)
	err := MigrateACMEToCertificateProvider(db)
	if err == nil || !strings.Contains(err.Error(), "unsupported legacy acme field") {
		t.Fatalf("expected unsupported-field error, got %v", err)
	}
}
