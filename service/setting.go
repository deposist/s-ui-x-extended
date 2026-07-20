package service

import (
	"encoding/json"
	"net"
	"net/netip"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/deposist/s-ui-x-extended/config"
	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/deposist/s-ui-x-extended/logger"
	"github.com/deposist/s-ui-x-extended/realtime"
	"github.com/deposist/s-ui-x-extended/util"
	"github.com/deposist/s-ui-x-extended/util/common"
	"github.com/deposist/s-ui-x-extended/util/ssrf"

	"gorm.io/gorm"
)

var defaultConfig = `{
  "log": {
    "level": "info"
  },
  "dns": {
    "servers": [],
    "rules": []
  },
  "route": {
    "rules": [
		  {
        "action": "sniff"
      },
      {
        "protocol": [
          "dns"
        ],
        "action": "hijack-dns"
      }
    ]
  },
  "experimental": {}
}`

var defaultValueMap = map[string]string{
	"webListen":                   "",
	"webDomain":                   "",
	"webPort":                     "2095",
	"secret":                      common.Random(32),
	"installSalt":                 common.Random(32),
	"webCertFile":                 "",
	"webKeyFile":                  "",
	"webPath":                     "/app/",
	"webURI":                      "",
	"sessionMaxAge":               "0",
	"forceCookieSecure":           "false",
	"sessionSameSiteStrict":       "false",
	"sessionGeneration":           "",
	"trafficAge":                  "30",
	"timeLocation":                "Europe/Moscow",
	"subListen":                   "",
	"subPort":                     "2096",
	"subPath":                     "/sub/",
	"subDomain":                   "",
	"subCertFile":                 "",
	"subKeyFile":                  "",
	"subUpdates":                  "12",
	"subEncode":                   "true",
	"subShowInfo":                 "false",
	"subSecretRequired":           "true",
	"subRateLimitPerIP":           "60",
	"subLinkEnable":               "true",
	"subJsonEnable":               "true",
	"subClashEnable":              "true",
	"subJsonPath":                 "/json/",
	"subClashPath":                "/clash/",
	"subJsonURI":                  "",
	"subClashURI":                 "",
	"subTitle":                    "",
	"subSupportUrl":               "",
	"subProfileUrl":               "",
	"subAnnounce":                 "",
	"subNameInRemark":             "false",
	"subJsonFragment":             "",
	"subJsonNoises":               "",
	"subJsonMux":                  "false",
	"subJsonDirectRules":          "false",
	"subURI":                      "",
	"subJsonExt":                  "",
	"subClashExt":                 "",
	"auditRetentionDays":          "30",
	"ipShowRaw":                   "false",
	"ipHistoryRetentionDays":      "30",
	"observabilityMemoryCapMB":    "32",
	"updateChannel":               "main",
	"telegramEnabled":             "false",
	"telegramBotToken":            "",
	"telegramChatID":              "",
	"telegramProxyURL":            "",
	"telegramProxyUsername":       "",
	"telegramProxyPassword":       "",
	"telegramTransportMode":       "proxy",
	"telegramOutboundTag":         "",
	"telegramCpuThreshold":        "90",
	"telegramNotifyCpu":           "false",
	"telegramReport":              "false",
	"telegramReportCron":          "",
	"telegramBackupEnabled":       "false",
	"telegramBackupPassphrase":    "",
	"telegramBackupCron":          "",
	"telegramBackupExcludeTables": "stats,client_ips,audit_events,changes",
	"telegramBackupMaxSizeMB":     "45",
	// Paid Subscriptions (experimental client bot + payments module). See paidsub package.
	"paidSubEnabled":              "false",
	"paidSubBotToken":             "",
	"paidSubBotPollSeconds":       "25",
	"paidSubUpdateOffset":         "0",
	"paidSubTransportMode":        "proxy",
	"paidSubProxyURL":             "",
	"paidSubProxyUsername":        "",
	"paidSubProxyPassword":        "",
	"paidSubOutboundTag":          "",
	"paidSubAutoRegister":         "false",
	"paidSubAutoInbounds":         "[]",
	"paidSubTrialDays":            "3",
	"paidSubTrialVolumeGB":        "0",
	"paidSubMaxClients":           "5000",
	"paidSubStartRateLimitPerMin": "3",
	"paidSubCurrency":             "RUB",
	"paidSubStarsEnabled":         "false",
	"paidSubYooKassaEnabled":      "false",
	"paidSubYooKassaToken":        "",
	"paidSubStripeEnabled":        "false",
	"paidSubStripeToken":          "",
	"paidSubPayMasterEnabled":     "false",
	"paidSubPayMasterToken":       "",
	"paidSubCryptoBotEnabled":     "false",
	"paidSubCryptoBotToken":       "",
	"paidSubExternalEnabled":      "false",
	"paidSubExternalUrlTemplate":  "",
	"paidSubOrderTTLMinutes":      "30",
	"paidSubGreeting":             "",
	"paidSubRefundRevoke":         "true",
	// Managed AmneziaWG devices.
	"awgEnabled":              "false",
	"awgEndpointTag":          "",
	"awgPublicEndpoint":       "",
	"awgSubnet":               "10.77.0.0/16",
	"awgDNS":                  "1.1.1.1,1.0.0.1",
	"awgDefaultDeviceLimit":   "3",
	"awgReconcileIntervalSec": "30",
	"awgStatsIntervalSec":     "60",
	"awgMTU":                  "0",
	// IP TLS certificate (Let's Encrypt shortlived profile, RFC 8738) issued
	// in-process via go-acme/lego. User-editable controls + machine-managed
	// state (account key, issued paths, expiry). See IpCertificateService.
	"ipCertEnabled":             "false",
	"ipCertTargetIP":            "",
	"ipCertEmail":               "",
	"ipCertChallengePort":       "80",
	"ipCertApplyTarget":         "panel",
	"ipCertAccountKey":          "",
	"ipCertAccountRegistration": "",
	"ipCertLastIP":              "",
	"ipCertCertPath":            "",
	"ipCertKeyPath":             "",
	"ipCertNotAfter":            "",
	"ipCertLastIssue":           "",
	"config":                    defaultConfig,
	"version":                   "",
}

// ipCertInternalSettingKeys are machine-managed: written only by
// IpCertificateService, never by the settings UI. They are stripped from
// GetAllSetting and rejected by isEditableSettingKey.
var ipCertInternalSettingKeys = []string{
	"ipCertAccountKey",
	"ipCertAccountRegistration",
	"ipCertLastIP",
	"ipCertCertPath",
	"ipCertKeyPath",
	"ipCertNotAfter",
	"ipCertLastIssue",
}

type SettingService struct {
}

type PanelLoadSettings struct {
	Config      string
	SubURI      string
	SubJsonURI  string
	SubClashURI string
	TrafficAge  int
}

func (s *SettingService) GetAllSetting() (*map[string]string, error) {
	db := database.GetDB()
	if err := s.ensureDefaultSettings(db); err != nil {
		return nil, err
	}

	settings := make([]*model.Setting, 0)
	err := db.Model(model.Setting{}).Find(&settings).Error
	if err != nil {
		return nil, err
	}
	allSetting := map[string]string{}

	for _, setting := range settings {
		if isEncryptedSettingKey(setting.Key) {
			writeSecretSettingMarker(allSetting, setting.Key, setting.Value)
			continue
		}
		allSetting[setting.Key] = setting.Value
	}

	// Due to security principles
	delete(allSetting, "secret")
	delete(allSetting, "installSalt")
	delete(allSetting, "sessionGeneration")
	delete(allSetting, "config")
	delete(allSetting, "version")
	delete(allSetting, "paidSubUpdateOffset") // internal bot cursor, not user-facing
	for _, key := range ipCertInternalSettingKeys {
		delete(allSetting, key)             // machine-managed IP cert state, not user-facing
		delete(allSetting, key+"HasSecret") // and its encrypted-marker, if any
	}

	return &allSetting, nil
}

func (s *SettingService) ensureDefaultSettings(db *gorm.DB) error {
	keys := defaultSettingKeys()
	// Fast path: once every default key is present, skip the seed transaction
	// entirely so GetAllSetting stays a pure read on the steady-state hot path
	// (e.g. GET /load). Previously this opened a ~90-statement write
	// transaction (INSERT ... WHERE NOT EXISTS per default key) on every call,
	// which serializes on SQLite's single writer and dominated /load CPU under
	// concurrency. The seed below is unchanged and still idempotent, so the
	// concurrent first-init path (issue #19) keeps its exactly-once semantics.
	var present int64
	if err := db.Model(model.Setting{}).Where("key IN ?", keys).Count(&present).Error; err != nil {
		return err
	}
	if int(present) == len(keys) {
		return nil
	}
	return db.Transaction(func(tx *gorm.DB) error {
		for _, key := range keys {
			value, _ := defaultSettingValue(key)
			if err := insertSettingIfMissing(tx, key, value); err != nil {
				return err
			}
		}
		return nil
	})
}

func defaultSettingKeys() []string {
	keys := make([]string, 0, len(defaultValueMap))
	for key := range defaultValueMap {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func insertSettingIfMissing(tx *gorm.DB, key string, value string) error {
	return tx.Exec(
		`INSERT INTO settings ("key", value)
		 SELECT ?, ?
		 WHERE NOT EXISTS (SELECT 1 FROM settings WHERE "key" = ?)`,
		key, value, key,
	).Error
}

func (s *SettingService) ResetSettings() error {
	db := database.GetDB()
	return db.Where("1 = 1").Delete(model.Setting{}).Error
}

// GetUpdateChannel returns the persisted self-update channel, defaulting to
// "main" when unset or invalid. Channel constants and validation live in the
// config package (config.UpdateChannelMain / config.NormalizeUpdateChannel).
func (s *SettingService) GetUpdateChannel() string {
	value, err := s.getString("updateChannel")
	if err != nil {
		return config.UpdateChannelMain
	}
	return config.NormalizeUpdateChannel(value)
}

// SetUpdateChannel persists the self-update channel after validating it.
func (s *SettingService) SetUpdateChannel(channel string) error {
	return s.setString("updateChannel", config.NormalizeUpdateChannel(channel))
}

func (s *SettingService) getSetting(key string) (*model.Setting, error) {
	db := database.GetDB()
	if db == nil {
		return nil, common.NewError("database is not initialized")
	}
	setting := &model.Setting{}
	err := db.Model(model.Setting{}).Where("key = ?", key).First(setting).Error
	if err != nil {
		return nil, err
	}
	return setting, nil
}

func (s *SettingService) getString(key string) (string, error) {
	setting, err := s.getSetting(key)
	if database.IsNotFound(err) {
		value, ok := defaultSettingValue(key)
		if !ok {
			return "", common.NewErrorf("key <%v> not in defaultValueMap", key)
		}
		return value, nil
	} else if err != nil {
		return "", err
	}
	if isEncryptedSettingKey(key) {
		return s.decryptSettingValue(key, setting.Value)
	}
	return setting.Value, nil
}

func defaultSettingValue(key string) (string, bool) {
	value, ok := defaultValueMap[key]
	if !ok {
		return "", false
	}
	if key == "version" {
		return config.GetVersion(), true
	}
	return value, true
}

func (s *SettingService) saveSetting(key string, value string) error {
	setting, err := s.getSetting(key)
	db := database.GetDB()
	if database.IsNotFound(err) {
		return db.Create(&model.Setting{
			Key:   key,
			Value: value,
		}).Error
	} else if err != nil {
		return err
	}
	setting.Key = key
	setting.Value = value
	return db.Save(setting).Error
}

func (s *SettingService) setString(key string, value string) error {
	return s.saveSetting(key, value)
}

func (s *SettingService) getBool(key string) (bool, error) {
	str, err := s.getString(key)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(str)
}

// func (s *SettingService) setBool(key string, value bool) error {
// 	return s.setString(key, strconv.FormatBool(value))
// }

func (s *SettingService) getInt(key string) (int, error) {
	str, err := s.getString(key)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(str)
}

func (s *SettingService) setInt(key string, value int) error {
	return s.setString(key, strconv.Itoa(value))
}
func (s *SettingService) GetListen() (string, error) {
	return s.getString("webListen")
}

func (s *SettingService) GetWebDomain() (string, error) {
	return s.getString("webDomain")
}

func (s *SettingService) GetWebURI() (string, error) {
	return s.getString("webURI")
}

// ClearWebDomainAndAddress clears the panel domain, listen address, web URI and
// the certificate paths used by the panel and subscriptions.
// It restores access by IP on all interfaces when a wrong domain or listen
// address or a broken certificate path locked the panel out. A panel restart is
// required for the change to take effect.
func (s *SettingService) ClearWebDomainAndAddress() error {
	keys := []string{
		"webDomain", "webListen", "webURI",
		"webCertFile", "webKeyFile", "subCertFile", "subKeyFile",
	}
	db := database.GetDB()
	if db == nil {
		return common.NewError("database is not initialized")
	}
	return db.Transaction(func(tx *gorm.DB) error {
		for _, key := range keys {
			setting := &model.Setting{}
			err := tx.Where("key = ?", key).First(setting).Error
			if database.IsNotFound(err) {
				if err := tx.Create(&model.Setting{Key: key, Value: ""}).Error; err != nil {
					return err
				}
				continue
			}
			if err != nil {
				return err
			}
			if err := tx.Model(setting).Update("value", "").Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *SettingService) GetPort() (int, error) {
	return s.getInt("webPort")
}

func (s *SettingService) SetPort(port int) error {
	return s.setInt("webPort", port)
}

func (s *SettingService) GetCertFile() (string, error) {
	return s.getString("webCertFile")
}

func (s *SettingService) GetKeyFile() (string, error) {
	return s.getString("webKeyFile")
}

func (s *SettingService) GetWebPath() (string, error) {
	webPath, err := s.getString("webPath")
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(webPath, "/") {
		webPath = "/" + webPath
	}
	if !strings.HasSuffix(webPath, "/") {
		webPath += "/"
	}
	return webPath, nil
}

func (s *SettingService) SetWebPath(webPath string) error {
	webPath, err := normalizeAndValidatePathSetting("webPath", webPath)
	if err != nil {
		return err
	}
	return s.setString("webPath", webPath)
}

func (s *SettingService) GetSecret() ([]byte, error) {
	setting, err := s.getSetting("secret")
	if database.IsNotFound(err) {
		secret := defaultValueMap["secret"]
		if saveErr := s.saveSetting("secret", secret); saveErr != nil {
			logger.Warning("save secret failed:", saveErr)
			return []byte(secret), saveErr
		}
		return []byte(secret), nil
	}
	if err != nil {
		return nil, err
	}
	return []byte(setting.Value), nil
}

func (s *SettingService) GetInstallSalt() ([]byte, error) {
	setting, err := s.getSetting("installSalt")
	if database.IsNotFound(err) {
		salt := defaultValueMap["installSalt"]
		if saveErr := s.saveSetting("installSalt", salt); saveErr != nil {
			logger.Warning("save install salt failed:", saveErr)
			return []byte(salt), saveErr
		}
		return []byte(salt), nil
	}
	if err != nil {
		return nil, err
	}
	return []byte(setting.Value), nil
}

func (s *SettingService) GetSessionMaxAge() (int, error) {
	return s.getInt("sessionMaxAge")
}

func (s *SettingService) GetForceCookieSecure() (bool, error) {
	if enabled, ok, err := config.GetForceCookieSecureEnv(); ok {
		if err != nil {
			return false, common.NewError("invalid SUI_FORCE_COOKIE_SECURE")
		}
		return enabled, nil
	}
	return s.getBool("forceCookieSecure")
}

func (s *SettingService) GetSessionSameSiteStrict() (bool, error) {
	return s.getBool("sessionSameSiteStrict")
}

func (s *SettingService) GetSessionGeneration() (string, error) {
	return s.getString("sessionGeneration")
}

func (s *SettingService) RotateSessionGeneration() (string, error) {
	generation := common.Random(32)
	if err := s.setString("sessionGeneration", generation); err != nil {
		return "", err
	}
	realtime.CloseAll("session_rotated")
	invalidated := invalidateWSTokensForSessionRotation()
	if err := (&AuditService{}).Record(AuditEvent{
		Actor:    "system",
		Event:    "ws_tokens_invalidated",
		Resource: "realtime",
		Severity: AuditSeverityInfo,
		Details: map[string]any{
			"count": invalidated,
		},
	}); err != nil {
		logger.Warning("ws token invalidation audit failed:", err)
	}
	return generation, nil
}

func (s *SettingService) GetTrafficAge() (int, error) {
	return s.getInt("trafficAge")
}

func (s *SettingService) LoadPanelSettingsForData(host string) (PanelLoadSettings, error) {
	keys := []string{"config", "subURI", "subKeyFile", "subCertFile", "subDomain", "subPort", "subPath", "subJsonURI", "subClashURI", "trafficAge"}
	values, err := s.getSettingsSnapshot(keys...)
	if err != nil {
		return PanelLoadSettings{}, err
	}
	trafficAge, err := strconv.Atoi(values["trafficAge"])
	if err != nil {
		return PanelLoadSettings{}, err
	}
	return PanelLoadSettings{
		Config:      values["config"],
		SubURI:      finalSubURIFromSettings(host, values),
		SubJsonURI:  values["subJsonURI"],
		SubClashURI: values["subClashURI"],
		TrafficAge:  trafficAge,
	}, nil
}

func (s *SettingService) getSettingsSnapshot(keys ...string) (map[string]string, error) {
	db := database.GetDB()
	if db == nil {
		return nil, common.NewError("database is not initialized")
	}
	settings := make([]model.Setting, 0, len(keys))
	if err := db.Model(model.Setting{}).Where("key IN ?", keys).Find(&settings).Error; err != nil {
		return nil, err
	}
	values := make(map[string]string, len(keys))
	for _, key := range keys {
		value, ok := defaultSettingValue(key)
		if !ok {
			return nil, common.NewErrorf("key <%v> not in defaultValueMap", key)
		}
		values[key] = value
	}
	for _, setting := range settings {
		if isEncryptedSettingKey(setting.Key) {
			value, err := s.decryptSettingValue(setting.Key, setting.Value)
			if err != nil {
				return nil, err
			}
			values[setting.Key] = value
			continue
		}
		values[setting.Key] = setting.Value
	}
	return values, nil
}

func (s *SettingService) GetAuditRetentionDays() (int, error) {
	return s.getInt("auditRetentionDays")
}

func (s *SettingService) GetIPHistoryRetentionDays() (int, error) {
	return s.getInt("ipHistoryRetentionDays")
}

func (s *SettingService) GetIPShowRaw() (bool, error) {
	return s.getBool("ipShowRaw")
}

func (s *SettingService) GetObservabilityMemoryCapMB() (int, error) {
	return s.getInt("observabilityMemoryCapMB")
}

func (s *SettingService) GetTimeLocation() (*time.Location, error) {
	l, err := s.getString("timeLocation")
	if err != nil {
		return nil, err
	}
	location, err := time.LoadLocation(l)
	if err != nil {
		logger.Warningf("location <%v> not exist, using local location", l)
		return time.Local, nil
	}
	return location, nil
}

func (s *SettingService) GetSubListen() (string, error) {
	return s.getString("subListen")
}

func (s *SettingService) GetSubPort() (int, error) {
	return s.getInt("subPort")
}

func (s *SettingService) SetSubPort(subPort int) error {
	return s.setInt("subPort", subPort)
}

func (s *SettingService) GetSubPath() (string, error) {
	subPath, err := s.getString("subPath")
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(subPath, "/") {
		subPath = "/" + subPath
	}
	if !strings.HasSuffix(subPath, "/") {
		subPath += "/"
	}
	return subPath, nil
}

func (s *SettingService) SetSubPath(subPath string) error {
	subPath, err := normalizeAndValidatePathSetting("subPath", subPath)
	if err != nil {
		return err
	}
	return s.setString("subPath", subPath)
}

func (s *SettingService) GetSubDomain() (string, error) {
	return s.getString("subDomain")
}

func (s *SettingService) GetSubCertFile() (string, error) {
	return s.getString("subCertFile")
}

func (s *SettingService) GetSubKeyFile() (string, error) {
	return s.getString("subKeyFile")
}

func (s *SettingService) GetSubUpdates() (int, error) {
	return s.getInt("subUpdates")
}

func (s *SettingService) GetSubEncode() (bool, error) {
	return s.getBool("subEncode")
}

func (s *SettingService) GetSubShowInfo() (bool, error) {
	return s.getBool("subShowInfo")
}

func (s *SettingService) GetSubSecretRequired() (bool, error) {
	return s.getBool("subSecretRequired")
}

func (s *SettingService) GetSubRateLimitPerIP() (int, error) {
	return s.getInt("subRateLimitPerIP")
}

func (s *SettingService) GetSubLinkEnable() (bool, error) {
	return s.getBool("subLinkEnable")
}

func (s *SettingService) GetSubJsonEnable() (bool, error) {
	return s.getBool("subJsonEnable")
}

func (s *SettingService) GetSubClashEnable() (bool, error) {
	return s.getBool("subClashEnable")
}

func (s *SettingService) GetSubJsonPath() (string, error) {
	subJsonPath, err := s.getString("subJsonPath")
	if err != nil {
		return "", err
	}
	return normalizeURLPath(subJsonPath), nil
}

func (s *SettingService) GetSubClashPath() (string, error) {
	subClashPath, err := s.getString("subClashPath")
	if err != nil {
		return "", err
	}
	return normalizeURLPath(subClashPath), nil
}

func (s *SettingService) GetSubJsonURI() (string, error) {
	return s.getString("subJsonURI")
}

func (s *SettingService) GetSubClashURI() (string, error) {
	return s.getString("subClashURI")
}

func (s *SettingService) GetSubTitle() (string, error) {
	return s.getString("subTitle")
}

func (s *SettingService) GetSubSupportUrl() (string, error) {
	return s.getString("subSupportUrl")
}

func (s *SettingService) GetSubProfileUrl() (string, error) {
	return s.getString("subProfileUrl")
}

func (s *SettingService) GetSubAnnounce() (string, error) {
	return s.getString("subAnnounce")
}

func (s *SettingService) GetSubNameInRemark() (bool, error) {
	return s.getBool("subNameInRemark")
}

func (s *SettingService) GetSubJsonFragment() (string, error) {
	return s.getString("subJsonFragment")
}

func (s *SettingService) GetSubJsonNoises() (string, error) {
	return s.getString("subJsonNoises")
}

func (s *SettingService) GetSubJsonMux() (bool, error) {
	return s.getBool("subJsonMux")
}

func (s *SettingService) GetSubJsonDirectRules() (bool, error) {
	return s.getBool("subJsonDirectRules")
}

func (s *SettingService) GetSubURI() (string, error) {
	return s.getString("subURI")
}

func (s *SettingService) GetFinalSubURI(host string) (string, error) {
	allSetting, err := s.GetAllSetting()
	if err != nil {
		return "", err
	}
	return finalSubURIFromSettings(host, *allSetting), nil
}

func finalSubURIFromSettings(host string, settings map[string]string) string {
	SubURI := settings["subURI"]
	if SubURI != "" {
		return SubURI
	}
	protocol := "http"
	if settings["subKeyFile"] != "" && settings["subCertFile"] != "" {
		protocol = "https"
	}
	if settings["subDomain"] != "" {
		host = settings["subDomain"]
	}
	portValue := settings["subPort"]
	authority := hostForURL(host)
	if (portValue == "80" && protocol == "http") || (portValue == "443" && protocol == "https") {
		portValue = ""
	}
	if portValue != "" {
		authority = net.JoinHostPort(host, portValue)
	}
	return protocol + "://" + authority + settings["subPath"]
}

func hostForURL(host string) string {
	if strings.Contains(host, ":") && !strings.HasPrefix(host, "[") {
		return "[" + host + "]"
	}
	return host
}

func (s *SettingService) GetConfig() (string, error) {
	return s.getString("config")
}

func (s *SettingService) SetConfig(config string) error {
	return s.setString("config", config)
}

func (s *SettingService) SaveConfig(tx *gorm.DB, config json.RawMessage) error {
	configs, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	result := tx.Model(model.Setting{}).Where("key = ?", "config").Update("value", string(configs))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return tx.Create(&model.Setting{Key: "config", Value: string(configs)}).Error
	}
	return nil
}

// ConfigBlobChanged reports whether saving the given config would change the
// stored blob. It compares against the exact persisted representation
// (SaveConfig's MarshalIndent form), so a byte-identical re-save is detected
// reliably and any doubt counts as changed.
func (s *SettingService) ConfigBlobChanged(tx *gorm.DB, config json.RawMessage) (bool, error) {
	configs, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return false, err
	}
	var stored model.Setting
	result := tx.Model(model.Setting{}).Where("key = ?", "config").Limit(1).Find(&stored)
	if result.Error != nil {
		return false, result.Error
	}
	if result.RowsAffected == 0 {
		return true, nil
	}
	return stored.Value != string(configs), nil
}

func (s *SettingService) Save(tx *gorm.DB, data json.RawMessage) error {
	var err error
	var settings map[string]string
	err = json.Unmarshal(data, &settings)
	if err != nil {
		return err
	}
	if err = s.validateSaveKeys(settings); err != nil {
		return err
	}
	if err = s.validateAll(settings); err != nil {
		return err
	}
	for _, key := range sortedSettingKeys(settings) {
		obj := settings[key]
		if strings.HasSuffix(key, "HasSecret") {
			continue
		}
		if key == "paidSubCurrency" {
			obj = canonicalPaidSubCurrency(obj)
		}
		if isEncryptedSettingKey(key) {
			if obj == StoredSecretMarker {
				continue
			}
			if obj == "" {
				if key != "telegramBackupPassphrase" {
					continue
				}
			} else {
				obj, err = s.encryptSettingValue(key, obj)
				if err != nil {
					return err
				}
			}
		}
		// Correct Pathes start and ends with `/`
		if isPathSetting(key) {
			obj, err = normalizeAndValidatePathSetting(key, obj)
			if err != nil {
				return err
			}
		}

		result := tx.Model(model.Setting{}).Where("key = ?", key).Update("value", obj)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			if err = tx.Create(&model.Setting{Key: key, Value: obj}).Error; err != nil {
				return err
			}
		}
	}
	return err
}

func (s *SettingService) validateSaveKeys(settings map[string]string) error {
	for _, key := range sortedSettingKeys(settings) {
		if strings.HasSuffix(key, "HasSecret") {
			baseKey := strings.TrimSuffix(key, "HasSecret")
			if isEncryptedSettingKey(baseKey) {
				continue
			}
			return common.NewError("invalid setting key: ", key)
		}
		if !isEditableSettingKey(key) {
			return common.NewError("invalid setting key: ", key)
		}
	}
	return nil
}

func isEditableSettingKey(key string) bool {
	if _, ok := defaultValueMap[key]; !ok {
		return false
	}
	switch key {
	case "secret", "installSalt", "sessionGeneration", "config", "version", "paidSubUpdateOffset":
		return false
	}
	for _, internal := range ipCertInternalSettingKeys {
		if key == internal {
			return false
		}
	}
	return true
}

func (s *SettingService) validateAll(settings map[string]string) error {
	if err := s.validateSubscriptionPathSettings(settings); err != nil {
		return err
	}
	for _, key := range sortedSettingKeys(settings) {
		obj := settings[key]
		if strings.HasSuffix(key, "HasSecret") {
			continue
		}
		if key == "trafficAge" {
			if age, err := strconv.Atoi(strings.TrimSpace(obj)); err == nil && age == 0 {
				return common.NewError("invalid trafficAge setting: 0 is reserved")
			}
		}
		if (key == "telegramProxyURL" || key == "paidSubProxyURL") && obj != "" && obj != StoredSecretMarker {
			if err := validateTelegramProxyURL(obj); err != nil {
				return err
			}
		}
		if err := validateTelegramSettingInput(key, obj); err != nil {
			return err
		}
		if err := validateObservabilitySettingInput(key, obj); err != nil {
			return err
		}
		if err := validateSubscriptionSettingInput(key, obj); err != nil {
			return err
		}
		if err := validatePaidSubSettingInput(key, obj); err != nil {
			return err
		}
		if err := validateAWGSettingInput(key, obj); err != nil {
			return err
		}
		if err := validateIpCertSettingInput(key, obj); err != nil {
			return err
		}
		if key == "forceCookieSecure" || key == "sessionSameSiteStrict" {
			if _, err := strconv.ParseBool(obj); err != nil {
				return common.NewError("invalid boolean setting: ", key)
			}
		}
		if isDomainSetting(key) {
			if err := util.ValidateHostname(obj); err != nil {
				return common.NewErrorf("%s: %v", key, err)
			}
		}
		if obj != "" && isCertificatePathSetting(key) {
			if err := s.fileExists(obj); err != nil {
				return common.NewError(" -> ", obj, " is not exists")
			}
		}
		if isPathSetting(key) {
			if _, err := normalizeAndValidatePathSetting(key, obj); err != nil {
				return err
			}
		}
	}
	return nil
}

func sortedSettingKeys(settings map[string]string) []string {
	keys := make([]string, 0, len(settings))
	for key := range settings {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func isCertificatePathSetting(key string) bool {
	switch key {
	case "webCertFile", "webKeyFile", "subCertFile", "subKeyFile":
		return true
	default:
		return false
	}
}

func isPathSetting(key string) bool {
	switch key {
	case "webPath", "subPath", "subJsonPath", "subClashPath":
		return true
	default:
		return false
	}
}

func isDomainSetting(key string) bool {
	switch key {
	case "webDomain", "subDomain":
		return true
	default:
		return false
	}
}

func (s *SettingService) GetSubJsonExt() (string, error) {
	return s.getString("subJsonExt")
}

func (s *SettingService) GetSubClashExt() (string, error) {
	return s.getString("subClashExt")
}

func (s *SettingService) GetTelegramCpuThreshold() (int, error) {
	return s.getInt("telegramCpuThreshold")
}

func (s *SettingService) GetTelegramNotifyCpu() (bool, error) {
	return s.getBool("telegramNotifyCpu")
}

func (s *SettingService) GetTelegramEnabled() (bool, error) {
	return s.getBool("telegramEnabled")
}

func (s *SettingService) GetTelegramReport() (bool, error) {
	return s.getBool("telegramReport")
}

func (s *SettingService) GetTelegramReportCron() (string, error) {
	return s.getString("telegramReportCron")
}

func (s *SettingService) GetTelegramBackupEnabled() (bool, error) {
	return s.getBool("telegramBackupEnabled")
}

func (s *SettingService) GetTelegramBackupCron() (string, error) {
	return s.getString("telegramBackupCron")
}

func (s *SettingService) GetTelegramBackupExcludeTables() (string, error) {
	return s.getString("telegramBackupExcludeTables")
}

func (s *SettingService) GetTelegramBackupMaxSizeMB() (int, error) {
	return s.getInt("telegramBackupMaxSizeMB")
}

func (s *SettingService) GetTelegramBackupPassphraseBytes() ([]byte, error) {
	setting, err := s.getSetting("telegramBackupPassphrase")
	if database.IsNotFound(err) {
		value, _ := defaultSettingValue("telegramBackupPassphrase")
		return []byte(value), nil
	}
	if err != nil {
		return nil, err
	}
	return s.decryptSettingBytes("telegramBackupPassphrase", setting.Value)
}

func (s *SettingService) HasTelegramBackupPassphrase() (bool, error) {
	setting, err := s.getSetting("telegramBackupPassphrase")
	if database.IsNotFound(err) {
		value, _ := defaultSettingValue("telegramBackupPassphrase")
		return value != "", nil
	}
	if err != nil {
		return false, err
	}
	return setting.Value != "", nil
}

func (s *SettingService) recordTelegramBackupPassphraseChanged(actor string, configured bool) {
	if database.GetDB() == nil {
		return
	}
	if err := (&AuditService{}).Record(AuditEvent{
		Actor:    actor,
		Event:    "tg_backup_passphrase_changed",
		Resource: "database",
		Severity: AuditSeverityInfo,
		Details: map[string]any{
			"configured": configured,
		},
	}); err != nil {
		logger.Warning("telegram backup passphrase audit failed:", err)
	}
}

func (s *SettingService) fileExists(path string) error {
	_, err := os.Stat(path)
	return err
}

func normalizeAndValidatePathSetting(key string, path string) (string, error) {
	path = normalizeURLPath(path)
	if err := util.ValidatePath(path, reservedPathPrefixesForSetting(key)); err != nil {
		return "", err
	}
	return path, nil
}

func normalizeURLPath(path string) string {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}
	return path
}

func reservedPathPrefixesForSetting(key string) []string {
	ownPrefix := ""
	switch key {
	case "subPath":
		ownPrefix = "/sub/"
	case "subJsonPath":
		ownPrefix = "/json/"
	case "subClashPath":
		ownPrefix = "/clash/"
	}
	if ownPrefix == "" {
		return util.ReservedPathPrefixes
	}
	reserved := make([]string, 0, len(util.ReservedPathPrefixes))
	for _, prefix := range util.ReservedPathPrefixes {
		if prefix == ownPrefix {
			continue
		}
		reserved = append(reserved, prefix)
	}
	return reserved
}

func (s *SettingService) validateSubscriptionPathSettings(settings map[string]string) error {
	pathKeys := []string{"subPath", "subJsonPath", "subClashPath"}
	touched := false
	for _, key := range pathKeys {
		if _, ok := settings[key]; ok {
			touched = true
			break
		}
	}
	if !touched {
		return nil
	}

	paths := make(map[string]string, len(pathKeys))
	for _, key := range pathKeys {
		value, ok := settings[key]
		if !ok {
			var err error
			value, err = s.getString(key)
			if err != nil {
				return err
			}
		}
		normalized, err := normalizeAndValidatePathSetting(key, value)
		if err != nil {
			return err
		}
		paths[key] = normalized
	}

	if paths["subJsonPath"] == paths["subClashPath"] {
		return common.NewError("subscription format paths must be unique")
	}
	if paths["subJsonPath"] == "/" || paths["subClashPath"] == "/" {
		return common.NewError("subscription format path cannot be root")
	}
	if paths["subPath"] != "/" {
		if pathHasPrefix(paths["subJsonPath"], paths["subPath"]) || pathHasPrefix(paths["subClashPath"], paths["subPath"]) {
			return common.NewError("subscription format path conflicts with subscription path")
		}
	}
	return nil
}

func pathHasPrefix(path string, prefix string) bool {
	path = normalizeURLPath(path)
	prefix = normalizeURLPath(prefix)
	return path == prefix || strings.HasPrefix(path, prefix)
}

func validateSubscriptionSettingInput(key string, value string) error {
	switch key {
	case "subLinkEnable", "subJsonEnable", "subClashEnable", "subNameInRemark", "subJsonMux", "subJsonDirectRules":
		if _, err := strconv.ParseBool(value); err != nil {
			return common.NewError("invalid boolean setting: ", key)
		}
	case "subRateLimitPerIP":
		limit, err := strconv.Atoi(value)
		if err != nil || limit <= 0 || limit > 10000 {
			return common.NewError("invalid rate-limit setting: ", key)
		}
	case "subJsonURI", "subClashURI", "subSupportUrl", "subProfileUrl":
		if err := validateOptionalHTTPURL(value); err != nil {
			return err
		}
	case "subJsonFragment":
		if err := validateOptionalJSONObject(value, key); err != nil {
			return err
		}
	case "subJsonNoises":
		if err := validateOptionalJSONArray(value, key); err != nil {
			return err
		}
	}
	return nil
}

func validateOptionalJSONObject(value string, key string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(value), &obj); err != nil {
		return common.NewError("invalid JSON setting: ", key)
	}
	if obj == nil {
		return common.NewError("invalid JSON setting: ", key)
	}
	return nil
}

func validateOptionalJSONArray(value string, key string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	var arr []interface{}
	if err := json.Unmarshal([]byte(value), &arr); err != nil {
		return common.NewError("invalid JSON array setting: ", key)
	}
	if arr == nil {
		return common.NewError("invalid JSON array setting: ", key)
	}
	return nil
}

func validateOptionalHTTPURL(value string) error {
	if containsControlCharacter(value) {
		return common.NewError("invalid URL setting")
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" {
		return common.NewError("invalid URL setting")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return common.NewError("invalid URL setting")
	}
	if parsed.User != nil {
		return common.NewError("invalid URL setting")
	}
	if host := parsed.Hostname(); host != "" {
		if addr, err := netip.ParseAddr(host); err == nil && ssrf.IsBlockedAddr(addr) {
			return common.NewError("invalid URL setting")
		}
	}
	if strings.Contains(value, "#") || parsed.Fragment != "" || parsed.RawFragment != "" {
		return common.NewError("invalid URL setting")
	}
	if containsControlCharacter(parsed.Path) ||
		containsControlCharacter(parsed.RawPath) ||
		containsControlCharacter(parsed.RawQuery) ||
		containsQueryControlCharacter(parsed) {
		return common.NewError("invalid URL setting")
	}
	return nil
}

func containsControlCharacter(value string) bool {
	return strings.ContainsFunc(value, func(r rune) bool {
		return r < 0x20 || r == 0x7f
	})
}

func containsQueryControlCharacter(parsed *url.URL) bool {
	for _, part := range strings.Split(parsed.RawQuery, "&") {
		key, value, _ := strings.Cut(part, "=")
		if queryComponentHasControl(key) || queryComponentHasControl(value) {
			return true
		}
	}
	return false
}

func queryComponentHasControl(value string) bool {
	decoded, err := url.QueryUnescape(value)
	if err != nil {
		return false
	}
	return containsControlCharacter(decoded)
}

func validateTelegramSettingInput(key string, value string) error {
	switch key {
	case "telegramNotifyCpu", "telegramReport":
		if _, err := strconv.ParseBool(value); err != nil {
			return common.NewError("invalid boolean setting: ", key)
		}
	case "telegramBackupEnabled":
		if value != "true" && value != "false" {
			return common.NewError("invalid boolean setting: ", key)
		}
	case "telegramCpuThreshold":
		threshold, err := strconv.Atoi(value)
		if err != nil || threshold <= 0 || threshold > 100 {
			return common.NewError("invalid cpu threshold setting")
		}
	case "telegramReportCron":
		if _, err := ParseTelegramReportCron(value); err != nil {
			return err
		}
	case "telegramBackupPassphrase":
		if value != "" && value != StoredSecretMarker && len([]rune(value)) < 12 {
			return common.NewError("weak_passphrase")
		}
	case "telegramBackupCron":
		if _, err := ParseTelegramReportCron(value); err != nil {
			return err
		}
	case "telegramBackupExcludeTables":
		if len(value) > 256 {
			return common.NewError("telegramBackupExcludeTables is too long")
		}
	case "telegramBackupMaxSizeMB":
		limit, err := strconv.Atoi(value)
		if err != nil || limit < 1 || limit > 50 {
			return common.NewError("invalid telegram backup max size setting")
		}
	case "telegramTransportMode":
		if err := validateTransportMode(value); err != nil {
			return err
		}
	case "telegramOutboundTag":
		if len(value) > 256 {
			return common.NewError("telegramOutboundTag is too long")
		}
	}
	return nil
}

func validateTransportMode(value string) error {
	switch value {
	case "proxy", "outbound":
		return nil
	default:
		return common.NewError("transport mode must be 'proxy' or 'outbound'")
	}
}

func validateObservabilitySettingInput(key string, value string) error {
	switch key {
	case "observabilityMemoryCapMB":
		capMB, err := strconv.Atoi(value)
		if err != nil || capMB <= 0 || capMB > 1024 {
			return common.NewError("invalid observability memory cap setting")
		}
	}
	return nil
}

var allowedPaidSubCurrencies = map[string]bool{
	// Keep in sync with the frontend picker (PaidSubscriptions.vue: `currencies`).
	// XTR is Telegram Stars; the Stars provider pins the order currency to XTR
	// regardless of paidSubCurrency, so listing it here only keeps the picker
	// and the backend validator from disagreeing.
	"RUB": true,
	"USD": true,
	"EUR": true,
	"GBP": true,
	"UAH": true,
	"KZT": true,
	"BYN": true,
	"XTR": true,
}

func canonicalPaidSubCurrency(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func validatePaidSubAutoInbounds(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	var raw []int64
	if err := json.Unmarshal([]byte(value), &raw); err != nil {
		return common.NewError("paidSubAutoInbounds must be a JSON array of inbound ids")
	}
	seen := make(map[int64]bool, len(raw))
	ids := make([]uint, 0, len(raw))
	for _, id := range raw {
		if id <= 0 {
			return common.NewError("paidSubAutoInbounds must contain positive inbound ids")
		}
		if seen[id] {
			return common.NewError("paidSubAutoInbounds must not contain duplicate inbound ids")
		}
		seen[id] = true
		ids = append(ids, uint(id))
	}
	if len(ids) == 0 {
		return nil
	}
	var count int64
	if err := database.GetDB().Model(&model.Inbound{}).Where("id IN ?", ids).Count(&count).Error; err != nil {
		return err
	}
	if count != int64(len(ids)) {
		return common.NewError("paidSubAutoInbounds contains unknown inbound ids")
	}
	return nil
}

func validatePaidSubExternalURLTemplate(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	if len(value) > 2048 {
		return common.NewError("paidSubExternalUrlTemplate is too long")
	}
	if strings.Contains(value, "{") || strings.Contains(value, "}") {
		return common.NewError("paidSubExternalUrlTemplate must not contain placeholders in the host")
	}
	if err := validateOptionalHTTPURL(value); err != nil {
		return common.NewError("invalid paidSubExternalUrlTemplate")
	}
	return nil
}

func validatePaidSubSettingInput(key string, value string) error {
	switch key {
	case "paidSubEnabled", "paidSubAutoRegister", "paidSubStarsEnabled",
		"paidSubYooKassaEnabled", "paidSubStripeEnabled", "paidSubPayMasterEnabled",
		"paidSubCryptoBotEnabled", "paidSubExternalEnabled", "paidSubRefundRevoke":
		if _, err := strconv.ParseBool(value); err != nil {
			return common.NewError("invalid boolean setting: ", key)
		}
	case "paidSubBotPollSeconds":
		if err := validateIntRange(key, value, 1, 50); err != nil {
			return err
		}
	case "paidSubTrialDays":
		if err := validateIntRange(key, value, 0, 3650); err != nil {
			return err
		}
	case "paidSubTrialVolumeGB":
		if err := validateIntRange(key, value, 0, 1048576); err != nil {
			return err
		}
	case "paidSubMaxClients":
		if err := validateIntRange(key, value, 0, 10000000); err != nil {
			return err
		}
	case "paidSubStartRateLimitPerMin":
		if err := validateIntRange(key, value, 0, 1000); err != nil {
			return err
		}
	case "paidSubOrderTTLMinutes":
		if err := validateIntRange(key, value, 1, 1440); err != nil {
			return err
		}
	case "paidSubAutoInbounds":
		if err := validatePaidSubAutoInbounds(value); err != nil {
			return err
		}
	case "paidSubCurrency":
		if !allowedPaidSubCurrencies[canonicalPaidSubCurrency(value)] {
			return common.NewError("paidSubCurrency is not supported")
		}
	case "paidSubExternalUrlTemplate":
		if err := validatePaidSubExternalURLTemplate(value); err != nil {
			return err
		}
	case "paidSubTransportMode":
		if err := validateTransportMode(value); err != nil {
			return err
		}
	case "paidSubOutboundTag":
		if len(value) > 256 {
			return common.NewError("paidSubOutboundTag is too long")
		}
	case "paidSubGreeting":
		if len([]rune(value)) > 4096 {
			return common.NewError("paidSubGreeting is too long (max 4096)")
		}
	}
	return nil
}

func validateIpCertSettingInput(key string, value string) error {
	switch key {
	case "ipCertEnabled":
		if _, err := strconv.ParseBool(value); err != nil {
			return common.NewError("invalid boolean setting: ", key)
		}
	case "ipCertTargetIP":
		if value == "" {
			return nil
		}
		if err := validateIssuableIP(value); err != nil {
			return err
		}
	case "ipCertEmail":
		if err := validateIpCertEmail(value, false); err != nil {
			return err
		}
	case "ipCertChallengePort":
		if err := validateIntRange(key, value, 1, 65535); err != nil {
			return err
		}
	case "ipCertApplyTarget":
		if err := validateIpCertApplyTarget(value); err != nil {
			return err
		}
	}
	return nil
}

// validateIpCertApplyTarget accepts "panel" or "inbound:<numericTlsId>".
func validateIpCertApplyTarget(value string) error {
	if value == "" || value == "panel" {
		return nil
	}
	rest, ok := strings.CutPrefix(value, "inbound:")
	if !ok {
		return common.NewError("ipCertApplyTarget must be 'panel' or 'inbound:<id>'")
	}
	if id, err := strconv.Atoi(rest); err != nil || id <= 0 {
		return common.NewError("ipCertApplyTarget inbound id must be a positive integer")
	}
	return nil
}

func validateIntRange(key string, value string, min int, max int) error {
	n, err := strconv.Atoi(value)
	if err != nil || n < min || n > max {
		return common.NewErrorf("invalid setting %s: must be an integer in [%d, %d]", key, min, max)
	}
	return nil
}
