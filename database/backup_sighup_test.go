package database

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func resetSighupTimeoutCacheForTest(t *testing.T) {
	t.Helper()
	SetSighupTimeoutForTest(0)
	sighupTimeoutOnce = sync.Once{}
	t.Cleanup(func() {
		SetSighupTimeoutForTest(0)
		sighupTimeoutOnce = sync.Once{}
	})
}

func TestSendSighupRespectsConfiguredTimeoutIssue10(t *testing.T) {
	t.Setenv("SUI_SIGHUP_TIMEOUT_SECONDS", "5")
	resetSighupTimeoutCacheForTest(t)
	if got := resolvedSighupTimeout(); got != 5*time.Second {
		t.Fatalf("expected 5s from env, got %s", got)
	}

	t.Setenv("SUI_SIGHUP_TIMEOUT_SECONDS", "not-a-number")
	resetSighupTimeoutCacheForTest(t)
	if got := resolvedSighupTimeout(); got != 3*time.Second {
		t.Fatalf("expected 3s default for invalid env, got %s", got)
	}

	t.Setenv("SUI_SIGHUP_TIMEOUT_SECONDS", "")
	resetSighupTimeoutCacheForTest(t)
	if got := resolvedSighupTimeout(); got != 3*time.Second {
		t.Fatalf("expected 3s default for empty env, got %s", got)
	}

	SetSighupTimeoutForTest(25 * time.Millisecond)
	if got := resolvedSighupTimeout(); got != 25*time.Millisecond {
		t.Fatalf("expected test override timeout, got %s", got)
	}

	var fired atomic.Bool
	SetSendSighupHook(func() error {
		fired.Store(true)
		return nil
	})
	t.Cleanup(func() { SetSendSighupHook(nil) })

	if err := SendSighup(); err != nil {
		t.Fatal(err)
	}
	if !fired.Load() {
		t.Fatal("hook should have fired synchronously when set")
	}

	for _, raw := range []string{"0", "61", "-5"} {
		t.Setenv("SUI_SIGHUP_TIMEOUT_SECONDS", raw)
		resetSighupTimeoutCacheForTest(t)
		if got := resolvedSighupTimeout(); got != 3*time.Second {
			t.Fatalf("expected 3s default for out-of-range %q, got %s", raw, got)
		}
	}
}
