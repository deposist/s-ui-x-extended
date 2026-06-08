package cronjob

import (
	"time"

	"github.com/robfig/cron/v3"
)

type CronJob struct {
	cron *cron.Cron
}

func NewCronJob() *CronJob {
	return &CronJob{}
}

func (c *CronJob) Start(loc *time.Location, trafficAge int) error {
	c.cron = cron.New(
		cron.WithLocation(loc),
		cron.WithSeconds(),
		// Recover keeps a panicking job (e.g. a nil-deref in a goroutine) from
		// taking down the whole panel process; SkipIfStillRunning prevents a
		// slow job from overlapping itself.
		cron.WithChain(
			cron.Recover(cronLogger{}),
			cron.SkipIfStillRunning(cronLogger{}),
		),
	)
	// Start stats job
	if _, err := c.cron.AddJob("@every 10s", NewStatsJob(trafficAge > 0)); err != nil {
		return err
	}
	// Start expiry job
	if _, err := c.cron.AddJob("@every 1m", NewDepleteJob()); err != nil {
		return err
	}
	// Start deleting old stats
	if trafficAge > 0 {
		if _, err := c.cron.AddJob("@daily", NewDelStatsJob(trafficAge)); err != nil {
			return err
		}
	}
	// Start core if it is not running
	if _, err := c.cron.AddJob("@every 5s", NewCheckCoreJob()); err != nil {
		return err
	}
	// CPU hysteresis notifications
	if _, err := c.cron.AddJob("@every 12s", NewCPUHysteresisJob()); err != nil {
		return err
	}
	// Observability history sampling
	if _, err := c.cron.AddJob("@every 2s", NewObservabilitySamplerJob()); err != nil {
		return err
	}
	// Telegram scheduled report dynamic replanning
	reportScheduler := NewTelegramReportScheduler(c.cron)
	reportScheduler.Run()
	if _, err := c.cron.AddJob("@every 1m", reportScheduler); err != nil {
		return err
	}
	// Telegram encrypted database backup dynamic replanning
	backupScheduler := NewTelegramBackupScheduler(c.cron)
	backupScheduler.Run()
	if _, err := c.cron.AddJob("@every 1m", backupScheduler); err != nil {
		return err
	}
	// database WAL checkpoint
	if _, err := c.cron.AddJob("@every 10m", NewWALCheckpointJob()); err != nil {
		return err
	}
	// retention cleanup
	if _, err := c.cron.AddJob("@every 1h", NewAuditGCJob()); err != nil {
		return err
	}
	// Paid Subscriptions: poll out-of-band payments + expire stale orders
	if _, err := c.cron.AddJob("@every 20s", NewPaidSubPollJob()); err != nil {
		return err
	}

	c.cron.Start()

	return nil
}

func (c *CronJob) Stop() {
	if c.cron != nil {
		c.cron.Stop()
	}
}
