package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x/service"
)

func TestDecryptBackupCommandRoundTripWithEnvPassphrase(t *testing.T) {
	dir := t.TempDir()
	payload := []byte("sqlite payload bytes")
	passphrase := "correct horse battery staple"
	inPath := filepath.Join(dir, "backup.db.aes")
	outPath := filepath.Join(dir, "backup.db")
	envelope, err := service.BuildTelegramBackupEnvelope(payload, []byte(passphrase))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(inPath, envelope, 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := runDecryptBackup([]string{"--in", inPath, "--out", outPath, "--passphrase-env", "SUI_TEST_PASSPHRASE"}, strings.NewReader(""), &stdout, &stderr, func(name string) string {
		if name == "SUI_TEST_PASSPHRASE" {
			return passphrase
		}
		return ""
	})
	if code != 0 {
		t.Fatalf("unexpected exit code %d stderr=%s", code, stderr.String())
	}
	got, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("decrypted payload mismatch: %q", got)
	}
	if strings.Contains(stdout.String(), passphrase) || strings.Contains(stderr.String(), passphrase) {
		t.Fatalf("passphrase leaked to command output stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestDecryptBackupCommandRoundTripWithStdinPassphrase(t *testing.T) {
	dir := t.TempDir()
	payload := []byte("payload from stdin passphrase")
	passphrase := "correct horse battery staple"
	inPath := filepath.Join(dir, "backup.db.aes")
	outPath := filepath.Join(dir, "backup.db")
	envelope, err := service.BuildTelegramBackupEnvelope(payload, []byte(passphrase))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(inPath, envelope, 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := runDecryptBackup([]string{"--in", inPath, "--out", outPath, "--passphrase-stdin"}, strings.NewReader(passphrase+"\n"), &stdout, &stderr, os.Getenv)
	if code != 0 {
		t.Fatalf("unexpected exit code %d stderr=%s", code, stderr.String())
	}
	got, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("decrypted payload mismatch: %q", got)
	}
}

func TestDecryptBackupCommandWrongPassphraseRemovesPartialOutput(t *testing.T) {
	dir := t.TempDir()
	passphrase := "correct horse battery staple"
	inPath := filepath.Join(dir, "backup.db.aes")
	outPath := filepath.Join(dir, "backup.db")
	envelope, err := service.BuildTelegramBackupEnvelope([]byte("payload"), []byte(passphrase))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(inPath, envelope, 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := runDecryptBackup([]string{"--in", inPath, "--out", outPath, "--passphrase-env", "SUI_TEST_PASSPHRASE"}, strings.NewReader(""), &stdout, &stderr, func(name string) string {
		return "wrong horse battery staple"
	})
	if code == 0 {
		t.Fatal("wrong passphrase should fail")
	}
	if _, err := os.Stat(outPath); !os.IsNotExist(err) {
		t.Fatalf("partial output was not removed, stat err=%v", err)
	}
	if strings.Contains(stdout.String(), passphrase) || strings.Contains(stderr.String(), passphrase) || strings.Contains(stderr.String(), "wrong horse") {
		t.Fatalf("passphrase leaked to command output stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestDecryptBackupCommandRejectsPassphraseArgument(t *testing.T) {
	dir := t.TempDir()
	inPath := filepath.Join(dir, "backup.db.aes")
	outPath := filepath.Join(dir, "backup.db")
	if err := os.WriteFile(inPath, []byte("not an envelope"), 0o600); err != nil {
		t.Fatal(err)
	}
	passphrase := "correct horse battery staple"
	var stdout, stderr bytes.Buffer
	code := runDecryptBackup([]string{"--in", inPath, "--out", outPath, "--passphrase", passphrase}, strings.NewReader(""), &stdout, &stderr, os.Getenv)
	if code != 2 {
		t.Fatalf("unexpected exit code %d stderr=%s", code, stderr.String())
	}
	if strings.Contains(stdout.String(), passphrase) || strings.Contains(stderr.String(), passphrase) {
		t.Fatalf("passphrase argv leaked to command output stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}
