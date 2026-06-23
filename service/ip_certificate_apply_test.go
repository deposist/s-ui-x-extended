package service

import (
	"encoding/json"
	"testing"
)

func TestPatchTlsServerBlock(t *testing.T) {
	// Empty server block → a fresh object carrying the managed file paths.
	out, err := patchTlsServerBlock(nil, "/c/cert.crt", "/c/key.key")
	if err != nil {
		t.Fatal(err)
	}
	m := map[string]any{}
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	if m["certificate_path"] != "/c/cert.crt" || m["key_path"] != "/c/key.key" {
		t.Fatalf("paths not set on empty block: %v", m)
	}

	// Inline certificate/key bytes must be dropped (they would shadow the file
	// paths in sing-box) while unrelated fields are preserved.
	existing := []byte(`{"server_name":"x","certificate":["AAA"],"key":["BBB"],"min_version":"1.3"}`)
	out, err = patchTlsServerBlock(existing, "/c2.crt", "/k2.key")
	if err != nil {
		t.Fatal(err)
	}
	m = map[string]any{}
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	if _, ok := m["certificate"]; ok {
		t.Error("inline certificate not removed")
	}
	if _, ok := m["key"]; ok {
		t.Error("inline key not removed")
	}
	if m["server_name"] != "x" || m["min_version"] != "1.3" {
		t.Errorf("unrelated fields lost: %v", m)
	}
	if m["certificate_path"] != "/c2.crt" || m["key_path"] != "/k2.key" {
		t.Errorf("paths not set on existing block: %v", m)
	}

	// A non-object server block is rejected rather than silently overwritten.
	if _, err := patchTlsServerBlock([]byte(`["not","an","object"]`), "c", "k"); err == nil {
		t.Fatal("patchTlsServerBlock on non-object = nil, want error")
	}
}

func TestApplyToTargetRoutingErrors(t *testing.T) {
	// These targets are rejected before any DB access or panel restart, so a
	// zero-value service is sufficient to exercise the routing guards.
	svc := &IpCertificateService{}
	bad := []string{"garbage", "inbound:", "inbound:abc", "inbound:0", "inbound:-3"}
	for _, target := range bad {
		if err := svc.applyToTarget(target, "/c.crt", "/k.key", ""); err == nil {
			t.Errorf("applyToTarget(%q) = nil, want error", target)
		}
	}
}
