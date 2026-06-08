package util

import (
	"strings"
	"testing"
)

// TestVlessLinkOmitsFlowOnNonTcpTransport covers the link-generator side
// of issue #1127. A vless link served from an inbound whose transport is
// grpc/ws/http must not advertise xtls-rprx-vision because Xray-core
// rejects the flow on those transports. The generator now suppresses the
// flow parameter unless the transport is TCP.
func TestVlessLinkOmitsFlowOnNonTcpTransport(t *testing.T) {
	addrs := []map[string]interface{}{
		{
			"server":      "1.2.3.4",
			"server_port": float64(443),
			"remark":      "node",
			"tls": map[string]interface{}{
				"enabled":     true,
				"server_name": "example.com",
			},
		},
	}

	cases := []struct {
		name        string
		transport   map[string]interface{}
		wantFlow    bool
		wantNetwork string
	}{
		{
			name:        "tcp keeps flow",
			transport:   nil,
			wantFlow:    true,
			wantNetwork: "tcp",
		},
		{
			name:        "grpc drops flow",
			transport:   map[string]interface{}{"type": "grpc", "service_name": "hello"},
			wantFlow:    false,
			wantNetwork: "grpc",
		},
		{
			name:        "ws drops flow",
			transport:   map[string]interface{}{"type": "ws", "path": "/ws"},
			wantFlow:    false,
			wantNetwork: "ws",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			inbound := map[string]interface{}{}
			if tc.transport != nil {
				inbound["transport"] = tc.transport
			}
			user := map[string]interface{}{
				"uuid": "11111111-1111-4111-8111-111111111111",
				"flow": "xtls-rprx-vision",
			}
			links := vlessLink(user, inbound, addrs)
			if len(links) != 1 {
				t.Fatalf("expected 1 link, got %d", len(links))
			}
			link := links[0]
			if !strings.Contains(link, "type="+tc.wantNetwork) {
				t.Fatalf("link %q missing type=%s", link, tc.wantNetwork)
			}
			hasFlow := strings.Contains(link, "flow=xtls-rprx-vision")
			if hasFlow != tc.wantFlow {
				t.Fatalf("link %q wantFlow=%v gotFlow=%v", link, tc.wantFlow, hasFlow)
			}
		})
	}
}
