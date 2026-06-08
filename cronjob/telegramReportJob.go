package cronjob

import (
	"sync"
	"time"

	"github.com/deposist/s-ui-x/logger"
	"github.com/deposist/s-ui-x/service"
	"github.com/robfig/cron/v3"
)

type TelegramReportJob struct {
	service.TelegramService
}

func NewTelegramReportJob() *TelegramReportJob {
	return &TelegramReportJob{}
}

func (j *TelegramReportJob) Run() {
	j.TelegramService.NotifyTelegramEvent("scheduled_report", map[string]string{
		"ts": time.Now().UTC().Format(time.RFC3339),
	})
}

type TelegramReportScheduler struct {
	service.SettingService

	cron        *cron.Cron
	mu          sync.Mutex
	currentSpec string
	entryID     cron.EntryID
}

func NewTelegramReportScheduler(c *cron.Cron) *TelegramReportScheduler {
	return &TelegramReportScheduler{cron: c}
}

func (s *TelegramReportScheduler) Run() {
	enabled, err := s.SettingService.GetTelegramReport()
	if err != nil {
		logger.Warning("telegram report setting read failed:", err)
		return
	}
	spec, err := s.SettingService.GetTelegramReportCron()
	if err != nil {
		logger.Warning("telegram report cron read failed:", err)
		return
	}
	if !enabled {
		spec = ""
	}
	if err := s.reconcile(spec); err != nil {
		logger.Warning("telegram report scheduler failed:", err)
	}
}

func (s *TelegramReportScheduler) reconcile(spec string) error {
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
	entryID := s.cron.Schedule(schedule, NewTelegramReportJob())
	s.entryID = entryID
	s.currentSpec = spec
	return nil
}
