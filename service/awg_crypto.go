package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
)

const (
	awgEncryptionKeyEnv  = "AWG_KEY_ENC"
	awgCiphertextVersion = byte(1)
	awgCryptoContextSize = 16
)

var ErrAWGEncryptionUnavailable = errors.New("AWG encryption key is unavailable")

type AWGCipher struct {
	aead cipher.AEAD
	rand io.Reader
}

func NewAWGCipherFromEnv() (*AWGCipher, error) {
	encoded := os.Getenv(awgEncryptionKeyEnv)
	key, err := decodeAWGMasterKey(encoded)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("initialize AWG encryption: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("initialize AWG encryption: %w", err)
	}
	return &AWGCipher{aead: aead, rand: rand.Reader}, nil
}

func decodeAWGMasterKey(encoded string) ([]byte, error) {
	if encoded == "" {
		return nil, ErrAWGEncryptionUnavailable
	}
	key, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		key, err = base64.RawStdEncoding.DecodeString(encoded)
	}
	if err != nil || len(key) != 32 {
		return nil, fmt.Errorf("%w: AWG_KEY_ENC must encode exactly 32 bytes", ErrAWGEncryptionUnavailable)
	}
	return key, nil
}

func NewAWGCryptoContext() ([]byte, error) {
	context := make([]byte, awgCryptoContextSize)
	if _, err := io.ReadFull(rand.Reader, context); err != nil {
		return nil, fmt.Errorf("generate AWG crypto context: %w", err)
	}
	return context, nil
}

func (c *AWGCipher) Encrypt(clientID uint, cryptoContext, plaintext []byte) ([]byte, error) {
	if c == nil || c.aead == nil {
		return nil, ErrAWGEncryptionUnavailable
	}
	aad, err := awgCipherAAD(clientID, cryptoContext)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := io.ReadFull(c.rand, nonce); err != nil {
		return nil, fmt.Errorf("encrypt AWG key material: generate nonce: %w", err)
	}
	ciphertext := make([]byte, 1, 1+len(nonce)+len(plaintext)+c.aead.Overhead())
	ciphertext[0] = awgCiphertextVersion
	ciphertext = append(ciphertext, nonce...)
	return c.aead.Seal(ciphertext, nonce, plaintext, aad), nil
}

func (c *AWGCipher) Decrypt(clientID uint, cryptoContext, ciphertext []byte) ([]byte, error) {
	if c == nil || c.aead == nil {
		return nil, ErrAWGEncryptionUnavailable
	}
	if len(ciphertext) < 1+c.aead.NonceSize()+c.aead.Overhead() || ciphertext[0] != awgCiphertextVersion {
		return nil, errors.New("invalid AWG encrypted key material")
	}
	aad, err := awgCipherAAD(clientID, cryptoContext)
	if err != nil {
		return nil, err
	}
	nonceEnd := 1 + c.aead.NonceSize()
	plaintext, err := c.aead.Open(nil, ciphertext[1:nonceEnd], ciphertext[nonceEnd:], aad)
	if err != nil {
		return nil, errors.New("invalid AWG encrypted key material")
	}
	return plaintext, nil
}

func awgCipherAAD(clientID uint, cryptoContext []byte) ([]byte, error) {
	if len(cryptoContext) != awgCryptoContextSize {
		return nil, errors.New("invalid AWG crypto context")
	}
	aad := make([]byte, 1+8+len(cryptoContext))
	aad[0] = awgCiphertextVersion
	binary.BigEndian.PutUint64(aad[1:9], uint64(clientID))
	copy(aad[9:], cryptoContext)
	return aad, nil
}
