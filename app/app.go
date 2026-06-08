package app

import (
	"context"
	"log"
	"time"

	"github.com/deposist/s-ui-x/cmd/migration"
	"github.com/deposist/s-ui-x/config"
	"github.com/deposist/s-ui-x/core"
	"github.com/deposist/s-ui-x/cronjob"
	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/ipmonitor"
	"github.com/deposist/s-ui-x/logger"
	"github.com/deposist/s-ui-x/paidsub"
	"github.com/deposist/s-ui-x/service"
	"github.com/deposist/s-ui-x/sub"
	"github.com/deposist/s-ui-x/web"
)

type APP struct {
	service.SettingService
	configService *service.ConfigService
	webServer     *web.Server
	subServer     *sub.Server
	cronJob       *cronjob.CronJob
	core          *core.Core
	runtime       *service.Runtime
}

func NewApp() *APP {
	return &APP{}
}

func (a *APP) Init() error {
	log.Printf("%v %v", config.GetName(), config.GetVersion())

	a.initLog()

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

	return nil
}

func (a *APP) Stop() {
	service.StopRestartManager()
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

func (a *APP) GetCore() *core.Core {
	return a.core
}
