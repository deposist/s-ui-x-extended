package database

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/deposist/s-ui-x-extended/config"
	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/deposist/s-ui-x-extended/util/common"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

var (
	dbMu sync.RWMutex
	db   *gorm.DB

	// maintenanceMu is held exclusively while a database restore replaces the
	// SQLite file. Request/job entry points hold a read lock for their complete
	// DB-using operation, so restore first drains them and then prevents new
	// work from acquiring a handle that is about to be closed.
	maintenanceMu      sync.RWMutex
	restoreMu          sync.Mutex
	restoreStartedHook func()
)

// ErrRestoreInProgress is returned instead of allowing two restores to race
// over the same live, temporary and fallback database paths.
var ErrRestoreInProgress = errors.New("database restore already in progress")

// EnterDBOperation brackets a request or scheduled job that may use GetDB.
// The returned function must be deferred by the caller. sync.RWMutex blocks
// new readers while beginRestore is waiting for its exclusive lock, so work
// admitted before restore drains and later work waits for the completed swap.
func EnterDBOperation() func() {
	maintenanceMu.RLock()
	return maintenanceMu.RUnlock
}

// beginRestore serializes restore attempts and drains operations that entered
// through EnterDBOperation. It must be paired with the returned function.
func beginRestore() (func(), error) {
	if !restoreMu.TryLock() {
		return nil, ErrRestoreInProgress
	}
	if restoreStartedHook != nil {
		restoreStartedHook()
	}
	maintenanceMu.Lock()
	return func() {
		maintenanceMu.Unlock()
		restoreMu.Unlock()
	}, nil
}

var adaptToCurrentVersion = AdaptToCurrentVersion

const (
	dbMaxOpenConnsEnv        = "SUI_DB_MAX_OPEN_CONNS"
	dbMaxIdleConnsEnv        = "SUI_DB_MAX_IDLE_CONNS"
	defaultDBMaxOpenConns    = 8
	defaultDBMaxIdleConns    = 4
	defaultDBConnMaxLifetime = time.Hour
)

type dbPoolConfig struct {
	maxOpenConns    int
	maxIdleConns    int
	connMaxLifetime time.Duration
}

type dbPoolSetter interface {
	SetMaxOpenConns(int)
	SetMaxIdleConns(int)
	SetConnMaxLifetime(time.Duration)
}

func initUser(dbPath string) error {
	var count int64
	err := db.Model(&model.User{}).Count(&count).Error
	if err != nil {
		return err
	}
	passwordPath := initialAdminPasswordPath(dbPath)
	if count == 0 {
		password := common.Random(24)
		passwordHash, err := common.HashPassword(password)
		if err != nil {
			return err
		}
		if err := writeInitialAdminPassword(passwordPath, password); err != nil {
			return err
		}
		user := &model.User{
			Username:           "admin",
			Password:           passwordHash,
			ForcePasswordReset: true,
		}
		if err := db.Create(user).Error; err != nil {
			_ = os.Remove(passwordPath)
			return err
		}
		notifyInitialAdminPasswordSaved(passwordPath)
		return nil
	}
	warnIfInitialAdminPasswordFileExists(passwordPath)
	return nil
}

func OpenDB(dbPath string) error {
	dir := filepath.Dir(dbPath)
	err := os.MkdirAll(dir, 0o750)
	if err != nil {
		return err
	}

	var gormLog gormlogger.Interface

	if config.IsDebug() {
		gormLog = gormlogger.Default
	} else {
		gormLog = gormlogger.Discard
	}

	c := &gorm.Config{
		Logger: gormLog,
	}
	sep := "?"
	if strings.Contains(dbPath, "?") {
		sep = "&"
	}
	dsn := dbPath + sep + "_busy_timeout=10000&_journal_mode=WAL&_synchronous=NORMAL&_foreign_keys=on"
	openedDB, err := gorm.Open(sqlite.Open(dsn), c)
	if err != nil {
		return err
	}

	sqlDB, err := openedDB.DB()
	if err != nil {
		return err
	}
	// SQLite is a single-writer database. Allowing many concurrent open
	// connections only spreads writers across them and produces SQLITE_BUSY
	// errors during stats inserts. Keep a small read pool plus one effective
	// writer driven through `_busy_timeout` to serialize gracefully.
	applyDBPoolConfig(sqlDB, resolvedDBPoolConfig())

	if config.IsDebug() {
		openedDB = openedDB.Debug()
	}
	dbMu.Lock()
	db = openedDB
	dbMu.Unlock()
	return nil
}

func resolvedDBPoolConfig() dbPoolConfig {
	maxOpen := parseDBPoolLimitEnv(dbMaxOpenConnsEnv, defaultDBMaxOpenConns, func(value int) bool {
		return value > 0
	})
	maxIdle := parseDBPoolLimitEnv(dbMaxIdleConnsEnv, defaultDBMaxIdleConns, func(value int) bool {
		return value >= 0
	})
	if maxIdle > maxOpen {
		maxIdle = maxOpen
	}
	return dbPoolConfig{
		maxOpenConns:    maxOpen,
		maxIdleConns:    maxIdle,
		connMaxLifetime: defaultDBConnMaxLifetime,
	}
}

func parseDBPoolLimitEnv(key string, fallback int, valid func(int) bool) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil || !valid(parsed) {
		return fallback
	}
	return parsed
}

func applyDBPoolConfig(pool dbPoolSetter, cfg dbPoolConfig) {
	pool.SetMaxOpenConns(cfg.maxOpenConns)
	pool.SetMaxIdleConns(cfg.maxIdleConns)
	pool.SetConnMaxLifetime(cfg.connMaxLifetime)
}

func InitDB(dbPath string) error {
	err := OpenDB(dbPath)
	if err != nil {
		return err
	}

	// Default Outbounds
	if err := ensureDefaultOutbound(gormDefaultOutboundStore{db: db}); err != nil {
		return err
	}

	err = db.AutoMigrate(
		&model.Setting{},
		&model.Tls{},
		&model.Inbound{},
		&model.Outbound{},
		&model.Service{},
		&model.Endpoint{},
		&model.Provider{},
		&model.User{},
		&model.Tokens{},
		&model.Stats{},
		&model.ClientIP{},
		&model.Client{},
		&model.Changes{},
		&model.AuditEvent{},
	)
	if err != nil {
		return err
	}
	// The AWG tables are shared with paidsub.EnsureSchema, which upgrades them
	// with raw, guarded DDL. AutoMigrate only CREATES them on fresh databases:
	// running it over an existing raw-SQL table makes GORM's SQLite migrator
	// rebuild the table (its rendered column defaults differ textually from
	// the raw DDL), and that rebuild silently dropped columns on the
	// 1.0.2-beta1 -> 1.0.2 upgrade, crashing every boot with
	// "no such column: endpoint_id".
	for _, awgModel := range []any{&model.ClientEndpointAccess{}, &model.AWGDevice{}} {
		if db.Migrator().HasTable(awgModel) {
			continue
		}
		if err := db.AutoMigrate(awgModel); err != nil {
			return err
		}
	}
	if err := dropDeprecatedTables(); err != nil {
		return err
	}
	if err := ensureNoTLSRow(); err != nil {
		return err
	}
	if err := ensureIndexes(); err != nil {
		return err
	}
	err = initUser(dbPath)
	if err != nil {
		return err
	}
	// Post-migration adaptation is part of startup safety: indexes, legacy
	// password rehashes and the settings.version pointer must be current
	// before the panel serves traffic.
	if err := adaptToCurrentVersion(); err != nil {
		return fmt.Errorf("post-migration adapt failed: %w", err)
	}

	return nil
}

type defaultOutboundStore interface {
	HasTable(value any) bool
	CreateTable(values ...any) error
	Create(value any) error
}

type gormDefaultOutboundStore struct {
	db *gorm.DB
}

func (s gormDefaultOutboundStore) HasTable(value any) bool {
	return s.db.Migrator().HasTable(value)
}

func (s gormDefaultOutboundStore) CreateTable(values ...any) error {
	return s.db.Migrator().CreateTable(values...)
}

func (s gormDefaultOutboundStore) Create(value any) error {
	return s.db.Create(value).Error
}

func ensureDefaultOutbound(store defaultOutboundStore) error {
	if store.HasTable(&model.Outbound{}) {
		return nil
	}
	if err := store.CreateTable(&model.Outbound{}); err != nil {
		return err
	}
	defaultOutbound := []model.Outbound{
		{Type: "direct", Tag: "direct", Options: json.RawMessage(`{}`)},
	}
	return store.Create(&defaultOutbound)
}

// dropDeprecatedTables removes tables that backed features which have since been
// retired, so a re-saved database does not keep dead schema around. The 3x-ui
// scheduled-sync feature (xui_sync_profiles) and its SSH host-key store
// (xui_known_hosts) were removed in favour of the one-shot .db upload importer;
// these tables are dropped on every startup. DROP TABLE IF EXISTS is idempotent
// and a no-op once the tables are gone.
func dropDeprecatedTables() error {
	statements := []string{
		"DROP TABLE IF EXISTS xui_sync_profiles",
		"DROP TABLE IF EXISTS xui_known_hosts",
	}
	for _, query := range statements {
		if err := db.Exec(query).Error; err != nil {
			return err
		}
	}
	return nil
}

func ensureNoTLSRow() error {
	return ensureNoTLSRowOn(db)
}

func ensureNoTLSRowOn(target *gorm.DB) error {
	if target == nil {
		return nil
	}
	return target.Exec("INSERT OR IGNORE INTO tls(id, name, server, client) VALUES(0, ?, ?, ?)", "__none__", []byte("{}"), []byte("{}")).Error
}

func ensureIndexes() error {
	obsoleteIndexes := []string{
		"DROP INDEX IF EXISTS idx_client_ips_client_ip",
	}
	for _, query := range obsoleteIndexes {
		if err := db.Exec(query).Error; err != nil {
			return err
		}
	}
	// settings.key must be unique, but the historical read-then-create path could
	// race two first-start inserts into duplicate rows. Drop any duplicates
	// (keeping the lowest id, which carries the original default) before enforcing
	// uniqueness, so the unique index below also closes that TOCTOU gap and makes
	// every keyed settings read an index lookup instead of a full table scan (P-a).
	if err := db.Exec("DELETE FROM settings WHERE id NOT IN (SELECT MIN(id) FROM settings GROUP BY key)").Error; err != nil {
		return err
	}
	indexes := []string{
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_settings_key ON settings(key)",
		"CREATE INDEX IF NOT EXISTS idx_stats_lookup ON stats(date_time, resource, tag)",
		// GetStats filters `resource IN ? AND tag = ? AND date_time > ?`; an index
		// leading with the equality columns lets SQLite seek to the resource+tag
		// and walk the time range instead of scanning every tag in the window. The
		// date_time-leading idx_stats_lookup is kept for DelOldStats (date_time < ?).
		"CREATE INDEX IF NOT EXISTS idx_stats_resource_tag_dt ON stats(resource, tag, date_time)",
		"CREATE INDEX IF NOT EXISTS idx_stats_resource_dt ON stats(resource, date_time)",
		"CREATE INDEX IF NOT EXISTS idx_changes_lookup ON changes(date_time, actor, key)",
		"CREATE INDEX IF NOT EXISTS idx_audit_events_lookup ON audit_events(date_time, actor, event)",
		"CREATE INDEX IF NOT EXISTS idx_audit_events_event_dt ON audit_events(event, date_time DESC)",
		"CREATE INDEX IF NOT EXISTS idx_audit_events_severity_dt ON audit_events(severity, date_time DESC)",
		"CREATE INDEX IF NOT EXISTS idx_clients_name ON clients(name)",
		"CREATE INDEX IF NOT EXISTS idx_clients_sub_secret ON clients(sub_secret)",
		"CREATE INDEX IF NOT EXISTS idx_client_ips_client_legacy_ip ON client_ips(client_name, ip) WHERE ip IS NOT NULL AND ip != ''",
		"CREATE INDEX IF NOT EXISTS idx_client_ips_last_seen ON client_ips(last_seen)",
	}
	for _, query := range indexes {
		if err := db.Exec(query).Error; err != nil {
			return err
		}
	}
	return nil
}

func GetDB() *gorm.DB {
	dbMu.RLock()
	defer dbMu.RUnlock()
	return db
}

func IsNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
