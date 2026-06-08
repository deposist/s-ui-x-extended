package util

import (
	"strings"
	"testing"
)

func TestValidateHostnameAcceptsValidInput(t *testing.T) {
	label63 := strings.Repeat("a", 63)
	validLength253 := strings.Join([]string{
		label63,
		label63,
		label63,
		strings.Repeat("b", 61),
	}, ".")

	tests := []string{
		"",
		"example.com",
		"admin-1.example.co.uk",
		"xn--e1afmkfd.xn--p1ai",
		"пример.рф",
		validLength253,
	}

	for _, host := range tests {
		t.Run(host, func(t *testing.T) {
			if err := ValidateHostname(host); err != nil {
				t.Fatalf("expected hostname to be accepted: %v", err)
			}
		})
	}
}

func TestValidateHostnameRejectsInvalidInput(t *testing.T) {
	label64 := strings.Repeat("a", 64)
	tooLong := strings.Join([]string{
		strings.Repeat("a", 63),
		strings.Repeat("b", 63),
		strings.Repeat("c", 63),
		strings.Repeat("d", 62),
	}, ".")

	tests := []string{
		" example.com",
		"example.com ",
		"example.com:443",
		"https://example.com",
		"bad_name.example",
		"-bad.example",
		"bad-.example",
		"bad..example",
		"bad.",
		label64 + ".example",
		tooLong,
	}

	for _, host := range tests {
		t.Run(host, func(t *testing.T) {
			if err := ValidateHostname(host); err == nil {
				t.Fatal("expected hostname to be rejected")
			}
		})
	}
}
