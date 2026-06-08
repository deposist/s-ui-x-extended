package importxui

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestApplyState_TLSReplaceDeleteError(t *testing.T) {
	initPlanTLSDeleteTestDB(t)
	db := database.GetDB()
	spec := realitySpec{
		PrivateKey:  "private-key-delete-error",
		Target:      "example.com:443",
		Host:        "example.com",
		Port:        443,
		ServerName:  "example.com",
		PublicKey:   "public-key-delete-error",
		Fingerprint: "chrome",
		ShortIDs:    []string{"abcd"},
	}
	existing, err := buildTLSRecord(spec)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&existing).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.Inbound{
		Type:    "vless",
		Tag:     "uses-existing-tls",
		TlsId:   existing.Id,
		Addrs:   []byte("[]"),
		OutJson: []byte("{}"),
		Options: []byte("{}"),
	}).Error; err != nil {
		t.Fatal(err)
	}

	src := createPlanTLSDeleteSource(t)
	plan, err := Plan(src, PlanOptions{Strategy: StrategyReplace})
	if err != nil {
		t.Fatal(err)
	}
	_, err = Apply(src, *plan, ApplyOptions{DryRun: true})
	if err == nil {
		t.Fatal("expected Apply to return TLS replace delete error")
	}
	errText := strings.ToLower(err.Error())
	if !strings.Contains(errText, "foreign key") && !strings.Contains(errText, "constraint") {
		t.Fatalf("expected foreign-key delete error, got %v", err)
	}
}

func initPlanTLSDeleteTestDB(t *testing.T) {
	t.Helper()
	closeMainDBForImportTest(t)
	dir := makeImportXUITempDir(t)
	t.Setenv("SUI_DB_FOLDER", dir)
	if err := database.InitDB(filepath.Join(dir, "s-ui.db")); err != nil {
		if strings.Contains(err.Error(), "go-sqlite3 requires cgo") {
			t.Skip(err)
		}
		t.Fatal(err)
	}
	t.Cleanup(func() {
		closeMainDBForImportTest(t)
	})
}

func createPlanTLSDeleteSource(t *testing.T) string {
	t.Helper()
	path := filepath.Join(makeImportXUITempDir(t), "x-ui.db")
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err == nil {
		defer sqlDB.Close()
	}
	if err := db.Exec(`CREATE TABLE inbounds (
		id integer primary key,
		user_id integer,
		up integer,
		down integer,
		total integer,
		all_time integer,
		remark text,
		enable integer,
		expiry_time integer,
		traffic_reset text,
		last_traffic_reset_time integer,
		listen text,
		port integer,
		protocol text,
		settings text,
		stream_settings text,
		tag text,
		sniffing text
	)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE client_traffics (
		id integer primary key,
		inbound_id integer,
		enable integer,
		email text,
		up integer,
		down integer,
		all_time integer,
		expiry_time integer,
		total integer,
		reset integer,
		last_online integer
	)`).Error; err != nil {
		t.Fatal(err)
	}
	streamSettings := `{
		"network":"tcp",
		"security":"reality",
		"realitySettings":{
			"target":"example.com:443",
			"privateKey":"private-key-delete-error",
			"serverNames":["example.com"],
			"shortIds":["abcd"],
			"settings":{
				"publicKey":"public-key-delete-error",
				"fingerprint":"chrome",
				"serverName":"example.com"
			}
		}
	}`
	if err := db.Exec(`INSERT INTO inbounds
		(id, user_id, up, down, total, all_time, remark, enable, expiry_time, traffic_reset,
		 last_traffic_reset_time, listen, port, protocol, settings, stream_settings, tag, sniffing)
		VALUES (1, 1, 0, 0, 0, 0, 'reality-delete-error', 1, 0, '', 0, '', 443, 'vless', '{"clients":[]}', ?, 'reality-delete-error', '{}')`,
		streamSettings,
	).Error; err != nil {
		t.Fatal(err)
	}
	return path
}
