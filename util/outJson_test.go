package util

import (
	"encoding/json"
	"testing"

	"github.com/deposist/s-ui-x-extended/database/model"
)

// TestAddTlsDoesNotPanicOnNonLockstepBlobs pins the H1 regression: addTls must be
// total over the operator-controlled (and unvalidated) Tls.Server/Tls.Client JSON
// blobs. A server that enables reality/ech while the client block omits the matching
// sub-map (or is JSON null, or carries a non-bool enabled flag) used to panic via
// nil-map writes and bare type assertions, crashing the save path.
func TestAddTlsDoesNotPanicOnNonLockstepBlobs(t *testing.T) {
	cases := []struct {
		name   string
		server string
		client string
	}{
		{"client null", `{"enabled":true}`, `null`},
		{"client empty object", `{"enabled":true}`, `{}`},
		{"reality enabled, client lacks reality map", `{"reality":{"enabled":true}}`, `{}`},
		{"reality enabled non-bool", `{"reality":{"enabled":"yes"}}`, `{}`},
		{"reality present without enabled", `{"reality":{}}`, `{}`},
		{"ech enabled, client lacks ech map", `{"ech":{"enabled":true}}`, `{}`},
		{"ech enabled non-bool", `{"ech":{"enabled":1}}`, `{}`},
		{"server null", `null`, `{}`},
		{"both null", `null`, `null`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("addTls panicked on %s: %v", tc.name, r)
				}
			}()
			out := map[string]interface{}{}
			addTls(&out, &model.Tls{
				Server: json.RawMessage(tc.server),
				Client: json.RawMessage(tc.client),
			})
		})
	}
}

// TestAddTlsRealityHappyPath verifies the merge still works when server and client
// are in lockstep (the UI-produced shape), so the hardening did not regress behavior.
func TestAddTlsRealityHappyPath(t *testing.T) {
	out := map[string]interface{}{}
	addTls(&out, &model.Tls{
		Server: json.RawMessage(`{"enabled":true,"server_name":"example.com","reality":{"enabled":true,"short_id":["ab"]}}`),
		Client: json.RawMessage(`{"reality":{},"utls":{"enabled":true}}`),
	})
	tlsBlock, ok := out["tls"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected tls block, got %T", out["tls"])
	}
	if tlsBlock["enabled"] != true {
		t.Fatalf("expected enabled=true, got %v", tlsBlock["enabled"])
	}
	reality, ok := tlsBlock["reality"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected reality map, got %T", tlsBlock["reality"])
	}
	if reality["enabled"] != true {
		t.Fatalf("expected reality.enabled=true, got %v", reality["enabled"])
	}
	if reality["short_id"] != "ab" {
		t.Fatalf("expected reality.short_id=ab, got %v", reality["short_id"])
	}
}
