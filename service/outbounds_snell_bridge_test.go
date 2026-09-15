package service

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x-extended/core"
	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
)

// The snell and bridge outbounds registered in 1.14 must survive the panel
// save path (Save -> DB -> GetConfig) with every nested field, false/0 value
// and list intact, and the generated config must validate against the core.
func TestSnellBridgeOutboundRoundTrip(t *testing.T) {
	initSettingTestDB(t)
	coreInstance := startTestCore(t)

	recorder := &outboundOpsRecorder{}
	recorder.stubOutboundHooks(t)

	configService := NewConfigServiceWithRuntime(NewRuntime(coreInstance))
	payloads := []json.RawMessage{
		json.RawMessage(`{"type":"snell","tag":"sn1","server":"127.0.0.1","server_port":4444,"version":6,"psk":"topsecret","mode":"unshaped","reuse":true,"network":"udp"}`),
		json.RawMessage(`{"type":"bridge","tag":"br1","interface":"br0","bridge_name":"brname","iproute2_table_index":100,"iproute2_rule_index":0}`),
	}
	for _, p := range payloads {
		if _, err := configService.Save("outbounds", "new", p, "", "admin", "example.com"); err != nil {
			t.Fatalf("save %s: %v", p, err)
		}
	}

	// Persisted rows carry the option payload byte-exact.
	var snellRow model.Outbound
	if err := database.GetDB().Where("tag = ?", "sn1").First(&snellRow).Error; err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(snellRow.Options), `"topsecret"`) {
		t.Fatalf("snell psk lost in DB: %s", snellRow.Options)
	}

	rawConfig, err := configService.GetConfig("")
	if err != nil {
		t.Fatal(err)
	}
	text := string(*rawConfig)
	for _, want := range []string{`"snell"`, `"bridge"`, `"topsecret"`, `"unshaped"`, `"brname"`} {
		if !strings.Contains(text, want) {
			t.Fatalf("generated config lost %s:\n%s", want, text)
		}
	}

	// false/0 survival: bridge iproute2_rule_index=0 must persist (not dropped),
	// and snell-only keys must not leak onto the bridge object.
	var cfg struct {
		Outbounds []map[string]any `json:"outbounds"`
	}
	if err := json.Unmarshal(*rawConfig, &cfg); err != nil {
		t.Fatal(err)
	}
	var bridge map[string]any
	for _, ob := range cfg.Outbounds {
		if ob["type"] == "bridge" {
			bridge = ob
		}
	}
	if bridge == nil {
		t.Fatal("bridge outbound missing")
	}
	if v, ok := bridge["iproute2_rule_index"]; !ok || v != float64(0) {
		t.Fatalf("iproute2_rule_index=0 lost: %v", bridge)
	}
	if v := bridge["reuse"]; v != nil {
		t.Fatalf("bridge should not carry snell reuse: %v", v)
	}

	if err := core.ValidateConfig(*rawConfig); err != nil {
		t.Fatalf("generated config rejected by core: %v", err)
	}
}
