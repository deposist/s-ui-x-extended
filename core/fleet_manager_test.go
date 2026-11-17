//go:build with_manager

package core

import (
	"os"
	"path/filepath"
	"testing"

	CM "github.com/sagernet/sing-box/service/manager/constant"
)

// fleetTempDB returns a throwaway fleet database path. It deliberately avoids
// t.TempDir(): a fork release that does not close the repository keeps the file
// open, and t.TempDir treats a failed cleanup as a test failure. That leak is
// the fork's bug to fix (fixed upstream in the next release), not a reason for
// this panel test to go red.
func fleetTempDB(t *testing.T, name string) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "sui-fleet-")
	if err != nil {
		t.Fatalf("temp dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return filepath.ToSlash(filepath.Join(dir, name))
}

// TestWithFleetManagerReachesRunningManager proves the accessor resolves the
// manager service of a live core: a controller role starts a real manager on a
// throwaway fleet database and the callback must see it on every call, which
// also shows the pointer never needs caching.
func TestWithFleetManagerReachesRunningManager(t *testing.T) {
	fleetDB := fleetTempDB(t, "fleet.db")
	config := `{
		"log":{"disabled":true},
		"dns":{"servers":[{"type":"local","tag":"local"}]},
		"outbounds":[{"type":"direct","tag":"direct"}],
		"route":{"final":"direct"},
		"services":[{"type":"manager","tag":"fleet","database":{"driver":"sqlite","dsn":"` + fleetDB + `"}}]
	}`

	core := NewCore()
	if err := core.Start([]byte(config)); err != nil {
		t.Fatalf("start core: %v", err)
	}
	defer func() {
		if err := core.Stop(); err != nil {
			t.Errorf("stop core: %v", err)
		}
	}()

	seen := 0
	for i := 0; i < 2; i++ {
		err := core.WithFleetManager(func(manager CM.Manager) error {
			seen++
			_, err := manager.GetNodes(map[string][]string{})
			return err
		})
		if err != nil {
			t.Fatalf("WithFleetManager call %d: %v", i, err)
		}
	}
	if seen != 2 {
		t.Fatalf("callback ran %d times, want 2", seen)
	}
}

// TestWithFleetManagerErrorsWithoutManager keeps the failure honest when the
// controller role is not configured: callers must get a distinguishable error
// instead of a nil panic.
func TestWithFleetManagerErrorsWithoutManager(t *testing.T) {
	core := NewCore()
	if err := core.Start([]byte(`{
		"log":{"disabled":true},
		"dns":{"servers":[{"type":"local","tag":"local"}]},
		"outbounds":[{"type":"direct","tag":"direct"}],
		"route":{"final":"direct"}
	}`)); err != nil {
		t.Fatalf("start core: %v", err)
	}
	defer func() {
		if err := core.Stop(); err != nil {
			t.Errorf("stop core: %v", err)
		}
	}()

	err := core.WithFleetManager(func(CM.Manager) error { return nil })
	if err == nil {
		t.Fatal("expected an error when no manager service is running")
	}
}
