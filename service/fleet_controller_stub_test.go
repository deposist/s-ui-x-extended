//go:build !with_manager

package service

import (
	"encoding/json"
	"testing"
)

// TestInjectFleetControllerSkipsUntaggedBuild is the honesty guard for builds
// without the manager: adding a service the binary does not contain would stop
// the core from starting, so the config must be left alone.
func TestInjectFleetControllerSkipsUntaggedBuild(t *testing.T) {
	existing := json.RawMessage(`{"type":"resolved","tag":"dns"}`)
	injected := InjectFleetController(nil, []json.RawMessage{existing})
	if len(injected) != 1 || string(injected[0]) != string(existing) {
		t.Fatalf("an untagged build must not gain a manager, got %v", injected)
	}
}
