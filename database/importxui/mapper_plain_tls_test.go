package importxui

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Single-line PEM placeholders so they stay valid inside a JSON string.
const (
	inlineCert = "-----BEGIN CERTIFICATE-----MIIBfakecertcontent-----END CERTIFICATE-----"
	inlineKey  = "-----BEGIN PRIVATE KEY-----MIIEfakekeycontent-----END PRIVATE KEY-----"
)

func tlsTestInbound(stream string) xuiInboundRow {
	return xuiInboundRow{
		ID:             1,
		Tag:            "tls-in",
		Protocol:       "vless",
		Port:           443,
		Enable:         true,
		Settings:       json.RawMessage(`{"clients":[{"email":"a@x.com","id":"u"}]}`),
		StreamSettings: json.RawMessage(stream),
	}
}

func TestExtractPlainTLS_InlineArrayAndSingle(t *testing.T) {
	// Array-of-strings form.
	rowArr := tlsTestInbound(`{"network":"tcp","security":"tls","tlsSettings":{"serverName":"example.com","alpn":["h2"],"certificates":[{"certificate":["` + inlineCert + `"],"key":["` + inlineKey + `"]}]}}`)
	spec, warnings, err := extractPlainTLS(rowArr)
	if err != nil {
		t.Fatal(err)
	}
	if spec == nil {
		t.Fatalf("expected a spec; warnings=%v", warnings)
	}
	if spec.ServerName != "example.com" || len(spec.Certificate) != 1 || spec.Certificate[0] != inlineCert || len(spec.KeyPEM) != 1 {
		t.Fatalf("unexpected spec: %#v", spec)
	}
	rec, err := buildPlainTLSRecord(*spec)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Name != "tls-example.com" {
		t.Errorf("record name = %q", rec.Name)
	}
	var server map[string]any
	if err := json.Unmarshal(rec.Server, &server); err != nil {
		t.Fatal(err)
	}
	if server["enabled"] != true || server["server_name"] != "example.com" {
		t.Errorf("server block = %v", server)
	}
	if c, _ := server["certificate"].([]any); len(c) != 1 || !strings.Contains(c[0].(string), "BEGIN CERTIFICATE") {
		t.Errorf("server certificate = %v", server["certificate"])
	}
	if _, ok := server["key"]; !ok {
		t.Errorf("server block missing key: %v", server)
	}

	// Single-string form must also parse (flexStringList).
	rowStr := tlsTestInbound(`{"network":"tcp","security":"tls","tlsSettings":{"serverName":"single.com","certificates":[{"certificate":"` + inlineCert + `","key":"` + inlineKey + `"}]}}`)
	specStr, _, err := extractPlainTLS(rowStr)
	if err != nil || specStr == nil {
		t.Fatalf("single-string cert should parse: spec=%v err=%v", specStr, err)
	}
	if len(specStr.Certificate) != 1 || specStr.Certificate[0] != inlineCert {
		t.Errorf("single-string cert = %v", specStr.Certificate)
	}
}

func TestExtractPlainTLS_FilePathOnlyWarns(t *testing.T) {
	row := tlsTestInbound(`{"network":"tcp","security":"tls","tlsSettings":{"serverName":"example.com","certificates":[{"certificateFile":"/etc/ssl/cert.pem","keyFile":"/etc/ssl/key.pem"}]}}`)
	spec, warnings, err := extractPlainTLS(row)
	if err != nil {
		t.Fatal(err)
	}
	if spec != nil {
		t.Fatalf("file-path-only certificate must not produce a spec, got %#v", spec)
	}
	if len(warnings) == 0 || !strings.Contains(warnings[0], "file path") {
		t.Errorf("expected a file-path warning, got %v", warnings)
	}
}

func TestExtractPlainTLS_NonTLSAndReality(t *testing.T) {
	if spec, _, err := extractPlainTLS(tlsTestInbound(`{"security":"reality","realitySettings":{}}`)); err != nil || spec != nil {
		t.Fatalf("reality must yield no plain-tls spec: spec=%v err=%v", spec, err)
	}
	if spec, _, err := extractPlainTLS(tlsTestInbound(`{"security":"none"}`)); err != nil || spec != nil {
		t.Fatalf("no-tls must yield no plain-tls spec: spec=%v err=%v", spec, err)
	}
}

// TestApply_PlainTLSInlineCert_CreatesTLSRecord drives the full Plan/Apply path
// over a source with a plain-TLS inbound carrying an inline certificate, and
// verifies an s-ui TLS record is created and the inbound references it.
func TestApply_PlainTLSInlineCert_CreatesTLSRecord(t *testing.T) {
	initCompatDest(t)
	dir := makeImportXUITempDir(t)
	src := filepath.Join(dir, "x-ui.db")
	buildCompatSource(t, forkVariant, src)

	db, err := gorm.Open(sqlite.Open(src), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	stream := `{"network":"tcp","security":"tls","tlsSettings":{"serverName":"tls.example.com","certificates":[{"certificate":["` + inlineCert + `"],"key":["` + inlineKey + `"]}]}}`
	settings := `{"clients":[{"email":"tlsuser","id":"11111111-1111-1111-1111-111111111111"}]}`
	if err := db.Exec(
		"INSERT INTO inbounds(user_id, up, down, total, remark, enable, expiry_time, listen, port, protocol, settings, stream_settings, tag, sniffing) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)",
		1, 0, 0, 0, "tls-in", 1, 0, "", 8443, "vless", settings, stream, "tls-in", "",
	).Error; err != nil {
		t.Fatal(err)
	}
	if sqlDB, err := db.DB(); err == nil {
		_ = sqlDB.Close()
	}

	plan, err := Plan(src, PlanOptions{Strategy: StrategyMerge, AdminMode: AdminModeSkip})
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if _, err := Apply(src, *plan, ApplyOptions{}); err != nil {
		t.Fatalf("apply: %v", err)
	}

	dest := database.GetDB()
	var tls model.Tls
	if err := dest.Where("name = ?", "tls-tls.example.com").First(&tls).Error; err != nil {
		t.Fatalf("plain TLS record not created: %v", err)
	}
	if !strings.Contains(string(tls.Server), "BEGIN CERTIFICATE") {
		t.Errorf("server certificate not stored: %s", tls.Server)
	}
	var in model.Inbound
	if err := dest.Where("tag = ?", "tls-in").First(&in).Error; err != nil {
		t.Fatalf("inbound missing: %v", err)
	}
	if in.TlsId == 0 {
		t.Errorf("inbound should reference the migrated TLS record, got TlsId=0")
	}
}

// TestApply_PlainTLSInlineCert_DedupSharedCert verifies two inbounds sharing one
// inline certificate produce a single TLS record that both reference (the dedup
// path must still resolve a non-zero TlsId for the second inbound).
func TestApply_PlainTLSInlineCert_DedupSharedCert(t *testing.T) {
	initCompatDest(t)
	dir := makeImportXUITempDir(t)
	src := filepath.Join(dir, "x-ui.db")
	buildCompatSource(t, forkVariant, src)

	db, err := gorm.Open(sqlite.Open(src), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	stream := `{"network":"tcp","security":"tls","tlsSettings":{"serverName":"shared.example.com","certificates":[{"certificate":["` + inlineCert + `"],"key":["` + inlineKey + `"]}]}}`
	rows := []struct {
		tag, email, id string
		port           int
	}{
		{"tls-a", "ua@x.com", "11111111-1111-1111-1111-1111111111aa", 8443},
		{"tls-b", "ub@x.com", "11111111-1111-1111-1111-1111111111bb", 8444},
	}
	for _, r := range rows {
		settings := `{"clients":[{"email":"` + r.email + `","id":"` + r.id + `"}]}`
		if err := db.Exec(
			"INSERT INTO inbounds(user_id, up, down, total, remark, enable, expiry_time, listen, port, protocol, settings, stream_settings, tag, sniffing) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)",
			1, 0, 0, 0, r.tag, 1, 0, "", r.port, "vless", settings, stream, r.tag, "",
		).Error; err != nil {
			t.Fatal(err)
		}
	}
	if sqlDB, err := db.DB(); err == nil {
		_ = sqlDB.Close()
	}

	plan, err := Plan(src, PlanOptions{Strategy: StrategyMerge, AdminMode: AdminModeSkip})
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if _, err := Apply(src, *plan, ApplyOptions{}); err != nil {
		t.Fatalf("apply: %v", err)
	}

	dest := database.GetDB()
	var count int64
	dest.Model(&model.Tls{}).Where("name = ?", "tls-shared.example.com").Count(&count)
	if count != 1 {
		t.Fatalf("shared certificate should create exactly 1 TLS record, got %d", count)
	}
	var a, b model.Inbound
	if err := dest.Where("tag = ?", "tls-a").First(&a).Error; err != nil {
		t.Fatal(err)
	}
	if err := dest.Where("tag = ?", "tls-b").First(&b).Error; err != nil {
		t.Fatal(err)
	}
	if a.TlsId == 0 || b.TlsId == 0 {
		t.Fatalf("both inbounds must reference the shared TLS record: a=%d b=%d", a.TlsId, b.TlsId)
	}
	if a.TlsId != b.TlsId {
		t.Errorf("shared certificate should yield the same TlsId: a=%d b=%d", a.TlsId, b.TlsId)
	}
}
