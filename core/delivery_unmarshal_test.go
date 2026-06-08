//go:build with_sudoku && with_trusttunnel

package core

import (
	"context"
	"testing"

	"github.com/sagernet/sing-box/option"
)

// TestExtendedDeliveryOutboundsUnmarshal proves the client out_json field names the
// Phase 1 builders emit (util/outJson.go: mieruOut/sshOut/sudokuOut/trustTunnelOut)
// are valid against the EXTENDED sing-box option structs — the gold-standard check
// for the main risk that fork field names differ from upstream. It unmarshals a
// config carrying one outbound per delivered extended protocol through the real
// registries. Tag-gated (sudoku/trusttunnel need their build tags); runs in the
// tagged CI build, not the bare local run.
func TestExtendedDeliveryOutboundsUnmarshal(t *testing.T) {
	ctx := Context(context.Background(), InboundRegistry(), OutboundRegistry(), EndpointRegistry(), ProviderRegistry(), DNSTransportRegistry(), ServiceRegistry())
	config := []byte(`{
  "outbounds": [
    {
      "type": "mieru", "tag": "mieru-out",
      "server": "example.com", "server_port": 443,
      "username": "alice", "password": "pw",
      "transport": "TCP", "traffic_pattern": "tls",
      "server_ports": ["2090:2099"]
    },
    {
      "type": "ssh", "tag": "ssh-out",
      "server": "example.com", "server_port": 22,
      "user": "carol", "password": "pw"
    },
    {
      "type": "sudoku", "tag": "sudoku-out",
      "server": "example.com", "server_port": 443,
      "key": "SHARED-KEY", "aead_method": "chacha20-poly1305",
      "table_type": "prefer_ascii", "padding_min": 10, "padding_max": 30,
      "enable_pure_downlink": true,
      "http_mask": { "enabled": true, "mode": "stream", "host": "cdn.example.com", "path_root": "/cdn", "multiplex": "auto" }
    },
    {
      "type": "trusttunnel", "tag": "trusttunnel-out",
      "server": "example.com", "server_port": 443,
      "username": "bob", "password": "pw",
      "network": ["tcp", "udp"], "quic": true,
      "congestion_controller": "bbr", "bbr_profile": "standard", "cwnd": 32
    }
  ]
}`)

	var options option.Options
	if err := options.UnmarshalJSONContext(ctx, config); err != nil {
		t.Fatalf("extended delivery outbounds failed to unmarshal through the fork registries: %v", err)
	}
}
