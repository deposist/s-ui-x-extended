package service

import (
	"testing"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
	"github.com/deposist/s-ui-x/util/secretbox"
)

func TestEncryptDecryptSettingValueRoundTripExtra(t *testing.T) {
	t.Setenv("SUI_SECRETBOX_KEY", encodedTestSecretboxKey())
	settingService := initSettingTestDB(t)

	encrypted, err := settingService.encryptSettingValue("telegramBotToken", "phase2-secret")
	if err != nil {
		t.Fatal(err)
	}
	if encrypted == "phase2-secret" || !secretbox.IsEncrypted(encrypted) {
		t.Fatalf("value was not encrypted: %q", encrypted)
	}
	decrypted, err := settingService.decryptSettingValue("telegramBotToken", encrypted)
	if err != nil {
		t.Fatal(err)
	}
	if decrypted != "phase2-secret" {
		t.Fatalf("unexpected decrypted value %q", decrypted)
	}
}

func TestDecryptPrimarySecretboxCandidateDoesNotAuditFallback(t *testing.T) {
	t.Setenv("SUI_SECRETBOX_KEY", encodedTestSecretboxKey())
	settingService := initSettingTestDB(t)

	encrypted, err := settingService.encryptSettingValue("telegramProxyPassword", "primary-secret")
	if err != nil {
		t.Fatal(err)
	}
	decrypted, err := settingService.decryptSettingValue("telegramProxyPassword", encrypted)
	if err != nil {
		t.Fatal(err)
	}
	if decrypted != "primary-secret" {
		t.Fatalf("unexpected decrypted value %q", decrypted)
	}

	var count int64
	if err := database.GetDB().Model(model.AuditEvent{}).Where("event = ?", "settings_secretbox_key_fallback").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("primary candidate decrypt wrote fallback audit events: %d", count)
	}
}

func TestLegacySecretboxFallbackDoesNotAuditAfterFix_XFAILIssue17(t *testing.T) {
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
	if _, err := settingService.decryptSettingValue("telegramBotToken", legacyValue); err != nil {
		t.Fatal(err)
	}

	var count int64
	if err := database.GetDB().Model(model.AuditEvent{}).Where("event = ?", "settings_secretbox_key_fallback").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("legacy fallback should not write audit noise after issue 17 fix, got %d", count)
	}
}
