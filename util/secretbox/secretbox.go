package secretbox

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"strings"

	"github.com/deposist/s-ui-x/util/common"
	"golang.org/x/crypto/hkdf"
)

const Prefix = "sbox:v1:"

var (
	salt = []byte("s-ui secretbox v1")
	info = []byte("settings secrets")
)

type Box struct {
	aead cipher.AEAD
}

func New(masterKey []byte) (*Box, error) {
	if len(masterKey) == 0 {
		return nil, common.NewError("empty secretbox key")
	}
	key := make([]byte, 32)
	defer zeroBytes(key)
	reader := hkdf.New(sha256.New, masterKey, salt, info)
	if _, err := io.ReadFull(reader, key); err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Box{aead: aead}, nil
}

func NewRawKey(key []byte) (*Box, error) {
	if len(key) < 32 {
		return nil, common.NewError("secretbox raw key must be at least 32 bytes")
	}
	aesKey := make([]byte, 32)
	defer zeroBytes(aesKey)
	copy(aesKey, key[:32])
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Box{aead: aead}, nil
}

func NewFromString(masterKey string) (*Box, error) {
	masterKey = strings.TrimSpace(masterKey)
	if decoded, err := base64.StdEncoding.DecodeString(masterKey); err == nil && len(decoded) > 0 {
		return New(decoded)
	}
	if decoded, err := base64.RawStdEncoding.DecodeString(masterKey); err == nil && len(decoded) > 0 {
		return New(decoded)
	}
	if decoded, err := base64.RawURLEncoding.DecodeString(masterKey); err == nil && len(decoded) > 0 {
		return New(decoded)
	}
	return New([]byte(masterKey))
}

func IsEncrypted(value string) bool {
	return strings.HasPrefix(value, Prefix)
}

func (b *Box) EncryptString(plaintext string, associatedData string) (string, error) {
	plain := []byte(plaintext)
	defer zeroBytes(plain)
	return b.EncryptBytes(plain, associatedData)
}

func (b *Box) DecryptString(value string, associatedData string) (string, error) {
	plaintext, err := b.DecryptBytes(value, associatedData)
	if err != nil {
		return "", err
	}
	defer zeroBytes(plaintext)
	return string(plaintext), nil
}

func (b *Box) EncryptBytes(plaintext []byte, associatedData string) (string, error) {
	nonce := make([]byte, b.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	ciphertext := b.aead.Seal(nil, nonce, plaintext, []byte(associatedData))
	payload := append(nonce, ciphertext...)
	return Prefix + base64.RawURLEncoding.EncodeToString(payload), nil
}

func (b *Box) DecryptBytes(value string, associatedData string) ([]byte, error) {
	if !IsEncrypted(value) {
		plaintext := []byte(value)
		return plaintext, nil
	}
	payload, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(value, Prefix))
	if err != nil {
		return nil, err
	}
	nonceSize := b.aead.NonceSize()
	if len(payload) < nonceSize {
		return nil, common.NewError("invalid secretbox payload")
	}
	nonce := payload[:nonceSize]
	ciphertext := payload[nonceSize:]
	plaintext, err := b.aead.Open(nil, nonce, ciphertext, []byte(associatedData))
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}

func zeroBytes(buf []byte) {
	for i := range buf {
		buf[i] = 0
	}
}
