package sub

import (
	"testing"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
	"github.com/deposist/s-ui-x/service"
)

func TestJsonServiceAddFragmentToSupportedOutbounds(t *testing.T) {
	initSubTestDB(t)
	settingService := &service.SettingService{}
	if _, err := settingService.GetAllSetting(); err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Model(model.Setting{}).Where("key = ?", "subJsonFragment").Update("value", `{"enabled":true,"packets":"tlshello"}`).Error; err != nil {
		t.Fatal(err)
	}

	outbounds := []map[string]interface{}{
		{"type": "selector", "tag": "proxy"},
		{"type": "vless", "tag": "vless-out"},
		{"type": "vmess", "tag": "vmess-out"},
		{"type": "trojan", "tag": "trojan-out"},
		{"type": "shadowsocks", "tag": "ss-out"},
	}
	config := map[string]interface{}{
		"outbounds": &outbounds,
	}
	if err := (&JsonService{}).addOthers(&config); err != nil {
		t.Fatal(err)
	}

	for _, outbound := range outbounds {
		_, hasFragment := outbound["fragment"]
		switch outbound["type"] {
		case "vless", "vmess", "trojan":
			if !hasFragment {
				t.Fatalf("%s outbound is missing fragment: %#v", outbound["type"], outbound)
			}
		default:
			if hasFragment {
				t.Fatalf("%s outbound should not receive fragment: %#v", outbound["type"], outbound)
			}
		}
	}
}

func TestJsonServiceAddNoisesToSupportedOutbounds(t *testing.T) {
	initSubTestDB(t)
	settingService := &service.SettingService{}
	if _, err := settingService.GetAllSetting(); err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Model(model.Setting{}).Where("key = ?", "subJsonNoises").Update("value", `[{"type":"rand","packet":"tlshello"}]`).Error; err != nil {
		t.Fatal(err)
	}

	outbounds := []map[string]interface{}{
		{"type": "selector", "tag": "proxy"},
		{"type": "vless", "tag": "vless-out"},
		{"type": "vmess", "tag": "vmess-out"},
		{"type": "trojan", "tag": "trojan-out"},
		{"type": "shadowsocks", "tag": "ss-out"},
	}
	config := map[string]interface{}{
		"outbounds": &outbounds,
	}
	if err := (&JsonService{}).addOthers(&config); err != nil {
		t.Fatal(err)
	}

	for _, outbound := range outbounds {
		_, hasNoises := outbound["noises"]
		switch outbound["type"] {
		case "vless", "vmess", "trojan":
			if !hasNoises {
				t.Fatalf("%s outbound is missing noises: %#v", outbound["type"], outbound)
			}
		default:
			if hasNoises {
				t.Fatalf("%s outbound should not receive noises: %#v", outbound["type"], outbound)
			}
		}
	}
}

func TestJsonServiceMuxToggle(t *testing.T) {
	initSubTestDB(t)
	settingService := &service.SettingService{}
	if _, err := settingService.GetAllSetting(); err != nil {
		t.Fatal(err)
	}

	outbounds := []map[string]interface{}{
		{"type": "vless", "tag": "vless-out"},
		{"type": "shadowsocks", "tag": "ss-out"},
	}
	config := map[string]interface{}{
		"outbounds": &outbounds,
	}
	if err := (&JsonService{}).addOthers(&config); err != nil {
		t.Fatal(err)
	}
	for _, outbound := range outbounds {
		if _, ok := outbound["multiplex"]; ok {
			t.Fatalf("mux should be absent when subJsonMux=false: %#v", outbound)
		}
	}

	if err := database.GetDB().Model(model.Setting{}).Where("key = ?", "subJsonMux").Update("value", "true").Error; err != nil {
		t.Fatal(err)
	}
	outbounds = []map[string]interface{}{
		{"type": "selector", "tag": "proxy"},
		{"type": "vless", "tag": "vless-out"},
		{"type": "vmess", "tag": "vmess-out"},
		{"type": "trojan", "tag": "trojan-out"},
		{"type": "shadowsocks", "tag": "ss-out"},
		{"type": "hysteria2", "tag": "hy2-out"},
	}
	config = map[string]interface{}{
		"outbounds": &outbounds,
	}
	if err := (&JsonService{}).addOthers(&config); err != nil {
		t.Fatal(err)
	}

	for _, outbound := range outbounds {
		mux, hasMux := outbound["multiplex"]
		switch outbound["type"] {
		case "vless", "vmess", "trojan", "shadowsocks":
			if !hasMux {
				t.Fatalf("%s outbound is missing mux: %#v", outbound["type"], outbound)
			}
			muxMap, ok := mux.(map[string]interface{})
			if !ok || muxMap["enabled"] != true || muxMap["protocol"] != "smux" {
				t.Fatalf("unexpected mux settings for %s: %#v", outbound["type"], mux)
			}
		default:
			if hasMux {
				t.Fatalf("%s outbound should not receive mux: %#v", outbound["type"], outbound)
			}
		}
	}
}

func TestJsonServiceDirectRulesToggle(t *testing.T) {
	initSubTestDB(t)
	settingService := &service.SettingService{}
	if _, err := settingService.GetAllSetting(); err != nil {
		t.Fatal(err)
	}

	config := map[string]interface{}{}
	if err := (&JsonService{}).addOthers(&config); err != nil {
		t.Fatal(err)
	}
	route := config["route"].(map[string]interface{})
	if _, ok := route["rule_set"]; ok {
		t.Fatalf("direct rule_sets should be absent when subJsonDirectRules=false: %#v", route)
	}

	if err := database.GetDB().Model(model.Setting{}).Where("key = ?", "subJsonDirectRules").Update("value", "true").Error; err != nil {
		t.Fatal(err)
	}
	config = map[string]interface{}{}
	if err := (&JsonService{}).addOthers(&config); err != nil {
		t.Fatal(err)
	}
	route = config["route"].(map[string]interface{})
	rules := route["rules"].([]interface{})
	if len(rules) < 2 {
		t.Fatalf("expected direct rule after sniff rule: %#v", rules)
	}
	directRule := rules[1].(map[string]interface{})
	if directRule["outbound"] != "direct" || directRule["action"] != "route" {
		t.Fatalf("unexpected direct rule: %#v", directRule)
	}
	ruleSetTags, ok := directRule["rule_set"].([]string)
	if !ok || len(ruleSetTags) != 2 || ruleSetTags[0] != "geosite-private" || ruleSetTags[1] != "geoip-private" {
		t.Fatalf("unexpected direct rule sets: %#v", directRule["rule_set"])
	}
	ruleSets := route["rule_set"].([]interface{})
	if !hasRuleSetTag(ruleSets, "geosite-private") || !hasRuleSetTag(ruleSets, "geoip-private") {
		t.Fatalf("private rule_sets missing: %#v", ruleSets)
	}
}

func hasRuleSetTag(ruleSets []interface{}, tag string) bool {
	for _, ruleSet := range ruleSets {
		if got, ok := ruleSetTag(ruleSet); ok && got == tag {
			return true
		}
	}
	return false
}
