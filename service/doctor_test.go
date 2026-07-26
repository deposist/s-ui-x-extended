package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"

	"gorm.io/gorm"
)

func initDoctorTestDB(t *testing.T) {
	t.Helper()
	tempDir, err := os.MkdirTemp("", "s-ui-doctor-test-")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("SUI_DB_FOLDER", tempDir)
	closeDoctorTestDB(database.GetDB())
	if err := database.InitDB(filepath.Join(tempDir, "s-ui.db")); err != nil {
		removeDoctorTestDir(t, tempDir)
		if strings.Contains(err.Error(), "go-sqlite3 requires cgo") {
			t.Skip(err)
		}
		t.Fatalf("InitDB: %v", err)
	}
	db := database.GetDB()
	t.Cleanup(func() {
		closeDoctorTestDB(db)
		removeDoctorTestDir(t, tempDir)
	})
}

func closeDoctorTestDB(db *gorm.DB) {
	if db == nil {
		return
	}
	_ = db.Exec("PRAGMA wal_checkpoint(TRUNCATE)").Error
	if sqlDB, err := db.DB(); err == nil {
		_ = sqlDB.Close()
	}
}

func removeDoctorTestDir(t *testing.T, dir string) {
	t.Helper()
	var err error
	for i := 0; i < 20; i++ {
		err = os.RemoveAll(dir)
		if err == nil || os.IsNotExist(err) {
			return
		}
		time.Sleep(time.Duration(i+1) * 10 * time.Millisecond)
	}
	t.Errorf("remove doctor test dir %q: %v", dir, err)
}

func TestDoctorRunReportsMalformedConfig(t *testing.T) {
	initDoctorTestDB(t)
	if err := (&SettingService{}).SetConfig("{bad json"); err != nil {
		t.Fatalf("set config: %v", err)
	}

	report := (&DoctorService{}).Run("example.com")
	if report.Status != DoctorSeverityError {
		t.Fatalf("status = %s, want error: %#v", report.Status, report.Items)
	}
}

func TestDoctorRunReportsMissingReferences(t *testing.T) {
	initDoctorTestDB(t)
	config := `{"log":{"disabled":true},"dns":{"servers":[],"final":"missing-dns","rules":[]},"route":{"final":"missing-out","rules":[{"outbound":"missing-rule"}],"rule_set":[]}}`
	if err := (&SettingService{}).SetConfig(config); err != nil {
		t.Fatalf("set config: %v", err)
	}

	report := (&DoctorService{}).Run("example.com")
	if !doctorReportHas(report, "dns-references", DoctorSeverityError) {
		t.Fatalf("missing dns reference error: %#v", report.Items)
	}
	if !doctorReportHas(report, "route-references", DoctorSeverityError) {
		t.Fatalf("missing route reference error: %#v", report.Items)
	}
}

func TestDoctorRunReportsMissingRuleConditions(t *testing.T) {
	initDoctorTestDB(t)
	config := `{"log":{"disabled":true},"dns":{"servers":[],"rules":[{"type":"logical","mode":"and","rules":[{}],"action":"route"}]},"route":{"rules":[{"action":"sniff"},{"type":"logical","mode":"and","rules":[{}],"action":"route","outbound":"direct"}],"rule_set":[]}}`
	if err := (&SettingService{}).SetConfig(config); err != nil {
		t.Fatalf("set config: %v", err)
	}

	report := (&DoctorService{}).Run("example.com")
	if !doctorReportHas(report, "rule-conditions", DoctorSeverityError) {
		t.Fatalf("missing rule condition error: %#v", report.Items)
	}
}

// TestDoctorRuleConditionDetailsNameDeepDescendant pins the exact "<path>:
// <message>" detail lines the doctor reports, for both route and DNS. The path
// is the only part of the report that tells an operator which nested branch to
// open, so a truncated path makes the item nearly useless on a deep tree.
func TestDoctorRuleConditionDetailsNameDeepDescendant(t *testing.T) {
	for _, tc := range []struct {
		name     string
		config   string
		wantPath string
	}{
		{
			name:     "route",
			config:   `{"route":{"rules":[{"type":"logical","mode":"and","rules":[{"domain":["a.com"]},{"type":"logical","mode":"or","rules":[]}],"outbound":"direct"}]}}`,
			wantPath: "route.rules[0].rules[1].rules",
		},
		{
			name:     "dns",
			config:   `{"dns":{"rules":[{"type":"logical","mode":"and","rules":[{"domain":["a.com"]},{"type":"logical","mode":"or","rules":[]}],"server":"local"}]}}`,
			wantPath: "dns.rules[0].rules[1].rules",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			items := doctorRuleConditionChecks([]byte(tc.config))
			if len(items) != 1 {
				t.Fatalf("expected exactly one item, got %#v", items)
			}
			if items[0].Severity != DoctorSeverityError {
				t.Fatalf("expected an error item, got %#v", items[0])
			}
			details, ok := items[0].Details.([]string)
			if !ok {
				t.Fatalf("expected string details, got %#v", items[0].Details)
			}
			if len(details) != 1 {
				t.Fatalf("expected only the broken branch, got %#v", details)
			}
			if !strings.HasPrefix(details[0], tc.wantPath+": ") {
				t.Fatalf("expected a %q: <message> detail, got %q", tc.wantPath, details[0])
			}
		})
	}
}

// TestDoctorRuleConditionChecksAcceptsDecodedEmptyBranches guards the former
// false positives. These shapes are all accepted by the core, so the doctor must
// not raise an error item for them; the old map-walking heuristic flagged every
// one of them.
func TestDoctorRuleConditionChecksAcceptsDecodedEmptyBranches(t *testing.T) {
	for _, tc := range []struct {
		name   string
		config string
	}{
		{"empty branch beside a sibling", `{"route":{"rules":[{"type":"logical","mode":"and","rules":[{"domain":["a.com"]},{}],"outbound":"direct"}]}}`},
		{"invert-only branch", `{"route":{"rules":[{"type":"logical","mode":"and","rules":[{"invert":true}],"outbound":"direct"}]}}`},
		{"action-only rule", `{"route":{"rules":[{"action":"sniff"}]}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			items := doctorRuleConditionChecks([]byte(tc.config))
			for _, item := range items {
				if item.Severity == DoctorSeverityError {
					t.Fatalf("unexpected error item: %#v", item)
				}
			}
		})
	}
}

// TestDoctorRuleConditionChecksWarnsOnDroppedRule keeps the two failure modes
// distinct: a discarded rule is silent data loss the core tolerates, so it is a
// warning, not a startup error.
func TestDoctorRuleConditionChecksWarnsOnDroppedRule(t *testing.T) {
	items := doctorRuleConditionChecks([]byte(`{"dns":{"rules":[{}]}}`))
	if len(items) != 1 || items[0].Severity != DoctorSeverityWarn {
		t.Fatalf("expected a single warning item, got %#v", items)
	}
	// The path is the collection, not an index: the discarded rule no longer has
	// a position in the decoded list, so the count is what identifies the loss.
	details, ok := items[0].Details.([]string)
	if !ok || len(details) != 1 || !strings.HasPrefix(details[0], "dns.rules: ") {
		t.Fatalf("expected a dns.rules detail line, got %#v", items[0].Details)
	}
}

// TestDoctorRuleConditionChecksReportsDecodeError proves malformed rule JSON
// becomes its own error item instead of being silently reported as healthy.
func TestDoctorRuleConditionChecksReportsDecodeError(t *testing.T) {
	items := doctorRuleConditionChecks([]byte(`{"route":{"rules":[{"type":"logical","mode":5}]}}`))
	if len(items) != 1 || items[0].Severity != DoctorSeverityError {
		t.Fatalf("expected a single error item, got %#v", items)
	}
}
func TestDoctorRunAllowsActionOnlyRules(t *testing.T) {
	initDoctorTestDB(t)
	config := `{"log":{"disabled":true},"dns":{"servers":[],"rules":[{"action":"route","server":"local"}]},"route":{"rules":[{"action":"sniff"}],"rule_set":[]}}`
	if err := (&SettingService{}).SetConfig(config); err != nil {
		t.Fatalf("set config: %v", err)
	}

	report := (&DoctorService{}).Run("example.com")
	if doctorReportHas(report, "rule-conditions", DoctorSeverityError) {
		t.Fatalf("unexpected rule condition error: %#v", report.Items)
	}
}

func TestDiagnoseClientReportsDisabledExpiredAndOverLimit(t *testing.T) {
	initDoctorTestDB(t)
	inbounds, _ := json.Marshal([]uint{})
	client := model.Client{
		Enable:   false,
		Name:     "alice",
		Inbounds: inbounds,
		Volume:   10,
		Up:       10,
		Down:     1,
		Expiry:   1,
	}
	if err := database.GetDB().Create(&client).Error; err != nil {
		t.Fatalf("create client: %v", err)
	}

	report, err := (&DoctorService{}).DiagnoseClient(DoctorClientRequest{ClientID: client.Id}, "example.com")
	if err != nil {
		t.Fatalf("DiagnoseClient: %v", err)
	}
	for _, id := range []string{"client-enabled", "client-expiry", "client-traffic", "client-inbounds"} {
		if !doctorReportHas(report, id, DoctorSeverityError) {
			t.Fatalf("missing %s error: %#v", id, report.Items)
		}
	}
}

func TestDiagnoseClientTrafficBoundary(t *testing.T) {
	initDoctorTestDB(t)
	inbounds, _ := json.Marshal([]uint{})

	// used == Volume must count as over-limit (error), and the enabled/non-expired
	// branches must report OK - pinning the OK direction the earlier test never asserts.
	atLimit := model.Client{Enable: true, Name: "atlimit", Inbounds: inbounds, Volume: 10, Up: 6, Down: 4}
	if err := database.GetDB().Create(&atLimit).Error; err != nil {
		t.Fatalf("create atlimit client: %v", err)
	}
	report, err := (&DoctorService{}).DiagnoseClient(DoctorClientRequest{ClientID: atLimit.Id}, "example.com")
	if err != nil {
		t.Fatalf("DiagnoseClient atlimit: %v", err)
	}
	if !doctorReportHas(report, "client-traffic", DoctorSeverityError) {
		t.Fatalf("used==Volume must be over-limit error: %#v", report.Items)
	}
	if !doctorReportHas(report, "client-enabled", DoctorSeverityOK) {
		t.Fatalf("enabled client must report client-enabled OK: %#v", report.Items)
	}
	if !doctorReportHas(report, "client-expiry", DoctorSeverityOK) {
		t.Fatalf("non-expired client must report client-expiry OK: %#v", report.Items)
	}

	// used == Volume-1 must count as within limit (OK).
	under := model.Client{Enable: true, Name: "under", Inbounds: inbounds, Volume: 10, Up: 5, Down: 4}
	if err := database.GetDB().Create(&under).Error; err != nil {
		t.Fatalf("create under client: %v", err)
	}
	report2, err := (&DoctorService{}).DiagnoseClient(DoctorClientRequest{ClientID: under.Id}, "example.com")
	if err != nil {
		t.Fatalf("DiagnoseClient under: %v", err)
	}
	if !doctorReportHas(report2, "client-traffic", DoctorSeverityOK) {
		t.Fatalf("used==Volume-1 must be within limit (OK): %#v", report2.Items)
	}
}

func doctorReportHas(report DoctorReport, id string, severity DoctorSeverity) bool {
	for _, item := range report.Items {
		if item.ID == id && item.Severity == severity {
			return true
		}
	}
	return false
}
