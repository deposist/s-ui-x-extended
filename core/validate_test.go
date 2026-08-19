package core

import (
	"net"
	"strings"
	"testing"
	"time"
)

func TestValidateConfigRejectsMalformedConfig(t *testing.T) {
	if err := ValidateConfig([]byte("{ this is not json")); err == nil {
		t.Fatal("ValidateConfig must reject malformed config")
	}
}

func TestValidateConfigAcceptsMinimalConfig(t *testing.T) {
	config := []byte(`{"log":{"disabled":true},"dns":{"servers":[],"rules":[]},"route":{"rules":[]}}`)
	if err := ValidateConfig(config); err != nil {
		t.Fatalf("ValidateConfig rejected minimal config: %v", err)
	}
}

func TestValidateInboundJSONAcceptsMinimalMixed(t *testing.T) {
	config := []byte(`{"type":"mixed","tag":"in","listen":"127.0.0.1","listen_port":1}`)
	if err := ValidateInboundJSON(config); err != nil {
		t.Fatalf("ValidateInboundJSON rejected minimal mixed inbound: %v", err)
	}
}

func TestValidateInboundJSONAcceptsAllPanelInboundFixtures(t *testing.T) {
	tests := []struct {
		name   string
		config string
	}{
		{name: "socks", config: `{"type":"socks","tag":"panel","listen":"127.0.0.1","listen_port":1,"users":[]}`},
		{name: "http", config: `{"type":"http","tag":"panel","listen":"127.0.0.1","listen_port":1,"users":[]}`},
		{name: "mixed", config: `{"type":"mixed","tag":"panel","listen":"127.0.0.1","listen_port":1,"users":[]}`},
		{name: "shadowsocks", config: `{"type":"shadowsocks","tag":"panel","listen":"127.0.0.1","listen_port":1,"method":"none","users":[]}`},
		{name: "vmess", config: `{"type":"vmess","tag":"panel","listen":"127.0.0.1","listen_port":1,"users":[]}`},
		{name: "vless", config: `{"type":"vless","tag":"panel","listen":"127.0.0.1","listen_port":1,"users":[],"decryption":"none"}`},
		{name: "trojan", config: `{"type":"trojan","tag":"panel","listen":"127.0.0.1","listen_port":1,"users":[]}`},
		{name: "naive", config: `{"type":"naive","tag":"panel","listen":"127.0.0.1","listen_port":1,"users":[],"network":["tcp"],"quic_congestion_control":"bbr"}`},
		{name: "hysteria", config: `{"type":"hysteria","tag":"panel","listen":"127.0.0.1","listen_port":1,"users":[],"up_mbps":100,"down_mbps":100}`},
		{name: "hysteria2", config: `{"type":"hysteria2","tag":"panel","listen":"127.0.0.1","listen_port":1,"users":[],"up_mbps":100,"down_mbps":100}`},
		{name: "tuic", config: `{"type":"tuic","tag":"panel","listen":"127.0.0.1","listen_port":1,"users":[],"congestion_control":"cubic"}`},
		{name: "anytls", config: `{"type":"anytls","tag":"panel","listen":"127.0.0.1","listen_port":1,"users":[],"padding_scheme":["stop=8"]}`},
		{name: "shadowtls", config: `{"type":"shadowtls","tag":"panel","listen":"127.0.0.1","listen_port":1,"version":3,"users":[]}`},
		{name: "mieru", config: `{"type":"mieru","tag":"panel","listen":"127.0.0.1","listen_port":1,"transport":"TCP","users":[]}`},
		{name: "sudoku", config: `{"type":"sudoku","tag":"panel","listen":"127.0.0.1","listen_port":1,"key":"public","aead_method":"chacha20-poly1305","padding_min":10,"padding_max":30,"enable_pure_downlink":true,"disable_http_mask":false,"http_mask_mode":"auto","path_root":"/mask","fallback":"https://example.com"}`},
		{name: "trusttunnel", config: `{"type":"trusttunnel","tag":"panel","listen":"127.0.0.1","listen_port":1,"users":[],"network":["tcp","udp"],"congestion_controller":"bbr","cwnd":32}`},
		{name: "ssh", config: `{"type":"ssh","tag":"panel","listen":"127.0.0.1","listen_port":1,"users":[]}`},
		{name: "mtproxy", config: `{"type":"mtproxy","tag":"panel","listen":"127.0.0.1","listen_port":1,"users":[]}`},
		{name: "direct", config: `{"type":"direct","tag":"panel","listen":"127.0.0.1","listen_port":1}`},
		{name: "call", config: `{"type":"call","tag":"panel","platform":"dion","cookies":[]}`},
		{name: "tun", config: `{"type":"tun","tag":"panel","address":["172.19.0.1/30"],"auto_route":true}`},
		{name: "redirect", config: `{"type":"redirect","tag":"panel","listen":"127.0.0.1","listen_port":1}`},
		{name: "tproxy", config: `{"type":"tproxy","tag":"panel","listen":"127.0.0.1","listen_port":1,"network":"tcp"}`},
		{name: "bond", config: `{"type":"bond","tag":"panel","inbounds":[{"type":"mixed","tag":"nested","listen":"127.0.0.1","listen_port":1}]}`},
		{name: "core-failover", config: `{"type":"failover","tag":"panel","inbounds":[{"type":"mixed","tag":"nested","listen":"127.0.0.1","listen_port":1}]}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateInboundJSON([]byte(tt.config)); err != nil {
				t.Fatalf("ValidateInboundJSON rejected %s fixture: %v", tt.name, err)
			}
		})
	}
}

func TestValidateInboundJSONRejectsKnownDirectionLeaks(t *testing.T) {
	tests := []struct {
		name   string
		config string
		field  string
	}{
		{name: "trojan-network", config: `{"type":"trojan","tag":"panel","listen":"127.0.0.1","listen_port":1,"network":"tcp","users":[]}`, field: "network"},
		// Pinned badjson elides unknown empty objects before strict decoding; a populated
		// outbound-shaped http_mask still exercises the strict schema rejection.
		{name: "sudoku-http-mask", config: `{"type":"sudoku","tag":"panel","listen":"127.0.0.1","listen_port":1,"http_mask":{"enabled":true}}`, field: "http_mask"},
		{name: "trusttunnel-quic", config: `{"type":"trusttunnel","tag":"panel","listen":"127.0.0.1","listen_port":1,"quic":true,"users":[]}`, field: "quic"},
		{name: "call-listen", config: `{"type":"call","tag":"panel","platform":"dion","cookies":[],"listen":"127.0.0.1","listen_port":1}`, field: "listen"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateInboundJSON([]byte(tt.config))
			if err == nil {
				t.Fatalf("ValidateInboundJSON accepted direction leak %s", tt.field)
			}
			if !strings.Contains(err.Error(), tt.field) {
				t.Fatalf("ValidateInboundJSON error = %v, want field %q", err, tt.field)
			}
		})
	}
}

func TestValidateInboundJSONRejectsUnknownField(t *testing.T) {
	config := []byte(`{"type":"mixed","tag":"in","listen":"127.0.0.1","listen_port":1,"bogus_top_level":true}`)
	if err := ValidateInboundJSON(config); err == nil {
		t.Fatal("ValidateInboundJSON accepted unknown inbound field")
	} else if !strings.Contains(err.Error(), "unknown field") || !strings.Contains(err.Error(), "bogus_top_level") {
		t.Fatalf("ValidateInboundJSON error = %v, want unknown field bogus_top_level", err)
	}
}

// TestTrustTunnelInboundUnmarshal is the gold-standard regression for the
// inbound core-start crash (`json: unknown field "quic"`): the SANITIZED
// trusttunnel inbound config that addUsers emits (outbound-only fields stripped,
// network/congestion_controller/cwnd kept) passes through the same strict parser
// used by AddInbound, while the otherwise identical UNSANITIZED variant with
// top-level `quic` fails and names that rejected field.
func TestTrustTunnelInboundUnmarshal(t *testing.T) {
	sanitized := []byte(`{"type":"trusttunnel","tag":"trusttunnel-in","listen":"0.0.0.0","listen_port":443,"network":["tcp","udp"],"congestion_controller":"bbr","cwnd":32,"users":[{"name":"alice","password":"pw"}]}`)
	if err := ValidateInboundJSON(sanitized); err != nil {
		t.Fatalf("sanitized trusttunnel inbound failed validation: %v", err)
	}

	unsanitized := []byte(`{"type":"trusttunnel","tag":"trusttunnel-in","listen":"0.0.0.0","listen_port":443,"quic":true,"network":["tcp","udp"],"congestion_controller":"bbr","cwnd":32,"users":[{"name":"alice","password":"pw"}]}`)
	if err := ValidateInboundJSON(unsanitized); err == nil {
		t.Fatal("unsanitized trusttunnel inbound with outbound-only `quic` unexpectedly passed validation")
	} else if !strings.Contains(err.Error(), "quic") {
		t.Fatalf("unsanitized trusttunnel inbound failed for an unexpected reason (want `quic` rejection): %v", err)
	}
}

// ValidateConfig must accept a selector group with static members. This covers
// the core-supported group type that panel-managed failover assembles into.
func TestValidateConfigAcceptsSelectorGroup(t *testing.T) {
	config := []byte(`{
		"log":{"disabled":true},
		"dns":{"servers":[],"rules":[]},
		"outbounds":[
			{"type":"direct","tag":"direct"},
			{"type":"socks","tag":"member-a","server":"127.0.0.1","server_port":1080},
			{"type":"selector","tag":"auto-group","outbounds":["member-a","direct"],"default":"member-a"}
		],
		"route":{"rules":[]}
	}`)
	if err := ValidateConfig(config); err != nil {
		t.Fatalf("ValidateConfig rejected selector group config: %v", err)
	}
}

// ValidateConfig must accept a urltest group with static members.
func TestValidateConfigAcceptsURLTestGroup(t *testing.T) {
	config := []byte(`{
		"log":{"disabled":true},
		"dns":{"servers":[],"rules":[]},
		"outbounds":[
			{"type":"direct","tag":"direct"},
			{"type":"socks","tag":"member-a","server":"127.0.0.1","server_port":1080},
			{"type":"urltest","tag":"latency-group","outbounds":["member-a"],"url":"https://www.gstatic.com/generate_204","interval":"1m"}
		],
		"route":{"rules":[]}
	}`)
	if err := ValidateConfig(config); err != nil {
		t.Fatalf("ValidateConfig rejected urltest group config: %v", err)
	}
}

// ValidateConfig must accept a fallback group with static members.
func TestValidateConfigAcceptsFallbackGroup(t *testing.T) {
	config := []byte(`{
		"log":{"disabled":true},
		"dns":{"servers":[],"rules":[]},
		"outbounds":[
			{"type":"direct","tag":"direct"},
			{"type":"socks","tag":"member-a","server":"127.0.0.1","server_port":1080},
			{"type":"fallback","tag":"fb-group","outbounds":["member-a","direct"]}
		],
		"route":{"rules":[]}
	}`)
	if err := ValidateConfig(config); err != nil {
		t.Fatalf("ValidateConfig rejected fallback group config: %v", err)
	}
}

// ValidateConfig must accept a selector group that a panel-managed failover
// would assemble into: ordered members, default pinned to the first member,
// and interrupt_exist_connections. This proves the assembled config is valid
// against the linked core registries without changing the core.
func TestValidateConfigAcceptsFailoverAssembledAsSelector(t *testing.T) {
	config := []byte(`{
		"log":{"disabled":true},
		"dns":{"servers":[],"rules":[]},
		"outbounds":[
			{"type":"direct","tag":"direct"},
			{"type":"socks","tag":"us-1","server":"127.0.0.1","server_port":1080},
			{"type":"socks","tag":"us-2","server":"127.0.0.1","server_port":1081},
			{"type":"selector","tag":"auto-us","outbounds":["us-1","us-2","direct"],"default":"us-1","interrupt_exist_connections":true}
		],
		"route":{"rules":[]}
	}`)
	if err := ValidateConfig(config); err != nil {
		t.Fatalf("ValidateConfig rejected failover-assembled-as-selector config: %v", err)
	}
}

// ValidateConfig must accept an inline provider with a simple outbound.
func TestValidateConfigAcceptsInlineProvider(t *testing.T) {
	config := []byte(`{
		"log":{"disabled":true},
		"dns":{"servers":[],"rules":[]},
		"outbounds":[
			{"type":"direct","tag":"direct"}
		],
		"providers":[
			{"type":"inline","tag":"inline-prov","outbounds":[{"type":"socks","tag":"prov-member","server":"127.0.0.1","server_port":1080}]}
		],
		"route":{"rules":[]}
	}`)
	if err := ValidateConfig(config); err != nil {
		t.Fatalf("ValidateConfig rejected inline provider config: %v", err)
	}
}

// ValidateConfig documents that it constructs and closes sing-box without
// starting its lifecycle: no listener binds and no remote rule-set downloads.
// A config with a listen inbound plus a remote rule-set whose URL is an
// unreachable host must therefore validate quickly (no blocking download) and
// leave the inbound port free afterwards (no leaked listener).
func TestValidateConfigDoesNotBindOrDownload(t *testing.T) {
	const port = 18099
	config := []byte(`{
		"log":{"disabled":true},
		"dns":{"servers":[],"rules":[]},
		"inbounds":[{"type":"mixed","tag":"in","listen":"127.0.0.1","listen_port":18099}],
		"outbounds":[{"type":"direct","tag":"direct"}],
		"route":{"rules":[{"rule_set":"remote-rs","outbound":"direct"}],"rule_set":[{"type":"remote","tag":"remote-rs","format":"binary","url":"https://10.255.255.1/never.srs","download_detour":"direct"}]}
	}`)

	done := make(chan error, 1)
	go func() { done <- ValidateConfig(config) }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("ValidateConfig must accept config without binding/downloading: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("ValidateConfig blocked — it appears to start listeners or download rule-sets")
	}

	ln, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", "18099"))
	if err != nil {
		t.Fatalf("inbound port %d still bound after ValidateConfig (listener leaked): %v", port, err)
	}
	_ = ln.Close()
}
