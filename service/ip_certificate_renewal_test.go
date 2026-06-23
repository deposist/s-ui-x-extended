package service

import (
	"testing"
	"time"
)

func TestShouldRenew(t *testing.T) {
	now := time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name     string
		notAfter time.Time
		want     bool
	}{
		{"zero notAfter renews", time.Time{}, true},
		{"already expired renews", now.Add(-1 * time.Hour), true},
		{"71h remaining renews", now.Add(71 * time.Hour), true},
		{"exactly threshold does not renew", now.Add(ipCertRenewThreshold), false},
		{"73h remaining does not renew", now.Add(73 * time.Hour), false},
		{"fresh 160h cert does not renew", now.Add(160 * time.Hour), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldRenew(tc.notAfter, now); got != tc.want {
				t.Fatalf("shouldRenew(%v, now) = %v, want %v", tc.notAfter, got, tc.want)
			}
		})
	}
}

func TestValidateIssuableIP(t *testing.T) {
	rejected := []string{
		"",
		"not-an-ip",
		"10.0.0.1",
		"192.168.1.1",
		"172.16.5.5",
		"127.0.0.1",
		"::1",
		"169.254.169.254",   // cloud metadata
		"0.0.0.0",           // unspecified
		"224.0.0.1",         // multicast
		"100.64.0.1",        // CGNAT
		"192.0.2.1",         // TEST-NET-1
		"198.51.100.1",      // TEST-NET-2
		"203.0.113.1",       // TEST-NET-3
		"255.255.255.255",   // broadcast / reserved 240.0.0.0/4
		"198.18.0.1",        // benchmarking 198.18.0.0/15
		"192.0.0.1",         // IETF protocol assignments
		"fe80::1",           // IPv6 link-local
		"fc00::1",           // IPv6 unique-local
		"fd12:3456:789a::1", // IPv6 ULA
		"2001:db8::1",       // IPv6 documentation
		"ff02::1",           // IPv6 multicast
		"::",                // IPv6 unspecified
	}
	for _, ip := range rejected {
		if err := validateIssuableIP(ip); err == nil {
			t.Errorf("validateIssuableIP(%q) = nil, want error", ip)
		}
	}

	accepted := []string{
		"93.184.216.34",
		"8.8.8.8",
		"2606:4700:4700::1111",
	}
	for _, ip := range accepted {
		if err := validateIssuableIP(ip); err != nil {
			t.Errorf("validateIssuableIP(%q) = %v, want nil", ip, err)
		}
	}
}

// TestValidateIssuableIPExportedWrapper pins that the exported wrapper delegates
// to the package-private validator with identical results, so callers outside
// the service package get the same SSRF guard (public accepted, private/loopback/
// metadata/malformed rejected).
func TestValidateIssuableIPExportedWrapper(t *testing.T) {
	if err := ValidateIssuableIP("8.8.8.8"); err != nil {
		t.Errorf("ValidateIssuableIP(public) = %v, want nil", err)
	}
	for _, ip := range []string{"169.254.169.254", "127.0.0.1", "10.0.0.1", "not-an-ip", ""} {
		if err := ValidateIssuableIP(ip); err == nil {
			t.Errorf("ValidateIssuableIP(%q) = nil, want error", ip)
		}
	}
}

func TestIpCertInternalKeysNotEditable(t *testing.T) {
	// Machine-managed keys (account key, issued paths, expiry) must never be
	// writable through the settings save path.
	for _, k := range ipCertInternalSettingKeys {
		if isEditableSettingKey(k) {
			t.Errorf("internal key %q is editable, want rejected", k)
		}
	}
	// The user-facing controls remain editable.
	for _, k := range []string{"ipCertEnabled", "ipCertTargetIP", "ipCertEmail", "ipCertChallengePort", "ipCertApplyTarget"} {
		if !isEditableSettingKey(k) {
			t.Errorf("control key %q is not editable, want editable", k)
		}
	}
}

func TestValidateIpCertApplyTarget(t *testing.T) {
	ok := []string{"", "panel", "inbound:1", "inbound:42"}
	for _, v := range ok {
		if err := validateIpCertApplyTarget(v); err != nil {
			t.Errorf("validateIpCertApplyTarget(%q) = %v, want nil", v, err)
		}
	}
	bad := []string{"inbound:", "inbound:0", "inbound:-1", "inbound:abc", "service:1", "panel:1"}
	for _, v := range bad {
		if err := validateIpCertApplyTarget(v); err == nil {
			t.Errorf("validateIpCertApplyTarget(%q) = nil, want error", v)
		}
	}
}

func TestValidateIpCertEmail(t *testing.T) {
	// Empty is allowed when optional, rejected when required.
	if err := validateIpCertEmail("", false); err != nil {
		t.Errorf("optional empty email = %v, want nil", err)
	}
	if err := validateIpCertEmail("", true); err == nil {
		t.Error("required empty email = nil, want error")
	}

	valid := []string{"admin@example.com", "a.b+tag@sub.example.co.uk"}
	for _, e := range valid {
		if err := validateIpCertEmail(e, true); err != nil {
			t.Errorf("validateIpCertEmail(%q) = %v, want nil", e, err)
		}
	}

	invalid := []string{
		"@",
		"@example.com",
		"admin@",
		"plainaddress",
		"ad\nmin@example.com",                // embedded newline (not trimmed)
		"admin@exa\tmple.com",                // embedded control char
		"two addr@a.com, b@b.com",            // not a single address
		"Display Name <a@b.com>",             // display-name form rejected
		string(make([]byte, 260)) + "@x.com", // over length
	}
	for _, e := range invalid {
		if err := validateIpCertEmail(e, false); err == nil {
			t.Errorf("validateIpCertEmail(%q) = nil, want error", e)
		}
	}
}

func TestValidateIpCertPort(t *testing.T) {
	for _, p := range []int{1, 80, 443, 8080, 65535} {
		if err := validateIpCertPort(p); err != nil {
			t.Errorf("validateIpCertPort(%d) = %v, want nil", p, err)
		}
	}
	for _, p := range []int{0, -1, 65536, 100000} {
		if err := validateIpCertPort(p); err == nil {
			t.Errorf("validateIpCertPort(%d) = nil, want error", p)
		}
	}
}
