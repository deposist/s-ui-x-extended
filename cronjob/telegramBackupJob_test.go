package cronjob

import (
	"strings"
	"testing"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
	"github.com/deposist/s-ui-x/service"
	"github.com/robfig/cron/v3"
)

func TestTelegramBackupSchedulerReplansFromSettings(t *testing.T) {
	initCronJobTestDB(t)
	if _, err := (&service.SettingService{}).GetAllSetting(); err != nil {
		t.Fatal(err)
	}
	updateTelegramBackupSchedulerSettings(t, map[string]string{
		"telegramEnabled":       "true",
		"telegramBackupEnabled": "true",
		"telegramBackupCron":    "*/5 * * * *",
	})
	c := cron.New()
	scheduler := NewTelegramBackupScheduler(c)
	scheduler.Run()
	firstEntry := scheduler.entryID
	if firstEntry == 0 || scheduler.currentSpec != "*/5 * * * *" {
		t.Fatalf("scheduler did not add backup job: %#v", scheduler)
	}

	updateTelegramBackupSchedulerSettings(t, map[string]string{
		"telegramBackupCron": "*/10 * * * *",
	})
	scheduler.Run()
	if scheduler.entryID == 0 || scheduler.entryID == firstEntry || scheduler.currentSpec != "*/10 * * * *" {
		t.Fatalf("scheduler did not replan backup job: %#v", scheduler)
	}

	updateTelegramBackupSchedulerSettings(t, map[string]string{
		"telegramBackupEnabled": "false",
	})
	scheduler.Run()
	if scheduler.entryID != 0 || scheduler.currentSpec != "" {
		t.Fatalf("scheduler did not remove disabled backup job: %#v", scheduler)
	}
}

func TestTelegramBackupSchedulerNoopWhenTelegramDisabled(t *testing.T) {
	initCronJobTestDB(t)
	if _, err := (&service.SettingService{}).GetAllSetting(); err != nil {
		t.Fatal(err)
	}
	updateTelegramBackupSchedulerSettings(t, map[string]string{
		"telegramEnabled":       "false",
		"telegramBackupEnabled": "true",
		"telegramBackupCron":    "*/5 * * * *",
	})
	c := cron.New()
	scheduler := NewTelegramBackupScheduler(c)
	scheduler.Run()
	if scheduler.entryID != 0 || scheduler.currentSpec != "" {
		t.Fatalf("scheduler planned while telegram disabled: %#v", scheduler)
	}
}

func TestTelegramBackupJobUsesScheduledTrigger(t *testing.T) {
	prevAuditSync := service.AuditSyncForTest
	service.AuditSyncForTest = true
	t.Cleanup(func() { service.AuditSyncForTest = prevAuditSync })
	initCronJobTestDB(t)
	if _, err := (&service.SettingService{}).GetAllSetting(); err != nil {
		t.Fatal(err)
	}
	job := NewTelegramBackupJob()
	job.Run()
	var event model.AuditEvent
	if err := database.GetDB().Where("event = ?", "tg_backup_failed").Order("id desc").First(&event).Error; err != nil {
		t.Fatal(err)
	}
	if event.Actor != "system" || !strings.Contains(string(event.Details), `"trigger":"scheduled"`) {
		t.Fatalf("scheduled job did not audit scheduled trigger: %#v details=%s", event, event.Details)
	}
}

func updateTelegramBackupSchedulerSettings(t *testing.T, settings map[string]string) {
	t.Helper()
	for key, value := range settings {
		if err := database.GetDB().Model(model.Setting{}).Where("key = ?", key).Update("value", value).Error; err != nil {
			t.Fatal(err)
		}
	}
}
