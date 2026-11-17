package service

import "testing"

func TestTelegramOutboundTransportUsesInjectedRuntime(t *testing.T) {
	settings := initSettingTestDB(t)
	seedTelegramHTTPClientCacheIssue21(t, telegramProxyConfig{})
	t.Cleanup(ReplaceDefaultRuntimeForTest(NewRuntimeWithCoreProvider(nil)))
	if err := settings.setString("telegramTransportMode", "outbound"); err != nil {
		t.Fatal(err)
	}
	if err := settings.setString("telegramOutboundTag", "direct"); err != nil {
		t.Fatal(err)
	}
	service := &TelegramService{Runtime: NewRuntime(startTestCore(t))}
	client, err := service.getTelegramHTTPClient()
	if err != nil {
		t.Fatalf("outbound transport ignored injected running core: %v", err)
	}
	if client == nil || client.Transport == nil {
		t.Fatal("outbound transport client was not constructed")
	}
	client.CloseIdleConnections()
}
