package service

import (
	"encoding/json"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/deposist/s-ui-x/config"
	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
	"gorm.io/gorm"
)

func TestDefaultSettingValueReadsCurrentVersion(t *testing.T) {
	if defaultValueMap["version"] != "" {
		t.Fatal("version default should be computed, not stored in defaultValueMap")
	}
	got, ok := defaultSettingValue("version")
	if !ok {
		t.Fatal("version key is missing")
	}
	if got != config.GetVersion() {
		t.Fatalf("default version = %q, want %q", got, config.GetVersion())
	}
}

func TestGetAllSettingConcurrentDefaultInitializationIssue19(t *testing.T) {
	initSettingTestDB(t)
	db := database.GetDB()
	if err := db.Where("1 = 1").Delete(&model.Setting{}).Error; err != nil {
		t.Fatal(err)
	}

	const workers = 16
	start := make(chan struct{})
	errCh := make(chan error, workers)
	resultCh := make(chan map[string]string, workers)
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			<-start
			settings, err := (&SettingService{}).GetAllSetting()
			if err != nil {
				errCh <- err
				return
			}
			resultCh <- *settings
		}()
	}

	close(start)
	wg.Wait()
	close(errCh)
	close(resultCh)

	for err := range errCh {
		if err != nil {
			t.Errorf("GetAllSetting returned error: %v", err)
		}
	}
	if t.Failed() {
		return
	}

	var rowCount int64
	if err := db.Model(&model.Setting{}).Count(&rowCount).Error; err != nil {
		t.Fatal(err)
	}
	if rowCount != int64(len(defaultValueMap)) {
		t.Fatalf("settings row count = %d, want %d", rowCount, len(defaultValueMap))
	}

	var duplicates []struct {
		Key   string
		Count int64 `gorm:"column:count"`
	}
	if err := db.Model(&model.Setting{}).
		Select("key, COUNT(*) AS count").
		Group("key").
		Having("COUNT(*) > 1").
		Scan(&duplicates).Error; err != nil {
		t.Fatal(err)
	}
	if len(duplicates) != 0 {
		t.Fatalf("duplicate default settings: %#v", duplicates)
	}

	var returned map[string]string
	for settings := range resultCh {
		if returned == nil {
			returned = settings
		}
	}
	if returned == nil {
		t.Fatal("no successful GetAllSetting result")
	}
	for _, key := range []string{"secret", "installSalt", "sessionGeneration", "config", "version"} {
		if _, ok := returned[key]; ok {
			t.Fatalf("returned settings should omit %q", key)
		}
	}
	if got, ok := returned["webPort"]; !ok || got != defaultValueMap["webPort"] {
		t.Fatalf("returned settings webPort = %q, present=%v", got, ok)
	}
}

func TestSaveConfigCreatesMissingConfigSetting(t *testing.T) {
	settingService := initSettingTestDB(t)
	tx := database.GetDB().Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}

	config := json.RawMessage(`{"dns":{"servers":[{"tag":"dns-umbrella"}]},"route":{"rules":[{"action":"sniff"}]}}`)
	if err := settingService.SaveConfig(tx, config); err != nil {
		tx.Rollback()
		t.Fatal(err)
	}
	if err := tx.Commit().Error; err != nil {
		t.Fatal(err)
	}

	var saved string
	if err := database.GetDB().Model(&model.Setting{}).Select("value").Where("key = ?", "config").Scan(&saved).Error; err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(saved, `"dns"`) || !strings.Contains(saved, `"route"`) {
		t.Fatalf("saved config does not contain DNS and route data: %s", saved)
	}
}

func TestGetFinalSubURIOmitsDefaultPorts(t *testing.T) {
	t.Setenv("SUI_DB_FOLDER", t.TempDir())
	if err := database.InitDB("file::memory:?cache=shared"); err != nil {
		if strings.Contains(err.Error(), "go-sqlite3 requires cgo") {
			t.Skip(err)
		}
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if d := database.GetDB(); d != nil {
			if sqlDB, err := d.DB(); err == nil {
				_ = sqlDB.Close()
			}
		}
	})

	settingService := &SettingService{}
	if _, err := settingService.GetAllSetting(); err != nil {
		t.Fatal(err)
	}
	db := database.GetDB()
	settings := map[string]string{
		"subPort":     "443",
		"subCertFile": "/tmp/cert.pem",
		"subKeyFile":  "/tmp/key.pem",
		"subPath":     "/sub/",
	}
	for key, value := range settings {
		if err := db.Model(model.Setting{}).Where("key = ?", key).Update("value", value).Error; err != nil {
			t.Fatal(err)
		}
	}
	uri, err := settingService.GetFinalSubURI("example.com")
	if err != nil {
		t.Fatal(err)
	}
	if uri != "https://example.com/sub/" {
		t.Fatalf("unexpected URI: %s", uri)
	}
}

func TestGetFinalSubURIFormatsIPv6Hosts(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		settings map[string]string
		want     string
	}{
		{
			name: "loopback with explicit port",
			host: "::1",
			settings: map[string]string{
				"subPort": "8443",
				"subPath": "/sub/",
			},
			want: "http://[::1]:8443/sub/",
		},
		{
			name: "documentation address with explicit port",
			host: "2001:db8::1",
			settings: map[string]string{
				"subPort": "8080",
				"subPath": "/sub/",
			},
			want: "http://[2001:db8::1]:8080/sub/",
		},
		{
			name: "default https port omitted",
			host: "::1",
			settings: map[string]string{
				"subPort":     "443",
				"subCertFile": "/tmp/cert.pem",
				"subKeyFile":  "/tmp/key.pem",
				"subPath":     "/sub/",
			},
			want: "https://[::1]/sub/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settingService := initSettingTestDB(t)
			if _, err := settingService.GetAllSetting(); err != nil {
				t.Fatal(err)
			}
			for key, value := range tt.settings {
				if err := database.GetDB().Model(model.Setting{}).Where("key = ?", key).Update("value", value).Error; err != nil {
					t.Fatal(err)
				}
			}
			uri, err := settingService.GetFinalSubURI(tt.host)
			if err != nil {
				t.Fatal(err)
			}
			if uri != tt.want {
				t.Fatalf("unexpected URI: %s, want %s", uri, tt.want)
			}
		})
	}
}

func TestSaveRejectsReservedWebPath(t *testing.T) {
	settingService := initSettingTestDB(t)
	if _, err := settingService.GetAllSetting(); err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(map[string]string{
		"webPath": "/api/",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Transaction(func(tx *gorm.DB) error {
		return settingService.Save(tx, payload)
	}); err == nil {
		t.Fatal("expected reserved webPath to be rejected")
	}
}

func TestSaveAllowsDefaultSubPathButRejectsOtherReservedSubPaths(t *testing.T) {
	settingService := initSettingTestDB(t)
	if _, err := settingService.GetAllSetting(); err != nil {
		t.Fatal(err)
	}
	validPayload, err := json.Marshal(map[string]string{
		"subPath": "/sub/",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Transaction(func(tx *gorm.DB) error {
		return settingService.Save(tx, validPayload)
	}); err != nil {
		t.Fatalf("default subPath should remain valid: %v", err)
	}

	invalidPayload, err := json.Marshal(map[string]string{
		"subPath": "/json/",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Transaction(func(tx *gorm.DB) error {
		return settingService.Save(tx, invalidPayload)
	}); err == nil {
		t.Fatal("expected reserved subPath to be rejected")
	}
}

func TestSubscriptionSettingsDefaultsAndValidation(t *testing.T) {
	settingService := initSettingTestDB(t)
	settings, err := settingService.GetAllSetting()
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{
		"subLinkEnable",
		"subJsonEnable",
		"subClashEnable",
		"subJsonPath",
		"subClashPath",
		"subJsonURI",
		"subClashURI",
		"subTitle",
		"subSupportUrl",
		"subProfileUrl",
		"subAnnounce",
		"subNameInRemark",
		"subJsonFragment",
		"subJsonNoises",
		"subJsonMux",
		"subJsonDirectRules",
		"subRateLimitPerIP",
	} {
		if _, ok := (*settings)[key]; !ok {
			t.Fatalf("missing default setting %s", key)
		}
	}

	validPayload, err := json.Marshal(map[string]string{
		"subJsonPath":       "/json/",
		"subClashPath":      "/clash/",
		"subSupportUrl":     "https://example.com/support",
		"subProfileUrl":     "https://example.com/profile",
		"subJsonEnable":     "false",
		"subRateLimitPerIP": "120",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Transaction(func(tx *gorm.DB) error {
		return settingService.Save(tx, validPayload)
	}); err != nil {
		t.Fatalf("valid subscription settings rejected: %v", err)
	}

	validCustomPaths, err := json.Marshal(map[string]string{
		"subJsonPath":  "/json-custom/",
		"subClashPath": "/clash-custom/",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Transaction(func(tx *gorm.DB) error {
		return settingService.Save(tx, validCustomPaths)
	}); err != nil {
		t.Fatalf("valid custom subscription paths rejected: %v", err)
	}

	validFragment, err := json.Marshal(map[string]string{
		"subJsonFragment": `{"enabled":true,"packets":"tlshello"}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Transaction(func(tx *gorm.DB) error {
		return settingService.Save(tx, validFragment)
	}); err != nil {
		t.Fatalf("valid JSON fragment setting rejected: %v", err)
	}
	validNoises, err := json.Marshal(map[string]string{
		"subJsonNoises": `[{"type":"rand","packet":"tlshello"}]`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Transaction(func(tx *gorm.DB) error {
		return settingService.Save(tx, validNoises)
	}); err != nil {
		t.Fatalf("valid JSON noises setting rejected: %v", err)
	}

	invalidPayload, err := json.Marshal(map[string]string{
		"subJsonEnable": "sometimes",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Transaction(func(tx *gorm.DB) error {
		return settingService.Save(tx, invalidPayload)
	}); err == nil {
		t.Fatal("expected invalid boolean setting to be rejected")
	}

	invalidURLPayload, err := json.Marshal(map[string]string{
		"subSupportUrl": "ftp://example.com/support",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Transaction(func(tx *gorm.DB) error {
		return settingService.Save(tx, invalidURLPayload)
	}); err == nil {
		t.Fatal("expected invalid URL setting to be rejected")
	}

	invalidFragment, err := json.Marshal(map[string]string{
		"subJsonFragment": "enabled",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Transaction(func(tx *gorm.DB) error {
		return settingService.Save(tx, invalidFragment)
	}); err == nil {
		t.Fatal("expected invalid JSON fragment setting to be rejected")
	}
	invalidNoises, err := json.Marshal(map[string]string{
		"subJsonNoises": `{"type":"rand"}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Transaction(func(tx *gorm.DB) error {
		return settingService.Save(tx, invalidNoises)
	}); err == nil {
		t.Fatal("expected invalid JSON noises setting to be rejected")
	}

	conflictingPaths, err := json.Marshal(map[string]string{
		"subJsonPath":  "/same/",
		"subClashPath": "/same/",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Transaction(func(tx *gorm.DB) error {
		return settingService.Save(tx, conflictingPaths)
	}); err == nil {
		t.Fatal("expected duplicate subscription format paths to be rejected")
	}

	subPathConflict, err := json.Marshal(map[string]string{
		"subPath":     "/custom-sub/",
		"subJsonPath": "/custom-sub/json/",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Transaction(func(tx *gorm.DB) error {
		return settingService.Save(tx, subPathConflict)
	}); err == nil {
		t.Fatal("expected subscription format path under subPath to be rejected")
	}
}

func TestSettingSaveDeterministicForDifferentPayloadOrders(t *testing.T) {
	payloads := []json.RawMessage{
		json.RawMessage(`{"telegramNotifyCpu":"true","telegramCpuThreshold":"75","observabilityMemoryCapMB":"64","subPath":"/sub-custom","subJsonPath":"/json-custom","subClashPath":"/clash-custom","subJsonEnable":"false"}`),
		json.RawMessage(`{"subJsonEnable":"false","subClashPath":"/clash-custom","subJsonPath":"/json-custom","subPath":"/sub-custom","observabilityMemoryCapMB":"64","telegramCpuThreshold":"75","telegramNotifyCpu":"true"}`),
	}
	keys := []string{
		"telegramNotifyCpu",
		"telegramCpuThreshold",
		"observabilityMemoryCapMB",
		"subPath",
		"subJsonPath",
		"subClashPath",
		"subJsonEnable",
	}

	var snapshots []map[string]string
	for _, payload := range payloads {
		settingService := initSettingTestDB(t)
		if _, err := settingService.GetAllSetting(); err != nil {
			t.Fatal(err)
		}
		if err := database.GetDB().Transaction(func(tx *gorm.DB) error {
			return settingService.Save(tx, payload)
		}); err != nil {
			t.Fatal(err)
		}
		snapshot := map[string]string{}
		for _, key := range keys {
			var setting model.Setting
			if err := database.GetDB().Where("key = ?", key).First(&setting).Error; err != nil {
				t.Fatal(err)
			}
			snapshot[key] = setting.Value
		}
		snapshots = append(snapshots, snapshot)
	}

	if !reflect.DeepEqual(snapshots[0], snapshots[1]) {
		t.Fatalf("settings differ for equivalent payload orders: %#v != %#v", snapshots[0], snapshots[1])
	}
}

func TestSaveValidatesTelegramProxyURLBeforeEncrypting(t *testing.T) {
	t.Setenv("SUI_SECRETBOX_KEY", encodedTestSecretboxKey())
	settingService := initSettingTestDB(t)
	if _, err := settingService.GetAllSetting(); err != nil {
		t.Fatal(err)
	}

	invalidPayload, err := json.Marshal(map[string]string{
		"telegramProxyURL": "http://127.0.0.1:8080",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Transaction(func(tx *gorm.DB) error {
		return settingService.Save(tx, invalidPayload)
	}); err == nil {
		t.Fatal("expected private telegramProxyURL to be rejected")
	}

	validPayload, err := json.Marshal(map[string]string{
		"telegramProxyURL": "socks5://8.8.8.8:1080",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Transaction(func(tx *gorm.DB) error {
		return settingService.Save(tx, validPayload)
	}); err != nil {
		t.Fatalf("expected public telegramProxyURL to be accepted: %v", err)
	}
	decrypted, err := settingService.getString("telegramProxyURL")
	if err != nil {
		t.Fatal(err)
	}
	if decrypted != "socks5://8.8.8.8:1080" {
		t.Fatalf("unexpected stored telegramProxyURL: %q", decrypted)
	}
}

func TestSaveValidatesTelegramCPUSettings(t *testing.T) {
	settingService := initSettingTestDB(t)
	if _, err := settingService.GetAllSetting(); err != nil {
		t.Fatal(err)
	}

	validPayload, err := json.Marshal(map[string]string{
		"telegramNotifyCpu":    "true",
		"telegramCpuThreshold": "85",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Transaction(func(tx *gorm.DB) error {
		return settingService.Save(tx, validPayload)
	}); err != nil {
		t.Fatalf("valid CPU settings rejected: %v", err)
	}

	invalidPayload, err := json.Marshal(map[string]string{
		"telegramCpuThreshold": "101",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Transaction(func(tx *gorm.DB) error {
		return settingService.Save(tx, invalidPayload)
	}); err == nil {
		t.Fatal("expected invalid CPU threshold to be rejected")
	}
}

func TestGetTimeLocationRespectsConfiguredLocation(t *testing.T) {
	settingService := initSettingTestDB(t)
	if _, err := settingService.GetAllSetting(); err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Model(model.Setting{}).Where("key = ?", "timeLocation").Update("value", "UTC").Error; err != nil {
		t.Fatal(err)
	}

	location, err := settingService.GetTimeLocation()
	if err != nil {
		t.Fatal(err)
	}
	if location.String() != "UTC" {
		t.Fatalf("expected configured timeLocation to be respected, got %q", location.String())
	}
}

func TestGetTimeLocationFallsBackToLocalForInvalidLocation(t *testing.T) {
	settingService := initSettingTestDB(t)
	if _, err := settingService.GetAllSetting(); err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Model(model.Setting{}).Where("key = ?", "timeLocation").Update("value", "Invalid/Nowhere").Error; err != nil {
		t.Fatal(err)
	}

	location, err := settingService.GetTimeLocation()
	if err != nil {
		t.Fatal(err)
	}
	if location != time.Local {
		t.Fatalf("expected invalid timeLocation to fall back to time.Local, got %q", location.String())
	}
}

func TestSaveValidatesDomainSettings(t *testing.T) {
	settingService := initSettingTestDB(t)
	if _, err := settingService.GetAllSetting(); err != nil {
		t.Fatal(err)
	}

	validPayload, err := json.Marshal(map[string]string{
		"webDomain": "example.com",
		"subDomain": "пример.рф",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Transaction(func(tx *gorm.DB) error {
		return settingService.Save(tx, validPayload)
	}); err != nil {
		t.Fatalf("valid domains rejected: %v", err)
	}

	invalidPayload, err := json.Marshal(map[string]string{
		"webDomain": "example.com:443",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Transaction(func(tx *gorm.DB) error {
		return settingService.Save(tx, invalidPayload)
	}); err == nil {
		t.Fatal("expected domain with port to be rejected")
	}
}

func TestSaveValidatesForceCookieSecureSetting(t *testing.T) {
	settingService := initSettingTestDB(t)
	if _, err := settingService.GetAllSetting(); err != nil {
		t.Fatal(err)
	}

	invalidPayload, err := json.Marshal(map[string]string{
		"forceCookieSecure": "maybe",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Transaction(func(tx *gorm.DB) error {
		return settingService.Save(tx, invalidPayload)
	}); err == nil {
		t.Fatal("expected invalid forceCookieSecure to be rejected")
	}
}

func TestGetForceCookieSecureReadsSettingAndEnvOverride(t *testing.T) {
	settingService := initSettingTestDB(t)
	if _, err := settingService.GetAllSetting(); err != nil {
		t.Fatal(err)
	}

	enabled, err := settingService.GetForceCookieSecure()
	if err != nil {
		t.Fatal(err)
	}
	if enabled {
		t.Fatal("expected default forceCookieSecure=false")
	}

	payload, err := json.Marshal(map[string]string{
		"forceCookieSecure": "true",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Transaction(func(tx *gorm.DB) error {
		return settingService.Save(tx, payload)
	}); err != nil {
		t.Fatal(err)
	}

	enabled, err = settingService.GetForceCookieSecure()
	if err != nil {
		t.Fatal(err)
	}
	if !enabled {
		t.Fatal("expected persisted forceCookieSecure=true")
	}

	t.Setenv("SUI_FORCE_COOKIE_SECURE", "false")
	enabled, err = settingService.GetForceCookieSecure()
	if err != nil {
		t.Fatal(err)
	}
	if enabled {
		t.Fatal("expected env override to force false")
	}
}

func TestGetForceCookieSecureRejectsInvalidEnv(t *testing.T) {
	settingService := initSettingTestDB(t)
	t.Setenv("SUI_FORCE_COOKIE_SECURE", "invalid")
	if _, err := settingService.GetForceCookieSecure(); err == nil {
		t.Fatal("expected invalid SUI_FORCE_COOKIE_SECURE to return error")
	}
}

func TestSaveRejectsUnknownSettingKey(t *testing.T) {
	settingService := initSettingTestDB(t)
	if _, err := settingService.GetAllSetting(); err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(map[string]string{
		"unexpectedKey": "value",
	})
	if err != nil {
		t.Fatal(err)
	}
	err = database.GetDB().Transaction(func(tx *gorm.DB) error {
		return settingService.Save(tx, payload)
	})
	if err == nil {
		t.Fatal("expected unknown setting key to be rejected")
	}
	if !strings.Contains(err.Error(), "invalid setting key") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSaveRejectsProtectedSettingKey(t *testing.T) {
	settingService := initSettingTestDB(t)
	if _, err := settingService.GetAllSetting(); err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(map[string]string{
		"secret": "override-not-allowed",
	})
	if err != nil {
		t.Fatal(err)
	}
	err = database.GetDB().Transaction(func(tx *gorm.DB) error {
		return settingService.Save(tx, payload)
	})
	if err == nil {
		t.Fatal("expected protected setting key to be rejected")
	}
	if !strings.Contains(err.Error(), "invalid setting key") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSaveRejectsUnknownHasSecretMarker(t *testing.T) {
	settingService := initSettingTestDB(t)
	if _, err := settingService.GetAllSetting(); err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(map[string]string{
		"unexpectedHasSecret": "true",
	})
	if err != nil {
		t.Fatal(err)
	}
	err = database.GetDB().Transaction(func(tx *gorm.DB) error {
		return settingService.Save(tx, payload)
	})
	if err == nil {
		t.Fatal("expected unknown HasSecret marker to be rejected")
	}
	if !strings.Contains(err.Error(), "invalid setting key") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetSecretCreatesSettingWhenMissing(t *testing.T) {
	settingService := initSettingTestDB(t)
	db := database.GetDB()

	if err := db.Where("key = ?", "secret").Delete(&model.Setting{}).Error; err != nil {
		t.Fatal(err)
	}

	secret, err := settingService.GetSecret()
	if err != nil {
		t.Fatal(err)
	}
	if len(secret) == 0 {
		t.Fatal("expected non-empty secret")
	}

	var stored model.Setting
	if err := db.Where("key = ?", "secret").First(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Value != string(secret) {
		t.Fatalf("stored secret mismatch: got %q want %q", stored.Value, string(secret))
	}

	secret2, err := settingService.GetSecret()
	if err != nil {
		t.Fatal(err)
	}
	if string(secret2) != string(secret) {
		t.Fatalf("secret changed between reads: got %q want %q", string(secret2), string(secret))
	}

	var count int64
	if err := db.Model(&model.Setting{}).Where("key = ?", "secret").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected single secret row, got %d", count)
	}
}

func TestGetInstallSaltCreatesSettingWhenMissing(t *testing.T) {
	settingService := initSettingTestDB(t)
	db := database.GetDB()

	if err := db.Where("key = ?", "installSalt").Delete(&model.Setting{}).Error; err != nil {
		t.Fatal(err)
	}

	salt, err := settingService.GetInstallSalt()
	if err != nil {
		t.Fatal(err)
	}
	if len(salt) == 0 {
		t.Fatal("expected non-empty install salt")
	}

	var stored model.Setting
	if err := db.Where("key = ?", "installSalt").First(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Value != string(salt) {
		t.Fatalf("stored installSalt mismatch: got %q want %q", stored.Value, string(salt))
	}

	salt2, err := settingService.GetInstallSalt()
	if err != nil {
		t.Fatal(err)
	}
	if string(salt2) != string(salt) {
		t.Fatalf("installSalt changed between reads: got %q want %q", string(salt2), string(salt))
	}

	var count int64
	if err := db.Model(&model.Setting{}).Where("key = ?", "installSalt").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected single installSalt row, got %d", count)
	}
}

func TestGetSecretReturnsErrorWhenDBClosed(t *testing.T) {
	settingService := initSettingTestDB(t)
	if _, err := settingService.GetAllSetting(); err != nil {
		t.Fatal(err)
	}
	liveDB := database.GetDB()
	sqlDB, err := liveDB.DB()
	if err != nil {
		t.Fatal(err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatal(err)
	}

	if _, err := settingService.GetSecret(); err == nil {
		t.Fatal("expected error when database is closed")
	}
}

func TestGetInstallSaltReturnsErrorWhenDBClosed(t *testing.T) {
	settingService := initSettingTestDB(t)
	if _, err := settingService.GetAllSetting(); err != nil {
		t.Fatal(err)
	}
	liveDB := database.GetDB()
	sqlDB, err := liveDB.DB()
	if err != nil {
		t.Fatal(err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatal(err)
	}

	if _, err := settingService.GetInstallSalt(); err == nil {
		t.Fatal("expected error when database is closed")
	}
}
