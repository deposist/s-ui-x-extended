package migrateutil

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/sagernet/sing-box/option"
	sbjson "github.com/sagernet/sing/common/json"
	"gorm.io/gorm"
)

func createOpenVPNMigrationTables(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.Exec(`
CREATE TABLE outbounds (
	id integer PRIMARY KEY AUTOINCREMENT,
	type text,
	tag text,
	options blob
)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`
CREATE TABLE endpoints (
	id integer PRIMARY KEY AUTOINCREMENT,
	type text,
	tag text,
	ext blob,
	options blob
)`).Error; err != nil {
		t.Fatal(err)
	}
}

func insertOutbound(t *testing.T, db *gorm.DB, typ, tag, options string) {
	t.Helper()
	if err := db.Exec("INSERT INTO outbounds(type, tag, options) VALUES(?, ?, ?)", typ, tag, []byte(options)).Error; err != nil {
		t.Fatal(err)
	}
}

func insertEndpoint(t *testing.T, db *gorm.DB, typ, tag, options string) {
	t.Helper()
	if err := db.Exec("INSERT INTO endpoints(type, tag, options) VALUES(?, ?, ?)", typ, tag, []byte(options)).Error; err != nil {
		t.Fatal(err)
	}
}

func endpointOptionsForTag(t *testing.T, db *gorm.DB, tag string) map[string]any {
	t.Helper()
	var endpoint model.Endpoint
	if err := db.Where("tag = ?", tag).First(&endpoint).Error; err != nil {
		t.Fatal(err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(endpoint.Options, &parsed); err != nil {
		t.Fatal(err)
	}
	return parsed
}

func outboundCountByTag(t *testing.T, db *gorm.DB, tag string) int64 {
	t.Helper()
	var count int64
	if err := db.Model(&model.Outbound{}).Where("tag = ?", tag).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	return count
}

func TestMigrateOpenVPNOutboundToEndpointConvertsTLSClient(t *testing.T) {
	db := openLegacyInboundMigrationTestDB(t)
	createOpenVPNMigrationTables(t, db)

	options := `{
		"proto": "udp4",
		"cipher": "aes-256-gcm",
		"auth": "sha256",
		"username": "user",
		"password": "pass",
		"tls_auth": "-----BEGIN tls-auth-----",
		"key_direction": 1,
		"allowed_ips": ["10.8.0.0/24"],
		"servers": [{"server":"vpn.example.com","server_port":1194}],
		"reconnect_delay": "5s",
		"ping_interval": "10s",
		"tls": {
			"ca": "CA-PEM",
			"ca_path": "/etc/ca.pem",
			"certificate": "CLIENT-CERT",
			"key": "CLIENT-KEY",
			"cipher_suites": ["TLS_AES_256_GCM_SHA384","TLS_CHACHA20_POLY1305_SHA256"],
			"verify_x509_name": "server",
			"verify_x509_name_mode": "name"
		}
	}`
	insertOutbound(t, db, "openvpn", "ovpn-out", options)

	if err := MigrateOpenVPNOutboundToEndpoint(db); err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	if got := outboundCountByTag(t, db, "ovpn-out"); got != 0 {
		t.Fatalf("openvpn outbound row still present, count=%d", got)
	}
	got := endpointOptionsForTag(t, db, "ovpn-out")

	if got["mode"] != "tls" {
		t.Fatalf("mode=%v, want tls", got["mode"])
	}
	if got["network"] != "udp" {
		t.Fatalf("network=%v, want udp (udp4 normalized)", got["network"])
	}
	if got["auth"] != "sha256" {
		t.Fatalf("auth=%v", got["auth"])
	}
	// cipher → data_ciphers, single element, uppercased
	dc, ok := got["data_ciphers"].([]any)
	if !ok || len(dc) != 1 || dc[0] != "AES-256-GCM" {
		t.Fatalf("data_ciphers=%v, want [AES-256-GCM]", got["data_ciphers"])
	}
	if _, present := got["cipher"]; present {
		t.Fatalf("cipher leaked into TLS-mode endpoint")
	}
	// allowed_ips → routes
	routes, ok := got["routes"].([]any)
	if !ok || len(routes) != 1 || routes[0] != "10.8.0.0/24" {
		t.Fatalf("routes=%v", got["routes"])
	}

	tls, ok := got["tls"].(map[string]any)
	if !ok {
		t.Fatalf("tls missing: %v", got)
	}
	if tls["certificate"] != "CA-PEM" || tls["certificate_path"] != "/etc/ca.pem" {
		t.Fatalf("CA material not preserved: %v", tls)
	}
	if tls["client_certificate"] != "CLIENT-CERT" || tls["client_key"] != "CLIENT-KEY" {
		t.Fatalf("client material not moved: %v", tls)
	}
	if tls["cipher"] != "TLS_AES_256_GCM_SHA384:TLS_CHACHA20_POLY1305_SHA256" {
		t.Fatalf("cipher=%v", tls["cipher"])
	}
	if tls["server_name"] != "server" || tls["server_name_type"] != "name" {
		t.Fatalf("verify_x509_name mapping wrong: %v", tls)
	}
	cw, ok := tls["control_wrap"].(map[string]any)
	if !ok {
		t.Fatalf("control_wrap missing: %v", tls)
	}
	if cw["type"] != "tls_auth" {
		t.Fatalf("control_wrap.type=%v, want tls_auth", cw["type"])
	}
	keys, ok := cw["key"].([]any)
	if !ok || len(keys) != 1 || keys[0] != "-----BEGIN tls-auth-----" {
		t.Fatalf("control_wrap.key=%v", cw["key"])
	}
	if cw["direction"] != "client" {
		t.Fatalf("control_wrap.direction=%v, want client (key_direction 1)", cw["direction"])
	}
}

func TestMigrateOpenVPNOutboundToEndpointDefaultsDirectionToServer(t *testing.T) {
	db := openLegacyInboundMigrationTestDB(t)
	createOpenVPNMigrationTables(t, db)

	// tls_auth present but no key_direction: legacy effective direction was 0
	// (server), not no-direction.
	options := `{"tls_auth":"KEY"}`
	insertOutbound(t, db, "openvpn", "auth-default", options)
	if err := MigrateOpenVPNOutboundToEndpoint(db); err != nil {
		t.Fatalf("migration failed: %v", err)
	}
	got := endpointOptionsForTag(t, db, "auth-default")
	cw := got["tls"].(map[string]any)["control_wrap"].(map[string]any)
	if cw["direction"] != "server" {
		t.Fatalf("direction=%v, want server for absent key_direction", cw["direction"])
	}
}

func TestMigrateOpenVPNOutboundToEndpointCryptV2(t *testing.T) {
	db := openLegacyInboundMigrationTestDB(t)
	createOpenVPNMigrationTables(t, db)

	options := `{"tls_crypt_v2": true, "tls_crypt": "CRYPT-KEY"}`
	insertOutbound(t, db, "openvpn", "crypt-v2", options)
	if err := MigrateOpenVPNOutboundToEndpoint(db); err != nil {
		t.Fatalf("migration failed: %v", err)
	}
	got := endpointOptionsForTag(t, db, "crypt-v2")
	cw := got["tls"].(map[string]any)["control_wrap"].(map[string]any)
	if cw["type"] != "tls_crypt_v2" {
		t.Fatalf("control_wrap.type=%v, want tls_crypt_v2", cw["type"])
	}
	if _, present := cw["direction"]; present {
		t.Fatalf("direction must be absent for tls_crypt branch")
	}
}

func TestMigrateOpenVPNOutboundToEndpointTagCollision(t *testing.T) {
	db := openLegacyInboundMigrationTestDB(t)
	createOpenVPNMigrationTables(t, db)

	insertOutbound(t, db, "openvpn", "dup", `{"auth":"sha1"}`)
	insertEndpoint(t, db, "wireguard", "dup", `{}`)

	err := MigrateOpenVPNOutboundToEndpoint(db)
	if err == nil || !strings.Contains(err.Error(), "already uses this tag") {
		t.Fatalf("expected tag collision error, got %v", err)
	}
	// Rollback semantics: the outbound row must be untouched.
	if got := outboundCountByTag(t, db, "dup"); got != 1 {
		t.Fatalf("outbound row modified despite collision")
	}
}

func TestMigrateOpenVPNOutboundToEndpointNestedBondAborts(t *testing.T) {
	db := openLegacyInboundMigrationTestDB(t)
	createOpenVPNMigrationTables(t, db)

	bond := `{"outbounds":[{"outbound":{"type":"openvpn","tag":"nested-ovpn"},"download_ratio":1,"upload_ratio":1}]}`
	insertOutbound(t, db, "bond", "mybond", bond)
	insertOutbound(t, db, "openvpn", "top-ovpn", `{"auth":"sha1"}`)

	err := MigrateOpenVPNOutboundToEndpoint(db)
	if err == nil || !strings.Contains(err.Error(), "endpoint equivalent") {
		t.Fatalf("expected nested-bond abort, got %v", err)
	}
	// Nothing migrated: the top-level openvpn must also remain.
	if got := outboundCountByTag(t, db, "top-ovpn"); got != 1 {
		t.Fatalf("top-level openvpn migrated despite nested-bond abort")
	}
}

func TestMigrateOpenVPNOutboundToEndpointRejectsBadProto(t *testing.T) {
	db := openLegacyInboundMigrationTestDB(t)
	createOpenVPNMigrationTables(t, db)

	insertOutbound(t, db, "openvpn", "badproto", `{"proto":"gre"}`)
	err := MigrateOpenVPNOutboundToEndpoint(db)
	if err == nil || !strings.Contains(err.Error(), "proto") {
		t.Fatalf("expected proto error, got %v", err)
	}
}

func TestMigrateOpenVPNOutboundToEndpointRejectsBadKeyDirection(t *testing.T) {
	db := openLegacyInboundMigrationTestDB(t)
	createOpenVPNMigrationTables(t, db)

	insertOutbound(t, db, "openvpn", "badkd", `{"tls_auth":"K","key_direction":7}`)
	err := MigrateOpenVPNOutboundToEndpoint(db)
	if err == nil || !strings.Contains(err.Error(), "key_direction") {
		t.Fatalf("expected key_direction error, got %v", err)
	}
}

func TestMigrateOpenVPNOutboundToEndpointIdempotent(t *testing.T) {
	db := openLegacyInboundMigrationTestDB(t)
	createOpenVPNMigrationTables(t, db)

	insertOutbound(t, db, "openvpn", "once", `{"auth":"sha1"}`)
	if err := MigrateOpenVPNOutboundToEndpoint(db); err != nil {
		t.Fatalf("first pass failed: %v", err)
	}
	// Second pass: no openvpn outbounds remain, must be a no-op.
	if err := MigrateOpenVPNOutboundToEndpoint(db); err != nil {
		t.Fatalf("second pass failed: %v", err)
	}
	var count int64
	if err := db.Model(&model.Endpoint{}).Where("tag = ?", "once").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected exactly one endpoint, got %d", count)
	}
}

func TestMigrateOpenVPNOutboundToEndpointRejectsUnsupportedField(t *testing.T) {
	db := openLegacyInboundMigrationTestDB(t)
	createOpenVPNMigrationTables(t, db)

	insertOutbound(t, db, "openvpn", "unknown", `{"auth":"sha1","some_future_field":123}`)
	err := MigrateOpenVPNOutboundToEndpoint(db)
	if err == nil || !strings.Contains(err.Error(), "unsupported legacy field") {
		t.Fatalf("expected unsupported-field error, got %v", err)
	}
}

func TestMigrateOpenVPNOutboundToEndpointLeavesOtherOutbounds(t *testing.T) {
	db := openLegacyInboundMigrationTestDB(t)
	createOpenVPNMigrationTables(t, db)

	insertOutbound(t, db, "vless", "myvless", `{"server":"x"}`)
	insertOutbound(t, db, "openvpn", "myovpn", `{"auth":"sha1"}`)
	if err := MigrateOpenVPNOutboundToEndpoint(db); err != nil {
		t.Fatalf("migration failed: %v", err)
	}
	if got := outboundCountByTag(t, db, "myvless"); got != 1 {
		t.Fatalf("non-openvpn outbound removed")
	}
}

// TestMigratedOptionsParseAgainstNewCoreSchema is the byte-level acceptance
// check: whatever the migrator emits must be accepted by the 1.14
// openvpn-client endpoint options under a strict unknown-field-rejecting parse,
// otherwise the migrated config would fail to start the new core.
func TestMigratedOptionsParseAgainstNewCoreSchema(t *testing.T) {
	db := openLegacyInboundMigrationTestDB(t)
	createOpenVPNMigrationTables(t, db)

	options := `{
		"proto": "tcp4-client",
		"cipher": "AES-256-GCM",
		"auth": "sha256",
		"tls_auth": "-----BEGIN OpenVPN Static key V1-----\\nabc\\n-----END OpenVPN Static key V1-----",
		"key_direction": 0,
		"allowed_ips": ["10.8.0.0/24", "fd00::/64"],
		"servers": [{"server":"vpn.example.com","server_port":1194}],
		"reconnect_delay": "5s",
		"ping_interval": "10s",
		"ping_restart": "60s",
		"system": false,
		"name": "tun0",
		"detour": "direct",
		"connect_timeout": "5s",
		"tls": {
			"ca": "CA-PEM",
			"certificate": "CLIENT-CERT",
			"key": "CLIENT-KEY",
			"cipher_suites": ["TLS_AES_256_GCM_SHA384"],
			"verify_x509_name": "vpn.example.com",
			"verify_x509_name_mode": "name"
		}
	}`
	insertOutbound(t, db, "openvpn", "accept", options)

	if err := MigrateOpenVPNOutboundToEndpoint(db); err != nil {
		t.Fatalf("migration failed: %v", err)
	}
	var endpoint model.Endpoint
	if err := db.Where("tag = ?", "accept").First(&endpoint).Error; err != nil {
		t.Fatal(err)
	}
	// Decode the options blob exactly the way the core does after the endpoint
	// envelope (type/tag) has been consumed by badjson.UnmarshallExcluded.
	docBytes := []byte(endpoint.Options)
	var decoded option.OpenVPNClientEndpointOptions
	if err := sbjson.UnmarshalContextDisallowUnknownFields(context.Background(), docBytes, &decoded); err != nil {
		t.Fatalf("migrated options rejected by 1.14 schema: %v\n%s", err, docBytes)
	}
	if decoded.Mode != "tls" {
		t.Fatalf("mode=%q, want tls", decoded.Mode)
	}
	if decoded.Network != "tcp" {
		t.Fatalf("network=%q, want tcp", decoded.Network)
	}
	if decoded.TLS == nil || decoded.TLS.ControlWrap == nil {
		t.Fatalf("control_wrap missing after strict parse")
	}
	if decoded.TLS.ControlWrap.Direction != "server" {
		t.Fatalf("direction=%q, want server", decoded.TLS.ControlWrap.Direction)
	}
}

var _ = json.Marshal // keep json import used across builds
