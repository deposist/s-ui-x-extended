package service

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
	"github.com/deposist/s-ui-x/util/secretbox"
	"gorm.io/gorm"
)

func initSettingTestDB(t *testing.T) *SettingService {
	t.Helper()
	prevAuditSync := AuditSyncForTest
	AuditSyncForTest = true
	t.Cleanup(func() { AuditSyncForTest = prevAuditSync })
	t.Setenv("SUI_DB_FOLDER", t.TempDir())
	if err := database.InitDB(filepath.Join(t.TempDir(), "s-ui.db")); err != nil {
		if strings.Contains(err.Error(), "go-sqlite3 requires cgo") {
			t.Skip(err)
		}
		t.Fatal(err)
	}
	testDB := database.GetDB()
	t.Cleanup(func() {
		if testDB != nil {
			if sqlDB, err := testDB.DB(); err == nil {
				_ = sqlDB.Close()
				time.Sleep(25 * time.Millisecond)
			}
		}
	})
	return &SettingService{}
}

func encodedTestSecretboxKey() string {
	return base64.RawURLEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
}

func TestSecretSettingIsEncryptedAndMasked(t *testing.T) {
	t.Setenv("SUI_SECRETBOX_KEY", encodedTestSecretboxKey())
	settingService := initSettingTestDB(t)

	if _, err := settingService.GetAllSetting(); err != nil {
		t.Fatal(err)
	}

	payload, err := json.Marshal(map[string]string{
		"telegramBotToken": "123456:secret-token",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Transaction(func(tx *gorm.DB) error {
		return settingService.Save(tx, payload)
	}); err != nil {
		t.Fatal(err)
	}

	var setting model.Setting
	if err := database.GetDB().Where("key = ?", "telegramBotToken").First(&setting).Error; err != nil {
		t.Fatal(err)
	}
	if setting.Value == "123456:secret-token" || !secretbox.IsEncrypted(setting.Value) {
		t.Fatalf("secret setting was not encrypted: %q", setting.Value)
	}

	decrypted, err := settingService.getString("telegramBotToken")
	if err != nil {
		t.Fatal(err)
	}
	if decrypted != "123456:secret-token" {
		t.Fatalf("unexpected decrypted value %q", decrypted)
	}

	settings, err := settingService.GetAllSetting()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := (*settings)["telegramBotToken"]; ok {
		t.Fatal("raw telegramBotToken leaked through settings API")
	}
	if (*settings)["telegramBotTokenHasSecret"] != "true" {
		t.Fatalf("expected has-secret marker, got %q", (*settings)["telegramBotTokenHasSecret"])
	}

	emptyPayload, err := json.Marshal(map[string]string{
		"telegramBotToken": "",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Transaction(func(tx *gorm.DB) error {
		return settingService.Save(tx, emptyPayload)
	}); err != nil {
		t.Fatal(err)
	}
	afterEmpty, err := settingService.getString("telegramBotToken")
	if err != nil {
		t.Fatal(err)
	}
	if afterEmpty != "123456:secret-token" {
		t.Fatalf("empty secret save should keep old value, got %q", afterEmpty)
	}
}

func TestLegacyPlaintextSecretRoundTripEncryptsOnSave(t *testing.T) {
	t.Setenv("SUI_SECRETBOX_KEY", encodedTestSecretboxKey())
	settingService := initSettingTestDB(t)
	if _, err := settingService.GetAllSetting(); err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Model(model.Setting{}).Where("key = ?", "telegramProxyPassword").Update("value", "legacy-plain-secret").Error; err != nil {
		t.Fatal(err)
	}

	got, err := settingService.getString("telegramProxyPassword")
	if err != nil {
		t.Fatal(err)
	}
	if got != "legacy-plain-secret" {
		t.Fatalf("legacy plaintext secret did not round-trip: %q", got)
	}

	payload, err := json.Marshal(map[string]string{
		"telegramProxyPassword": got,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Transaction(func(tx *gorm.DB) error {
		return settingService.Save(tx, payload)
	}); err != nil {
		t.Fatal(err)
	}

	var stored model.Setting
	if err := database.GetDB().Where("key = ?", "telegramProxyPassword").First(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Value == got || !secretbox.IsEncrypted(stored.Value) {
		t.Fatalf("legacy plaintext secret was not encrypted on save: %q", stored.Value)
	}
	after, err := settingService.getString("telegramProxyPassword")
	if err != nil {
		t.Fatal(err)
	}
	if after != got {
		t.Fatalf("encrypted legacy secret did not round-trip: %q", after)
	}
}

func TestTelegramBackupPassphraseEncryptedMaskedAndClearable(t *testing.T) {
	t.Setenv("SUI_SECRETBOX_KEY", encodedTestSecretboxKey())
	settingService := initSettingTestDB(t)

	settings, err := settingService.GetAllSetting()
	if err != nil {
		t.Fatal(err)
	}
	if (*settings)["telegramBackupPassphrase"] != "" || (*settings)["telegramBackupPassphraseHasSecret"] != "false" {
		t.Fatalf("unexpected default passphrase markers: %#v", *settings)
	}

	weakPayload, err := json.Marshal(map[string]string{
		"telegramBackupPassphrase": "too-short",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Transaction(func(tx *gorm.DB) error {
		return settingService.Save(tx, weakPayload)
	}); err == nil || !strings.Contains(err.Error(), "weak_passphrase") {
		t.Fatalf("expected weak passphrase validation, got %v", err)
	}

	passphrase := "correct horse battery staple"
	payload, err := json.Marshal(map[string]string{
		"telegramBackupPassphrase": passphrase,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := (&ConfigService{}).Save("settings", "set", payload, "", "admin", "localhost"); err != nil {
		t.Fatal(err)
	}
	var stored model.Setting
	if err := database.GetDB().Where("key = ?", "telegramBackupPassphrase").First(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Value == passphrase || !secretbox.IsEncrypted(stored.Value) {
		t.Fatalf("backup passphrase was not encrypted: %q", stored.Value)
	}
	decrypted, err := settingService.GetTelegramBackupPassphraseBytes()
	if err != nil {
		t.Fatal(err)
	}
	if string(decrypted) != passphrase {
		t.Fatalf("unexpected passphrase %q", string(decrypted))
	}
	zeroBytes(decrypted)

	settings, err = settingService.GetAllSetting()
	if err != nil {
		t.Fatal(err)
	}
	if (*settings)["telegramBackupPassphrase"] != StoredSecretMarker || (*settings)["telegramBackupPassphraseHasSecret"] != "true" {
		t.Fatalf("passphrase was not masked: %#v", *settings)
	}

	var event model.AuditEvent
	if err := database.GetDB().Where("event = ?", "tg_backup_passphrase_changed").First(&event).Error; err != nil {
		t.Fatal(err)
	}
	if event.Actor != "admin" || event.Severity != AuditSeverityInfo || strings.Contains(string(event.Details), passphrase) {
		t.Fatalf("unexpected audit event: %#v details=%s", event, event.Details)
	}

	clearPayload, err := json.Marshal(map[string]string{
		"telegramBackupPassphrase": "",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := (&ConfigService{}).Save("settings", "set", clearPayload, "", "admin", "localhost"); err != nil {
		t.Fatal(err)
	}
	decrypted, err = settingService.GetTelegramBackupPassphraseBytes()
	if err != nil {
		t.Fatal(err)
	}
	if len(decrypted) != 0 {
		t.Fatalf("passphrase was not cleared: %q", string(decrypted))
	}
}

func TestGetCookieKeysDerivedFromSecretByDefault(t *testing.T) {
	settingService := initSettingTestDB(t)

	secret, err := settingService.GetSecret()
	if err != nil {
		t.Fatal(err)
	}
	keys, err := settingService.GetCookieKeys()
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) == 0 {
		t.Fatalf("expected at least one cookie key, got %d", len(keys))
	}
	if len(keys[0]) != 32 {
		t.Fatalf("expected 32-byte cookie key, got %d", len(keys[0]))
	}
	if bytes.Equal(keys[0], secret) {
		t.Fatal("cookie key must be domain-separated from settings.secret")
	}
	if len(keys) < 2 {
		t.Fatalf("expected legacy cookie fallback key, got %d keys", len(keys))
	}
	keys2, err := settingService.GetCookieKeys()
	if err != nil {
		t.Fatal(err)
	}
	if len(keys2) != len(keys) {
		t.Fatalf("cookie key count changed between calls: %d != %d", len(keys2), len(keys))
	}
	for i := range keys {
		if !bytes.Equal(keys[i], keys2[i]) {
			t.Fatalf("derived cookie key %d changed between calls", i)
		}
	}
}

func TestDerivedSettingKeysUseDomainSeparatedInfo(t *testing.T) {
	master := []byte("test-master-key-material-32-bytes!!")
	cookieKey, err := deriveHKDFKey(master, nil, cookieKeyHKDFInfo)
	if err != nil {
		t.Fatal(err)
	}
	secretboxKey, err := deriveHKDFKey(master, nil, settingsSecretboxKeyHKDFInfo)
	if err != nil {
		t.Fatal(err)
	}
	cookieKeyAgain, err := deriveHKDFKey(master, nil, cookieKeyHKDFInfo)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(cookieKey, cookieKeyAgain) {
		t.Fatal("cookie HKDF derivation is not deterministic")
	}
	if bytes.Equal(cookieKey, secretboxKey) {
		t.Fatal("cookie and settings secretbox keys must use distinct HKDF info")
	}
}

func TestGetCookieKeysUsesEnvRolloverList(t *testing.T) {
	settingService := initSettingTestDB(t)

	key1 := []byte("0123456789abcdef0123456789abcdef")
	key2 := []byte("abcdef0123456789abcdef0123456789")
	t.Setenv("SUI_COOKIE_KEY", base64.RawURLEncoding.EncodeToString(key1)+","+base64.RawURLEncoding.EncodeToString(key2))

	keys, err := settingService.GetCookieKeys()
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 2 {
		t.Fatalf("expected two cookie keys, got %d", len(keys))
	}
	if !bytes.Equal(keys[0], key1) || !bytes.Equal(keys[1], key2) {
		t.Fatalf("unexpected cookie key order/values: %q %q", keys[0], keys[1])
	}
}

func TestGetCookieKeysInvalidEnvFallsBackToDerivedKey(t *testing.T) {
	t.Setenv("SUI_COOKIE_KEY", "short")
	settingService := initSettingTestDB(t)

	keys, err := settingService.GetCookieKeys()
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) < 2 || len(keys[0]) != 32 {
		t.Fatalf("expected derived 32-byte cookie key, got %d keys len=%d", len(keys), len(keys[0]))
	}
	var count int64
	if err := database.GetDB().Model(model.Setting{}).Where("key = ?", "secret").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected fallback to create secret setting row, got %d", count)
	}
}

func TestInvalidSecretboxEnvFallsBackToSettingsSecret(t *testing.T) {
	t.Setenv("SUI_SECRETBOX_KEY", "short")
	settingService := initSettingTestDB(t)

	encrypted, err := settingService.encryptSettingValue("telegramBotToken", "fallback-secret")
	if err != nil {
		t.Fatal(err)
	}
	if !secretbox.IsEncrypted(encrypted) {
		t.Fatalf("expected encrypted value, got %q", encrypted)
	}
	decrypted, err := settingService.decryptSettingValue("telegramBotToken", encrypted)
	if err != nil {
		t.Fatal(err)
	}
	if decrypted != "fallback-secret" {
		t.Fatalf("unexpected decrypted value %q", decrypted)
	}

	var count int64
	if err := database.GetDB().Model(model.Setting{}).Where("key = ?", "secret").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected fallback to create secret setting row, got %d", count)
	}
}

func TestSecretboxUsesEnvRawKeyOverride(t *testing.T) {
	rawKey := []byte("abcdef0123456789abcdef0123456789")
	t.Setenv("SUI_SECRETBOX_KEY", base64.RawURLEncoding.EncodeToString(rawKey))
	settingService := initSettingTestDB(t)

	encrypted, err := settingService.encryptSettingValue("telegramBotToken", "env-secret")
	if err != nil {
		t.Fatal(err)
	}
	rawBox, err := secretbox.NewRawKey(rawKey)
	if err != nil {
		t.Fatal(err)
	}
	decrypted, err := rawBox.DecryptString(encrypted, "telegramBotToken")
	if err != nil {
		t.Fatal(err)
	}
	if decrypted != "env-secret" {
		t.Fatalf("unexpected decrypted value %q", decrypted)
	}

	legacyBox, err := secretbox.New(rawKey)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := legacyBox.DecryptString(encrypted, "telegramBotToken"); err == nil {
		t.Fatal("env raw-key ciphertext should not decrypt with legacy HKDF constructor")
	}
}

func TestSecretboxLegacyFallbackAudits(t *testing.T) {
	settingService := initSettingTestDB(t)
	if _, err := settingService.GetAllSetting(); err != nil {
		t.Fatal(err)
	}
	secret, err := settingService.GetSecret()
	if err != nil {
		t.Fatal(err)
	}
	legacyBox, err := secretbox.New(secret)
	if err != nil {
		t.Fatal(err)
	}
	legacyValue, err := legacyBox.EncryptString("legacy-secret", "telegramBotToken")
	if err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Model(model.Setting{}).Where("key = ?", "telegramBotToken").Update("value", legacyValue).Error; err != nil {
		t.Fatal(err)
	}

	got, err := settingService.getString("telegramBotToken")
	if err != nil {
		t.Fatal(err)
	}
	if got != "legacy-secret" {
		t.Fatalf("unexpected legacy decrypted value %q", got)
	}

	var event model.AuditEvent
	if err := database.GetDB().Where("event = ?", "settings_secretbox_key_fallback").First(&event).Error; err != nil {
		t.Fatal(err)
	}
	if event.Resource != "settings" || event.Severity != AuditSeverityWarn {
		t.Fatalf("unexpected fallback audit event: %#v", event)
	}
	if !strings.Contains(string(event.Details), `"key":"telegramBotToken"`) ||
		!strings.Contains(string(event.Details), `"candidate":"legacy_settings_secret"`) {
		t.Fatalf("unexpected fallback audit details: %s", event.Details)
	}
	if strings.Contains(string(event.Details), "legacy-secret") {
		t.Fatalf("secret leaked to fallback audit details: %s", event.Details)
	}
}

func TestSecretboxEnvOverrideCanReadSettingsSecretLegacyCiphertext(t *testing.T) {
	settingService := initSettingTestDB(t)
	if _, err := settingService.GetAllSetting(); err != nil {
		t.Fatal(err)
	}
	secret, err := settingService.GetSecret()
	if err != nil {
		t.Fatal(err)
	}
	legacyBox, err := secretbox.New(secret)
	if err != nil {
		t.Fatal(err)
	}
	legacyValue, err := legacyBox.EncryptString("legacy-before-env", "telegramProxyPassword")
	if err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Model(model.Setting{}).Where("key = ?", "telegramProxyPassword").Update("value", legacyValue).Error; err != nil {
		t.Fatal(err)
	}

	t.Setenv("SUI_SECRETBOX_KEY", encodedTestSecretboxKey())
	got, err := settingService.getString("telegramProxyPassword")
	if err != nil {
		t.Fatal(err)
	}
	if got != "legacy-before-env" {
		t.Fatalf("unexpected fallback value %q", got)
	}

	var event model.AuditEvent
	if err := database.GetDB().Where("event = ?", "settings_secretbox_key_fallback").First(&event).Error; err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(event.Details), `"candidate":"legacy_settings_secret"`) {
		t.Fatalf("unexpected fallback audit details: %s", event.Details)
	}
}
