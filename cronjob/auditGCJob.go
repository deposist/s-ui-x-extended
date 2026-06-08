package cronjob

import (
	"time"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
	"github.com/deposist/s-ui-x/ipmonitor"
	"github.com/deposist/s-ui-x/logger"
	"github.com/deposist/s-ui-x/service"
)

type AuditGCJob struct {
	service.AuditService
	service.SettingService
}

func NewAuditGCJob() *AuditGCJob {
	return &AuditGCJob{}
}

func (s *AuditGCJob) Run() {
	auditRetentionDays, err := s.SettingService.GetAuditRetentionDays()
	if err != nil {
		logger.Warning("Reading audit retention failed: ", err)
	} else if err := s.AuditService.Prune(auditRetentionDays); err != nil {
		logger.Warning("Deleting old audit events failed: ", err)
	}

	ipRetentionDays, err := s.SettingService.GetIPHistoryRetentionDays()
	if err != nil {
		logger.Warning("Reading IP history retention failed: ", err)
	} else if err := pruneClientIPs(ipRetentionDays); err != nil {
		logger.Warning("Deleting old client IP history failed: ", err)
	}
}

func pruneClientIPs(retentionDays int) error {
	if retentionDays <= 0 {
		return nil
	}
	db := database.GetDB()
	if db == nil {
		return nil
	}
	before := time.Now().Add(-time.Duration(retentionDays) * 24 * time.Hour).Unix()
	if err := db.Where("last_seen < ?", before).Delete(&model.ClientIP{}).Error; err != nil {
		return err
	}
	ipmonitor.InvalidateAllCache()
	return nil
}
