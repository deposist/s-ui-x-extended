package model

import (
	"encoding/json"
	"testing"
)

func TestInboundJSONDropsLegacyRuleActionFields(t *testing.T) {
	payload := []byte(`{
		"id": 7,
		"type": "vless",
		"tag": "vless-in",
		"tls_id": 0,
		"listen": "::",
		"listen_port": 443,
		"sniff": true,
		"sniff_override_destination": true,
		"sniff_timeout": "300ms",
		"domain_strategy": "prefer_ipv4",
		"udp_disable_domain_unmapping": true,
		"proxy_protocol": true
	}`)
	var inbound Inbound
	if err := json.Unmarshal(payload, &inbound); err != nil {
		t.Fatal(err)
	}
	if !hasAnyLegacyInboundOption(t, inbound.Options) {
		t.Fatalf("legacy inbound fields should remain in stored options until the DB migration can move them to route rules: %s", inbound.Options)
	}
	full, err := inbound.MarshalFull()
	if err != nil {
		t.Fatal(err)
	}
	core, err := inbound.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	var coreMap map[string]any
	if err := json.Unmarshal(core, &coreMap); err != nil {
		t.Fatal(err)
	}
	for _, data := range []map[string]any{*full, coreMap} {
		for key := range legacyInboundRuleActionFields {
			if _, ok := data[key]; ok {
				t.Fatalf("legacy key %s leaked into inbound JSON: %#v", key, data)
			}
		}
		if data["proxy_protocol"] != true {
			t.Fatalf("non-legacy option was not preserved: %#v", data)
		}
	}
}

func hasAnyLegacyInboundOption(t *testing.T, raw json.RawMessage) bool {
	t.Helper()
	var options map[string]any
	if err := json.Unmarshal(raw, &options); err != nil {
		t.Fatal(err)
	}
	for key := range legacyInboundRuleActionFields {
		if _, ok := options[key]; ok {
			return true
		}
	}
	return false
}
