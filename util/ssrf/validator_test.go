package ssrf

import (
	"context"
	"net/netip"
	"testing"
)

func TestValidateOutboundURLRejectsUnsafeTargets(t *testing.T) {
	tests := []string{
		"file:///tmp/data",
		"http://localhost/test",
		"http://127.0.0.1/test",
		"http://10.0.0.1/test",
		"http://100.64.0.1/test",
		"http://169.254.169.254/latest/meta-data",
		"http://[::1]/test",
		"http://[::ffff:127.0.0.1]/test",
		"http://[64:ff9b::808:808]/test",
		"http://example/test",
		"http://bad_host.example/test",
	}
	for _, rawURL := range tests {
		t.Run(rawURL, func(t *testing.T) {
			if err := ValidateOutboundURL(context.Background(), rawURL); err == nil {
				t.Fatal("expected URL to be rejected")
			}
			if IsSafeOutboundURL(rawURL) {
				t.Fatal("IsSafeOutboundURL returned true for rejected URL")
			}
		})
	}
}

func TestValidateOutboundURLAllowsPublicTargetsAndProxySchemes(t *testing.T) {
	tests := []string{
		"https://1.1.1.1/generate_204",
		"http://8.8.8.8:8080",
		"socks5://user:pass@8.8.4.4:1080",
	}
	for _, rawURL := range tests {
		t.Run(rawURL, func(t *testing.T) {
			if err := ValidateOutboundURL(context.Background(), rawURL); err != nil {
				t.Fatalf("expected URL to be accepted: %v", err)
			}
			if !IsSafeOutboundURL(rawURL) {
				t.Fatal("IsSafeOutboundURL returned false for accepted URL")
			}
		})
	}
}

func TestValidateOutboundURLHonorsSchemeAllowlist(t *testing.T) {
	if err := ValidateOutboundURL(context.Background(), "http://8.8.8.8/test", "https"); err == nil {
		t.Fatal("expected HTTP URL to be rejected by HTTPS-only allowlist")
	}
	if err := ValidateOutboundURL(context.Background(), "https://8.8.8.8/test", "https"); err != nil {
		t.Fatalf("expected HTTPS URL to be accepted: %v", err)
	}
}

func TestBlockedAddrCoversReservedAndMappedRanges(t *testing.T) {
	rejected := []string{
		"0.0.0.1",
		"10.0.0.1",
		"100.64.0.1",
		"127.0.0.1",
		"169.254.169.254",
		"172.16.0.1",
		"192.0.0.1",
		"192.0.2.1",
		"192.168.0.1",
		"198.18.0.1",
		"198.51.100.1",
		"203.0.113.1",
		"224.0.0.1",
		"240.0.0.1",
		"::",
		"::1",
		"::ffff:127.0.0.1",
		"64:ff9b::808:808",
		"100::1",
		"2001::1",
		"2001:db8::1",
		"fc00::1",
		"fe80::1",
		"ff00::1",
	}
	for _, ip := range rejected {
		t.Run(ip, func(t *testing.T) {
			if !isBlockedAddr(netip.MustParseAddr(ip)) {
				t.Fatal("expected address to be blocked")
			}
		})
	}
	allowed := []string{"1.1.1.1", "8.8.8.8", "2606:4700:4700::1111"}
	for _, ip := range allowed {
		t.Run(ip, func(t *testing.T) {
			if isBlockedAddr(netip.MustParseAddr(ip)) {
				t.Fatal("expected public address to be allowed")
			}
		})
	}
}
