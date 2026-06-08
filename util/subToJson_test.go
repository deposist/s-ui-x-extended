package util

import "testing"

func TestValidateExternalURLRejectsUnsafeTargets(t *testing.T) {
	tests := []string{
		"file:///tmp/sub.txt",
		"http://localhost/sub.txt",
		"http://127.0.0.1/sub.txt",
		"http://10.0.0.1/sub.txt",
		"http://[::1]/sub.txt",
	}
	for _, rawURL := range tests {
		if err := validateExternalURL(rawURL); err == nil {
			t.Fatalf("expected %s to be rejected", rawURL)
		}
	}
}

func TestValidateExternalURLAllowsPrivateTargetsWhenExplicitlyEnabled(t *testing.T) {
	t.Setenv("SUI_ALLOW_PRIVATE_SUB_URLS", "true")
	if err := validateExternalURL("http://127.0.0.1/sub.txt"); err != nil {
		t.Fatal(err)
	}
}
