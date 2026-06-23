package core

import (
	"errors"
	"testing"
)

func TestGroupSelect(t *testing.T) {
	c := NewCore()
	cfg := []byte(`{"log":{"disabled":true},"outbounds":[` +
		`{"type":"direct","tag":"direct"},` +
		`{"type":"socks","tag":"proxy","server":"127.0.0.1","server_port":1080},` +
		`{"type":"selector","tag":"g","outbounds":["proxy","direct"],"default":"proxy"}` +
		`]}`)
	if err := c.Start(cfg); err != nil {
		t.Skipf("minimal core start unavailable: %v", err)
	}
	t.Cleanup(func() { _ = c.Stop() })

	if active, ok := c.GroupNow("g"); !ok || active != "proxy" {
		t.Fatalf("GroupNow after start = %q,%v; want proxy,true", active, ok)
	}

	// Switching to the direct member is the all-down fallback path.
	if err := c.SelectGroupMember("g", "direct"); err != nil {
		t.Fatalf("select direct: %v", err)
	}
	if active, _ := c.GroupNow("g"); active != "direct" {
		t.Fatalf("GroupNow after select = %q; want direct", active)
	}

	if err := c.SelectGroupMember("g", "missing"); !errors.Is(err, ErrMemberNotInGroup) {
		t.Fatalf("select missing member err = %v; want ErrMemberNotInGroup", err)
	}
	if err := c.SelectGroupMember("direct", "x"); !errors.Is(err, ErrNotASelectorGroup) {
		t.Fatalf("select on non-group err = %v; want ErrNotASelectorGroup", err)
	}
	if err := c.SelectGroupMember("nope", "x"); !errors.Is(err, ErrGroupNotFound) {
		t.Fatalf("select on unknown group err = %v; want ErrGroupNotFound", err)
	}
	if _, ok := c.GroupNow("nope"); ok {
		t.Fatal("GroupNow on unknown group must report ok=false")
	}
}
