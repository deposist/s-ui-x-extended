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
