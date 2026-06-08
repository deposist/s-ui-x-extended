package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
)

func TestSecurityTelegramBackupAuditOmitsPayloadPassphraseAndToken(t *testing.T) {
	passphrase := "correct horse battery staple"
	settingService := initSettingTestDB(t)
	configureTelegramBackupSettings(t, settingService, telegramBackupSettings{
		TelegramEnabled: true,
		BackupEnabled:   true,
		Passphrase:      passphrase,
	})
	restoreSend := replaceTelegramBackupSendDocumentForTest(t, func(_ *TelegramService, _ string, _ []byte, _ string) TelegramResult {
		return TelegramResult{Success: true}
	})
	defer restoreSend()

	result := (&TelegramBackupService{}).RunOnce(ContextWithTelegramBackupActor(context.Background(), "admin"), TelegramBackupTriggerManual)
	if !result.Success {
		t.Fatalf("backup failed: %#v", result)
	}
	var event model.AuditEvent
	if err := database.GetDB().Where("event = ?", "tg_backup_sent").Order("id desc").First(&event).Error; err != nil {
		t.Fatal(err)
	}
	details := string(event.Details)
	for _, forbidden := range []string{
		passphrase,
		"123456:test-token",
		"SQLite format 3",
	} {
		if strings.Contains(details, forbidden) {
			t.Fatalf("backup audit leaked %q in details: %s", forbidden, details)
		}
	}
	for _, expected := range []string{`"payloadSizeBytes"`, `"envelopeSizeBytes"`, `"channel":"telegram"`} {
		if !strings.Contains(details, expected) {
			t.Fatalf("backup audit missing %s in details: %s", expected, details)
		}
	}
}

func TestSecurityConfigChangeRedactsTelegramBackupPassphrase(t *testing.T) {
	t.Setenv("SUI_SECRETBOX_KEY", encodedTestSecretboxKey())
	initSettingTestDB(t)
	passphrase := "correct horse battery staple"
	payload, err := json.Marshal(map[string]string{
		"telegramBackupPassphrase": passphrase,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := (&ConfigService{}).Save("settings", "set", payload, "", "admin", "example.com"); err != nil {
		t.Fatal(err)
	}
	var change model.Changes
	if err := database.GetDB().Where("key = ?", "settings").Order("id desc").First(&change).Error; err != nil {
		t.Fatal(err)
	}
	stored := string(change.Obj)
	if strings.Contains(stored, passphrase) {
		t.Fatalf("change payload leaked telegramBackupPassphrase: %s", stored)
	}
	if !strings.Contains(stored, `"telegramBackupPassphrase":"[REDACTED]"`) {
		t.Fatalf("change payload did not redact telegramBackupPassphrase: %s", stored)
	}
}

func TestSecurityTelegramBackupSecretBagZeroizationIssue25(t *testing.T) {
	payload := []byte("SQLite format 3\x00sensitive payload")
	passphrase := []byte("correct horse battery staple")
	bag := telegramBackupSecretBag{}

	bag.setPayload(payload)
	bag.setPassphrase(passphrase)
	bag.zeroPassphrase()
	assertZeroedBytes(t, "passphrase", passphrase)
	if bag.passphrase != nil {
		t.Fatal("passphrase should be released from secret bag after zeroPassphrase")
	}
	if allBytesZero(payload) {
		t.Fatal("payload should remain owned until payload zeroization")
	}

	bag.zero()
	assertZeroedBytes(t, "payload", payload)
	if bag.payload != nil {
		t.Fatal("payload should be released from secret bag after zero")
	}
}

func assertZeroedBytes(t *testing.T, label string, buf []byte) {
	t.Helper()
	if !allBytesZero(buf) {
		t.Fatalf("%s was not zeroized: %q", label, string(buf))
	}
}

func allBytesZero(buf []byte) bool {
	for _, value := range buf {
		if value != 0 {
			return false
		}
	}
	return true
}
