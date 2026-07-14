package core

import (
	"strings"
	"testing"
)

func TestWithWireGuardIPCRequiresRunningCore(t *testing.T) {
	err := NewCore().WithWireGuardIPC("awg", func(WireGuardIPC) error {
		t.Fatal("callback must not run while core is stopped")
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "sing-box is not running") {
		t.Fatalf("WithWireGuardIPC error = %v; want stopped-core error", err)
	}
}

func TestWithWireGuardIPCRejectsNilCallback(t *testing.T) {
	err := NewCore().WithWireGuardIPC("awg", nil)
	if err == nil || err.Error() != "wireguard IPC callback is nil" {
		t.Fatalf("WithWireGuardIPC error = %v; want nil-callback error", err)
	}
}

func TestWithWireGuardIPCReportsMissingEndpoint(t *testing.T) {
	core := NewCore()
	config := []byte(`{"log":{"disabled":true},"outbounds":[{"type":"direct","tag":"direct"}]}`)
	if err := core.Start(config); err != nil {
		t.Skipf("minimal core start unavailable: %v", err)
	}
	t.Cleanup(func() { _ = core.Stop() })

	err := core.WithWireGuardIPC("missing", func(WireGuardIPC) error {
		t.Fatal("callback must not run for a missing endpoint")
		return nil
	})
	if err == nil || err.Error() != `wireguard endpoint "missing" not found` {
		t.Fatalf("WithWireGuardIPC error = %v; want missing-endpoint error", err)
	}
}
