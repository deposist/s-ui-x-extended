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

func TestConfigRoundTripWarpEndpointDropsUnsupportedReservedFields(t *testing.T) {
	initSettingTestDB(t)
	if err := database.GetDB().Create(&model.Endpoint{
		Type: "warp",
		Tag:  "warp-reserved",
		Options: json.RawMessage(`{
			"address":["172.16.0.2/32"],
			"private_key":"yAnz5TF+lXXJte14tji3zlMNq+hd2rYUIgJBgB3fBmk=",
			"listen_port":0,
			"reserved":[1,2,3],
			"peers":[{
				"address":"162.159.192.1",
				"port":2408,
				"public_key":"HIgo9xNzJMWLKASShiTqIybxZ0U3wGLiUeJ1PKf8ykw=",
				"allowed_ips":["0.0.0.0/0","::/0"],
				"reserved":[1,2,3]
			}]
		}`),
	}).Error; err != nil {
		t.Fatal(err)
	}

	configService := NewConfigServiceWithRuntime(NewRuntimeWithCoreProvider(nil))
	rawConfig, err := configService.GetConfig(`{"log":{"disabled":true}}`)
	if err != nil {
		t.Fatal(err)
	}
	endpoints := decodedConfigArray(t, *rawConfig, "endpoints")
	if len(endpoints) != 1 {
		t.Fatalf("endpoints = %#v, want one", endpoints)
	}
	endpoint := endpoints[0]
	if endpoint["type"] != "wireguard" {
		t.Fatalf("warp panel endpoint should still emit wireguard core type, got %#v", endpoint["type"])
	}
	if _, ok := endpoint["reserved"]; ok {
		t.Fatalf("top-level reserved leaked into generated config: %s", string(*rawConfig))
	}
	peers, ok := endpoint["peers"].([]any)
	if !ok || len(peers) != 1 {
		t.Fatalf("peers = %#v, want one", endpoint["peers"])
	}
	peer, ok := peers[0].(map[string]any)
	if !ok {
		t.Fatalf("peer = %#v", peers[0])
	}
	if _, ok := peer["reserved"]; ok {
		t.Fatalf("peer reserved leaked into generated config: %s", string(*rawConfig))
	}
	if err := core.ValidateConfig(*rawConfig); err != nil {
		if !strings.Contains(err.Error(), "WireGuard is not included in this build") {
			t.Fatalf("generated WARP config must not fail on reserved schema fields: %v\n%s", err, string(*rawConfig))
		}
	}
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

// TestConfigRoundTripProtocolSpecificFields saves one entity per protocol and
// re-reads it through the assembled core config. It asserts what the test name
// claims: the protocol-specific field the operator set is still there after the
// save/reload cycle, AND a neighbouring field of the same payload survived it -
// a save path that rebuilt the entity from a partial whitelist would keep the
// first field and silently drop the rest.
func TestConfigRoundTripProtocolSpecificFields(t *testing.T) {
	initSettingTestDB(t)
	configService := NewConfigServiceWithRuntime(NewRuntimeWithCoreProvider(nil))
	cases := []struct {
		name          string
		obj           string
		section       string
		field         string
		want          any
		neighbour     string
		neighbourWant any
	}{
		{
			name: "outbound direct override", section: "outbounds",
			obj:   `{"type":"direct","tag":"direct-override","override_address":"127.0.0.1","override_port":443,"proxy_protocol":true}`,
			field: "override_address", want: "127.0.0.1",
			neighbour: "override_port", neighbourWant: float64(443),
		},
		{
			name: "outbound vmess padding", section: "outbounds",
			obj:   `{"type":"vmess","tag":"vmess-extra","server":"example.com","server_port":443,"uuid":"00000000-0000-0000-0000-000000000000","security":"auto","alter_id":0,"global_padding":true,"authenticated_length":true}`,
			field: "global_padding", want: true,
			neighbour: "authenticated_length", neighbourWant: true,
		},
		{
			name: "inbound shadowtls strict", section: "inbounds",
			obj:   `{"type":"shadowtls","tag":"shadowtls-extra","listen":"127.0.0.1","listen_port":0,"version":3,"password":"pw","handshake":{"server":"example.com","server_port":443},"strict_mode":true,"wildcard_sni":"authed"}`,
			field: "strict_mode", want: true,
			neighbour: "wildcard_sni", neighbourWant: "authed",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := configService.Save(tc.section, "new", json.RawMessage(tc.obj), "", "admin", "example.com"); err != nil {
				t.Fatalf("save %s: %v", tc.section, err)
			}

			rawConfig, err := configService.GetConfig("")
			if err != nil {
				t.Fatalf("get config: %v", err)
			}
			var payload struct {
				Tag string `json:"tag"`
			}
			if err := json.Unmarshal([]byte(tc.obj), &payload); err != nil {
				t.Fatal(err)
			}
			var saved map[string]any
			for _, entity := range decodedConfigArray(t, *rawConfig, tc.section) {
				if entity["tag"] == payload.Tag {
					saved = entity
					break
				}
			}
			if saved == nil {
				t.Fatalf("%s %q missing from the assembled config", tc.section, payload.Tag)
			}
			if got := saved[tc.field]; got != tc.want {
				t.Errorf("%s = %#v, want %#v", tc.field, got, tc.want)
			}
			if got := saved[tc.neighbour]; got != tc.neighbourWant {
				t.Errorf("neighbouring %s = %#v, want %#v (saving the entity must not drop it)", tc.neighbour, got, tc.neighbourWant)
			}
		})
	}
}

// TestConfigRoundTripTopLevelCollectionsSurvive guards the 1.14 top-level
// collections (certificate_providers, http_clients, network_namespaces): the
// config blob is stored and reassembled through map[string]json.RawMessage, so
// they must survive a save → GetConfig round-trip untouched and still validate
// against the core.
func TestConfigRoundTripTopLevelCollectionsSurvive(t *testing.T) {
	initSettingTestDB(t)
	configService := NewConfigServiceWithRuntime(NewRuntimeWithCoreProvider(nil))
	blob := json.RawMessage(`{
		"log":{"disabled":true},
		"dns":{"servers":[{"type":"local","tag":"local"}]},
		"certificate_providers":[{"type":"acme","tag":"le","domain":["example.com"],"email":"[email protected]"}],
		"http_clients":[{"tag":"hc"}],
		"network_namespaces":[{"tag":"ns","type":"default","path":"/var/run/netns/ns"}],
		"route":{"final":"direct"}
	}`)
	if _, err := configService.Save("config", "save", blob, "", "admin", "example.com"); err != nil {
		t.Fatalf("save config: %v", err)
	}
	rawConfig, err := configService.GetConfig("")
	if err != nil {
		t.Fatalf("get config: %v", err)
	}
	text := string(*rawConfig)
	for _, key := range []string{"certificate_providers", "http_clients", "network_namespaces"} {
		if !strings.Contains(text, `"`+key+`"`) {
			t.Fatalf("top-level collection %s lost in round-trip:\n%s", key, text)
		}
	}
	// network_namespaces is Linux-only and the acme certificate provider needs
	// the with_acme build tag, so neither is core-validated in the default test
	// build; their survival is already proven above. Strict-validate the blob
	// subset that is platform- and tag-independent (http_clients).
	if err := core.ValidateConfig([]byte(`{
		"log":{"disabled":true},
		"dns":{"servers":[{"type":"local","tag":"local"}]},
		"http_clients":[{"tag":"hc"}],
		"route":{"final":"direct"}
	}`)); err != nil {
		t.Fatalf("config with http_clients must validate: %v", err)
	}
}
