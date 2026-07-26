package cronjob

import (
	"github.com/deposist/s-ui-x-extended/logger"
	"github.com/deposist/s-ui-x-extended/service"
)

// RuleSetRefreshJob re-downloads the materialized .srs assets.
//
// This takes over the job that `update_interval` did for a remote rule-set.
// Since the core now reads the files from disk, nothing else would ever update
// them, and the routing data would slowly go stale.
//
// A failure here is intentionally harmless: the previously downloaded files
// stay in place and the core keeps using them, so a blocked or flaky network
// only means the data is older than desired, never that startup breaks.
type RuleSetRefreshJob struct {
	service.RuleSetAssetService
}

func NewRuleSetRefreshJob() *RuleSetRefreshJob {
	return &RuleSetRefreshJob{}
}

func (j *RuleSetRefreshJob) Run() {
	count, err := j.RuleSetAssetService.Refresh()
	if err != nil {
		logger.Warning("rule-set refresh failed, keeping existing files: ", err)
		return
	}
	if count > 0 {
		logger.Info("refreshed rule-set files: ", count)
	}
}
