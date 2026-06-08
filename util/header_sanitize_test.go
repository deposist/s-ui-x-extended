package util

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestSafeHeaderStripsControlsAndNUL(t *testing.T) {
	got := SafeHeader("ok\r\nInjected: bad\x00\tend", 0)
	if strings.ContainsAny(got, "\r\n\x00\t") {
		t.Fatalf("header contains control characters: %q", got)
	}
	if got != "okInjected: badend" {
		t.Fatalf("unexpected sanitized header: %q", got)
	}
}

func TestSafeHeaderTrimsWithoutSplittingUTF8(t *testing.T) {
	got := SafeHeader("абв", 5)
	if !utf8.ValidString(got) {
		t.Fatalf("header is not valid UTF-8: %q", got)
	}
	if got != "аб" {
		t.Fatalf("unexpected UTF-8 trim result: %q", got)
	}
}
