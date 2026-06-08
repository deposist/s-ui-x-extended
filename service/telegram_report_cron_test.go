package service

import (
	"encoding/json"
	"testing"

	"github.com/deposist/s-ui-x/database"
	"gorm.io/gorm"
)

func TestParseTelegramReportCronRequiresFiveFieldMinuteSchedule(t *testing.T) {
	if _, err := ParseTelegramReportCron("*/5 * * * *"); err != nil {
		t.Fatalf("valid five-field cron rejected: %v", err)
	}
	if _, err := ParseTelegramReportCron("* * * * * *"); err == nil {
		t.Fatal("six-field cron should be rejected")
	}
	if _, err := ParseTelegramReportCron("@every 30s"); err == nil {
		t.Fatal("@every seconds cron should be rejected")
	}
}

func TestSaveValidatesTelegramReportCron(t *testing.T) {
	settingService := initSettingTestDB(t)
	if _, err := settingService.GetAllSetting(); err != nil {
		t.Fatal(err)
	}
	validPayload, err := json.Marshal(map[string]string{
		"telegramReport":     "true",
		"telegramReportCron": "*/10 * * * *",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Transaction(func(tx *gorm.DB) error {
		return settingService.Save(tx, validPayload)
	}); err != nil {
		t.Fatalf("valid telegram report cron rejected: %v", err)
	}

	invalidPayload, err := json.Marshal(map[string]string{
		"telegramReportCron": "* * * * * *",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Transaction(func(tx *gorm.DB) error {
		return settingService.Save(tx, invalidPayload)
	}); err == nil {
		t.Fatal("expected invalid telegram report cron to be rejected")
	}
}
