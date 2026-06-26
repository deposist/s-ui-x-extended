package core

import (
	"net"
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
