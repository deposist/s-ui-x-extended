package app

import (
	"context"
	"log"
	"os"
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
	configService *service.ConfigService
	webServer     *web.Server
	subServer     *sub.Server
	cronJob       *cronjob.CronJob
	core          *core.Core
	runtime       *service.Runtime
	awgManager    *service.AWGManager
	awgCancel     context.CancelFunc
	awgDone       chan struct{}
}

func NewApp() *APP {
	return &APP{}
}

func (a *APP) Init() error {
	log.Printf("%v %v", config.GetName(), config.GetVersion())

	a.initLog()

	// Self-update safety net (SR-012): if a freshly-applied binary keeps failing
	// to boot, roll back to the backed-up previous binary and exit so systemd
	// restarts into the restored version. Runs once per process (not on the
	// in-process SIGHUP RestartApp, which does not re-run Init).
	if exe, err := os.Executable(); err == nil && exe != "" {
		if service.CheckPendingUpdate(exe) {
			logger.Warning("self-update: new binary failed to boot; rolled back to previous version, restarting")
			os.Exit(1)
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

	// Experimental Paid Subscriptions module owns its own schema; create it
	// idempotently at startup. Non-fatal: a failure here must not block core.
	if err := paidsub.EnsureSchema(database.GetDB()); err != nil {
		logger.Warning("failed to ensure paidsub schema: ", err)
	}
	if awgSettings, awgErr := a.SettingService.GetAWGSettings(); awgErr != nil {
		logger.Warning("failed to load AWG settings: ", awgErr)
	} else {
		a.awgManager = service.NewAWGManager(a.runtime, service.NewAWGProvisioner(a.runtime, awgSettings.EndpointTag), 128)
		if startErr := a.awgManager.Start(); startErr != nil {
			logger.Warning("failed to start AWG manager: ", startErr)
		} else {
			a.runtime.SetAWGClientStateHook(a.awgManager)
			a.runtime.SetAWGDeviceService(a.awgManager)
			a.runtime.SetAWGReconcileHook(func(ctx context.Context) error {
				_, err := a.awgManager.Reconcile(ctx)
				return err
			})
		}
	}
	// Outbound failover observability table (non-authoritative; idempotent).
	if err := service.EnsureFailoverSchema(database.GetDB()); err != nil {
		logger.Warning("failed to ensure failover_state schema: ", err)
	}

	return nil
}

func (a *APP) Start() error {
	if a.awgManager != nil {
		if err := a.awgManager.Start(); err != nil {
			return err
		}
	}
	loc, err := a.SettingService.GetTimeLocation()
	if err != nil {
		return err
	}

	trafficAge, err := a.SettingService.GetTrafficAge()
	if err != nil {
		return err
	}

	err = a.cronJob.Start(loc, trafficAge)
	if err != nil {
		return err
	}

	err = a.webServer.Start()
	if err != nil {
		return err
	}

	err = a.subServer.Start()
	if err != nil {
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
		service.ClearPendingUpdate(exe)
	}

	return nil
}

func (a *APP) Stop() {
	service.StopRestartManager()
	a.stopAWGLoops()
	paidsub.StopAWGCommands()
	if a.awgManager != nil {
		awgCtx, awgCancel := context.WithTimeout(context.Background(), 10*time.Second)
		if err := a.awgManager.Stop(awgCtx); err != nil {
			logger.Warning("stop AWG manager err:", err)
		}
		awgCancel()
	}
	a.cronJob.Stop()
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
	paidSubCtx, paidSubCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer paidSubCancel()
	if err := paidsub.StopBot(paidSubCtx); err != nil {
		logger.Warning("stop paidsub bot err:", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := service.StopAuditWriter(ctx); err != nil {
		logger.Warning("stop audit writer err:", err)
	}
}

func (a *APP) startAWGLoops() {
	if a.awgManager == nil || a.awgCancel != nil {
		return
	}
	settings, err := a.SettingService.GetAWGSettings()
	if err != nil || !settings.Enabled {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	a.awgCancel, a.awgDone = cancel, done
	go func() {
		defer close(done)
		reconcileTicker := time.NewTicker(time.Duration(settings.ReconcileIntervalSec) * time.Second)
		statsTicker := time.NewTicker(time.Duration(settings.StatsIntervalSec) * time.Second)
		defer reconcileTicker.Stop()
		defer statsTicker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-reconcileTicker.C:
				if _, err := a.awgManager.Reconcile(ctx); err != nil {
					logger.Warning("periodic AWG reconcile failed: ", err)
				}
			case <-statsTicker.C:
				if _, err := a.awgManager.CollectStats(ctx); err != nil {
					logger.Warning("periodic AWG stats failed: ", err)
				}
			}
		}
	}()
}

func (a *APP) stopAWGLoops() {
	if a.awgCancel == nil {
		return
	}
	a.awgCancel()
	if a.awgDone != nil {
		<-a.awgDone
	}
	a.awgCancel, a.awgDone = nil, nil
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
