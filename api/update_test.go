package api

import (
	"testing"
	"time"

	"github.com/deposist/s-ui-x-extended/config"
	"github.com/deposist/s-ui-x-extended/service"
)

// T048 / SR-009: forced version checks are rate-limited.
func TestAllowForcedUpdateCheckRateLimits(t *testing.T) {
	updateCheckMu.Lock()
	updateCheckLastAt = time.Time{}
	updateCheckMu.Unlock()
	t.Cleanup(func() {
		updateCheckMu.Lock()
		updateCheckLastAt = time.Time{}
		updateCheckMu.Unlock()
	})

	base := time.Unix(1_700_000_000, 0)
	if !allowForcedUpdateCheck(base) {
		t.Fatal("first forced check should be allowed")
	}
	if allowForcedUpdateCheck(base.Add(time.Second)) {
		t.Fatal("a check within the interval should be rate-limited")
	}
	if !allowForcedUpdateCheck(base.Add(updateCheckMinInterval + time.Second)) {
		t.Fatal("a check after the interval should be allowed")
	}
}

// T011 / SR-004: invalid channels normalize to the safe default ("main"); only
// "beta" is accepted as the non-default.
func TestNormalizeUpdateChannelAllowlist(t *testing.T) {
	cases := map[string]string{
		"main":             config.UpdateChannelMain,
		"beta":             config.UpdateChannelBeta,
		"":                 config.UpdateChannelMain,
		"../../etc/passwd": config.UpdateChannelMain,
		"BETA":             config.UpdateChannelMain, // case-sensitive allowlist
	}
	for in, want := range cases {
		if got := config.NormalizeUpdateChannel(in); got != want {
			t.Fatalf("NormalizeUpdateChannel(%q) = %q, want %q", in, got, want)
		}
	}
}

// FR-016: apply only proceeds for the confirmed target version.
func TestTargetVersionMatches(t *testing.T) {
	target := service.ReleaseTarget{Tag: "v1.5.9", Version: "1.5.9"}
	if !targetVersionMatches("v1.5.9", target) || !targetVersionMatches("1.5.9", target) {
		t.Fatal("matching tag or normalized version should pass")
	}
	if targetVersionMatches("", target) {
		t.Fatal("empty requested version must be rejected (FR-016)")
	}
	if targetVersionMatches("1.5.8", target) {
		t.Fatal("stale requested version must be rejected")
	}
}
