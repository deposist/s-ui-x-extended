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
