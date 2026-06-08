package service

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
	"gorm.io/gorm"
)

func TestTelegramBackupRunOnceDisabledAndMissingPassphrase(t *testing.T) {
	settingService := initSettingTestDB(t)
	if _, err := settingService.GetAllSetting(); err != nil {
		t.Fatal(err)
	}
	result := (&TelegramBackupService{}).RunOnce(context.Background(), TelegramBackupTriggerManual)
	if result.Success || result.ErrorClass != "disabled" {
		t.Fatalf("expected disabled, got %#v", result)
	}
	assertTelegramBackupAudit(t, "tg_backup_failed", "disabled", 0, 0)

	configureTelegramBackupSettings(t, settingService, telegramBackupSettings{
		TelegramEnabled: true,
		BackupEnabled:   true,
		Passphrase:      "",
	})
	result = (&TelegramBackupService{}).RunOnce(context.Background(), TelegramBackupTriggerManual)
	if result.Success || result.ErrorClass != "missing_passphrase" {
		t.Fatalf("expected missing_passphrase, got %#v", result)
	}
}

func TestTelegramBackupWeakPassphraseValidation(t *testing.T) {
	settingService := initSettingTestDB(t)
	if _, err := settingService.GetAllSetting(); err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(map[string]string{
		"telegramBackupPassphrase": "short",
	})
	if err != nil {
		t.Fatal(err)
	}
	err = database.GetDB().Transaction(func(tx *gorm.DB) error {
		return settingService.Save(tx, payload)
	})
	if err == nil || !strings.Contains(err.Error(), "weak_passphrase") {
		t.Fatalf("expected weak_passphrase validation, got %v", err)
	}
}

func TestTelegramBackupRunOnceSuccessSendsEnvelopeAndAuditsSizes(t *testing.T) {
	passphrase := "correct horse battery staple"
	settingService := initSettingTestDB(t)
	configureTelegramBackupSettings(t, settingService, telegramBackupSettings{
		TelegramEnabled: true,
		BackupEnabled:   true,
		Passphrase:      passphrase,
	})
	rt := &captureRoundTripper{}
	t.Cleanup(setTelegramHTTPClient(&http.Client{Transport: rt, Timeout: time.Second}))

	ctx := ContextWithTelegramBackupActor(context.Background(), "admin")
	result := (&TelegramBackupService{}).RunOnce(ctx, TelegramBackupTriggerManual)
	if !result.Success || result.Filename == "" || result.Trigger != TelegramBackupTriggerManual {
		t.Fatalf("unexpected result: %#v", result)
	}
	document := telegramDocumentFromCapture(t, rt)
	if !IsTelegramBackupEnvelope(document) {
		t.Fatalf("sent document is not a backup envelope: %x", document[:min(len(document), 16)])
	}
	if bytes.Contains(document, []byte("SQLite format 3")) {
		t.Fatal("sent document contains plaintext SQLite signature")
	}
	plaintext, err := OpenTelegramBackupEnvelope(document, []byte(passphrase))
	if err != nil {
		t.Fatal(err)
	}
	defer zeroBytes(plaintext)
	if !bytes.HasPrefix(plaintext, []byte("SQLite format 3\x00")) {
		t.Fatal("decrypted Telegram document is not SQLite")
	}

	event := assertTelegramBackupAudit(t, "tg_backup_sent", "", result.PayloadSizeBytes, result.EnvelopeSizeBytes)
	if event.Actor != "admin" {
		t.Fatalf("unexpected audit actor: %q", event.Actor)
	}
}

func TestTelegramBackupRunOnceOversizeSkipsSendAndAuditRedactsIssue25(t *testing.T) {
	passphrase := "correct horse battery staple"
	settingService := initSettingTestDB(t)
	configureTelegramBackupSettings(t, settingService, telegramBackupSettings{
		TelegramEnabled: true,
		BackupEnabled:   true,
		Passphrase:      passphrase,
		MaxSizeMB:       "1",
	})
	bigDesc := strings.Repeat("x", 2<<20)
	if err := database.GetDB().Create(&model.Client{Name: "large-client", Desc: bigDesc}).Error; err != nil {
		t.Fatal(err)
	}
	rt := &countingRoundTripper{}
	t.Cleanup(setTelegramHTTPClient(&http.Client{Transport: rt, Timeout: time.Second}))

	result := (&TelegramBackupService{}).RunOnce(context.Background(), TelegramBackupTriggerManual)
	if result.Success || result.ErrorClass != "oversize" {
		t.Fatalf("expected oversize, got %#v", result)
	}
	if rt.Count() != 0 {
		t.Fatalf("oversize backup made %d outbound calls", rt.Count())
	}
	if result.PayloadSizeBytes == 0 || result.EnvelopeSizeBytes == 0 {
		t.Fatalf("sizes were not recorded: %#v", result)
	}
	event := assertTelegramBackupAudit(t, "tg_backup_failed", "oversize", result.PayloadSizeBytes, result.EnvelopeSizeBytes)
	details := string(event.Details)
	for _, forbidden := range []string{
		passphrase,
		"123456:test-token",
		"SQLite format 3",
	} {
		if strings.Contains(details, forbidden) {
			t.Fatalf("oversize backup audit leaked %q in details: %s", forbidden, details)
		}
	}
}

func TestTelegramBackupRunOnceConcurrentGuard(t *testing.T) {
	passphrase := "correct horse battery staple"
	settingService := initSettingTestDB(t)
	configureTelegramBackupSettings(t, settingService, telegramBackupSettings{
		TelegramEnabled: true,
		BackupEnabled:   true,
		Passphrase:      passphrase,
	})
	started := make(chan struct{})
	release := make(chan struct{})
	restoreSend := replaceTelegramBackupSendDocumentForTest(t, func(_ *TelegramService, _ string, _ []byte, _ string) TelegramResult {
		close(started)
		<-release
		return TelegramResult{Success: true}
	})
	defer restoreSend()

	done := make(chan TelegramBackupResult, 1)
	go func() {
		done <- (&TelegramBackupService{}).RunOnce(context.Background(), TelegramBackupTriggerManual)
	}()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("first backup did not reach send")
	}
	result := (&TelegramBackupService{}).RunOnce(context.Background(), TelegramBackupTriggerScheduled)
	if result.Success || result.ErrorClass != "concurrent_run" || result.Trigger != TelegramBackupTriggerScheduled {
		t.Fatalf("expected concurrent_run, got %#v", result)
	}
	close(release)
	if first := <-done; !first.Success {
		t.Fatalf("first backup failed: %#v", first)
	}
}

func TestTelegramBackupRunOncePassesThroughTelegramErrorClassAndFallback(t *testing.T) {
	passphrase := "correct horse battery staple"
	settingService := initSettingTestDB(t)
	configureTelegramBackupSettings(t, settingService, telegramBackupSettings{
		TelegramEnabled: true,
		BackupEnabled:   true,
		Passphrase:      passphrase,
	})

	restoreSend := replaceTelegramBackupSendDocumentForTest(t, func(_ *TelegramService, _ string, _ []byte, _ string) TelegramResult {
		return TelegramResult{ErrorClass: "proxy"}
	})
	result := (&TelegramBackupService{}).RunOnce(context.Background(), TelegramBackupTriggerManual)
	if result.Success || result.ErrorClass != "proxy" {
		t.Fatalf("expected proxy pass-through, got %#v", result)
	}
	restoreSend()

	restoreSend = replaceTelegramBackupSendDocumentForTest(t, func(_ *TelegramService, _ string, _ []byte, _ string) TelegramResult {
		return TelegramResult{}
	})
	defer restoreSend()
	result = (&TelegramBackupService{}).RunOnce(context.Background(), TelegramBackupTriggerManual)
	if result.Success || result.ErrorClass != "internal" {
		t.Fatalf("expected internal fallback, got %#v", result)
	}
}

type telegramBackupSettings struct {
	TelegramEnabled bool
	BackupEnabled   bool
	Passphrase      string
	MaxSizeMB       string
}

func configureTelegramBackupSettings(t *testing.T, settingService *SettingService, cfg telegramBackupSettings) {
	t.Helper()
	t.Setenv("SUI_SECRETBOX_KEY", encodedTestSecretboxKey())
	if _, err := settingService.GetAllSetting(); err != nil {
		t.Fatal(err)
	}
	if cfg.MaxSizeMB == "" {
		cfg.MaxSizeMB = "45"
	}
	payload, err := json.Marshal(map[string]string{
		"telegramEnabled":             boolString(cfg.TelegramEnabled),
		"telegramBotToken":            "123456:test-token",
		"telegramChatID":              "42",
		"telegramBackupEnabled":       boolString(cfg.BackupEnabled),
		"telegramBackupPassphrase":    cfg.Passphrase,
		"telegramBackupMaxSizeMB":     cfg.MaxSizeMB,
		"telegramBackupExcludeTables": "stats,client_ips,audit_events,changes",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Transaction(func(tx *gorm.DB) error {
		return settingService.Save(tx, payload)
	}); err != nil {
		t.Fatal(err)
	}
}

func boolString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func replaceTelegramBackupSendDocumentForTest(t *testing.T, send func(*TelegramService, string, []byte, string) TelegramResult) func() {
	t.Helper()
	old := telegramBackupSendDocument
	telegramBackupSendDocument = send
	return func() {
		telegramBackupSendDocument = old
	}
}

func telegramDocumentFromCapture(t *testing.T, rt *captureRoundTripper) []byte {
	t.Helper()
	if rt.req == nil {
		t.Fatal("no Telegram request captured")
	}
	_, params, err := mime.ParseMediaType(rt.req.Header.Get("Content-Type"))
	if err != nil {
		t.Fatal(err)
	}
	reader := multipart.NewReader(bytes.NewReader(rt.body), params["boundary"])
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if part.FormName() != "document" {
			continue
		}
		content, err := io.ReadAll(part)
		if err != nil {
			t.Fatal(err)
		}
		return content
	}
	t.Fatal("document multipart field not found")
	return nil
}

func assertTelegramBackupAudit(t *testing.T, eventName string, errorClass string, payloadSize int64, envelopeSize int64) model.AuditEvent {
	t.Helper()
	var event model.AuditEvent
	if err := database.GetDB().Where("event = ?", eventName).Order("id desc").First(&event).Error; err != nil {
		t.Fatal(err)
	}
	var details map[string]any
	if err := json.Unmarshal(event.Details, &details); err != nil {
		t.Fatal(err)
	}
	if details["channel"] != "telegram" || details["trigger"] == "" {
		t.Fatalf("unexpected audit details: %#v", details)
	}
	if errorClass != "" && details["errorClass"] != errorClass {
		t.Fatalf("unexpected errorClass in audit details: %#v", details)
	}
	if payloadSize > 0 && int64(details["payloadSizeBytes"].(float64)) != payloadSize {
		t.Fatalf("unexpected payload size: %#v want %d", details, payloadSize)
	}
	if envelopeSize > 0 && int64(details["envelopeSizeBytes"].(float64)) != envelopeSize {
		t.Fatalf("unexpected envelope size: %#v want %d", details, envelopeSize)
	}
	if strings.Contains(string(event.Details), "correct horse battery staple") {
		t.Fatalf("audit leaked passphrase: %s", string(event.Details))
	}
	return event
}
