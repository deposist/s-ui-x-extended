package service

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x-extended/core"
	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
)

func decodedConfigArray(t *testing.T, raw []byte, key string) []map[string]any {
	t.Helper()
	var cfg map[string]json.RawMessage
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatal(err)
	}
	var items []map[string]any
	if err := json.Unmarshal(cfg[key], &items); err != nil {
		t.Fatalf("decode %s: %v\n%s", key, err, string(cfg[key]))
	}
	return items
}

func TestConfigRoundTripVLESSInboundLegacyFieldsMigrateToRouteRules(t *testing.T) {
	initSettingTestDB(t)
	configService := NewConfigServiceWithRuntime(NewRuntimeWithCoreProvider(nil))
	payload := json.RawMessage(`{"type":"vless","tag":"vless-advanced","listen":"127.0.0.1","listen_port":0,"sniff":true,"sniff_timeout":"1s","sniff_override_destination":true,"domain_strategy":"prefer_ipv4","udp_disable_domain_unmapping":true,"users":[]}`)
	if _, err := configService.Save("inbounds", "new", payload, "", "admin", "example.com"); err != nil {
		t.Fatalf("save vless inbound: %v", err)
	}

	var row model.Inbound
	if err := database.GetDB().Where("tag = ?", "vless-advanced").First(&row).Error; err != nil {
		t.Fatal(err)
	}
	full, err := row.MarshalFull()
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"sniff", "sniff_timeout", "sniff_override_destination", "domain_strategy", "udp_disable_domain_unmapping"} {
		if _, ok := (*full)[key]; ok {
			t.Fatalf("legacy inbound key %s leaked into MarshalFull: %#v", key, *full)
		}
	}

	rawConfig, err := configService.GetConfig("")
	if err != nil {
		t.Fatal(err)
	}
	var cfg struct {
		Route struct {
			Rules []map[string]any `json:"rules"`
		} `json:"route"`
	}
	if err := json.Unmarshal(*rawConfig, &cfg); err != nil {
		t.Fatal(err)
	}
	if !hasRouteRuleForInbound(cfg.Route.Rules, "resolve", "vless-advanced", "strategy", "prefer_ipv4") {
		t.Fatalf("resolve rule was not migrated: %#v", cfg.Route.Rules)
	}
	if !hasRouteRuleForInbound(cfg.Route.Rules, "sniff", "vless-advanced", "timeout", "1s") {
		t.Fatalf("sniff rule was not migrated: %#v", cfg.Route.Rules)
	}
	if !hasRouteRuleForInbound(cfg.Route.Rules, "route-options", "vless-advanced", "udp_disable_domain_unmapping", true) {
		t.Fatalf("route-options rule was not migrated: %#v", cfg.Route.Rules)
	}
}

func hasRouteRuleForInbound(rules []map[string]any, action string, inbound string, key string, value any) bool {
	for _, rule := range rules {
		if rule["action"] != action || rule[key] != value {
			continue
		}
		switch inbounds := rule["inbound"].(type) {
		case string:
			if inbounds == inbound {
				return true
			}
		case []any:
			for _, item := range inbounds {
				if item == inbound {
					return true
				}
			}
		}
	}
	return false
}

func TestConfigRoundTripNewNativeTypesProduceCoreConfig(t *testing.T) {
	initSettingTestDB(t)
	db := database.GetDB()
	if err := db.Create(&model.Outbound{Type: "direct", Tag: "plain-direct", Options: json.RawMessage(`{}`)}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.Outbound{Type: "block", Tag: "block-core", Options: json.RawMessage(`{}`)}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.Outbound{Type: CoreFailoverType, Tag: "native-fo", Options: json.RawMessage(`{"outbounds":["plain-direct"],"strategy":"sequential","delay":"1s"}`)}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.Inbound{Type: "direct", Tag: "member-in", Options: json.RawMessage(`{"listen":"127.0.0.1","listen_port":0}`)}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.Inbound{Type: "bond", Tag: "bond-core", Options: json.RawMessage(`{"inbounds":["member-in"]}`)}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.Inbound{Type: CoreFailoverType, Tag: "failover-in-core", Options: json.RawMessage(`{"inbounds":["member-in"]}`)}).Error; err != nil {
		t.Fatal(err)
	}

	configService := NewConfigServiceWithRuntime(NewRuntimeWithCoreProvider(nil))
	rawConfig, err := configService.GetConfig(`{"log":{"disabled":true},"route":{"final":"plain-direct"}}`)
	if err != nil {
		t.Fatalf("get config: %v", err)
	}
	if strings.Contains(string(*rawConfig), `"type":"core-failover"`) {
		t.Fatalf("core config leaked panel alias core-failover: %s", string(*rawConfig))
	}
	outbounds := decodedConfigArray(t, *rawConfig, "outbounds")
	var foundNativeFailover bool
	for _, outbound := range outbounds {
		if outbound["tag"] == "native-fo" {
			foundNativeFailover = outbound["type"] == "failover"
			members, ok := outbound["outbounds"].([]any)
			if !ok || len(members) != 1 {
				t.Fatalf("native failover outbounds = %#v, want one nested outbound", outbound["outbounds"])
			}
			member, ok := members[0].(map[string]any)
			if !ok || member["type"] != "direct" {
				t.Fatalf("nested outbound = %#v, want direct", members[0])
			}
		}
	}
	if !foundNativeFailover {
		t.Fatalf("native failover outbound not assembled as type failover: %#v", outbounds)
	}

	inbounds := decodedConfigArray(t, *rawConfig, "inbounds")
	for _, inbound := range inbounds {
		if inbound["tag"] == "member-in" {
			t.Fatalf("member inbound must be nested, not duplicated as top-level: %#v", inbounds)
		}
	}
	if err := core.ValidateConfig(*rawConfig); err != nil {
		t.Fatalf("generated config must validate: %v\n%s", err, string(*rawConfig))
	}
}

func TestConfigRoundTripProtocolSpecificFields(t *testing.T) {
	initSettingTestDB(t)
	configService := NewConfigServiceWithRuntime(NewRuntimeWithCoreProvider(nil))
	cases := []struct {
		name   string
		obj    string
		field  string
		want   any
		entity string
	}{
		{"outbound direct override", `{"type":"direct","tag":"direct-override","override_address":"127.0.0.1","override_port":443,"proxy_protocol":true}`, "override_address", "127.0.0.1", "outbounds"},
		{"outbound vmess padding", `{"type":"vmess","tag":"vmess-extra","server":"example.com","server_port":443,"uuid":"00000000-0000-0000-0000-000000000000","security":"auto","alter_id":0,"global_padding":true,"authenticated_length":true}`, "global_padding", true, "outbounds"},
		{"inbound shadowtls strict", `{"type":"shadowtls","tag":"shadowtls-extra","listen":"127.0.0.1","listen_port":0,"version":3,"password":"pw","handshake":{"server":"example.com","server_port":443},"strict_mode":true,"wildcard_sni":"authed"}`, "strict_mode", true, "inbounds"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := configService.Save(tc.entity, "new", json.RawMessage(tc.obj), "", "admin", "example.com"); err != nil {
				t.Fatalf("save %s: %v", tc.entity, err)
			}
		})
	}
}
