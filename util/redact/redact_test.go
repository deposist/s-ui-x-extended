package redact

import (
	"strings"
	"testing"
)

func TestValueRedactsSensitiveKeys(t *testing.T) {
	input := map[string]any{
		"user":                     "admin",
		"token":                    "secret-token",
		"telegramBackupPassphrase": "secret-passphrase",
		"nested": map[string]any{
			"password": "secret-password",
			"port":     2095,
		},
	}
	redacted := Value(input).(map[string]any)
	if redacted["user"] != "admin" {
		t.Fatalf("non-secret field changed: %#v", redacted["user"])
	}
	if redacted["token"] != Marker {
		t.Fatalf("token was not redacted: %#v", redacted["token"])
	}
	if redacted["telegramBackupPassphrase"] != Marker {
		t.Fatalf("passphrase was not redacted: %#v", redacted["telegramBackupPassphrase"])
	}
	nested := redacted["nested"].(map[string]any)
	if nested["password"] != Marker {
		t.Fatalf("password was not redacted: %#v", nested["password"])
	}
	if nested["port"] != 2095 {
		t.Fatalf("non-secret nested field changed: %#v", nested["port"])
	}
}

// TestValueRedactsProxyKeys pins S-e: proxy-URL secret settings (which can carry
// embedded user:pass credentials) are masked by key. "proxy" is a conservative
// fragment — proxy config keys are redacted in logs by design.
func TestValueRedactsProxyKeys(t *testing.T) {
	input := map[string]any{
		"telegramProxyURL": "socks5://user:pass@10.0.0.1:1080",
		"paidSubProxyURL":  "http://user:pass@proxy.local:3128",
		"proxyType":        "socks5",
		"webPort":          2095,
	}
	redacted := Value(input).(map[string]any)
	if redacted["telegramProxyURL"] != Marker {
		t.Fatalf("telegramProxyURL was not redacted: %#v", redacted["telegramProxyURL"])
	}
	if redacted["paidSubProxyURL"] != Marker {
		t.Fatalf("paidSubProxyURL was not redacted: %#v", redacted["paidSubProxyURL"])
	}
	if redacted["proxyType"] != Marker {
		t.Fatalf("proxy* keys are conservatively redacted: %#v", redacted["proxyType"])
	}
	if redacted["webPort"] != 2095 {
		t.Fatalf("non-secret field changed: %#v", redacted["webPort"])
	}
}

func TestValueRedactsSensitiveStringValues(t *testing.T) {
	botToken := "1234567890:" + strings.Repeat("A", 35)
	base32Secret := "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567"
	input := map[string]any{
		"message": "Authorization: Bearer secret.jwt.value",
		"nested": map[string]any{
			"caption": "telegram token " + botToken,
			"codes":   []string{"Token: legacy-token", "totp=" + base32Secret},
		},
	}
	redacted := Value(input).(map[string]any)
	if got := redacted["message"].(string); got != "Authorization: Bearer "+Marker {
		t.Fatalf("authorization header was not redacted: %q", got)
	}
	nested := redacted["nested"].(map[string]any)
	if got := nested["caption"].(string); strings.Contains(got, botToken) || !strings.Contains(got, Marker) {
		t.Fatalf("telegram token was not redacted: %q", got)
	}
	codes := nested["codes"].([]string)
	if codes[0] != "Token: "+Marker {
		t.Fatalf("legacy token header was not redacted: %q", codes[0])
	}
	if codes[1] != "totp="+Marker {
		t.Fatalf("base32 secret was not redacted: %q", codes[1])
	}
}

func TestValueRedactsMapStringString(t *testing.T) {
	input := map[string]string{
		"plain":        "Token: legacy-token",
		"refreshToken": "secret",
	}
	redacted := Value(input).(map[string]string)
	if redacted["plain"] != "Token: "+Marker {
		t.Fatalf("plain string value was not redacted: %q", redacted["plain"])
	}
	if redacted["refreshToken"] != Marker {
		t.Fatalf("sensitive key was not redacted: %q", redacted["refreshToken"])
	}
}

func TestStringRedactsContextualTOTPSecrets(t *testing.T) {
	base32Secret := "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567"

	tests := map[string]string{
		"totp=" + base32Secret:                           "totp=" + Marker,
		`"otp":"` + base32Secret + `"`:                   `"otp":"` + Marker + `"`,
		"two_factor_secret: " + base32Secret:             "two_factor_secret: " + Marker,
		"secret='" + strings.ToLower(base32Secret) + "'": "secret='" + Marker + "'",
	}

	for input, want := range tests {
		t.Run(input, func(t *testing.T) {
			if got := String(input); got != want {
				t.Fatalf("unexpected redaction: %q, want %q", got, want)
			}
		})
	}
}

func TestStringDoesNotRedactStandaloneBase32Identifiers(t *testing.T) {
	base32Secret := "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567"
	inputs := []string{
		base32Secret,
		"geo_id=" + base32Secret,
		"uuid_base32 " + base32Secret,
		"ABCDEFGHIJKLMNOPQRSTUVWXYZ234567ABCDEFGHIJKLMNOPQRSTUVWXYZ234567",
	}

	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			if got := String(input); got != input {
				t.Fatalf("standalone base32 value was redacted: %q -> %q", input, got)
			}
		})
	}
}

func TestValueRedactsExactOTPKeys(t *testing.T) {
	redacted := Value(map[string]any{
		"otp":       "123456",
		"totp":      "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567",
		"biotope":   "not-sensitive",
		"totpLabel": "not-sensitive",
	}).(map[string]any)

	if redacted["otp"] != Marker || redacted["totp"] != Marker {
		t.Fatalf("otp/totp keys were not redacted: %#v", redacted)
	}
	if redacted["biotope"] != "not-sensitive" || redacted["totpLabel"] != "not-sensitive" {
		t.Fatalf("non-exact otp/totp keys were redacted: %#v", redacted)
	}
}

// TestStringRedactsURLUserinfo pins H-10: inline credentials in a URL that
// reaches a free-text log line (not as a sensitive-keyed value) are masked,
// keeping scheme and host so the message stays useful.
func TestStringRedactsURLUserinfo(t *testing.T) {
	tests := map[string]string{
		"dial socks5://user:pass@10.0.0.1:1080 failed": "dial socks5://" + Marker + "@10.0.0.1:1080 failed",
		"http://admin:s3cr3t@proxy.local:3128/path":    "http://" + Marker + "@proxy.local:3128/path",
		"https://plain.example.com:8443/no-creds":      "https://plain.example.com:8443/no-creds",
	}
	for input, want := range tests {
		t.Run(input, func(t *testing.T) {
			if got := String(input); got != want {
				t.Fatalf("unexpected redaction: %q, want %q", got, want)
			}
		})
	}
}
