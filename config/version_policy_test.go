package config

import (
	"os"
	"strings"
	"testing"
)

func TestCurrentVersionFollowsReleasePolicy(t *testing.T) {
	if err := ValidateReleaseVersion(GetVersion()); err != nil {
		t.Fatalf("config/version does not follow release policy: %v", err)
	}
}

func TestValidateReleaseVersion(t *testing.T) {
	valid := []string{
		"1.5.2",
		"1.5.2-beta-hotfix2",
		"2.0.0-rc.1",
	}
	for _, version := range valid {
		if err := ValidateReleaseVersion(version); err != nil {
			t.Fatalf("ValidateReleaseVersion(%q) returned %v", version, err)
		}
	}

	invalid := []string{
		"",
		"v1.5.2",
		"1.5",
		"1.5.2+build",
		"1.05.2",
		"1.5.2-BETA",
		"1.5.2-01",
	}
	for _, version := range invalid {
		if err := ValidateReleaseVersion(version); err == nil {
			t.Fatalf("ValidateReleaseVersion(%q) succeeded unexpectedly", version)
		}
	}
}

func TestCompareVersionsUsesSemverPrecedence(t *testing.T) {
	tests := []struct {
		left  string
		right string
		want  int
	}{
		{left: "v1.5.10", right: "1.5.2", want: 1},
		{left: "1.5.2", right: "1.5.2-beta-hotfix2", want: 1},
		{left: "1.5.2-rc.2", right: "1.5.2-rc.1", want: 1},
		{left: "1.5.2-alpha.1", right: "1.5.2-alpha.beta", want: -1},
		{left: "1.2", right: "1.2.1", want: -1},
		{left: "1.2.0", right: "1.2", want: 0},
	}
	for _, tt := range tests {
		got, ok := CompareVersions(tt.left, tt.right)
		if !ok || got != tt.want {
			t.Fatalf("CompareVersions(%q, %q) = %d, %v; want %d, true", tt.left, tt.right, got, ok, tt.want)
		}
	}
}

func TestHotfixOutranksBaseRelease(t *testing.T) {
	tests := []struct {
		left  string
		right string
		want  int
	}{
		// A hotfix ranks above its base release (repository rule).
		{left: "1.0.4-hotfix1", right: "1.0.4", want: 1},
		{left: "1.0.4", right: "1.0.4-hotfix1", want: -1},
		// hotfixN ordering by number.
		{left: "1.0.4-hotfix2", right: "1.0.4-hotfix1", want: 1},
		{left: "1.0.4-hotfix1", right: "1.0.4-hotfix1", want: 0},
		{left: "1.0.4-hotfix10", right: "1.0.4-hotfix2", want: 1},
		// A later patch outranks a hotfix on the previous patch.
		{left: "1.0.5", right: "1.0.4-hotfix1", want: 1},
		{left: "1.0.4-hotfix9", right: "1.0.5", want: -1},
		// A hotfix on the same core outranks a beta/rc on that core.
		{left: "1.0.4-hotfix1", right: "1.0.4-rc.1", want: 1},
		{left: "1.0.4-beta.1", right: "1.0.4-hotfix1", want: -1},
	}
	for _, tt := range tests {
		got, ok := CompareVersions(tt.left, tt.right)
		if !ok || got != tt.want {
			t.Fatalf("CompareVersions(%q, %q) = %d, %v; want %d, true", tt.left, tt.right, got, ok, tt.want)
		}
	}
}

// The hotfix rule must not touch beta-line versions: a "beta-hotfixN"
// prerelease is a single non-numeric identifier, not the lone "hotfixN" form,
// so it keeps normal SemVer precedence and stays below the base release.
func TestBetaHotfixKeepsSemverPrecedence(t *testing.T) {
	tests := []struct {
		left  string
		right string
		want  int
	}{
		{left: "1.5.2", right: "1.5.2-beta-hotfix2", want: 1},
		{left: "1.5.2-beta-hotfix2", right: "1.5.2", want: -1},
		{left: "1.5.2-beta-hotfix2", right: "1.5.2-beta-hotfix1", want: 1},
	}
	for _, tt := range tests {
		got, ok := CompareVersions(tt.left, tt.right)
		if !ok || got != tt.want {
			t.Fatalf("CompareVersions(%q, %q) = %d, %v; want %d, true", tt.left, tt.right, got, ok, tt.want)
		}
	}
}

func TestVersionIsNewerHotfix(t *testing.T) {
	if !VersionIsNewer("1.0.4-hotfix1", "1.0.4") {
		t.Fatal("1.0.4-hotfix1 must be newer than 1.0.4")
	}
	if VersionIsNewer("1.0.4", "1.0.4-hotfix1") {
		t.Fatal("1.0.4 must NOT be newer than 1.0.4-hotfix1")
	}
	if !VersionIsNewer("1.0.5", "1.0.4-hotfix1") {
		t.Fatal("1.0.5 must be newer than 1.0.4-hotfix1")
	}
}

func TestReleasePolicyDocCoversVersionRules(t *testing.T) {
	doc, err := os.ReadFile("../docs/release-policy.md")
	if err != nil {
		t.Fatal(err)
	}
	text := string(doc)
	for _, phrase := range []string{
		"config/version",
		"MAJOR.MINOR.PATCH[-PRERELEASE]",
		"Git tag",
		"settings.version",
		"must never be downgraded",
	} {
		if !strings.Contains(text, phrase) {
			t.Fatalf("release policy doc missing %q", phrase)
		}
	}
}
