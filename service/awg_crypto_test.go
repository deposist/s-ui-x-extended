package service

import (
	"bytes"
	"encoding/base64"
	"errors"
	"strings"
	"testing"
)

func newTestAWGCipher(t *testing.T) *AWGCipher {
	t.Helper()
	t.Setenv(awgEncryptionKeyEnv, base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0x42}, 32)))
	cipher, err := NewAWGCipherFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	return cipher
}

func TestAWGCipherRoundTrip(t *testing.T) {
	cipher := newTestAWGCipher(t)
	context := bytes.Repeat([]byte{0x11}, awgCryptoContextSize)
	plaintext := []byte("private-key-material")

	first, err := cipher.Encrypt(7, context, plaintext)
	if err != nil {
		t.Fatal(err)
	}
	second, err := cipher.Encrypt(7, context, plaintext)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(first, second) {
		t.Fatal("random nonce did not produce distinct ciphertext")
	}
	decrypted, err := cipher.Decrypt(7, context, first)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("decrypted = %q; want %q", decrypted, plaintext)
	}
}

func TestAWGCipherRejectsWrongAADAndCorruptionWithoutLeaks(t *testing.T) {
	cipher := newTestAWGCipher(t)
	context := bytes.Repeat([]byte{0x22}, awgCryptoContextSize)
	plaintext := []byte("secret-private-key")
	ciphertext, err := cipher.Encrypt(9, context, plaintext)
	if err != nil {
		t.Fatal(err)
	}
	corrupt := append([]byte(nil), ciphertext...)
	corrupt[len(corrupt)-1] ^= 1
	wrongContext := bytes.Repeat([]byte{0x23}, awgCryptoContextSize)

	for name, decrypt := range map[string]func() error{
		"wrong client":  func() error { _, err := cipher.Decrypt(10, context, ciphertext); return err },
		"wrong context": func() error { _, err := cipher.Decrypt(9, wrongContext, ciphertext); return err },
		"corrupt":       func() error { _, err := cipher.Decrypt(9, context, corrupt); return err },
		"version": func() error {
			invalid := append([]byte(nil), ciphertext...)
			invalid[0]++
			_, err := cipher.Decrypt(9, context, invalid)
			return err
		},
	} {
		t.Run(name, func(t *testing.T) {
			err := decrypt()
			if err == nil {
				t.Fatal("invalid ciphertext was accepted")
			}
			if strings.Contains(err.Error(), string(plaintext)) || strings.Contains(err.Error(), base64.StdEncoding.EncodeToString(ciphertext)) {
				t.Fatalf("error leaked key material: %v", err)
			}
		})
	}
}

func TestNewAWGCipherFromEnvRequiresExactKey(t *testing.T) {
	for name, value := range map[string]string{
		"missing": "",
		"invalid": "not-base64",
		"short":   base64.StdEncoding.EncodeToString(make([]byte, 31)),
	} {
		t.Run(name, func(t *testing.T) {
			t.Setenv(awgEncryptionKeyEnv, value)
			_, err := NewAWGCipherFromEnv()
			if !errors.Is(err, ErrAWGEncryptionUnavailable) {
				t.Fatalf("error = %v; want ErrAWGEncryptionUnavailable", err)
			}
		})
	}
}

func TestNewAWGCryptoContext(t *testing.T) {
	first, err := NewAWGCryptoContext()
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewAWGCryptoContext()
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != awgCryptoContextSize || bytes.Equal(first, second) {
		t.Fatal("crypto contexts must be random 128-bit values")
	}
}
