package importxui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// realityStream mirrors the reality stream_settings shape exported by recent
// 3x-ui builds (nested settings.publicKey, post-quantum mldsa65 fields, etc.).
const realityStream = `{
  "network": "tcp",
  "security": "reality",
  "realitySettings": {
    "show": false,
    "target": "www.apple.com:443",
    "serverNames": ["www.apple.com"],
    "privateKey": "UF2GdUBplZ268703D0dNVZPZ7DU5PvAhtbZiylmCOHk",
    "shortIds": ["1e7baa1246ce"],
    "mldsa65Seed": "",
    "settings": {"publicKey": "dlhJts-G09NunPK_RnUQCxoqTuXOgwcU-bbowxLAKA4", "fingerprint": "chrome", "spiderX": "/"}
  },
  "tcpSettings": {"header": {"type": "none"}}
}`

const grpcStream = `{"network":"grpc","security":"none","grpcSettings":{"serviceName":"hello"}}`

const wireguardSettings = `{"mtu":1280,"secretKey":"aGVsbG8=","peers":[{"publicKey":"cGsx","allowedIPs":["0.0.0.0/0"],"keepAlive":25}]}`

func vlessSettings(emails ...string) string {
	clients := make([]map[string]any, 0, len(emails))
	for i, email := range emails {
		clients = append(clients, map[string]any{
			"email":   email,
			"id":      deterministicUUID(email),
			"flow":    "xtls-rprx-vision",
			"enable":  true,
			"subId":   "sub-" + email,
			"totalGB": int64(0),
			"comment": "note-" + string(rune('a'+i)),
		})
	}
	raw, _ := json.Marshal(map[string]any{"clients": clients, "decryption": "none"})
	return string(raw)
}

func trojanSettings(emails ...string) string {
	clients := make([]map[string]any, 0, len(emails))
	for _, email := range emails {
		clients = append(clients, map[string]any{
			"email":    email,
			"password": "pw-" + email,
			"enable":   true,
		})
	}
	raw, _ := json.Marshal(map[string]any{"clients": clients})
	return string(raw)
}

type compatInbound struct {
	tag      string
	protocol string
	port     int
	settings string
	stream   string
	enable   int
}

// schemaVariant describes how to materialize a 3x-ui source database whose
// inbounds/client_traffics columns deliberately differ from the importer's old
// hard-coded SELECT list.
type schemaVariant struct {
	name              string
	inboundsDDL       string
	clientTrafficsDDL string
	// insertInbound returns the column list and value placeholders used to
	// insert one inbound row for this schema.
	inboundCols string
	inboundVals func(in compatInbound) []any
}

var forkVariant = schemaVariant{
	name: "normalized_fork_no_all_time",
	// Matches the real x-ui (6).db export: has node_id/traffic_reset/
	// last_traffic_reset_time/last_online, but NO all_time column.
	inboundsDDL: `CREATE TABLE inbounds (
		id integer PRIMARY KEY AUTOINCREMENT, user_id integer, up integer, down integer,
		total integer, remark text, enable numeric, expiry_time integer,
		traffic_reset text DEFAULT 'never', last_traffic_reset_time integer DEFAULT 0,
		listen text, port integer, protocol text, settings text, stream_settings text,
		tag text, sniffing text, node_id integer)`,
	clientTrafficsDDL: `CREATE TABLE client_traffics (
		id integer PRIMARY KEY AUTOINCREMENT, inbound_id integer, enable numeric, email text,
		up integer, down integer, expiry_time integer, total integer,
		reset integer DEFAULT 0, last_online integer DEFAULT 0)`,
	inboundCols: "user_id, up, down, total, remark, enable, expiry_time, listen, port, protocol, settings, stream_settings, tag, sniffing",
	inboundVals: func(in compatInbound) []any {
		return []any{1, 0, 0, 0, in.tag, in.enable, 0, "", in.port, in.protocol, in.settings, in.stream, in.tag, ""}
	},
}

var vanillaVariant = schemaVariant{
	name: "vanilla_mhsanaei_no_all_time_no_last_online",
	// Minimal upstream mhsanaei schema: no all_time, no last_online, no
	// traffic_reset, no last_traffic_reset_time, no node_id.
	inboundsDDL: `CREATE TABLE inbounds (
		id integer PRIMARY KEY AUTOINCREMENT, user_id integer, up integer, down integer,
		total integer, remark text, enable numeric, expiry_time integer, listen text,
		port integer, protocol text, settings text, stream_settings text, tag text,
		sniffing text, allocate text)`,
	clientTrafficsDDL: `CREATE TABLE client_traffics (
		id integer PRIMARY KEY AUTOINCREMENT, inbound_id integer, enable numeric, email text,
		up integer, down integer, expiry_time integer, total integer, reset integer DEFAULT 0)`,
	inboundCols: "user_id, up, down, total, remark, enable, expiry_time, listen, port, protocol, settings, stream_settings, tag, sniffing",
	inboundVals: func(in compatInbound) []any {
		return []any{1, 0, 0, 0, in.tag, in.enable, 0, "", in.port, in.protocol, in.settings, in.stream, in.tag, ""}
	},
}

func buildCompatSource(t *testing.T, variant schemaVariant, path string) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()

	for _, ddl := range []string{
		variant.inboundsDDL,
		variant.clientTrafficsDDL,
		`CREATE TABLE settings (id integer PRIMARY KEY AUTOINCREMENT, key text, value text)`,
		`CREATE TABLE users (id integer PRIMARY KEY AUTOINCREMENT, username text, password text)`,
		`CREATE TABLE outbound_traffics (id integer PRIMARY KEY AUTOINCREMENT, tag text, up integer, down integer, total integer)`,
	} {
		if err := db.Exec(ddl).Error; err != nil {
			t.Fatalf("ddl: %v", err)
		}
	}

	inbounds := []compatInbound{
		{tag: "inbound-443", protocol: "vless", port: 443, settings: vlessSettings("alice", "bob"), stream: realityStream, enable: 1},
		{tag: "inbound-12223", protocol: "trojan", port: 12223, settings: trojanSettings("carol"), stream: grpcStream, enable: 1},
		{tag: "inbound-12555", protocol: "wireguard", port: 12555, settings: wireguardSettings, stream: `{}`, enable: 1},
	}
	insert := "INSERT INTO inbounds(" + variant.inboundCols + ") VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)"
	for _, in := range inbounds {
		if err := db.Exec(insert, variant.inboundVals(in)...).Error; err != nil {
			t.Fatalf("insert inbound %s: %v", in.tag, err)
		}
	}

	// client_traffics carries traffic + the email->inbound mapping. Insert one
	// row per client email, using only the columns common to both variants.
	traffics := []struct {
		inbound int
		email   string
		up, dn  int64
	}{
		{1, "alice", 100, 200},
		{1, "bob", 0, 0},
		{2, "carol", 5, 6},
	}
	for _, tr := range traffics {
		if err := db.Exec(
			"INSERT INTO client_traffics(inbound_id, enable, email, up, down, expiry_time, total, reset) VALUES(?,?,?,?,?,?,?,?)",
			tr.inbound, 1, tr.email, tr.up, tr.dn, 0, 0, 0,
		).Error; err != nil {
			t.Fatalf("insert traffic %s: %v", tr.email, err)
		}
	}

	for _, kv := range [][2]string{
		// host/domain-specific (must be skipped by default): distinct values so
		// we can prove they did NOT overwrite this host's config.
		{"webDomain", "old.example.com"},
		{"webListen", "10.0.0.9"},
		// portable (must migrate): ports + paths are logical config.
		{"webPort", "9172"},
		{"subPort", "8443"},
		{"webBasePath", "/panel/"},
		{"subPath", "/mysub/"},
		// no s-ui equivalent (dropped):
		{"subEnable", "true"},
	} {
		if err := db.Exec("INSERT INTO settings(key, value) VALUES(?, ?)", kv[0], kv[1]).Error; err != nil {
			t.Fatalf("insert setting %s: %v", kv[0], err)
		}
	}
	if err := db.Exec("INSERT INTO users(username, password) VALUES(?, ?)", "admin", "$2a$10$abcdefghijklmnopqrstuv").Error; err != nil {
		t.Fatalf("insert user: %v", err)
	}
}

func initCompatDest(t *testing.T) {
	t.Helper()
	closeMainDBForImportTest(t)
	dir := makeImportXUITempDir(t)
	t.Setenv("SUI_DB_FOLDER", dir)
	if err := database.InitDB(filepath.Join(dir, "s-ui.db")); err != nil {
		t.Fatalf("init destination: %v", err)
	}
	t.Cleanup(func() { closeMainDBForImportTest(t) })
}

// TestImport_SchemaWithoutAllTime is the regression guard for the
// "no such column: all_time" failure: the old dialect hard-coded all_time and
// last_online, which neither vanilla mhsanaei nor the normalized fork export
// actually contain, so every real-world import aborted before reading a row.
func TestImport_SchemaWithoutAllTime(t *testing.T) {
	for _, variant := range []schemaVariant{forkVariant, vanillaVariant} {
		t.Run(variant.name, func(t *testing.T) {
			initCompatDest(t)
			dir := makeImportXUITempDir(t)
			src := filepath.Join(dir, "x-ui.db")
			buildCompatSource(t, variant, src)

			plan, err := Plan(src, PlanOptions{
				Strategy:        StrategyMerge,
				IncludeSettings: true,
				AdminMode:       AdminModeSkip,
				IncludeHistory:  true,
				IncludeRouting:  true,
			})
			if err != nil {
				t.Fatalf("Plan failed (regression: %v)", err)
			}
			report, err := Apply(src, *plan, ApplyOptions{})
			if err != nil {
				t.Fatalf("Apply failed (regression: %v)", err)
			}

			// 2 routable inbounds (vless+reality, trojan) and 1 wireguard endpoint.
			if got := report.Summary.Inbounds.Imported; got != 2 {
				t.Fatalf("inbounds imported = %d, want 2; report=%#v", got, report.Summary.Inbounds)
			}
			if got := report.Summary.Endpoints.Imported; got != 1 {
				t.Fatalf("endpoints imported = %d, want 1", got)
			}
			if report.Summary.TLS.Created == 0 {
				t.Fatalf("expected reality TLS row to be created")
			}

			db := database.GetDB()
			for _, tag := range []string{"inbound-443", "inbound-12223"} {
				var n int64
				if err := db.Model(model.Inbound{}).Where("tag = ?", tag).Count(&n).Error; err != nil || n != 1 {
					t.Fatalf("inbound %s missing (count=%d, err=%v)", tag, n, err)
				}
			}
			var wg int64
			if err := db.Model(model.Endpoint{}).Where("tag = ?", "inbound-12555").Count(&wg).Error; err != nil || wg != 1 {
				t.Fatalf("wireguard endpoint missing (count=%d, err=%v)", wg, err)
			}
			for _, email := range []string{"alice", "bob", "carol"} {
				var n int64
				if err := db.Model(model.Client{}).Where("name = ?", email).Count(&n).Error; err != nil || n != 1 {
					t.Fatalf("client %s missing (count=%d, err=%v)", email, n, err)
				}
			}
			// Aggregated client traffic must survive (alice: up=100, down=200).
			var alice model.Client
			if err := db.Where("name = ?", "alice").First(&alice).Error; err != nil {
				t.Fatal(err)
			}
			if alice.Up != 100 || alice.Down != 200 {
				t.Fatalf("alice traffic = up:%d down:%d, want up:100 down:200", alice.Up, alice.Down)
			}
			// Portable settings must migrate under their s-ui keys:
			// webBasePath -> webPath (renamed), subPath, and ports.
			assertSetting(t, db, "webPath", "/panel/")
			assertSetting(t, db, "subPath", "/mysub/")
			assertSetting(t, db, "webPort", "9172")
			assertSetting(t, db, "subPort", "8443")

			// Host/domain-specific settings must NOT carry the source server's
			// values onto this host (would break a different host/domain).
			assertNotMigrated(t, db, "webDomain", "old.example.com")
			assertNotMigrated(t, db, "webListen", "10.0.0.9")

			// The plan must keep host-specific items visible but skipped + warned.
			domainItem := findSettingItem(t, plan, "webDomain")
			if domainItem.Action != ActionSkip {
				t.Fatalf("webDomain plan action = %q, want skip (host-specific)", domainItem.Action)
			}
			if len(domainItem.Warnings) == 0 {
				t.Fatalf("webDomain plan item should carry a host-specific warning")
			}

			// subEnable has no s-ui equivalent and must NOT be written verbatim.
			var subEnableCount int64
			if err := db.Model(model.Setting{}).Where("key = ?", "subEnable").Count(&subEnableCount).Error; err != nil {
				t.Fatal(err)
			}
			if subEnableCount != 0 {
				t.Fatalf("subEnable should not be migrated (no s-ui key), found %d", subEnableCount)
			}
		})
	}
}

func assertSetting(t *testing.T, db *gorm.DB, key, want string) {
	t.Helper()
	var setting model.Setting
	if err := db.Where("key = ?", key).First(&setting).Error; err != nil {
		t.Fatalf("setting %q missing: %v", key, err)
	}
	if setting.Value != want {
		t.Fatalf("setting %q = %q, want %q", key, setting.Value, want)
	}
}

// assertNotMigrated verifies a host-specific source value did not land in the
// destination. InitDB does not seed default settings, so a correctly skipped
// key is simply absent; the value must never equal the source server's value.
func assertNotMigrated(t *testing.T, db *gorm.DB, key, forbidden string) {
	t.Helper()
	var n int64
	if err := db.Model(model.Setting{}).Where("key = ? AND value = ?", key, forbidden).Count(&n).Error; err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("host-specific setting %q was migrated with source value %q (would break a different host)", key, forbidden)
	}
}

func findSettingItem(t *testing.T, plan *MigrationPlan, srcKey string) PlanItem {
	t.Helper()
	for _, item := range plan.Items {
		if item.Kind == KindSetting && item.SrcTag == srcKey {
			return item
		}
	}
	t.Fatalf("plan has no setting item for %q", srcKey)
	return PlanItem{}
}

// TestImport_RealXUIBackup imports an operator-supplied real x-ui.db when the
// IMPORT_XUI_REAL_DB env var points at one. It is skipped in CI (no fixture)
// but lets us prove the migration end-to-end against an actual export, e.g.
//
//	IMPORT_XUI_REAL_DB="C:\\CheckErrorS-ui\\x-ui (6).db" go test ./database/importxui/ -run RealXUIBackup -v
func TestImport_RealXUIBackup(t *testing.T) {
	path := os.Getenv("IMPORT_XUI_REAL_DB")
	if path == "" {
		t.Skip("set IMPORT_XUI_REAL_DB to a real x-ui.db to run this test")
	}
	if err := ValidateSQLiteSource(path); err != nil {
		t.Fatalf("source validation failed: %v", err)
	}
	initCompatDest(t)

	// Exercise the exact UI path (api/import-xui/plan + /apply) with every
	// optional category enabled, so the run proves routes, inbounds, clients,
	// settings, admins, history and routing all migrate.
	plan, err := Plan(path, PlanOptions{
		Strategy:        StrategyMerge,
		IncludeSettings: true,
		IncludeHistory:  true,
		IncludeRouting:  true,
		AdminMode:       AdminModeNewPassword,
	})
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}
	report, err := Apply(path, *plan, ApplyOptions{})
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}
	t.Logf("real-db import: inbounds=%d/%d skipped=%d, endpoints=%d, tls created=%d reused=%d, clients=%d unique/%d created, history=%d, routing imported=%d, admins=%d, warnings=%d",
		report.Summary.Inbounds.Imported, report.Summary.Inbounds.Total, report.Summary.Inbounds.Skipped,
		report.Summary.Endpoints.Imported, report.Summary.TLS.Created, report.Summary.TLS.Reused,
		report.Summary.Clients.UniqueEmails, report.Summary.Clients.Created,
		report.Summary.Historical.Imported, report.Summary.Routing.Imported,
		len(report.GeneratedAdmins), len(report.Warnings))

	if report.Summary.Inbounds.Imported == 0 && report.Summary.Endpoints.Imported == 0 {
		t.Fatalf("real import migrated nothing: %#v", report.Summary)
	}

	db := database.GetDB()
	// Portable settings migrate: webBasePath (/whatafuck/) -> webPath, and the
	// panel port carries over.
	assertSetting(t, db, "webPath", "/whatafuck/")
	assertSetting(t, db, "webPort", "9172")
	// Host/domain-specific keys must default to skip in the plan so they are not
	// forced onto a different host/domain.
	domainItem := findSettingItem(t, plan, "webDomain")
	if domainItem.Action != ActionSkip {
		t.Fatalf("webDomain plan action = %q, want skip (host-specific)", domainItem.Action)
	}
	t.Logf("real-db settings: webPath/webPort migrated; host/domain-specific keys (webDomain etc.) skipped by default")
	// No setting may be written under a dead 3x-ui key that s-ui ignores.
	var deadKeys int64
	if err := db.Model(model.Setting{}).Where("key IN ?", []string{"webBasePath", "subEnable", "tgBotRunTime", "tgBotEnable", "tgBotToken"}).Count(&deadKeys).Error; err != nil {
		t.Fatal(err)
	}
	if deadKeys != 0 {
		t.Fatalf("found %d settings written under dead 3x-ui keys s-ui ignores", deadKeys)
	}
}

func TestMapSettingKey_TargetsAreCanonicalSUIKeys(t *testing.T) {
	// The renamed keys are the ones most likely to regress; assert them
	// explicitly. Direct same-name keys are covered by the map itself.
	renamed := map[string]string{
		"webBasePath": "webPath",
		"tgBotEnable": "telegramEnabled",
		"tgBotToken":  "telegramBotToken",
		"tgBotChatId": "telegramChatID",
		"tgRunTime":   "telegramReportCron",
		"tgCpu":       "telegramCpuThreshold",
		"tgBotBackup": "telegramBackupEnabled",
		"tgBotProxy":  "telegramProxyURL",
	}
	for src, want := range renamed {
		got, ok := mapSettingKey(src)
		if !ok || got != want {
			t.Errorf("mapSettingKey(%q) = (%q, %v), want (%q, true)", src, got, ok, want)
		}
	}
	// subEnable has no s-ui equivalent and must not map.
	if got, ok := mapSettingKey("subEnable"); ok {
		t.Errorf("mapSettingKey(\"subEnable\") mapped to %q; expected no mapping", got)
	}
	// Every target must be a key s-ui actually recognizes; guard against typos
	// by ensuring no target equals a known-wrong legacy key.
	for src, target := range xuiSettingKeyMap {
		if target == "" {
			t.Errorf("empty target for %q", src)
		}
		if target == "tgBotRunTime" || target == "webBasePath" {
			t.Errorf("%q maps to dead key %q", src, target)
		}
	}
}
