package service

import "testing"

func TestValidateSubscriptionPathSettingsRejectsPhase2Conflicts(t *testing.T) {
	settingService := &SettingService{}
	tests := []map[string]string{
		{
			"subPath":      "/sub/",
			"subJsonPath":  "/same/",
			"subClashPath": "/same/",
		},
		{
			"subPath":      "/subscriptions/",
			"subJsonPath":  "/subscriptions/json/",
			"subClashPath": "/clash/",
		},
	}
	for _, settings := range tests {
		if err := settingService.validateSubscriptionPathSettings(settings); err == nil {
			t.Fatalf("expected subscription path conflict for %#v", settings)
		}
	}
}

func TestValidateTelegramSettingInputRejectsWeakBackupPassphrase(t *testing.T) {
	if err := validateTelegramSettingInput("telegramBackupPassphrase", "too-short"); err == nil {
		t.Fatal("weak telegram backup passphrase should be rejected")
	}
	if err := validateTelegramSettingInput("telegramBackupPassphrase", "correct horse battery staple"); err != nil {
		t.Fatalf("strong telegram backup passphrase should be accepted: %v", err)
	}
}

func TestValidateOptionalHTTPURLRejectsUserInfo(t *testing.T) {
	if err := validateOptionalHTTPURL("https://user:pass@example.com/profile"); err == nil {
		t.Fatal("URL with user-info should be rejected")
	}
	if err := validateOptionalHTTPURL("https://example.com/profile"); err != nil {
		t.Fatalf("plain HTTPS URL should be accepted: %v", err)
	}
}

func TestValidateOptionalHTTPURLRejectsUnsafePartsIssue30(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{
			name:    "reject fragment",
			value:   "https://example.com/profile#token",
			wantErr: true,
		},
		{
			name:    "reject raw newline",
			value:   "https://example.com/profile\nX-Test: value",
			wantErr: true,
		},
		{
			name:    "reject leading raw newline",
			value:   "\nhttps://example.com/profile",
			wantErr: true,
		},
		{
			name:    "reject trailing raw CRLF and tab",
			value:   "https://example.com/profile\r\n\t",
			wantErr: true,
		},
		{
			name:    "reject raw tab in path",
			value:   "https://example.com/pro\tfile",
			wantErr: true,
		},
		{
			name:    "reject encoded newline in path",
			value:   "https://example.com/%0a",
			wantErr: true,
		},
		{
			name:    "accept query string",
			value:   "https://example.com/profile?from=sub",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateOptionalHTTPURL(tt.value)
			if tt.wantErr && err == nil {
				t.Fatal("expected URL setting to be rejected")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected URL setting to be accepted: %v", err)
			}
		})
	}
}
