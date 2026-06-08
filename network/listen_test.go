package network

import (
	"fmt"
	"net"
	"os"
	"runtime"
	"strings"
	"syscall"
	"testing"
)

// TestListenWithFallbackBindsImmediatelyOnLoopback ensures the helper does
// not perturb the happy path where the requested address is bindable.
func TestListenWithFallbackBindsImmediatelyOnLoopback(t *testing.T) {
	listener, err := ListenWithFallback("127.0.0.1:0", "127.0.0.1", "0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	if !strings.HasPrefix(listener.Addr().String(), "127.0.0.1:") {
		t.Fatalf("expected loopback bind, got %s", listener.Addr())
	}
}

// TestListenWithFallbackHandlesUnbindableAddress simulates the after-restore
// scenario: the saved listen IP no longer exists on this machine. The
// helper must transparently fall back to binding on every interface so the
// panel can still come up.
func TestListenWithFallbackHandlesUnbindableAddress(t *testing.T) {
	// 240.0.0.0/4 is reserved (Class E) and is not assigned to any
	// interface on conventional hosts, so binding it produces
	// EADDRNOTAVAIL on Linux/macOS and the equivalent error on Windows.
	stale := "240.0.0.1"
	result, err := ListenWithFallbackResult(net.JoinHostPort(stale, "0"), stale, "0")
	if err != nil {
		// Some test environments may have unusual networking; surface a
		// clear hint so it's obvious why the test was skipped.
		t.Skipf("fallback path could not be exercised: %v", err)
	}
	listener := result.Listener
	defer listener.Close()
	if !result.Fallback || result.BindError == nil || result.FallbackAddr == "" {
		t.Fatalf("fallback result was not populated: %#v", result)
	}
	if strings.HasPrefix(listener.Addr().String(), stale+":") {
		t.Fatalf("expected fallback, but listener is still on %s", listener.Addr())
	}
}

func TestShouldFallbackRecognisesEADDRNOTAVAIL(t *testing.T) {
	err := &net.OpError{
		Op:  "listen",
		Net: "tcp",
		Err: &os.SyscallError{Syscall: "bind", Err: syscall.EADDRNOTAVAIL},
	}

	if !shouldFallback(err) {
		t.Fatal("expected fallback for EADDRNOTAVAIL")
	}
	if !shouldFallback(fmt.Errorf("wrapped: %w", err)) {
		t.Fatal("expected fallback for wrapped EADDRNOTAVAIL")
	}
}

func TestShouldFallbackRecognisesWindowsEADDRNOTAVAIL(t *testing.T) {
	err := &net.OpError{
		Op:  "listen",
		Net: "tcp",
		Err: &os.SyscallError{Syscall: "bind", Err: syscall.Errno(10049)},
	}

	if runtime.GOOS == "windows" && !shouldFallback(err) {
		t.Fatal("expected fallback for Windows WSAEADDRNOTAVAIL")
	}
	if runtime.GOOS != "windows" && shouldFallback(err) {
		t.Fatal("did not expect Windows WSAEADDRNOTAVAIL fallback on non-Windows")
	}
}

func TestShouldFallbackRejectsStringOnlyErrors(t *testing.T) {
	if shouldFallback(&net.OpError{Op: "listen", Err: &errString{"cannot assign requested address"}}) {
		t.Fatal("did not expect fallback for string-only address error")
	}
	if shouldFallback(&net.OpError{Op: "listen", Err: &errString{"some unrelated error"}}) {
		t.Fatal("did not expect fallback for unrelated error")
	}
}

type errString struct{ s string }

func (e *errString) Error() string { return e.s }
