package service

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/sagernet/sing-box/adapter"
	singService "github.com/sagernet/sing/service"
)

func TestProviderSavePreservesReferencedProvider(t *testing.T) {
	for _, action := range []string{"del", "edit"} {
		t.Run(action, func(t *testing.T) {
			initSettingTestDB(t)
			db := database.GetDB()
			provider := model.Provider{Type: "inline", Tag: "provider-a", Options: json.RawMessage(`{"outbounds":[]}`)}
			if err := db.Create(&provider).Error; err != nil {
				t.Fatal(err)
			}
			group := model.Outbound{Type: "selector", Tag: "provider-group", Options: json.RawMessage(`{"providers":["provider-a"]}`)}
			if err := db.Create(&group).Error; err != nil {
				t.Fatal(err)
			}
			payload := json.RawMessage(`"provider-a"`)
			if action == "edit" {
				payload = json.RawMessage(fmt.Sprintf(`{"id":%d,"type":"inline","tag":"provider-b","outbounds":[]}`, provider.Id))
			}
			if err := (&ProviderService{}).Save(db, action, payload); err == nil {
				t.Fatal("provider referenced by a selector must not be removed or renamed")
			}
			var saved model.Provider
			if err := db.First(&saved, provider.Id).Error; err != nil || saved.Tag != provider.Tag {
				t.Fatalf("referenced provider was changed: tag=%q err=%v", saved.Tag, err)
			}
		})
	}
}

func TestConfigSaveProviderAppliesToRunningCore(t *testing.T) {
	initSettingTestDB(t)
	instance := startTestCore(t)
	configService := NewConfigServiceWithRuntime(NewRuntime(instance))
	payload := json.RawMessage(`{"type":"inline","tag":"created-provider","outbounds":[{"type":"socks","tag":"provider-member","server":"127.0.0.1","server_port":1080}]}`)
	if _, err := configService.Save("providers", "new", payload, "", "admin", "example.com"); err != nil {
		t.Fatal(err)
	}
	manager := singService.FromContext[adapter.ProviderManager](instance.GetCtx())
	if manager == nil {
		t.Fatal("running core has no provider manager")
	}
	if _, exists := manager.Get("created-provider"); !exists {
		t.Fatal("saved provider was not applied to the running core")
	}
}
