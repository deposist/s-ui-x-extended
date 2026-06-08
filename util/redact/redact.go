package redact

import (
	"regexp"
	"strings"
)

const Marker = "[REDACTED]"

var sensitiveKeyFragments = []string{
	"authorization",
	"cookie",
	"passphrase",
	"password",
	"private",
	"proxy",
	"secret",
	"token",
	"access_key",
	"client_secret",
	"subscription",
}

var sensitiveExactKeys = []string{
	"otp",
	"totp",
	"mfa",
	"2fa",
}

var sensitiveValuePatterns = []struct {
	pattern     *regexp.Regexp
	replacement string
}{
	{
		pattern:     regexp.MustCompile(`\b\d{8,10}:[A-Za-z0-9_-]{35}\b`),
		replacement: Marker,
	},
	{
		// Strip inline URL credentials: scheme://user:pass@host -> scheme://[REDACTED]@host
		pattern:     regexp.MustCompile(`([a-zA-Z][a-zA-Z0-9+.\-]*://)[^/?#\s:@]+:[^/?#\s@]+@`),
		replacement: `${1}` + Marker + `@`,
	},
	{
		pattern:     regexp.MustCompile(`(?i)(\bAuthorization\s*:\s*Bearer\s+)[^\s,;]+`),
		replacement: `${1}` + Marker,
	},
	{
		pattern:     regexp.MustCompile(`(?i)(\bToken\s*:\s*)[^\s,;]+`),
		replacement: `${1}` + Marker,
	},
	{
		pattern:     regexp.MustCompile(`(?i)(\b(?:totp|otp|mfa|2fa|secret|otp[_-]?secret|totp[_-]?secret|two[_-]?factor(?:[_-]?secret)?)\b["']?\s*[:=]\s*["']?)\b[A-Z2-7]{32}\b(["']?)`),
		replacement: `${1}` + Marker + `${2}`,
	},
}

func Value(value any) any {
	switch v := value.(type) {
	case map[string]any:
		redacted := make(map[string]any, len(v))
		for key, item := range v {
			if IsSensitiveKey(key) {
				redacted[key] = Marker
				continue
			}
			redacted[key] = Value(item)
		}
		return redacted
	case map[string]string:
		redacted := make(map[string]string, len(v))
		for key, item := range v {
			if IsSensitiveKey(key) {
				redacted[key] = Marker
				continue
			}
			redacted[key] = String(item)
		}
		return redacted
	case []any:
		redacted := make([]any, len(v))
		for i, item := range v {
			redacted[i] = Value(item)
		}
		return redacted
	case []string:
		redacted := make([]string, len(v))
		for i, item := range v {
			redacted[i] = String(item)
		}
		return redacted
	case string:
		return String(v)
	default:
		return value
	}
}

func String(value string) string {
	for _, item := range sensitiveValuePatterns {
		value = item.pattern.ReplaceAllString(value, item.replacement)
	}
	return value
}

func IsSensitiveKey(key string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(key, "-", "_"))
	for _, exact := range sensitiveExactKeys {
		if normalized == exact {
			return true
		}
	}
	for _, fragment := range sensitiveKeyFragments {
		if strings.Contains(normalized, fragment) {
			return true
		}
	}
	return false
}
