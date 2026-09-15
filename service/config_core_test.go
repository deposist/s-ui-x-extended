package service

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestConfigCoreMethodsHandleNilCore(t *testing.T) {
	t.Cleanup(ReplaceDefaultRuntimeForTest(NewRuntimeWithCoreProvider(nil)))

	configService := &ConfigService{}
	if configService.IsCoreRunning() {
		t.Fatal("nil core should not report running")
	}

	tests := map[string]func() error{
		"StartCore":   configService.StartCore,
		"RestartCore": configService.RestartCore,
		"StopCore":    configService.StopCore,
	}
	for name, call := range tests {
		err := call()
		if err == nil || !strings.Contains(err.Error(), "core not initialized") {
			t.Fatalf("%s returned %v, want core not initialized", name, err)
		}
	}
}

func TestGetConfigPreservesTopLevelCertificateAndUnknownFields(t *testing.T) {
	initSettingTestDB(t)

	input := `{
  "log": { "level": "info" },
  "dns": { "servers": [], "rules": [] },
  "route": { "rules": [] },
  "experimental": {},
  "certificate": {
    "store": "mozilla",
    "certificate_path": ["/etc/ssl/custom.pem"]
  },
  "future_top_level": { "enabled": true }
}`

	rawConfig, err := (&ConfigService{}).GetConfig(input)
	if err != nil {
		t.Fatal(err)
	}

	var config map[string]json.RawMessage
	if err := json.Unmarshal(*rawConfig, &config); err != nil {
		t.Fatal(err)
	}
	if _, ok := config["certificate"]; !ok {
		t.Fatalf("certificate was not preserved in runtime config: %s", string(*rawConfig))
	}
	if _, ok := config["future_top_level"]; !ok {
		t.Fatalf("unknown top-level field was not preserved in runtime config: %s", string(*rawConfig))
	}
}

// The advanced config-blob collections edited from the Basics panel must reach
// the generated core config unchanged: GetConfig only overwrites
// inbounds/outbounds/services/endpoints/providers, so these must pass through.
func TestGetConfigPreservesConfigBlobCollections(t *testing.T) {
	initSettingTestDB(t)

	input := `{
  "log": { "level": "info" },
  "dns": { "servers": [], "rules": [] },
  "route": { "rules": [] },
  "experimental": {},
  "certificate_providers": [
    { "type": "acme", "tag": "cp1", "domain": ["example.com"], "email": "a@example.com", "disable_http_challenge": false }
  ],
  "http_clients": [
    { "tag": "hc1", "engine": "go", "version": 2, "disable_version_fallback": false }
  ],
  "network_namespaces": [
    { "type": "unshare", "tag": "ns1", "pid_file": "/run/ns1.pid" }
  ]
}`

	rawConfig, err := (&ConfigService{}).GetConfig(input)
	if err != nil {
		t.Fatal(err)
	}

	var config map[string]json.RawMessage
	if err := json.Unmarshal(*rawConfig, &config); err != nil {
		t.Fatal(err)
	}

	for _, key := range []string{"certificate_providers", "http_clients", "network_namespaces"} {
		var items []map[string]json.RawMessage
		if err := json.Unmarshal(config[key], &items); err != nil {
			t.Fatalf("collection %s did not survive GetConfig: %v", key, err)
		}
		if len(items) != 1 {
			t.Fatalf("collection %s: got %d items, want 1", key, len(items))
		}
	}

	// false/0 values and nested blocks must not be dropped by the passthrough.
	var cp []map[string]any
	if err := json.Unmarshal(config["certificate_providers"], &cp); err != nil {
		t.Fatal(err)
	}
	if v, ok := cp[0]["disable_http_challenge"]; !ok || v != false {
		t.Fatalf("disable_http_challenge=false lost: %v", cp[0])
	}
	var hc []map[string]any
	if err := json.Unmarshal(config["http_clients"], &hc); err != nil {
		t.Fatal(err)
	}
	if v, ok := hc[0]["disable_version_fallback"]; !ok || v != false {
		t.Fatalf("disable_version_fallback=false lost: %v", hc[0])
	}
}
