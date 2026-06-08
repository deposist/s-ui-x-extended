package service

import (
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
)

func TestTelegramHTTPClientConcurrentReloadRaceAnchorIssue21(t *testing.T) {
	settingService := initSettingTestDB(t)
	setTelegramProxyConfigIssue21(t, settingService, telegramProxyConfig{})
	seed := seedTelegramHTTPClientCacheIssue21(t, telegramProxyConfig{URL: "http://8.8.8.8:8080"})

	const workers = 64

	service := &TelegramService{}
	clients := make([]*http.Client, workers)
	errs := make(chan error, workers)
	start := make(chan struct{})

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			<-start
			client, err := service.getTelegramHTTPClient()
			if err != nil {
				errs <- err
				return
			}
			clients[index] = client
		}(i)
	}
	close(start)
	wg.Wait()
	close(errs)

	for err := range errs {
		t.Fatal(err)
	}
	if clients[0] == nil {
		t.Fatal("first worker received nil telegram http client")
	}
	if clients[0] == seed {
		t.Fatal("telegram http client should be replaced when config changes")
	}
	for i, client := range clients {
		if client != clients[0] {
			t.Fatalf("worker %d received a different telegram http client", i)
		}
	}
}

func TestTelegramHTTPClientReusesClientOnSameConfigIssue21(t *testing.T) {
	settingService := initSettingTestDB(t)
	setTelegramProxyConfigIssue21(t, settingService, telegramProxyConfig{})
	seed := seedTelegramHTTPClientCacheIssue21(t, telegramProxyConfig{})

	service := &TelegramService{}
	first, err := service.getTelegramHTTPClient()
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.getTelegramHTTPClient()
	if err != nil {
		t.Fatal(err)
	}
	if first != seed {
		t.Fatal("expected first call to reuse the seeded client")
	}
	if second != first {
		t.Fatal("expected same config to reuse the telegram http client")
	}
}

func TestTelegramHTTPClientReplacesClientOnDifferentConfigIssue21(t *testing.T) {
	settingService := initSettingTestDB(t)
	setTelegramProxyConfigIssue21(t, settingService, telegramProxyConfig{})
	seed := seedTelegramHTTPClientCacheIssue21(t, telegramProxyConfig{})
	setTelegramProxyConfigIssue21(t, settingService, telegramProxyConfig{URL: "http://8.8.8.8:8080"})

	client, err := (&TelegramService{}).getTelegramHTTPClient()
	if err != nil {
		t.Fatal(err)
	}
	if client == nil {
		t.Fatal("telegram http client should not be nil")
	}
	if client == seed {
		t.Fatal("expected different config to replace the telegram http client")
	}
}

func setTelegramProxyConfigIssue21(t *testing.T, settingService *SettingService, cfg telegramProxyConfig) {
	t.Helper()
	if _, err := settingService.GetAllSetting(); err != nil {
		t.Fatal(err)
	}
	settings := map[string]string{
		"telegramProxyURL":      cfg.URL,
		"telegramProxyUsername": cfg.Username,
		"telegramProxyPassword": cfg.Password,
	}
	for key, value := range settings {
		if err := database.GetDB().Model(model.Setting{}).Where("key = ?", key).Update("value", value).Error; err != nil {
			t.Fatal(err)
		}
	}
}

func seedTelegramHTTPClientCacheIssue21(t *testing.T, cfg telegramProxyConfig) *http.Client {
	t.Helper()

	client := &http.Client{Timeout: time.Second}

	telegramHTTPClientMu.Lock()
	oldClient := telegramHTTPClient
	oldOverride := telegramHTTPOverride
	oldConfig := telegramHTTPConfig
	telegramHTTPClient = client
	telegramHTTPOverride = false
	telegramHTTPConfig = cfg
	telegramHTTPClientMu.Unlock()

	t.Cleanup(func() {
		telegramHTTPClientMu.Lock()
		telegramHTTPClient = oldClient
		telegramHTTPOverride = oldOverride
		telegramHTTPConfig = oldConfig
		telegramHTTPClientMu.Unlock()
	})

	return client
}
