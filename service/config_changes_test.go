package service

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
)

func TestConfigSaveRedactsSensitiveChangePayload(t *testing.T) {
	t.Setenv("SUI_SECRETBOX_KEY", encodedTestSecretboxKey())
	initSettingTestDB(t)
	t.Cleanup(ReplaceDefaultRuntimeForTest(NewRuntimeWithCoreProvider(nil)))
	payload, err := json.Marshal(map[string]string{
		"telegramBotToken": "1234567890:" + strings.Repeat("A", 35),
		"telegramChatID":   "42",
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := (&ConfigService{}).Save("settings", "set", payload, "", "admin", "example.com"); err != nil {
		t.Fatal(err)
	}

	var change model.Changes
	if err := database.GetDB().Where("key = ?", "settings").First(&change).Error; err != nil {
		t.Fatal(err)
	}
	stored := string(change.Obj)
	if strings.Contains(stored, "1234567890:") || strings.Contains(stored, strings.Repeat("A", 35)) {
		t.Fatalf("change payload leaked secret: %s", stored)
	}
	if !strings.Contains(stored, `"telegramBotToken":"[REDACTED]"`) {
		t.Fatalf("change payload was not redacted: %s", stored)
	}
	if !strings.Contains(stored, `"telegramChatID":"42"`) {
		t.Fatalf("non-sensitive setting was unexpectedly removed: %s", stored)
	}
}
