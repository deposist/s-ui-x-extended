package sub

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestConvertToClashMetaTUICKeepsUDPRelayMode(t *testing.T) {
	outbounds := []map[string]interface{}{
		{
			"type":               "tuic",
			"tag":                "tuic-test",
			"server":             "example.com",
			"server_port":        443,
			"uuid":               "11111111-1111-4111-8111-111111111111",
			"password":           "secret",
			"congestion_control": "bbr",
			"udp_relay_mode":     "quic",
			"tls": map[string]interface{}{
				"enabled": true,
			},
		},
	}

	got, err := (&ClashService{}).ConvertToClashMeta(&outbounds, basicClashConfig)
	if err != nil {
		t.Fatal(err)
	}

	var config map[string]interface{}
	if err := yaml.Unmarshal([]byte(got), &config); err != nil {
		t.Fatal(err)
	}
	proxies, ok := config["proxies"].([]interface{})
	if !ok || len(proxies) != 1 {
		t.Fatalf("expected one proxy, got %#v", config["proxies"])
	}
	proxy, ok := proxies[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected proxy map, got %#v", proxies[0])
	}

	if got := proxy["udp-relay-mode"]; got != "quic" {
		t.Fatalf("expected udp-relay-mode=quic, got %#v", got)
	}
	if got := proxy["congestion-controller"]; got != "bbr" {
		t.Fatalf("expected congestion-controller=bbr, got %#v", got)
	}
}
