package cronjob

import (
	"context"
	"strings"
	"sync"

	"github.com/deposist/s-ui-x/logger"
	"github.com/deposist/s-ui-x/service"
	"github.com/robfig/cron/v3"
)

type TelegramBackupJob struct {
	service.TelegramBackupService
}

func NewTelegramBackupJob() *TelegramBackupJob {
	return &TelegramBackupJob{}
}

func (j *TelegramBackupJob) Run() {
	j.TelegramBackupService.RunOnce(context.Background(), service.TelegramBackupTriggerScheduled)
}

type TelegramBackupScheduler struct {
	service.SettingService

	cron        *cron.Cron
	mu          sync.Mutex
	currentSpec string
	entryID     cron.EntryID
}

func NewTelegramBackupScheduler(c *cron.Cron) *TelegramBackupScheduler {
	return &TelegramBackupScheduler{cron: c}
}

func (s *TelegramBackupScheduler) Run() {
	telegramEnabled, err := s.SettingService.GetTelegramEnabled()
	if err != nil {
		logger.Warning("telegram backup telegram-enabled setting read failed:", err)
		return
	}
	backupEnabled, err := s.SettingService.GetTelegramBackupEnabled()
	if err != nil {
		logger.Warning("telegram backup enabled setting read failed:", err)
		return
	}
	spec, err := s.SettingService.GetTelegramBackupCron()
	if err != nil {
		logger.Warning("telegram backup cron read failed:", err)
		return
	}
	spec = strings.TrimSpace(spec)
	if !telegramEnabled || !backupEnabled {
		spec = ""
	}
	if err := s.reconcile(spec); err != nil {
		logger.Warning("telegram backup scheduler failed:", err)
	}
}

func (s *TelegramBackupScheduler) reconcile(spec string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if spec == s.currentSpec {
		return nil
	}
	if s.entryID != 0 {
		s.cron.Remove(s.entryID)
		s.entryID = 0
	}
	s.currentSpec = ""
	if spec == "" {
		return nil
	}
	schedule, err := service.ParseTelegramReportCron(spec)
	if err != nil {
		return err
	}
	entryID := s.cron.Schedule(schedule, NewTelegramBackupJob())
	s.entryID = entryID
	s.currentSpec = spec
	return nil
}
