package service

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func validTestAWGKey() string {
	return base64.StdEncoding.EncodeToString(make([]byte, 32))
}

func provisionEnvFile(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "secretbox.env")
}

func TestEnsureAWGEncryptionKeyGeneratesAndIsStable(t *testing.T) {
	t.Setenv(awgEncryptionKeyEnv, "")
	envFile := provisionEnvFile(t)
	t.Setenv(awgEnvFileOverride, envFile)

	if err := EnsureAWGEncryptionKey(); err != nil {
		t.Fatal(err)
	}
	first := os.Getenv(awgEncryptionKeyEnv)
	if _, err := decodeAWGMasterKey(first); err != nil {
		t.Fatalf("generated key is invalid: %v", err)
	}
	raw, err := os.ReadFile(envFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), awgEncryptionKeyEnv+"="+first) {
		t.Fatalf("env file does not contain the generated key: %s", raw)
	}

	// A restart must reuse the persisted key instead of rotating it.
	t.Setenv(awgEncryptionKeyEnv, "")
	if err := EnsureAWGEncryptionKey(); err != nil {
		t.Fatal(err)
	}
	if os.Getenv(awgEncryptionKeyEnv) != first {
		t.Fatal("existing key was rotated on the second run")
	}
	again, err := os.ReadFile(envFile)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(again), awgEncryptionKeyEnv+"=") != 1 {
		t.Fatalf("duplicate key entries written: %s", again)
	}
}

func TestEnsureAWGEncryptionKeyKeepsEnvValue(t *testing.T) {
	key := validTestAWGKey()
	t.Setenv(awgEncryptionKeyEnv, key)
	envFile := provisionEnvFile(t)
	t.Setenv(awgEnvFileOverride, envFile)

	if err := EnsureAWGEncryptionKey(); err != nil {
		t.Fatal(err)
	}
	if os.Getenv(awgEncryptionKeyEnv) != key {
		t.Fatal("valid environment key was replaced")
	}
	if _, err := os.Stat(envFile); !os.IsNotExist(err) {
		t.Fatalf("env file created although the environment already had a key: %v", err)
	}
}

func TestEnsureAWGEncryptionKeyLoadsExistingFromFile(t *testing.T) {
	t.Setenv(awgEncryptionKeyEnv, "")
	key := validTestAWGKey()
	envFile := provisionEnvFile(t)
	content := "SUI_SECRETBOX_KEY=other-value\n" + awgEncryptionKeyEnv + "=" + key + "\n"
	if err := os.WriteFile(envFile, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(awgEnvFileOverride, envFile)

	if err := EnsureAWGEncryptionKey(); err != nil {
		t.Fatal(err)
	}
	if os.Getenv(awgEncryptionKeyEnv) != key {
		t.Fatal("key from the environment file was not loaded")
	}
	raw, err := os.ReadFile(envFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != content {
		t.Fatalf("environment file was modified although it had a valid key: %s", raw)
	}
}

func TestEnsureAWGEncryptionKeyReplacesInvalidEntry(t *testing.T) {
	t.Setenv(awgEncryptionKeyEnv, "")
	envFile := provisionEnvFile(t)
	if err := os.WriteFile(envFile, []byte("KEEP=1\n"+awgEncryptionKeyEnv+"=not-base64!!!\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(awgEnvFileOverride, envFile)

	if err := EnsureAWGEncryptionKey(); err != nil {
		t.Fatal(err)
	}
	if _, err := decodeAWGMasterKey(os.Getenv(awgEncryptionKeyEnv)); err != nil {
		t.Fatalf("replacement key is invalid: %v", err)
	}
	raw, err := os.ReadFile(envFile)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	if !strings.Contains(text, "KEEP=1") {
		t.Fatalf("unrelated entries were dropped: %s", text)
	}
	if strings.Count(text, awgEncryptionKeyEnv+"=") != 1 || strings.Contains(text, "not-base64") {
		t.Fatalf("stale entry was not replaced: %s", text)
	}
}
