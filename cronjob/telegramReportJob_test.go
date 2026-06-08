package cronjob

import (
	"testing"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
	"github.com/deposist/s-ui-x/service"
	"github.com/robfig/cron/v3"
)

func TestTelegramReportSchedulerReplansFromSettings(t *testing.T) {
	initCronJobTestDB(t)
	if _, err := (&service.SettingService{}).GetAllSetting(); err != nil {
		t.Fatal(err)
	}
	for key, value := range map[string]string{
		"telegramReport":     "true",
		"telegramReportCron": "*/5 * * * *",
	} {
		if err := database.GetDB().Model(model.Setting{}).Where("key = ?", key).Update("value", value).Error; err != nil {
			t.Fatal(err)
		}
	}
	c := cron.New()
	scheduler := NewTelegramReportScheduler(c)
	scheduler.Run()
	if scheduler.entryID == 0 || scheduler.currentSpec != "*/5 * * * *" {
		t.Fatalf("scheduler did not add report job: %#v", scheduler)
	}

	if err := database.GetDB().Model(model.Setting{}).Where("key = ?", "telegramReport").Update("value", "false").Error; err != nil {
		t.Fatal(err)
	}
	scheduler.Run()
	if scheduler.entryID != 0 || scheduler.currentSpec != "" {
		t.Fatalf("scheduler did not remove report job: %#v", scheduler)
	}
}
