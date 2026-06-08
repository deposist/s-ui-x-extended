package importxui

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x/database"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestApplyExtraReturnsErrPlanStale(t *testing.T) {
	initPlanExtraMainDB(t)
	src := createPlanExtraSource(t, nil)
	plan, err := Plan(src, PlanOptions{Strategy: StrategyMerge})
	if err != nil {
		t.Fatal(err)
	}
	file, err := os.OpenFile(src, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteString("changed"); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	if _, err := Apply(src, *plan, ApplyOptions{DryRun: true}); !errors.Is(err, ErrPlanStale) {
		t.Fatalf("expected ErrPlanStale, got %v", err)
	}
}

func TestApplyExtraReturnsErrBusyWhenApplyLockHeld(t *testing.T) {
	applyMu.Lock()
	defer applyMu.Unlock()

	if _, err := Apply("unused.db", MigrationPlan{}, ApplyOptions{DryRun: true}); !errors.Is(err, ErrBusy) {
		t.Fatalf("expected ErrBusy, got %v", err)
	}
}

func TestIssue6ApplyWireguardNoPeersCountsEndpointSkip(t *testing.T) {
	initPlanExtraMainDB(t)
	src := createPlanExtraSource(t, []planExtraInbound{{
		id:       1,
		port:     51820,
		protocol: "wireguard",
		tag:      "wg-empty",
		settings: `{"mtu":1280,"secretKey":"private-key","peers":[]}`,
	}})
	plan, err := Plan(src, PlanOptions{Strategy: StrategyMerge})
	if err != nil {
		t.Fatal(err)
	}
	report, err := Apply(src, *plan, ApplyOptions{DryRun: true})
	if err != nil {
		t.Fatal(err)
	}

	if report.Summary.Inbounds.Skipped != 0 {
		t.Fatalf("wireguard endpoint skip should not count as inbound skip: %#v", report.Summary)
	}
	if report.Summary.Endpoints.Imported != 0 {
		t.Fatalf("wireguard endpoint with no peers should not be imported: %#v", report.Summary.Endpoints)
	}
	if report.Summary.Endpoints.Skipped != 1 {
		t.Fatalf("wireguard endpoint with no peers should count as endpoint skip: %#v", report.Summary.Endpoints)
	}
	if !containsWarning(report.Warnings, "wireguard has no peers") {
		t.Fatalf("expected no-peers warning, got %v", report.Warnings)
	}
}

func TestIssue6ImportWireguardNoPeersCountsEndpointSkip(t *testing.T) {
	initPlanExtraMainDB(t)
	src := createPlanExtraSource(t, []planExtraInbound{{
		id:       1,
		port:     51820,
		protocol: "wireguard",
		tag:      "wg-empty",
		settings: `{"mtu":1280,"secretKey":"private-key","peers":[]}`,
	}})
	report, err := Import(src, Options{DryRun: true, Strategy: StrategyMerge})
	if err != nil {
		t.Fatal(err)
	}

	if report.Summary.Inbounds.Skipped != 0 {
		t.Fatalf("wireguard endpoint skip should not count as inbound skip: %#v", report.Summary)
	}
	if report.Summary.Endpoints.Imported != 0 {
		t.Fatalf("wireguard endpoint with no peers should not be imported: %#v", report.Summary.Endpoints)
	}
	if report.Summary.Endpoints.Skipped != 1 {
		t.Fatalf("wireguard endpoint with no peers should count as endpoint skip: %#v", report.Summary.Endpoints)
	}
	if !containsWarning(report.Warnings, "wireguard has no peers") {
		t.Fatalf("expected no-peers warning, got %v", report.Warnings)
	}
}

type planExtraInbound struct {
	id       int64
	port     int
	protocol string
	tag      string
	settings string
}

func initPlanExtraMainDB(t *testing.T) {
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

func createPlanExtraSource(t *testing.T, inbounds []planExtraInbound) string {
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
	for _, inbound := range inbounds {
		if err := db.Exec(`INSERT INTO inbounds
			(id, user_id, up, down, total, all_time, remark, enable, expiry_time, traffic_reset,
			 last_traffic_reset_time, listen, port, protocol, settings, stream_settings, tag, sniffing)
			VALUES (?, 1, 0, 0, 0, 0, ?, 1, 0, '', 0, '', ?, ?, ?, '{}', ?, '{}')`,
			inbound.id, inbound.tag, inbound.port, inbound.protocol, inbound.settings, inbound.tag,
		).Error; err != nil {
			t.Fatal(err)
		}
	}
	return path
}

func containsWarning(warnings []string, needle string) bool {
	for _, warning := range warnings {
		if strings.Contains(warning, needle) {
			return true
		}
	}
	return false
}
