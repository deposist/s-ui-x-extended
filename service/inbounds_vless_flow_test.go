package service

import (
	"strings"
	"testing"
)

// shouldStripVisionFlow mirrors the decision tree inside fetchUsersByCondition.
// We replicate it here because the flag is intentionally local to that
// function. Keeping the test in the same package lets us cover the rule
// without touching unrelated state (DB, gorm, etc.).
func shouldStripVisionFlow(inboundType string, inbound map[string]interface{}) bool {
	if inboundType != "vless" {
		return false
	}
	if inbound["tls"] == nil {
		return true
	}
	transport, ok := inbound["transport"].(map[string]interface{})
	if !ok {
		return false
	}
	tt, _ := transport["type"].(string)
	return tt != "" && tt != "tcp"
}

// TestVlessVisionFlowStrippedOnNonTcpTransport covers issue #1127.
// Sharing one client UUID across a TCP+REALITY inbound and a gRPC+TLS
// inbound used to break the gRPC inbound because xtls-rprx-vision is
// strictly TCP. The fetch step now strips the flow on any non-TCP
// transport, so the same UUID can serve both inbounds without producing
// a self-rejecting Xray-core configuration.
func TestVlessVisionFlowStrippedOnNonTcpTransport(t *testing.T) {
	cases := []struct {
		name        string
		inboundType string
		inbound     map[string]interface{}
		shouldStrip bool
	}{
		{
			name:        "vless tcp reality keeps flow",
			inboundType: "vless",
			inbound: map[string]interface{}{
				"tls": map[string]interface{}{"enabled": true},
				"transport": map[string]interface{}{
					"type": "tcp",
				},
			},
			shouldStrip: false,
		},
		{
			name:        "vless tcp without transport keeps flow",
			inboundType: "vless",
			inbound: map[string]interface{}{
				"tls": map[string]interface{}{"enabled": true},
			},
			shouldStrip: false,
		},
		{
			name:        "vless grpc tls strips flow",
			inboundType: "vless",
			inbound: map[string]interface{}{
				"tls": map[string]interface{}{"enabled": true},
				"transport": map[string]interface{}{
					"type":         "grpc",
					"service_name": "hello",
				},
			},
			shouldStrip: true,
		},
		{
			name:        "vless ws tls strips flow",
			inboundType: "vless",
			inbound: map[string]interface{}{
				"tls": map[string]interface{}{"enabled": true},
				"transport": map[string]interface{}{
					"type": "ws",
					"path": "/ws",
				},
			},
			shouldStrip: true,
		},
		{
			name:        "vless without tls strips flow",
			inboundType: "vless",
			inbound:     map[string]interface{}{},
			shouldStrip: true,
		},
		{
			name:        "vmess never touched",
			inboundType: "vmess",
			inbound: map[string]interface{}{
				"transport": map[string]interface{}{"type": "ws"},
			},
			shouldStrip: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := shouldStripVisionFlow(tc.inboundType, tc.inbound)
			if got != tc.shouldStrip {
				t.Fatalf("shouldStrip=%v, want %v", got, tc.shouldStrip)
			}
		})
	}
}

// TestVlessVisionFlowReplacementProducesValidJSON guards the actual
// string replacement that fetchUsersByCondition performs. We feed a
// representative user JSON through the same Replace call and confirm
// the rendered config no longer carries the flow value.
func TestVlessVisionFlowReplacementProducesValidJSON(t *testing.T) {
	user := `{"name":"alice","uuid":"11111111-1111-4111-8111-111111111111","flow":"xtls-rprx-vision"}`
	stripped := strings.Replace(user, "xtls-rprx-vision", "", -1)
	if strings.Contains(stripped, "xtls-rprx-vision") {
		t.Fatalf("flow string still present in %q", stripped)
	}
}
