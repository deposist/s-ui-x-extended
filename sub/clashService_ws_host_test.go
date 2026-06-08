package sub

import (
	"testing"

	"gopkg.in/yaml.v3"
)

// TestConvertToClashMetaWSPropagatesHostHeader covers issue #1126:
// external WS nodes whose transport carries an explicit Host header must
// surface the header inside `ws-opts.headers.Host` so Mihomo's WebSocket
// handshake passes through strict CDN / Nginx upstreams. Without the fix
// the header was silently dropped because the previous []interface{}
// cast against a map[string]interface{} value never matched.
func TestConvertToClashMetaWSPropagatesHostHeader(t *testing.T) {
	outbounds := []map[string]interface{}{
		{
			"type":        "vless",
			"tag":         "vless-ws",
			"server":      "1.2.3.4",
			"server_port": 443,
			"uuid":        "11111111-1111-4111-8111-111111111111",
			"tls": map[string]interface{}{
				"enabled":     true,
				"server_name": "bbb.example.com",
			},
			"transport": map[string]interface{}{
				"type": "ws",
				"path": "/yourpath",
				"headers": map[string]interface{}{
					"Host": "bbb.example.com",
				},
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
	proxy, _ := proxies[0].(map[string]interface{})
	wsOpts, ok := proxy["ws-opts"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected ws-opts map, got %#v", proxy["ws-opts"])
	}
	headers, ok := wsOpts["headers"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected ws-opts.headers map, got %#v", wsOpts["headers"])
	}
	if got := headers["Host"]; got != "bbb.example.com" {
		t.Fatalf("expected ws-opts.headers.Host=bbb.example.com, got %#v", got)
	}
}

// TestConvertToClashMetaWSFallsBackToSNIForHost covers the second half of
// issue #1126: subscriptions imported without an explicit Host header (a
// common case when the operator only set SNI) still need a Host so the
// upstream CDN does not reject the request. We backfill from the TLS
// server_name.
func TestConvertToClashMetaWSFallsBackToSNIForHost(t *testing.T) {
	outbounds := []map[string]interface{}{
		{
			"type":        "vless",
			"tag":         "vless-ws",
			"server":      "1.2.3.4",
			"server_port": 443,
			"uuid":        "11111111-1111-4111-8111-111111111111",
			"tls": map[string]interface{}{
				"enabled":     true,
				"server_name": "bbb.example.com",
			},
			"transport": map[string]interface{}{
				"type": "ws",
				"path": "/yourpath",
				// no headers
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
	proxies, _ := config["proxies"].([]interface{})
	proxy, _ := proxies[0].(map[string]interface{})
	wsOpts, ok := proxy["ws-opts"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected ws-opts map, got %#v", proxy["ws-opts"])
	}
	headers, ok := wsOpts["headers"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected ws-opts.headers map populated from SNI, got %#v", wsOpts["headers"])
	}
	if got := headers["Host"]; got != "bbb.example.com" {
		t.Fatalf("expected SNI fallback Host=bbb.example.com, got %#v", got)
	}
}
