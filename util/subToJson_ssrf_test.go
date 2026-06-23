package util

import (
	"net/netip"
	"testing"
)

// TestIsBlockedExternalAddrCoversReservedRanges pins the L3 fix: the external
// subscription fetch path now delegates to the central SSRF validator, so the
// CGNAT/benchmark/reserved ranges it previously let through are blocked.
func TestIsBlockedExternalAddrCoversReservedRanges(t *testing.T) {
	blocked := []string{
		"100.64.0.1",      // CGNAT 100.64.0.0/10 (previously NOT blocked)
		"100.127.255.1",   // CGNAT upper bound
		"192.0.0.1",       // 192.0.0.0/24 (previously NOT blocked)
		"198.18.0.1",      // benchmarking 198.18.0.0/15 (previously NOT blocked)
		"198.19.255.1",    // benchmarking upper bound
		"240.0.0.1",       // reserved 240.0.0.0/4 (previously NOT blocked)
		"10.0.0.1",        // RFC1918 (already blocked before)
		"127.0.0.1",       // loopback
		"169.254.169.254", // cloud-metadata link-local (always blocked)
	}
	for _, s := range blocked {
		if !isBlockedExternalAddr(netip.MustParseAddr(s)) {
			t.Errorf("expected %s to be blocked", s)
		}
	}

	allowed := []string{"1.1.1.1", "8.8.8.8", "93.184.216.34"}
	for _, s := range allowed {
		if isBlockedExternalAddr(netip.MustParseAddr(s)) {
			t.Errorf("expected public %s to be allowed", s)
		}
	}
}
