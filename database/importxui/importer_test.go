package importxui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestImport_DryRun_NoMutation(t *testing.T) {
	src, _ := setupImportTestDB(t)
	before := tableCounts(t, "inbounds", "endpoints", "tls", "clients", "audit_events")
	report, err := Import(src, Options{DryRun: true, Strategy: StrategyMerge})
	if err != nil {
		t.Fatal(err)
	}
	if report.Summary.Inbounds.Imported == 0 {
		t.Fatalf("dry-run should still report imported inbounds: %#v", report.Summary.Inbounds)
	}
	after := tableCounts(t, "inbounds", "endpoints", "tls", "clients", "audit_events")
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("dry-run mutated table counts: before=%v after=%v", before, after)
	}
}

func TestImport_Reality_DedupAndShape(t *testing.T) {
	src, _ := setupImportTestDB(t)
	if _, err := Import(src, Options{Strategy: StrategyMerge, Now: func() int64 { return 1 }}); err != nil {
		t.Fatal(err)
	}
	rows := tlsByRealityPrivateKey(t, "UF2GdUBplZ268703D0dNVZPZ7DU5PvAhtbZiylmCOHk")
	if len(rows) != 1 {
		t.Fatalf("expected one deduped TLS row for shared private key, got %d", len(rows))
	}
	var server map[string]any
	if err := json.Unmarshal(rows[0].Server, &server); err != nil {
		t.Fatal(err)
	}
	reality := server["reality"].(map[string]any)
	handshake := reality["handshake"].(map[string]any)
	if reality["private_key"] == "" || handshake["server"] != "www.apple.com" {
		t.Fatalf("unexpected reality server shape: %s", rows[0].Server)
	}
}

func TestImport_Trojan_Grpc(t *testing.T) {
	src, _ := setupImportTestDB(t)
	if _, err := Import(src, Options{Strategy: StrategyMerge}); err != nil {
		t.Fatal(err)
	}
	inbound := inboundByTag(t, "inbound-12223")
	if inbound.Type != "trojan" {
		t.Fatalf("expected trojan inbound, got %q", inbound.Type)
	}
	var options map[string]any
	if err := json.Unmarshal(inbound.Options, &options); err != nil {
		t.Fatal(err)
	}
	transport := options["transport"].(map[string]any)
	if transport["type"] != "grpc" || transport["service_name"] != "hello" {
		t.Fatalf("unexpected grpc transport: %#v", transport)
	}
}

func TestImport_Wireguard_AsEndpoint(t *testing.T) {
	src, _ := setupImportTestDB(t)
	if _, err := Import(src, Options{Strategy: StrategyMerge}); err != nil {
		t.Fatal(err)
	}
	var inboundCount int64
	if err := database.GetDB().Model(model.Inbound{}).Where("tag = ?", "inbound-12555").Count(&inboundCount).Error; err != nil {
		t.Fatal(err)
	}
	if inboundCount != 0 {
		t.Fatal("wireguard source inbound must not be stored in inbounds")
	}
	var endpoint model.Endpoint
	if err := database.GetDB().Where("tag = ?", "inbound-12555").First(&endpoint).Error; err != nil {
		t.Fatal(err)
	}
	if endpoint.Type != "wireguard" {
		t.Fatalf("expected wireguard endpoint, got %q", endpoint.Type)
	}
	var options map[string]any
	if err := json.Unmarshal(endpoint.Options, &options); err != nil {
		t.Fatal(err)
	}
	if options["mtu"].(float64) != 1280 || len(options["peers"].([]any)) == 0 {
		t.Fatalf("unexpected wireguard options: %s", endpoint.Options)
	}
}

func TestImport_Clients_AggregateByEmail(t *testing.T) {
	src, _ := setupImportTestDB(t)
	addDuplicateClientToInbound(t, src, "AndPh1", 13)
	if _, err := Import(src, Options{Strategy: StrategyMerge}); err != nil {
		t.Fatal(err)
	}
	client := clientByName(t, "AndPh1")
	var inbounds []uint
	if err := json.Unmarshal(client.Inbounds, &inbounds); err != nil {
		t.Fatal(err)
	}
	if len(inbounds) < 2 {
		t.Fatalf("expected duplicated email to be linked to multiple inbounds, got %v", inbounds)
	}
}

func TestImport_Clients_DefaultsDeterministic(t *testing.T) {
	src1, _ := setupImportTestDB(t)
	if _, err := Import(src1, Options{Strategy: StrategyMerge}); err != nil {
		t.Fatal(err)
	}
	config1 := append([]byte(nil), clientByName(t, "AndPh1").Config...)

	src2, _ := setupImportTestDB(t)
	if _, err := Import(src2, Options{Strategy: StrategyMerge}); err != nil {
		t.Fatal(err)
	}
	config2 := clientByName(t, "AndPh1").Config
	if string(config1) != string(config2) {
		t.Fatal("deterministic client config changed between fresh imports")
	}
}

func TestImport_Idempotent(t *testing.T) {
	src, _ := setupImportTestDB(t)
	if _, err := Import(src, Options{Strategy: StrategyMerge}); err != nil {
		t.Fatal(err)
	}
	before := tableCounts(t, "inbounds", "endpoints", "tls", "clients")
	if _, err := Import(src, Options{Strategy: StrategyMerge}); err != nil {
		t.Fatal(err)
	}
	after := tableCounts(t, "inbounds", "endpoints", "tls", "clients")
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("second import changed row counts: before=%v after=%v", before, after)
	}
}

func TestImport_StrategyReplace_OverwritesInbound(t *testing.T) {
	src, _ := setupImportTestDB(t)
	placeholder := placeholderInbound(t, "inbound-12223")
	if _, err := Import(src, Options{Strategy: StrategyReplace}); err != nil {
		t.Fatal(err)
	}
	inbound := inboundByTag(t, "inbound-12223")
	if inbound.Type != "trojan" {
		t.Fatalf("replace should overwrite inbound type, got %q", inbound.Type)
	}
	if inbound.Id == placeholder.Id {
		t.Fatal("replace should recreate the inbound row")
	}
}

func TestImport_StrategySkip_KeepsExisting(t *testing.T) {
	src, _ := setupImportTestDB(t)
	placeholderInbound(t, "inbound-12223")
	report, err := Import(src, Options{Strategy: StrategySkip})
	if err != nil {
		t.Fatal(err)
	}
	inbound := inboundByTag(t, "inbound-12223")
	if inbound.Type != "http" {
		t.Fatalf("skip should keep existing inbound, got %q", inbound.Type)
	}
	if report.Summary.Inbounds.Skipped == 0 {
		t.Fatalf("skip strategy should report skipped conflicts: %#v", report.Summary.Inbounds)
	}
}

func TestImport_AuditEntryCreated(t *testing.T) {
	src, _ := setupImportTestDB(t)
	if _, err := Import(src, Options{Strategy: StrategyMerge, Now: func() int64 { return 123 }}); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := database.GetDB().Model(model.AuditEvent{}).Where("event = ?", "xui_import").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count == 0 {
		t.Fatal("xui_import audit event was not created")
	}
}

func TestImport_Clients_AllTrafficEmailsPresent(t *testing.T) {
	src, _ := setupImportTestDB(t)
	emails := sourceTrafficEmails(t, src)
	if len(emails) != 42 {
		t.Fatalf("fixture expectation changed: got %d unique emails", len(emails))
	}
	if _, err := Import(src, Options{Strategy: StrategyMerge}); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := database.GetDB().Model(model.Client{}).Where("name IN ?", emails).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if int(count) != len(emails) {
		t.Fatalf("expected all source emails in clients, got %d/%d", count, len(emails))
	}
}

func setupImportTestDB(t *testing.T) (string, string) {
	t.Helper()
	closeMainDBForImportTest(t)
	dir := makeImportXUITempDir(t)
	t.Setenv("SUI_DB_FOLDER", dir)
	src := copyFixture(t, "x-ui.db", filepath.Join(dir, "x-ui.db"))
	dst := copyFixture(t, "s-ui.db", filepath.Join(dir, "s-ui.db"))
	if err := database.InitDB(dst); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		closeMainDBForImportTest(t)
	})
	return src, dst
}

func fixturePath(t *testing.T, name string) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(wd, "..", "..", "test-db", name)
	if _, err := os.Stat(path); err != nil {
		// test-db/ holds real production data and is intentionally not
		// committed to the repository (see .gitignore). Tests that need
		// the fixtures are skipped on CI; run them locally with the
		// fixtures present in test-db/.
		t.Skipf("test-db fixture %q not available: %v", name, err)
	}
	return path
}

func copyFixture(t *testing.T, name string, dst string) string {
	t.Helper()
	data, err := os.ReadFile(fixturePath(t, name))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return dst
}

func closeMainDBForImportTest(t *testing.T) {
	t.Helper()
	if db := database.GetDB(); db != nil {
		_ = db.Exec("PRAGMA wal_checkpoint(TRUNCATE)").Error
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	}
}

func makeImportXUITempDir(tb testing.TB) string {
	tb.Helper()
	dir, err := os.MkdirTemp("", "s-ui-importxui-test-")
	if err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(func() {
		var removeErr error
		for i := 0; i < 20; i++ {
			removeErr = os.RemoveAll(dir)
			if removeErr == nil || os.IsNotExist(removeErr) {
				return
			}
			time.Sleep(time.Duration(i+1) * 10 * time.Millisecond)
		}
		tb.Errorf("remove importxui temp dir %q: %v", dir, removeErr)
	})
	return dir
}

func tableCounts(t *testing.T, tables ...string) map[string]int64 {
	t.Helper()
	counts := map[string]int64{}
	for _, table := range tables {
		var count int64
		if err := database.GetDB().Table(table).Count(&count).Error; err != nil {
			t.Fatal(err)
		}
		counts[table] = count
	}
	return counts
}

func tlsByRealityPrivateKey(t *testing.T, privateKey string) []model.Tls {
	t.Helper()
	var rows []model.Tls
	if err := database.GetDB().Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	var matched []model.Tls
	for _, row := range rows {
		var server struct {
			Reality struct {
				PrivateKey string `json:"private_key"`
			} `json:"reality"`
		}
		if err := json.Unmarshal(row.Server, &server); err == nil && server.Reality.PrivateKey == privateKey {
			matched = append(matched, row)
		}
	}
	return matched
}

func inboundByTag(t *testing.T, tag string) model.Inbound {
	t.Helper()
	var inbound model.Inbound
	if err := database.GetDB().Where("tag = ?", tag).First(&inbound).Error; err != nil {
		t.Fatal(err)
	}
	return inbound
}

func clientByName(t *testing.T, name string) model.Client {
	t.Helper()
	var client model.Client
	if err := database.GetDB().Where("name = ?", name).First(&client).Error; err != nil {
		t.Fatal(err)
	}
	return client
}

func placeholderInbound(t *testing.T, tag string) model.Inbound {
	t.Helper()
	options := json.RawMessage(`{"listen":"0.0.0.0","listen_port":12223}`)
	inbound := model.Inbound{
		Type:    "http",
		Tag:     tag,
		Addrs:   json.RawMessage(`[]`),
		OutJson: json.RawMessage(`{}`),
		Options: options,
	}
	if err := database.GetDB().Create(&inbound).Error; err != nil {
		t.Fatal(err)
	}
	return inbound
}

func addDuplicateClientToInbound(t *testing.T, src string, email string, inboundID int64) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(src), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err == nil {
		defer sqlDB.Close()
	}
	var settingsText string
	if err := db.Raw("SELECT settings FROM inbounds WHERE id = ?", inboundID).Scan(&settingsText).Error; err != nil {
		t.Fatal(err)
	}
	var settings xuiInboundSettings
	if err := json.Unmarshal([]byte(settingsText), &settings); err != nil {
		t.Fatal(err)
	}
	enabled := true
	settings.Clients = append(settings.Clients, xuiClientSetting{
		Email:  email,
		Enable: &enabled,
		ID:     deterministicUUID(email + ":duplicate"),
		Flow:   "xtls-rprx-vision",
		SubID:  "duplicate-sub",
	})
	next, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("UPDATE inbounds SET settings = ? WHERE id = ?", string(next), inboundID).Error; err != nil {
		t.Fatal(err)
	}
}

func sourceTrafficEmails(t *testing.T, srcPath string) []string {
	t.Helper()
	src, err := openSource(srcPath)
	if err != nil {
		t.Fatal(err)
	}
	defer src.close()
	seen := map[string]struct{}{}
	if err := src.eachClientTraffic(func(row xuiClientTraffic) error {
		if row.Email != "" {
			seen[row.Email] = struct{}{}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	emails := make([]string, 0, len(seen))
	for email := range seen {
		emails = append(emails, email)
	}
	sortStrings(emails)
	return emails
}
