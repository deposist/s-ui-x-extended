package cronjob

import (
	"context"
	"sync"
	"time"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/robfig/cron/v3"
)

const defaultStopTimeout = 10 * time.Second

// databaseMaintenanceJob makes cron jobs participate in the restore drain
// barrier. It is deliberately applied at the scheduler boundary so every
// current and future scheduled job cannot obtain a DB handle while restore is
// swapping it. While the restore barrier is held the tick is skipped entirely:
// blocking would occupy the cron run slot and make SkipIfStillRunning log
// "cron: skip" at every schedule until the restore finishes. Periodic jobs
// tolerate the missed tick and run again after the swap.
type databaseMaintenanceJob struct {
	cron.Job
}

func (j databaseMaintenanceJob) Run() {
	leave, ok := database.TryEnterDBOperation()
	if !ok {
		return
	}
	defer leave()
	j.Job.Run()
}

type CronJob struct {
	mu          sync.Mutex
	cron        *cron.Cron
	ctx         context.Context
	cancel      context.CancelFunc
	stopTimeout time.Duration
}

func NewCronJob() *CronJob {
	return &CronJob{stopTimeout: defaultStopTimeout}
}

func (c *CronJob) Start(loc *time.Location, trafficAge int) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	// A new scheduler generation must never overlap an old one. In particular,
	// cron.Stop returns a context that is closed only after running jobs finish.
	if err := c.stopLocked(); err != nil {
		return err
	}

	c.ctx, c.cancel = context.WithCancel(context.Background())
	c.cron = cron.New(
		cron.WithLocation(loc),
		cron.WithSeconds(),
		cron.WithChain(
			cron.Recover(cronLogger{}),
			cron.SkipIfStillRunning(cronLogger{}),
			cron.JobWrapper(func(job cron.Job) cron.Job { return databaseMaintenanceJob{Job: job} }),
		),
	)
	if _, err := c.cron.AddJob("@every 10s", NewStatsJob(trafficAge > 0)); err != nil {
		return err
	}
	if _, err := c.cron.AddJob("@every 1m", NewDepleteJob()); err != nil {
		return err
	}
	if trafficAge > 0 {
		if _, err := c.cron.AddJob("@daily", NewDelStatsJob(trafficAge)); err != nil {
			return err
		}
	}
	if _, err := c.cron.AddJob("@every 5s", NewCheckCoreJob()); err != nil {
		return err
	}
	if _, err := c.cron.AddJob("@every 12s", NewCPUHysteresisJob()); err != nil {
		return err
	}
	if _, err := c.cron.AddJob("@every 2s", NewObservabilitySamplerJob()); err != nil {
		return err
	}
	if _, err := c.cron.AddJob("@every 5s", NewFailoverJob()); err != nil {
		return err
	}
	reportScheduler := NewTelegramReportScheduler(c.cron)
	reportScheduler.Run()
	if _, err := c.cron.AddJob("@every 1m", reportScheduler); err != nil {
		return err
	}
	backupScheduler := NewTelegramBackupScheduler(c.cron)
	backupScheduler.Run()
	if _, err := c.cron.AddJob("@every 1m", backupScheduler); err != nil {
		return err
	}
	if _, err := c.cron.AddJob("@every 10m", NewWALCheckpointJob()); err != nil {
		return err
	}
	if _, err := c.cron.AddJob("@every 1h", NewAuditGCJob()); err != nil {
		return err
	}
	if _, err := c.cron.AddJob("@every 20s", NewPaidSubPollJob(c.ctx)); err != nil {
		return err
	}
	if _, err := c.cron.AddJob("@every 12h", NewCertRenewJob(c.ctx)); err != nil {
		return err
	}
	// Replaces the rule-set `update_interval`, which no longer applies now that
	// the core reads these files from disk instead of fetching them.
	if _, err := c.cron.AddJob("@daily", NewRuleSetRefreshJob()); err != nil {
		return err
	}
	c.cron.Start()
	return nil
}

// Stop cancels jobs and waits, up to a bounded timeout, for cron's active jobs.
// Its error lets callers decide how to report a shutdown timeout without allowing
// Start to launch an overlapping scheduler generation.
func (c *CronJob) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.stopLocked()
}

func (c *CronJob) stopLocked() error {
	if c.cancel != nil {
		c.cancel()
		c.cancel = nil
	}
	if c.cron == nil {
		return nil
	}
	stopped := c.cron.Stop()
	timeout := c.stopTimeout
	if timeout <= 0 {
		timeout = defaultStopTimeout
	}
	select {
	case <-stopped.Done():
		c.cron = nil
		return nil
	case <-time.After(timeout):
		return context.DeadlineExceeded
	}
}
