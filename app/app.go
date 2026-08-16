package app

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/deposist/s-ui-x-extended/cmd/migration"
	"github.com/deposist/s-ui-x-extended/config"
	"github.com/deposist/s-ui-x-extended/core"
	"github.com/deposist/s-ui-x-extended/cronjob"
	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/ipmonitor"
	"github.com/deposist/s-ui-x-extended/logger"
	"github.com/deposist/s-ui-x-extended/paidsub"
	"github.com/deposist/s-ui-x-extended/service"
	"github.com/deposist/s-ui-x-extended/sub"
	"github.com/deposist/s-ui-x-extended/web"
)

type APP struct {
	service.SettingService
	configService      *service.ConfigService
	webServer          *web.Server
	subServer          *sub.Server
	cronJob            *cronjob.CronJob
	core               *core.Core
	runtime            *service.Runtime
	awgEndpointManager *service.AWGEndpointManager
	awgMu              sync.Mutex
	awgRun             *awgLoopRun
}

type awgLoopRun struct {
	cancel context.CancelFunc
	done   chan struct{}
}

var awgLoopIntervals = func() (time.Duration, time.Duration) {
	return 30 * time.Second, 60 * time.Second
}

func NewApp() *APP {
	return &APP{}
}

func (a *APP) Init() error {
	log.Printf("%v %v", config.GetName(), config.GetVersion())

	service.PanelUpdateDatabaseSnapshot = database.PreparePanelUpdateSnapshot
	service.PanelUpdateDatabaseRestore = database.RestorePanelUpdateSnapshot
	a.initLog()

	// Resolve updater recovery before schema migration or database initialization.
	// Any unresolved marker/rollback error is fatal: continuing could migrate a
	// database under an unconfirmed binary and destroy rollback compatibility.
	if exe, err := os.Executable(); err == nil && exe != "" {
		rolledBack, recoveryErr := service.RecoverPendingUpdate(exe)
		if recoveryErr != nil {
			return fmt.Errorf("self-update recovery: %w", recoveryErr)
		}
		if rolledBack {
			return service.ErrPanelUpdateRolledBack
		}
	}
	if exe, err := os.Executable(); err == nil && exe != "" {
		if err := service.MarkPendingUpdateBooting(exe); err != nil {
			return fmt.Errorf("prepare self-update boot: %w", err)
		}
	}

	// Run schema migrations against the on-disk DB before opening it. This
	// turns the upgrade flow into a one-step procedure: drop in the new
	// binary, restart, and the panel adapts the legacy schema in place. The
	// run is a no-op if the database is already at the current version or if
	// it does not yet exist (first install).
	if err := migration.MigrateDb(); err != nil {
		return err
	}

	err := database.InitDB(config.GetDBPath())
	if err != nil {
		return err
	}

	// Init Setting
	if _, err := a.SettingService.GetAllSetting(); err != nil {
		logger.Warning("failed to initialize settings: ", err)
	}

	// Re-seal any secret settings still encrypted under a DB-derived key once an
	// out-of-database SUI_SECRETBOX_KEY is configured. No-op without the env key,
	// idempotent, and fail-safe per row; a failure here must not block startup.
	if n, err := a.SettingService.ResealSecretSettings(); err != nil {
		logger.Warning("failed to re-seal secret settings: ", err)
	} else if n > 0 {
		logger.Info("re-sealed ", n, " secret setting(s) under SUI_SECRETBOX_KEY")
	}
	if err := ipmonitor.WarmUp(); err != nil {
		return err
	}

	a.core = core.NewCore()
	a.runtime = service.NewRuntime(a.core)
	service.SetDefaultRuntime(a.runtime)

	// Backfill of sudoku/mieru client links (so existing clients get their
	// sudoku://mierus:// links and QR without a manual re-save) runs lazily on
	// the first panel data load, not here: startup has no request host and
	// settings.webDomain is usually blank, which would generate links with an
	// empty server. See ClientService.RegenerateMissingLocalLinksOnce.

	// Mirror ipmonitor IP-limit enforcement into the durable audit log (D-5).
	// Set via a hook to avoid an import cycle; debounced upstream so it cannot
	// flood the audit log.
	ipmonitor.SecurityEventAuditHook = func(clientName string, kind string, payload map[string]any) {
		_ = (&service.AuditService{}).Record(service.AuditEvent{
			Actor:    "system",
			Event:    "ip_limit_enforced",
			Resource: "ipmonitor",
			Severity: service.AuditSeverityWarn,
			Details:  payload,
		})
	}

	a.cronJob = cronjob.NewCronJob()
	a.webServer, err = web.NewServer(web.WithRuntime(a.runtime))
	if err != nil {
		return err
	}
	a.subServer = sub.NewServer()

	a.configService = service.NewConfigServiceWithRuntime(a.runtime)

	// Paid Subscriptions owns its schema and cannot safely run unless its
	// uniqueness invariants were verified. Propagate the error through Init so
	// startup fails actionably instead of exposing a partially migrated module.
	if err := paidsub.EnsureSchema(database.GetDB()); err != nil {
		return err
	}
	// Provision the AWG device-key encryption key before any component can
	// need it: web-panel self-updates swap the binary without running
	// install.sh, so startup is the one path every update shares.
	if err := service.EnsureAWGEncryptionKey(); err != nil {
		logger.Warning("AWG device encryption unavailable: ", err)
	}
	// Convert the removed settings-based managed-AWG mode into endpoint
	// metadata before the managers come up; failures leave the panel running
	// but loudly logged so the admin can fix the named endpoint and restart.
	if migrated, err := service.MigrateLegacyAWGSettings(database.GetDB()); err != nil {
		logger.Warning("legacy managed AWG migration failed: ", err)
	} else if migrated {
		logger.Info("migrated legacy managed AWG settings to endpoint metadata")
	}
	a.awgEndpointManager = service.NewAWGEndpointManager(a.runtime)
	a.runtime.SetAWGEndpointDeviceService(a.awgEndpointManager)
	a.runtime.SetAWGReconcileHook(a.awgEndpointManager.ReconcileAll)
	a.runtime.SetAWGClientStateHook(a.awgEndpointManager)
	// Outbound failover observability table (non-authoritative; idempotent).
	if err := service.EnsureFailoverSchema(database.GetDB()); err != nil {
		logger.Warning("failed to ensure failover_state schema: ", err)
	}

	return nil
}

// lifecycleStep is a small, testable start/rollback seam. A component is added
// to the rollback stack only after its Start succeeds.
type lifecycleStep struct {
	name  string
	start func() error
	stop  func()
}

func runStartLifecycle(steps []lifecycleStep) error {
	started := make([]lifecycleStep, 0, len(steps))
	for _, step := range steps {
		if err := step.start(); err != nil {
			for i := len(started) - 1; i >= 0; i-- {
				started[i].stop()
			}
			return err
		}
		started = append(started, step)
	}
	return nil
}

func (a *APP) Start() error {
	loc, err := a.SettingService.GetTimeLocation()
	if err != nil {
		return err
	}
	trafficAge, err := a.SettingService.GetTrafficAge()
	if err != nil {
		return err
	}
	if err := runStartLifecycle([]lifecycleStep{
		{name: "awg endpoints", start: func() error {
			if a.awgEndpointManager == nil {
				return nil
			}
			return a.awgEndpointManager.Start()
		}, stop: func() {
			if a.awgEndpointManager == nil {
				return
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := a.awgEndpointManager.StopAll(ctx); err != nil {
				logger.Warning("rollback endpoint AWG managers err: ", err)
			}
		}},
		{name: "cron", start: func() error { return a.cronJob.Start(loc, trafficAge) }, stop: func() {
			if err := a.cronJob.Stop(); err != nil {
				logger.Warning("rollback cron err: ", err)
			}
		}},
		{name: "web", start: a.webServer.Start, stop: func() {
			if err := a.webServer.Stop(); err != nil {
				logger.Warning("rollback Web Server err: ", err)
			}
		}},
		{name: "sub", start: a.subServer.Start, stop: func() {
			if err := a.subServer.Stop(); err != nil {
				logger.Warning("rollback Sub Server err: ", err)
			}
		}},
	}); err != nil {
		return err
	}

	// Experimental Paid Subscriptions client bot. Self-gates on paidSubEnabled
	// internally, so starting unconditionally is safe and lets the admin toggle
	// it at runtime without a restart.
	paidsub.StartBot()

	// A core start failure is intentionally non-fatal: the web/sub panel must
	// stay up so the admin can fix a bad sing-box config through the UI. The
	// failure is surfaced loudly here and reflected in the panel's core status.
	if err = a.configService.StartCore(); err != nil {
		logger.Error("sing-box core failed to start; panel stays up so you can fix the config: ", err)
	}
	a.startAWGLoops()

	// Healthy boot reached: clear any pending self-update marker so this start is
	// not counted as a failed update attempt (SR-012). No-op when no update is
	// pending (e.g. normal restarts).
	if exe, err := os.Executable(); err == nil && exe != "" {
		if err := service.ConfirmPendingUpdate(exe); err != nil {
			return fmt.Errorf("confirm self-update: %w", err)
		}
	}

	return nil
}

func (a *APP) Stop() {
	updateCtx, updateCancel := context.WithTimeout(context.Background(), 10*time.Second)
	if err := service.StopPanelUpdate(updateCtx); err != nil {
		logger.Warning("stop panel update err:", err)
	}
	updateCancel()
	service.StopRestartManager()
	paidsub.StopAWGCommands()
	paidSubCtx, paidSubCancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := paidsub.StopBot(paidSubCtx); err != nil {
		logger.Warning("stop paidsub bot err:", err)
	}
	paidSubCancel()
	awgLoopCtx, awgLoopCancel := context.WithTimeout(context.Background(), 10*time.Second)
	if err := a.stopAWGLoops(awgLoopCtx); err != nil {
		logger.Warning("stop AWG loops err:", err)
	}
	awgLoopCancel()
	if a.awgEndpointManager != nil {
		awgCtx, awgCancel := context.WithTimeout(context.Background(), 10*time.Second)
		if err := a.awgEndpointManager.StopAll(awgCtx); err != nil {
			logger.Warning("stop endpoint AWG managers err:", err)
		}
		awgCancel()
	}
	if err := a.cronJob.Stop(); err != nil {
		logger.Warning("stop cron err:", err)
	}
	err := a.subServer.Stop()
	if err != nil {
		logger.Warning("stop Sub Server err:", err)
	}
	err = a.webServer.Stop()
	if err != nil {
		logger.Warning("stop Web Server err:", err)
	}
	err = a.configService.StopCore()
	if err != nil {
		logger.Warning("stop Core err:", err)
	}
	tokenCtx, tokenCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer tokenCancel()
	if err := service.StopTokenUseDebouncer(tokenCtx); err != nil {
		logger.Warning("stop token use debouncer err:", err)
	}
	telegramCtx, telegramCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer telegramCancel()
	if err := service.StopTelegramNotifier(telegramCtx); err != nil {
		logger.Warning("stop telegram notifier err:", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := service.StopAuditWriter(ctx); err != nil {
		logger.Warning("stop audit writer err:", err)
	}
}

func (a *APP) startAWGLoops() {
	a.awgMu.Lock()
	defer a.awgMu.Unlock()
	if a.awgRun != nil {
		select {
		case <-a.awgRun.done:
			a.awgRun = nil
		default:
			return
		}
	}
	reconcileInterval, statsInterval := awgLoopIntervals()
	ctx, cancel := context.WithCancel(context.Background())
	run := &awgLoopRun{cancel: cancel, done: make(chan struct{})}
	a.awgRun = run
	go func() {
		defer close(run.done)
		reconcileTicker := time.NewTicker(reconcileInterval)
		statsTicker := time.NewTicker(statsInterval)
		defer reconcileTicker.Stop()
		defer statsTicker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-reconcileTicker.C:
				if a.awgEndpointManager != nil {
					if err := a.awgEndpointManager.ReconcileAll(ctx); err != nil && ctx.Err() == nil {
						logger.Warning("periodic endpoint AWG reconcile failed: ", err)
					}
				}
			case <-statsTicker.C:
				if a.awgEndpointManager != nil {
					if err := a.awgEndpointManager.CollectStatsAll(ctx); err != nil && ctx.Err() == nil {
						logger.Warning("periodic endpoint AWG stats failed: ", err)
					}
				}
			}
		}
	}()
}

func (a *APP) stopAWGLoops(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	a.awgMu.Lock()
	run := a.awgRun
	if run == nil {
		a.awgMu.Unlock()
		return nil
	}
	run.cancel()
	a.awgMu.Unlock()

	select {
	case <-run.done:
		a.awgMu.Lock()
		if a.awgRun == run {
			a.awgRun = nil
		}
		a.awgMu.Unlock()
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (a *APP) initLog() {
	switch config.GetLogLevel() {
	case config.Debug:
		logger.Init(logger.LevelDebug)
	case config.Info:
		logger.Init(logger.LevelInfo)
	case config.Warn:
		logger.Init(logger.LevelWarning)
	case config.Error:
		logger.Init(logger.LevelError)
	default:
		logger.Init(logger.LevelInfo)
	}
}

func (a *APP) RestartApp() {
	a.Stop()
	if err := a.Start(); err != nil {
		logger.Warning("failed to restart app: ", err)
	}
}
