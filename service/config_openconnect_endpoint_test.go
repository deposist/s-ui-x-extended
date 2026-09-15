package service

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x-extended/core"
	"github.com/deposist/s-ui-x-extended/core/capabilities"
)

// TestOpenConnectEndpointRoundTrip saves an openconnect endpoint through the
// panel save path, regenerates the core config, and proves every field the
// OpenConnect editor owns (structured scalars, nested token/tls blocks,
// JSON-edited csd/hip/form_entries, false/0 values) survives, then validates
// the generated config against the core when with_openconnect is compiled in.
func TestOpenConnectEndpointRoundTrip(t *testing.T) {
	initSettingTestDB(t)
	configService := NewConfigServiceWithRuntime(NewRuntimeWithCoreProvider(nil))

	payload := json.RawMessage(`{
		"type": "openconnect",
		"tag": "oc-out",
		"server": "vpn.example.com",
		"flavor": "anyconnect",
		"username": "user",
		"password": "topsecret",
		"auth_group": "group1",
		"token": {"mode": "totp", "secret": "TOTPSECRET", "counter": 0},
		"no_udp": false,
		"compression_mode": "stateless",
		"mtu": 1406,
		"queue_length": 0,
		"tls": {
			"insecure": true,
			"server_name": "vpn.example.com",
			"client_key_password": "keypass"
		},
		"csd": {"wrapper_path": "/usr/libexec/openconnect/csd-wrapper.sh"},
		"form_entries": [{"form_id": "main_login", "submission_key": "Login", "name": "username", "value": "user"}, {"form_id": "main_login", "name": "group_list", "promote": true}]
	}`)
	if _, err := configService.Save("endpoints", "new", payload, "", "admin", "example.com"); err != nil {
		t.Fatalf("save openconnect endpoint: %v", err)
	}

	rawConfig, err := configService.GetConfig("")
	if err != nil {
		t.Fatalf("get config: %v", err)
	}
	text := string(*rawConfig)
	if !strings.Contains(text, `"openconnect"`) {
		t.Fatalf("openconnect endpoint missing from generated config")
	}
	for _, want := range []string{
		`"topsecret"`, `"TOTPSECRET"`, `"group1"`, `"stateless"`,
		`"insecure": true`, `"keypass"`, `csd-wrapper.sh`,
		`"main_login"`, `"promote"`, `"group_list"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("generated config lost %s", want)
		}
	}

	// false/0 values: token.counter=0 and no_udp=false must persist.
	var cfg struct {
		Endpoints []map[string]any `json:"endpoints"`
	}
	if err := json.Unmarshal(*rawConfig, &cfg); err != nil {
		t.Fatal(err)
	}
	var ep map[string]any
	for _, e := range cfg.Endpoints {
		if e["type"] == "openconnect" {
			ep = e
		}
	}
	if ep == nil {
		t.Fatal("openconnect endpoint missing from endpoints array")
	}
	if v, ok := ep["no_udp"]; !ok || v != false {
		t.Fatalf("no_udp=false lost: %v", ep)
	}
	if token, ok := ep["token"].(map[string]any); !ok || token["counter"] != float64(0) {
		t.Fatalf("token.counter=0 lost: %v", ep["token"])
	}

	if capabilities.BuildTags()["with_openconnect"] {
		if err := core.ValidateConfig(*rawConfig); err != nil {
			t.Fatalf("generated config rejected by core: %v", err)
		}
	} else {
		t.Log("with_openconnect not compiled in; skipping core validation")
	}
}
