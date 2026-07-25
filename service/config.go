package service

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/deposist/s-ui-x-extended/config"
	"github.com/deposist/s-ui-x-extended/core"
	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/deposist/s-ui-x-extended/ipmonitor"
	"github.com/deposist/s-ui-x-extended/logger"
	"github.com/deposist/s-ui-x-extended/realtime"
	"github.com/deposist/s-ui-x-extended/util/common"
	"github.com/deposist/s-ui-x-extended/util/redact"

	"gorm.io/gorm"
)

type ConfigService struct {
	ClientService
	TlsService
	SettingService
	InboundService
	OutboundService
	ServicesService
	EndpointService
	ProviderService
	Runtime *Runtime
}

type SingBoxConfig struct {
	Log          json.RawMessage   `json:"log"`
	Dns          json.RawMessage   `json:"dns"`
	Ntp          json.RawMessage   `json:"ntp"`
	Inbounds     []json.RawMessage `json:"inbounds"`
	Outbounds    []json.RawMessage `json:"outbounds"`
	Services     []json.RawMessage `json:"services"`
	Endpoints    []json.RawMessage `json:"endpoints"`
	Providers    []json.RawMessage `json:"providers"`
	Route        json.RawMessage   `json:"route"`
	Experimental json.RawMessage   `json:"experimental"`
}

var restartInboundsAfterSave = func(s *ConfigService, inboundIds []uint) error {
	return s.InboundService.RestartInbounds(database.GetDB(), inboundIds)
}

var restartServicesAfterSave = func(s *ConfigService, serviceIds []uint) error {
	return s.ServicesService.RestartServices(database.GetDB(), serviceIds)
}

var removeServicesFromCoreAfterSave = func(s *ConfigService, tags []string) error {
	return s.ServicesService.RemoveServicesFromCore(tags)
}

var removeInboundsFromCoreAfterSave = func(s *ConfigService, tags []string) error {
	return s.InboundService.RemoveInboundsFromCore(tags)
}

var restartOutboundsAfterSave = func(s *ConfigService, outboundIds []uint) error {
	return s.OutboundService.RestartOutbounds(database.GetDB(), outboundIds)
}

var removeOutboundsFromCoreAfterSave = func(s *ConfigService, tags []string) error {
	return s.OutboundService.RemoveOutboundsFromCore(tags)
}

var restartEndpointsAfterSave = func(s *ConfigService, endpointIds []uint) error {
	return s.EndpointService.RestartEndpoints(database.GetDB(), endpointIds)
}

var removeEndpointsFromCoreAfterSave = func(s *ConfigService, tags []string) error {
	return s.EndpointService.RemoveEndpointsFromCore(tags)
}

var invalidateClientPolicyCacheAfterSave = ipmonitor.InvalidateAllCache
var invalidateSubscriptionCacheAfterSave func()
var invalidateClientSubscriptionCacheAfterSave func([]string)

func RegisterSubscriptionCacheInvalidator(fn func()) {
	invalidateSubscriptionCacheAfterSave = fn
}

func RegisterClientSubscriptionCacheInvalidator(fn func([]string)) {
	invalidateClientSubscriptionCacheAfterSave = fn
}

func NewConfigService(core *core.Core) *ConfigService {
	runtime := NewRuntime(core)
	SetDefaultRuntime(runtime)
	return NewConfigServiceWithRuntime(runtime)
}

func NewConfigServiceWithRuntime(runtime *Runtime) *ConfigService {
	runtime = runtimeOrDefault(runtime)
	return &ConfigService{
		ClientService:   ClientService{Runtime: runtime},
		TlsService:      TlsService{Runtime: runtime, InboundService: InboundService{Runtime: runtime, ClientService: ClientService{Runtime: runtime}}, ServicesService: ServicesService{Runtime: runtime}},
		SettingService:  SettingService{},
		InboundService:  InboundService{Runtime: runtime, ClientService: ClientService{Runtime: runtime}},
		OutboundService: OutboundService{Runtime: runtime},
		ServicesService: ServicesService{Runtime: runtime},
		EndpointService: EndpointService{Runtime: runtime},
		ProviderService: ProviderService{},
		Runtime:         runtime,
	}
}

func (s *ConfigService) GetConfig(data string) (*[]byte, error) {
	var err error
	if len(data) == 0 {
		data, err = s.SettingService.GetConfig()
		if err != nil {
			return nil, err
		}
	}
	var singboxConfig map[string]json.RawMessage
	err = json.Unmarshal([]byte(data), &singboxConfig)
	if err != nil {
		return nil, err
	}

	inbounds, err := s.InboundService.GetAllConfig(database.GetDB())
	if err != nil {
		return nil, err
	}
	singboxConfig["inbounds"], err = json.Marshal(inbounds)
	if err != nil {
		return nil, err
	}
	outbounds, err := s.OutboundService.GetAllConfig(database.GetDB())
	if err != nil {
		return nil, err
	}
	singboxConfig["outbounds"], err = json.Marshal(outbounds)
	if err != nil {
		return nil, err
	}
	services, err := s.ServicesService.GetAllConfig(database.GetDB())
	if err != nil {
		return nil, err
	}
	singboxConfig["services"], err = json.Marshal(services)
	if err != nil {
		return nil, err
	}
	endpoints, err := s.EndpointService.GetAllConfig(database.GetDB())
	if err != nil {
		return nil, err
	}
	if awgSettings, settingsErr := s.SettingService.GetAWGSettings(); settingsErr == nil {
		var injectErr error
		endpoints, injectErr = injectAWGManagedEndpointPeers(database.GetDB(), awgSettings, endpoints)
		if injectErr != nil {
			logger.Warning("managed AWG peers omitted from core config: ", injectErr)
		}
	}
	if injected, injectErr := InjectManagedAWGEndpointPeers(database.GetDB(), endpoints); injectErr != nil {
		logger.Warning("endpoint-managed AWG peers omitted from core config: ", injectErr)
	} else {
		endpoints = injected
	}
	singboxConfig["endpoints"], err = json.Marshal(endpoints)
	if err != nil {
		return nil, err
	}
	providers, err := s.ProviderService.GetAllConfig(database.GetDB())
	if err != nil {
		return nil, err
	}
	singboxConfig["providers"], err = json.Marshal(providers)
	if err != nil {
		return nil, err
	}
	rawConfig, err := json.MarshalIndent(singboxConfig, "", "  ")
	if err != nil {
		return nil, err
	}
	return &rawConfig, nil
}

// startCore starts sing-box. When force is true, the cool-down between failed
// starts is bypassed, which is required for user-initiated restarts so the API
// reflects the real start status instead of silently succeeding.
func (s *ConfigService) startCore(force bool) error {
	manager := s.runtime().restart()
	if manager == nil {
		return common.NewError("restart manager not initialized")
	}
	return manager.run(func() error {
		return s.startCoreLocked(force)
	})
}

func (s *ConfigService) startCoreLocked(force bool) error {
	coreInstance := s.coreInstance()
	if coreInstance == nil {
		return common.NewError("core not initialized")
	}
	if coreInstance.IsRunning() {
		return nil
	}
	runtime := s.runtime()
	if !force && runtime.startCooldownActive() {
		logger.Info("start core cooldown ", runtime.coreStartCooldownDuration()/time.Second, " seconds")
		return nil
	}

	logger.Info("starting core")
	rawConfig, err := s.GetConfig("")
	if err != nil {
		return err
	}
	err = coreInstance.Start(*rawConfig)
	if err != nil {
		runtime.markCoreStartFailed()
		logger.Error("start sing-box err:", err.Error())
		return err
	}
	runtime.markCoreStartSucceeded()
	logger.Info("sing-box started")
	if err := runtime.TriggerAWGReconcile(context.Background()); err != nil {
		logger.Warning("AWG reconcile after core start failed: ", err)
	}
	return nil
}

// StartCore is the cron-friendly variant: it respects the cooldown so a
// failing core does not get hammered every 5 seconds.
func (s *ConfigService) StartCore() error {
	return s.startCore(false)
}

// RestartCore is invoked from user actions; it bypasses the cooldown so the
// caller observes the true start status. It waits for any in-flight core
// operation instead of being silently skipped.
func (s *ConfigService) RestartCore() error {
	manager := s.runtime().restart()
	if manager == nil {
		return common.NewError("restart manager not initialized")
	}
	return manager.runBlocking(s.restartCoreLocked)
}

// restartCoreLocked must only run inside a restart-manager section: it calls
// the lock-free primitives directly and never re-enters the manager.
func (s *ConfigService) restartCoreLocked() error {
	if err := s.stopCoreLocked(); err != nil {
		return err
	}
	return s.startCoreLocked(true)
}

func (s *ConfigService) StopCore() error {
	manager := s.runtime().restart()
	if manager == nil {
		return common.NewError("restart manager not initialized")
	}
	return manager.runBlocking(s.stopCoreLocked)
}

func (s *ConfigService) stopCoreLocked() error {
	coreInstance := s.coreInstance()
	if coreInstance == nil {
		return common.NewError("core not initialized")
	}
	err := coreInstance.Stop()
	if err != nil {
		return err
	}
	logger.Info("sing-box stopped")
	return nil
}

func (s *ConfigService) IsCoreRunning() bool {
	coreInstance := s.coreInstance()
	return coreInstance != nil && coreInstance.IsRunning()
}

func (s *ConfigService) CheckOutbound(tag string, link string) core.CheckOutboundResult {
	if tag == "" {
		return core.CheckOutboundResult{Error: "missing query parameter: tag"}
	}
	coreInstance := s.coreInstance()
	if coreInstance == nil || !coreInstance.IsRunning() {
		result := core.CheckOutboundResult{Error: "core not running"}
		SetOutboundHealth(tag, false, 0, result.Error)
		return result
	}
	result := coreInstance.CheckOutbound(coreInstance.GetCtx(), tag, link)
	SetOutboundHealth(tag, result.OK, result.Delay, result.Error)
	return result
}

func (s *ConfigService) CheckOutboundWithContext(ctx context.Context, tag string, link string) core.CheckOutboundResult {
	if tag == "" {
		return core.CheckOutboundResult{Error: "missing query parameter: tag"}
	}
	coreInstance := s.coreInstance()
	if coreInstance == nil || !coreInstance.IsRunning() {
		result := core.CheckOutboundResult{Error: "core not running"}
		SetOutboundHealth(tag, false, 0, result.Error)
		return result
	}
	result := coreInstance.CheckOutbound(ctx, tag, link)
	SetOutboundHealth(tag, result.OK, result.Delay, result.Error)
	return result
}

func (s *ConfigService) Save(obj string, act string, data json.RawMessage, initUsers string, loginUser string, hostname string) (objs []string, err error) {
	var plan postCommitCorePlan
	invalidateClientPolicyCache := false
	clientSubscriptionIDs := clientSubscriptionCacheIDs(obj, data)
	auditTelegramBackupPassphrase, auditTelegramBackupPassphraseConfigured, err := s.telegramBackupPassphraseAuditState(obj, data)
	if err != nil {
		return nil, err
	}

	db := database.GetDB()
	tx := db.Begin()
	defer func() {
		// A panic inside dispatchSave (e.g. a malformed entity blob) leaves the
		// named return err == nil, which would otherwise take the commit branch
		// and persist a half-applied transaction. Roll back first, then re-raise
		// so gin.Recovery still surfaces a 500 but the DB stays consistent.
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
		if err == nil {
			if commitErr := tx.Commit().Error; commitErr != nil {
				err = commitErr
				return
			}
			if auditTelegramBackupPassphrase {
				s.SettingService.recordTelegramBackupPassphraseChanged(loginUser, auditTelegramBackupPassphraseConfigured)
			}
			if invalidateClientPolicyCache {
				invalidateClientPolicyCacheAfterSave()
			}
			if obj == "clients" && invalidateClientSubscriptionCacheAfterSave != nil {
				invalidateClientSubscriptionCacheAfterSave(clientSubscriptionIDs)
			} else if invalidateSubscriptionCacheAfterSave != nil {
				invalidateSubscriptionCacheAfterSave()
			}
			// Advance the change marker only after the tx actually committed,
			// so a failed commit cannot make CheckChanges report phantom changes.
			s.setLastUpdate(time.Now().Unix())
			realtime.Publish(realtime.TopicConfigInvalidated, nil)
			s.applyPostCommitCoreChanges(plan)
		} else {
			tx.Rollback()
		}
	}()

	objs, plan, invalidateClientPolicyCache, err = s.dispatchSave(tx, obj, act, data, initUsers, hostname)
	if err != nil {
		return nil, err
	}

	dt := time.Now().Unix()
	err = tx.Create(&model.Changes{
		DateTime: dt,
		Actor:    loginUser,
		Key:      obj,
		Action:   act,
		Obj:      redactChangePayload(data),
	}).Error
	if err != nil {
		return nil, err
	}

	return objs, nil
}

func clientSubscriptionCacheIDs(obj string, data json.RawMessage) []string {
	if obj != "clients" {
		return nil
	}
	var clientIDs []uint
	var one model.Client
	if json.Unmarshal(data, &one) == nil {
		if one.SubSecret != "" {
			return []string{one.SubSecret}
		}
		if one.Id > 0 {
			clientIDs = append(clientIDs, one.Id)
		}
	} else {
		var many []model.Client
		if json.Unmarshal(data, &many) != nil {
			return nil
		}
		ids := make([]string, 0, len(many))
		for _, client := range many {
			if client.SubSecret != "" {
				ids = append(ids, client.SubSecret)
			} else if client.Id > 0 {
				clientIDs = append(clientIDs, client.Id)
			}
		}
		if len(clientIDs) == 0 {
			return ids
		}
	}
	var secrets []string
	if err := database.GetDB().Model(model.Client{}).Where("id in ?", clientIDs).Pluck("sub_secret", &secrets).Error; err != nil {
		return nil
	}
	return secrets
}

// dispatchSave routes the save to the owning entity service and translates its
// outcome into the post-commit core plan. The bool result reports whether the
// client policy cache must be invalidated after commit.
func (s *ConfigService) dispatchSave(tx *gorm.DB, obj string, act string, data json.RawMessage, initUsers string, hostname string) ([]string, postCommitCorePlan, bool, error) {
	objs := []string{obj}
	var plan postCommitCorePlan
	if err := validateEntityIdentity(obj, act, data); err != nil {
		return nil, plan, false, err
	}
	switch obj {
	case "clients":
		inboundIds, err := s.ClientService.Save(tx, act, data, hostname)
		if err != nil {
			return nil, plan, false, err
		}
		if len(inboundIds) > 0 {
			objs = append(objs, "inbounds")
			plan.inboundIds = inboundIds
		}
		return objs, plan, true, nil
	case "tls":
		inboundIds, serviceIds, err := s.TlsService.Save(tx, act, data, hostname)
		if err != nil {
			return nil, plan, false, err
		}
		objs = append(objs, "clients", "inbounds")
		plan.inboundIds = inboundIds
		plan.serviceIds = serviceIds
		return objs, plan, false, nil
	case "inbounds":
		change, err := s.InboundService.Save(tx, act, data, initUsers, hostname)
		if err != nil {
			return nil, plan, false, err
		}
		objs = append(objs, "clients")
		plan.mergeInboundChange(change)
		return objs, plan, false, nil
	case "outbounds":
		change, err := s.OutboundService.Save(tx, act, data)
		if err != nil {
			return nil, plan, false, err
		}
		plan.mergeOutboundChange(change)
		return objs, plan, false, nil
	case "services":
		change, err := s.ServicesService.Save(tx, act, data)
		if err != nil {
			return nil, plan, false, err
		}
		plan.mergeServiceChange(change)
		return objs, plan, false, nil
	case "endpoints":
		change, err := s.EndpointService.Save(tx, act, data)
		if err != nil {
			return nil, plan, false, err
		}
		plan.mergeEndpointChange(change)
		return objs, plan, false, nil
	case "providers":
		err := s.ProviderService.Save(tx, act, data)
		if err != nil {
			return nil, plan, false, err
		}
		return objs, plan, false, nil
	case "config":
		if err := validateConfigLogOutput(data); err != nil {
			return nil, plan, false, err
		}
		changed, err := s.SettingService.ConfigBlobChanged(tx, data)
		if err != nil {
			return nil, plan, false, err
		}
		if err := s.SettingService.SaveConfig(tx, data); err != nil {
			return nil, plan, false, err
		}
		// A byte-identical re-save keeps the audit trail but must not drop
		// every active connection through a pointless core restart.
		plan.needsCoreRestart = changed
		return objs, plan, false, nil
	case "settings":
		err := s.SettingService.Save(tx, data)
		if err != nil {
			return nil, plan, false, err
		}
		return objs, plan, false, nil
	default:
		return nil, plan, false, common.NewError("unknown object: ", obj)
	}
}

// entityIdentityField reports the JSON field used to reference an entity from
// the assembled sing-box config. Client names are descriptive, not references.
func entityIdentityField(obj string) (string, bool) {
	switch obj {
	case "inbounds", "outbounds", "services", "endpoints", "providers":
		return "tag", true
	case "tls":
		return "name", true
	default:
		return "", false
	}
}

// validateEntityIdentity rejects blank identities before an entity save can
// persist a row that other config objects cannot reference reliably.
func validateEntityIdentity(obj string, act string, data json.RawMessage) error {
	if act != "new" && act != "edit" {
		return nil
	}
	field, ok := entityIdentityField(obj)
	if !ok {
		return nil
	}

	var payload map[string]json.RawMessage
	if err := json.Unmarshal(data, &payload); err != nil {
		// The entity save path reports malformed body shapes with better context.
		return nil
	}
	raw, ok := payload[field]
	if !ok {
		return common.NewErrorf("%s: %s is required", obj, field)
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return common.NewErrorf("%s: %s must be a string", obj, field)
	}
	if strings.TrimSpace(value) == "" {
		return common.NewErrorf("%s: %s must not be empty", obj, field)
	}
	return nil
}

// validateConfigLogOutput rejects unsafe sing-box log.output paths before the
// config blob is persisted, so the core (often running as root) cannot be
// pointed at an arbitrary file on disk. See config.IsSafeLogOutputPath.
func validateConfigLogOutput(data json.RawMessage) error {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(data, &top); err != nil {
		// Not a JSON object: it cannot carry a log.output value, so there is
		// nothing to validate; the existing save/assembly path handles
		// malformed config.
		return nil
	}
	logRaw, ok := top["log"]
	if !ok {
		return nil
	}
	var logBlock struct {
		Output string `json:"output"`
	}
	if err := json.Unmarshal(logRaw, &logBlock); err != nil {
		return err
	}
	if !config.IsSafeLogOutputPath(logBlock.Output) {
		return common.NewError("log.output must be a relative path within the panel directory; absolute paths and '..' are not allowed")
	}
	return nil
}

func (s *ConfigService) coreInstance() *core.Core {
	if s == nil {
		return DefaultRuntime().Core()
	}
	return s.runtime().Core()
}

func (s *ConfigService) runtime() *Runtime {
	if s != nil {
		return runtimeOrDefault(s.Runtime)
	}
	return DefaultRuntime()
}

func (s *ConfigService) telegramBackupPassphraseAuditState(obj string, data json.RawMessage) (bool, bool, error) {
	if obj != "settings" {
		return false, false, nil
	}
	var settings map[string]string
	if err := json.Unmarshal(data, &settings); err != nil {
		return false, false, err
	}
	newPassphrase, ok := settings["telegramBackupPassphrase"]
	if !ok || newPassphrase == StoredSecretMarker {
		return false, false, nil
	}
	oldPassphrase, err := s.SettingService.GetTelegramBackupPassphraseBytes()
	if err != nil {
		return false, false, err
	}
	defer zeroBytes(oldPassphrase)
	if string(oldPassphrase) == newPassphrase {
		return false, false, nil
	}
	return true, newPassphrase != "", nil
}

func redactChangePayload(data json.RawMessage) json.RawMessage {
	var payload any
	if err := json.Unmarshal(data, &payload); err != nil {
		encoded, marshalErr := json.Marshal(redact.String(string(data)))
		if marshalErr != nil {
			return json.RawMessage(`"[REDACTED]"`)
		}
		return encoded
	}
	redactSudokuSplitKeys(payload)
	encoded, err := json.Marshal(redact.Value(payload))
	if err != nil {
		return json.RawMessage(`"[REDACTED]"`)
	}
	return encoded
}

func redactSudokuSplitKeys(value any) {
	switch typed := value.(type) {
	case map[string]any:
		if sudoku, ok := typed["sudoku"].(map[string]any); ok {
			if _, exists := sudoku["key"]; exists {
				sudoku["key"] = redact.Marker
			}
		}
		for _, child := range typed {
			redactSudokuSplitKeys(child)
		}
	case []any:
		for _, child := range typed {
			redactSudokuSplitKeys(child)
		}
	}
}

func (s *ConfigService) CheckChanges(lu string) (bool, error) {
	if lu == "" {
		return true, nil
	}
	lastUpdate := s.getLastUpdate()
	if lastUpdate == 0 {
		db := database.GetDB()
		var count int64
		intLu, err := strconv.ParseInt(lu, 10, 64)
		if err != nil {
			return false, err
		}
		err = db.Model(model.Changes{}).Where("date_time > ?", intLu).Count(&count).Error
		if err == nil {
			s.setLastUpdate(time.Now().Unix())
		}
		return count > 0, err
	}
	intLu, err := strconv.ParseInt(lu, 10, 64)
	return lastUpdate > intLu, err
}

func (s *ConfigService) GetChanges(actor string, chngKey string, count string) []model.Changes {
	c, _ := strconv.Atoi(count)
	if c <= 0 || c > 200 {
		c = 20
	}
	db := database.GetDB().Model(model.Changes{})
	if len(actor) > 0 {
		db = db.Where("actor = ?", actor)
	}
	if len(chngKey) > 0 {
		db = db.Where("key = ?", chngKey)
	}
	var chngs []model.Changes
	err := db.Order("id desc").Limit(c).Scan(&chngs).Error
	if err != nil {
		logger.Warning(err)
	}
	return chngs
}

func (s *ConfigService) setLastUpdate(value int64) {
	s.runtime().updates().Set(value)
}

func (s *ConfigService) getLastUpdate() int64 {
	return s.runtime().updates().Get()
}
