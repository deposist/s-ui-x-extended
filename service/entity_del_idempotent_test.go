package service

import (
	"encoding/json"
	"testing"
)

// Deleting an entity that is already gone must be an idempotent success for
// every entity type the panel manages - a stale UI row, a concurrent delete
// from another session, or a resubmitted request must not surface
// "record not found" (or any other error) to the operator.
func TestConfigSaveDelOfMissingEntityIsIdempotent(t *testing.T) {
	cases := []struct {
		obj  string
		data string
	}{
		{"inbounds", `"missing-tag"`},
		{"outbounds", `"missing-tag"`},
		{"endpoints", `"missing-tag"`},
		{"services", `"missing-tag"`},
		{"tls", `999999`},
	}
	for _, tc := range cases {
		t.Run(tc.obj, func(t *testing.T) {
			initSettingTestDB(t)
			cs := NewConfigServiceWithRuntime(NewRuntimeWithCoreProvider(nil))
			if _, err := cs.Save(tc.obj, "del", json.RawMessage(tc.data), "", "admin", "example.com"); err != nil {
				t.Fatalf("deleting a missing %s should be a no-op success, got: %v", tc.obj, err)
			}
		})
	}
}
