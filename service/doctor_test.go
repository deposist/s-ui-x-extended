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
