package service

import "testing"

func TestSecurityConstantTimeStringEqualAllowedScopes(t *testing.T) {
	for _, scope := range allowedAPITokenScopes {
		t.Run(scope, func(t *testing.T) {
			if constantTimeStringEqual(scope, scope, maxAPITokenScopeLen) != 1 {
				t.Fatalf("scope %q should compare equal", scope)
			}
			if !apiTokenScopeAllowed(scope) {
				t.Fatalf("scope %q should be allowed", scope)
			}
		})
	}
	if constantTimeStringEqual("admin", "read", maxAPITokenScopeLen) != 0 {
		t.Fatal("different scopes compared equal")
	}
	if apiTokenScopeAllowed("admin" + string(make([]byte, maxAPITokenScopeLen+1))) {
		t.Fatal("oversized scope should not be allowed")
	}
}
