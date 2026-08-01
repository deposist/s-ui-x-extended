package core

import (
	"context"
	"errors"
	"runtime"
	"strings"
	"testing"
	"time"
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

func TestWithWireGuardIPCContextCancelsWhileWaitingForCriticalSection(t *testing.T) {
	core := NewCore()
	<-core.wireGuardIPCAccess
	defer func() { core.wireGuardIPCAccess <- struct{}{} }()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	called := false
	err := core.WithWireGuardIPCContext(ctx, "awg", func(WireGuardIPC) error {
		called = true
		return nil
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("WithWireGuardIPCContext error = %v, want context deadline", err)
	}
	if called {
		t.Fatal("callback ran after context expired while waiting for IPC lock")
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

func TestWireGuardIPCLockSerializesEndpointRemoval(t *testing.T) {
	core := NewCore()
	<-core.wireGuardIPCAccess

	acquired := make(chan struct{})
	done := make(chan struct{})
	go func() {
		<-core.wireGuardIPCAccess
		close(acquired)
		core.wireGuardIPCAccess <- struct{}{}
		close(done)
	}()
	runtime.Gosched()

	select {
	case <-acquired:
		t.Fatal("a competing endpoint operation entered the IPC critical section")
	default:
	}
	core.wireGuardIPCAccess <- struct{}{}
	<-done
}
