//go:build with_naive_outbound

package core

import (
	"context"
	"testing"

	"github.com/sagernet/sing-box/option"
)

func TestSingBox113RepresentativeConfigUnmarshals(t *testing.T) {
	ctx := Context(context.Background(), InboundRegistry(), OutboundRegistry(), EndpointRegistry(), DNSTransportRegistry(), ServiceRegistry())
	config := []byte(`{
  "dns": {
    "servers": [
      { "type": "local", "tag": "local" }
    ],
    "rules": [
      {
        "query_type": ["A", "AAAA"],
        "network": ["tcp"],
        "network_type": ["wifi"],
        "network_is_expensive": true,
        "network_is_constrained": true,
        "wifi_ssid": ["office"],
        "wifi_bssid": ["00:11:22:33:44:55"],
        "interface_address": { "eth0": ["192.0.2.0/24"] },
        "network_interface_address": { "wifi": ["2001:db8::/32"] },
        "default_interface_address": ["198.51.100.0/24"],
        "action": "route",
        "server": "local"
      }
    ]
  },
  "route": {
    "rules": [
      {
        "network": "icmp",
        "network_type": ["wifi"],
        "wifi_ssid": ["office"],
        "interface_address": { "eth0": ["192.0.2.0/24"] },
        "default_interface_address": ["198.51.100.0/24"],
        "action": "reject",
        "method": "reply"
      },
      {
        "domain": ["example.com"],
        "action": "bypass",
        "outbound": "direct",
        "override_address": "example.org",
        "override_port": 443,
        "network_strategy": "fallback",
        "fallback_delay": 300
      }
    ],
    "rule_set": [
      {
        "type": "inline",
        "tag": "inline-headless",
        "rules": [
          {
            "query_type": ["A"],
            "network_type": ["wifi"],
            "wifi_ssid": ["office"],
            "network_interface_address": { "wifi": ["10.0.0.0/8"] },
            "default_interface_address": ["172.16.0.0/12"]
          }
        ]
      }
    ],
    "final": "direct"
  },
  "inbounds": [
    {
      "type": "tun",
      "tag": "tun-in",
      "address": ["172.19.0.1/30"],
      "auto_route": true,
      "auto_redirect": true,
      "auto_redirect_reset_mark": "0x2024",
      "auto_redirect_nfqueue": 100
    }
  ],
  "outbounds": [
    { "type": "direct", "tag": "direct" },
    {
      "type": "naive",
      "tag": "naive",
      "server": "example.com",
      "server_port": 443,
      "username": "u",
      "password": "p",
      "stream_receive_window": "8 MB",
      "udp_over_tcp": { "enabled": true, "version": 2 },
      "quic": true,
      "quic_session_receive_window": "16 MB",
      "tls": {
        "enabled": true,
        "server_name": "example.com",
        "certificate_public_key_sha256": ["YWJjZA=="],
        "client_certificate_path": "client.crt",
        "client_key_path": "client.key",
        "kernel_tx": true,
        "kernel_rx": true
      }
    }
  ],
  "endpoints": [
    {
      "type": "tailscale",
      "tag": "ts",
      "state_directory": "tailscale-state",
      "advertise_tags": ["tag:dev"]
    }
  ],
  "services": [
    {
      "type": "oom-killer",
      "tag": "oom",
      "memory_limit": "512 MB",
      "safety_margin": "64 MB",
      "min_interval": "1s",
      "max_interval": "5s",
      "checks_before_limit": 2
    },
    {
      "type": "ocm",
      "tag": "ocm",
      "listen": "127.0.0.1",
      "listen_port": 8080,
      "headers": { "X-Test": ["a", "b"] },
      "users": [{ "name": "u", "token": "t" }]
    },
    {
      "type": "ccm",
      "tag": "ccm",
      "listen": "127.0.0.1",
      "listen_port": 8081,
      "headers": { "X-Test": "a" },
      "users": [{ "name": "u", "token": "t" }]
    }
  ]
}`)

	var options option.Options
	if err := options.UnmarshalJSONContext(ctx, config); err != nil {
		t.Fatalf("unmarshal representative sing-box 1.13 config: %v", err)
	}
}
