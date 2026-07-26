package ssrf

import (
	"context"
	"net"
	"net/netip"
	"strings"
	"testing"
)

func TestPublicDialContextRejectsUnsafeResolvedAddress(t *testing.T) {
	lookupCalls := 0
	dialCalls := 0
	dialer := NewPublicDialContext(
		func(context.Context, string, string) ([]netip.Addr, error) {
			lookupCalls++
			return []netip.Addr{netip.MustParseAddr("93.184.216.34"), netip.MustParseAddr("127.0.0.1")}, nil
		},
		func(context.Context, string, string) (net.Conn, error) {
			dialCalls++
			return nil, nil
		},
	)
	if _, err := dialer(context.Background(), "tcp", "example.com:443"); err == nil {
		t.Fatal("mixed public/private DNS answer must be rejected")
	}
	if lookupCalls != 1 || dialCalls != 0 {
		t.Fatalf("lookup calls = %d, dial calls = %d; want 1, 0", lookupCalls, dialCalls)
	}
}

func TestPublicDialContextDialsValidatedLiteral(t *testing.T) {
	var dialed string
	dialer := NewPublicDialContext(
		func(context.Context, string, string) ([]netip.Addr, error) {
			return []netip.Addr{netip.MustParseAddr("93.184.216.34")}, nil
		},
		func(_ context.Context, network, address string) (net.Conn, error) {
			if network != "tcp" {
				t.Fatalf("network = %q", network)
			}
			dialed = address
			return nil, context.Canceled
		},
	)
	_, err := dialer(context.Background(), "tcp", "example.com:443")
	if err == nil || !strings.Contains(err.Error(), "context canceled") {
		t.Fatalf("unexpected dial error: %v", err)
	}
	if dialed != "93.184.216.34:443" {
		t.Fatalf("dialed %q; want resolved literal", dialed)
	}
}

func TestPublicDialContextRejectsBlockedLiteralWithoutLookup(t *testing.T) {
	lookupCalls := 0
	dialer := NewPublicDialContext(
		func(context.Context, string, string) ([]netip.Addr, error) {
			lookupCalls++
			return nil, nil
		},
		nil,
	)
	if _, err := dialer(context.Background(), "tcp", "169.254.169.254:443"); err == nil {
		t.Fatal("metadata address must be rejected")
	}
	if lookupCalls != 0 {
		t.Fatalf("literal triggered %d lookups", lookupCalls)
	}
}
