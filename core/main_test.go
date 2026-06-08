package core

import (
	"context"
	"testing"
)

func TestCoreManagersAreUnavailableWhenStopped(t *testing.T) {
	c := NewCore()

	if c.Router() != nil {
		t.Fatal("router must be nil before core starts")
	}
	if c.OutboundManager() != nil {
		t.Fatal("outbound manager must be nil before core starts")
	}
	if _, ok := c.runtime(); ok {
		t.Fatal("runtime must be unavailable before core starts")
	}
}

func TestCoreCheckOutboundRequiresRunningCore(t *testing.T) {
	c := NewCore()

	result := c.CheckOutbound(context.Background(), "direct", "https://example.com")
	if result.Error != "core not running" {
		t.Fatalf("expected core not running error, got %q", result.Error)
	}
}

func TestNewCoreContextsDoNotInterfere(t *testing.T) {
	type contextKey string
	const key contextKey = "core"
	first := NewCore()
	second := NewCore()

	first.access.Lock()
	first.ctx = context.WithValue(first.ctx, key, "first")
	first.access.Unlock()

	if got := first.GetCtx().Value(key); got != "first" {
		t.Fatalf("first context value=%v", got)
	}
	if got := second.GetCtx().Value(key); got != nil {
		t.Fatalf("second core observed first context value: %v", got)
	}
}
