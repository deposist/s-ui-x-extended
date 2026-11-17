package importxui

import (
	"encoding/json"
	"testing"

	"github.com/deposist/s-ui-x-extended/database/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRoutingImportHandlesNullConfigSections(t *testing.T) {
	for _, current := range []string{`{"route":null}`, `{"dns":null}`, `null`} {
		t.Run(current, func(t *testing.T) {
			db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
			if err != nil {
				t.Fatal(err)
			}
			sqlDB, err := db.DB()
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := sqlDB.Close(); err != nil {
					t.Error(err)
				}
			})
			if err := db.AutoMigrate(&model.Setting{}); err != nil {
				t.Fatal(err)
			}
			if err := db.Create(&model.Setting{Key: "config", Value: current}).Error; err != nil {
				t.Fatal(err)
			}
			mapped := map[string]any{
				"route": map[string]any{"rules": []any{map[string]any{"domain": []string{"example.com"}, "outbound": "direct"}}},
				"dns":   map[string]any{"servers": []any{map[string]any{"type": "local", "tag": "imported-dns"}}, "final": "imported-dns"},
			}
			if err := mergeRoutingIntoConfig(db, mapped); err != nil {
				t.Fatal(err)
			}
			var setting model.Setting
			if err := db.Where("key = ?", "config").First(&setting).Error; err != nil {
				t.Fatal(err)
			}
			var cfg struct {
				Route struct{ Rules []json.RawMessage }   `json:"route"`
				DNS   struct{ Servers []json.RawMessage } `json:"dns"`
			}
			if err := json.Unmarshal([]byte(setting.Value), &cfg); err != nil {
				t.Fatal(err)
			}
			if len(cfg.Route.Rules) != 1 || len(cfg.DNS.Servers) != 1 {
				t.Fatalf("routing/DNS additions lost: %s", setting.Value)
			}
		})
	}
}

func TestImportedVLESSPreservesEmptyFlow(t *testing.T) {
	row := xuiInboundRow{
		ID: 1, Tag: "vless-ws", Protocol: "vless", Enable: true, Port: 443,
		Settings:       json.RawMessage(`{"clients":[{"email":"alice","id":"572bf9c8-610f-4669-af8d-fbfbecc1c476","flow":""}]}`),
		StreamSettings: json.RawMessage(`{"network":"ws","security":"none","wsSettings":{"path":"/proxy"}}`),
	}
	mapped, err := mapInbound(row, 0, nil, "example.com")
	if err != nil {
		t.Fatal(err)
	}
	if len(mapped.ClientRefs) != 1 {
		t.Fatalf("expected one client ref, got %d", len(mapped.ClientRefs))
	}
	agg := newClientAggregate("alice")
	applyProtocolConfig(agg.Config, mapped.ClientRefs[0])
	client, err := agg.toModel()
	if err != nil {
		t.Fatal(err)
	}
	var config map[string]map[string]any
	if err := json.Unmarshal(client.Config, &config); err != nil {
		t.Fatal(err)
	}
	if flow := config["vless"]["flow"]; flow != "" {
		t.Fatalf("source VLESS flow was empty, imported flow = %v", flow)
	}
}
