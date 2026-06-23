package util

import "testing"

// TestAddParamsDoesNotPanicOnUnparsableURI pins the L7 fix: a control byte in the
// operator-controlled server address makes url.Parse return (nil, err); addParams
// must return the input unchanged instead of nil-dereferencing and panicking the
// subscription/link-generation path.
func TestAddParamsDoesNotPanicOnUnparsableURI(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("addParams panicked: %v", r)
		}
	}()
	bad := "vless://uuid@evil\x00host:443"
	got := addParams(bad, []LinkParam{{Key: "type", Value: "tcp"}}, "remark")
	if got != bad {
		t.Fatalf("expected unparsable uri returned unchanged, got %q", got)
	}

	// Sanity: a well-formed uri still gets params/fragment applied.
	ok := addParams("vless://uuid@example.com:443", []LinkParam{{Key: "type", Value: "tcp"}}, "node")
	if ok == "" || ok == "vless://uuid@example.com:443" {
		t.Fatalf("expected params applied to a valid uri, got %q", ok)
	}
}
