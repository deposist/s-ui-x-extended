package migration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x-extended/config"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// buildCurrentVersionDB creates a database stamped with the current binary
// version so MigrateDb takes the data-migration path, pre-loaded with the three
// legacy 1.13 shapes: an openvpn outbound, an inline tls.acme block, and a
// deprecated DNS config.
func buildCurrentVersionDB(t *testing.T) (dbPath string) {
	t.Helper()
	dbDir := t.TempDir()
	t.Setenv("SUI_DB_FOLDER", dbDir)
	dbPath = filepath.Join(dbDir, config.GetName()+".db")

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		if strings.Contains(err.Error(), "go-sqlite3 requires cgo") {
			t.Skip(err)
		}
		t.Fatal(err)
	}
	stmts := []string{
		`CREATE TABLE settings (id integer PRIMARY KEY AUTOINCREMENT, key text, value text)`,
		`CREATE TABLE outbounds (id integer PRIMARY KEY AUTOINCREMENT, type text, tag text, options blob)`,
		`CREATE TABLE endpoints (id integer PRIMARY KEY AUTOINCREMENT, type text, tag text, ext blob, options blob)`,
		`CREATE TABLE tls (id integer PRIMARY KEY AUTOINCREMENT, name text, server blob, client blob)`,
		`INSERT INTO settings(key, value) VALUES('version', '` + config.GetVersion() + `')`,
		`INSERT INTO settings(key, value) VALUES('config', '{"dns":{"servers":[],"independent_cache":true,"rules":[{"ip_is_private":true,"action":"route","server":"local"}]},"experimental":{"cache_file":{"enabled":true,"store_rdrc":true}}}')`,
	}
	for _, s := range stmts {
		if err := db.Exec(s).Error; err != nil {
			t.Fatalf("setup %q: %v", s, err)
		}
	}
	// blob columns must be written as bytes so gorm scans them into
	// json.RawMessage rather than erroring on a TEXT value.
	if err := db.Exec("INSERT INTO outbounds(type, tag, options) VALUES(?, ?, ?)",
		"openvpn", "ovpn-legacy",
		[]byte(`{"proto":"udp","auth":"sha256","servers":[{"server":"vpn.example.com","server_port":1194}],"tls":{"ca":"CA"}}`)).Error; err != nil {
		t.Fatalf("setup outbound: %v", err)
	}
	if err := db.Exec("INSERT INTO tls(name, server, client) VALUES(?, ?, ?)",
		"main",
		[]byte(`{"enabled":true,"acme":{"domain":["example.com"],"email":"a@b.c"}}`),
		[]byte(`{}`)).Error; err != nil {
		t.Fatalf("setup tls: %v", err)
	}
	if sqlDB, err := db.DB(); err == nil {
		if err := sqlDB.Close(); err != nil {
			t.Fatal(err)
		}
	}
	return dbPath
}

// TestMigrateDbAppliesAllDataMigrationsIdempotently runs the full data-migration
// path on a current-version DB twice and confirms the second run is a no-op.
func TestMigrateDbAppliesAllDataMigrationsIdempotently(t *testing.T) {
	dbPath := buildCurrentVersionDB(t)

	if err := MigrateDb(); err != nil {
		t.Fatalf("first MigrateDb failed: %v", err)
	}

	db := openMigrationDBAtPath(t, dbPath)

	// OpenVPN outbound -> endpoint
	var outboundCount int64
	if err := db.Raw("SELECT COUNT(*) FROM outbounds WHERE type = 'openvpn'").Scan(&outboundCount).Error; err != nil {
		t.Fatal(err)
	}
	if outboundCount != 0 {
		t.Fatalf("openvpn outbound not migrated, count=%d", outboundCount)
	}
	var endpointType string
	if err := db.Raw("SELECT type FROM endpoints WHERE tag = 'ovpn-legacy'").Scan(&endpointType).Error; err != nil {
		t.Fatal(err)
	}
	if endpointType != "openvpn-client" {
		t.Fatalf("endpoint type=%q, want openvpn-client", endpointType)
	}

	// tls.acme -> certificate_provider
	var tlsServer string
	if err := db.Raw("SELECT server FROM tls WHERE name = 'main'").Scan(&tlsServer).Error; err != nil {
		t.Fatal(err)
	}
	// The deprecated "acme" KEY must be gone; the string "acme" still appears as
	// certificate_provider.type, so match the key form `"acme":` specifically.
	if strings.Contains(tlsServer, `"acme":`) {
		t.Fatalf("tls.acme not migrated: %s", tlsServer)
	}
	if !strings.Contains(tlsServer, `"certificate_provider"`) {
		t.Fatalf("certificate_provider missing: %s", tlsServer)
	}

	// DNS deprecations
	var cfg string
	if err := db.Raw("SELECT value FROM settings WHERE key = 'config'").Scan(&cfg).Error; err != nil {
		t.Fatal(err)
	}
	for _, gone := range []string{`"independent_cache"`, `"store_rdrc"`} {
		if strings.Contains(cfg, gone) {
			t.Fatalf("deprecated %s still present: %s", gone, cfg)
		}
	}
	if !strings.Contains(cfg, `"store_dns"`) {
		t.Fatalf("store_dns missing: %s", cfg)
	}
	if !strings.Contains(cfg, `"match_response"`) {
		t.Fatalf("address filter not converted: %s", cfg)
	}

	// Second run must be a no-op (idempotent).
	if err := MigrateDb(); err != nil {
		t.Fatalf("second MigrateDb failed: %v", err)
	}
	var endpointCount int64
	if err := db.Raw("SELECT COUNT(*) FROM endpoints WHERE tag = 'ovpn-legacy'").Scan(&endpointCount).Error; err != nil {
		t.Fatal(err)
	}
	if endpointCount != 1 {
		t.Fatalf("second run duplicated endpoint, count=%d", endpointCount)
	}
}

// TestMigrateDbCurrentVersionWritesNoBackup confirms the data-migration path
// (same binary version) does NOT produce a rollback backup file — the
// idempotent transforms roll back via the transaction on error instead.
func TestMigrateDbCurrentVersionWritesNoBackup(t *testing.T) {
	dbPath := buildCurrentVersionDB(t)
	if err := MigrateDb(); err != nil {
		t.Fatalf("MigrateDb failed: %v", err)
	}
	matches, err := filepath.Glob(dbPath + ".pre-*.bak")
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("current-version data migration must not write a backup, found %v", matches)
	}
	_ = os.RemoveAll(filepath.Dir(dbPath))
}
