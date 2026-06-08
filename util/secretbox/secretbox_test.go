package secretbox

import "testing"

func TestEncryptDecryptRoundTrip(t *testing.T) {
	box, err := NewFromString("test-master-key")
	if err != nil {
		t.Fatal(err)
	}
	encrypted, err := box.EncryptString("telegram-token", "telegramBotToken")
	if err != nil {
		t.Fatal(err)
	}
	if encrypted == "telegram-token" || !IsEncrypted(encrypted) {
		t.Fatalf("value was not encrypted: %q", encrypted)
	}
	decrypted, err := box.DecryptString(encrypted, "telegramBotToken")
	if err != nil {
		t.Fatal(err)
	}
	if decrypted != "telegram-token" {
		t.Fatalf("unexpected plaintext %q", decrypted)
	}
}

func TestDecryptRejectsWrongAssociatedData(t *testing.T) {
	box, err := NewFromString("test-master-key")
	if err != nil {
		t.Fatal(err)
	}
	encrypted, err := box.EncryptString("telegram-token", "telegramBotToken")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := box.DecryptString(encrypted, "telegramProxyURL"); err == nil {
		t.Fatal("expected decrypt to fail with wrong associated data")
	}
}

func TestEncryptDecryptBytesRoundTrip(t *testing.T) {
	box, err := NewFromString("test-master-key")
	if err != nil {
		t.Fatal(err)
	}
	plain := []byte{0, 1, 2, 3, 255}
	encrypted, err := box.EncryptBytes(plain, "telegramBackupPassphrase")
	if err != nil {
		t.Fatal(err)
	}
	if encrypted == string(plain) || !IsEncrypted(encrypted) {
		t.Fatalf("value was not encrypted: %q", encrypted)
	}
	decrypted, err := box.DecryptBytes(encrypted, "telegramBackupPassphrase")
	if err != nil {
		t.Fatal(err)
	}
	if string(decrypted) != string(plain) {
		t.Fatalf("unexpected plaintext %v", decrypted)
	}
}

func TestNewRawKeyUsesProvidedAESKey(t *testing.T) {
	rawKey := []byte("0123456789abcdef0123456789abcdef")
	box, err := NewRawKey(rawKey)
	if err != nil {
		t.Fatal(err)
	}
	encrypted, err := box.EncryptString("telegram-token", "telegramBotToken")
	if err != nil {
		t.Fatal(err)
	}
	decrypted, err := box.DecryptString(encrypted, "telegramBotToken")
	if err != nil {
		t.Fatal(err)
	}
	if decrypted != "telegram-token" {
		t.Fatalf("unexpected plaintext %q", decrypted)
	}

	legacyBox, err := New(rawKey)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := legacyBox.DecryptString(encrypted, "telegramBotToken"); err == nil {
		t.Fatal("raw-key ciphertext should not decrypt with legacy HKDF constructor")
	}
}
