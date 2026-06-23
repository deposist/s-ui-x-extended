package config

import "testing"

func TestGetLogLevelFallsBackForInvalidEnv(t *testing.T) {
	t.Setenv("SUI_DEBUG", "")
	t.Setenv("SUI_LOG_LEVEL", "verbose")

	if got := GetLogLevel(); got != Info {
		t.Fatalf("GetLogLevel() = %q, want %q", got, Info)
	}
}

func TestGetLogLevelNormalizesValidEnv(t *testing.T) {
	t.Setenv("SUI_DEBUG", "")
	t.Setenv("SUI_LOG_LEVEL", " WARN ")

	if got := GetLogLevel(); got != Warn {
		t.Fatalf("GetLogLevel() = %q, want %q", got, Warn)
	}
}

func TestIsSafeLogOutputPath(t *testing.T) {
	cases := []struct {
		output string
		want   bool
	}{
		{"", true},
		{"stdout", true},
		{"stderr", true},
		{"box.log", true},
		{"logs/box.log", true},
		{"my..log", true}, // ".." as a substring of a filename is fine
		{"/etc/cron.d/s-ui", false},
		{"/var/log/s-ui/box.log", false},
		{"../../etc/passwd", false},
		{"logs/../../../etc/passwd", false},
		{"a/../b", false},
		{"..\\..\\windows", false},       // backslash traversal
		{"C:\\Windows\\system32", false}, // volume-qualified
	}
	for _, tc := range cases {
		if got := IsSafeLogOutputPath(tc.output); got != tc.want {
			t.Errorf("IsSafeLogOutputPath(%q) = %v, want %v", tc.output, got, tc.want)
		}
	}
}

// TestGetSecret pins the secret-derivation contract: an explicit SUI_SECRET is
// honored verbatim, and an empty/absent one falls back to a deterministic,
// non-empty name:db-folder derivation (no randomness, stable across calls).
func TestGetSecret(t *testing.T) {
	t.Setenv("SUI_SECRET", "top-secret-value")
	if got := GetSecret(); got != "top-secret-value" {
		t.Fatalf("GetSecret() with SUI_SECRET = %q, want %q", got, "top-secret-value")
	}

	t.Setenv("SUI_SECRET", "")
	fallback := GetSecret()
	if fallback == "" {
		t.Fatal("GetSecret() fallback is empty")
	}
	if want := GetName() + ":" + GetDBFolderPath(); fallback != want {
		t.Fatalf("GetSecret() fallback = %q, want %q", fallback, want)
	}
	if again := GetSecret(); again != fallback {
		t.Fatalf("GetSecret() fallback not deterministic: %q vs %q", again, fallback)
	}
}
