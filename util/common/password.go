package common

import (
	"crypto/subtle"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

const passwordHashPrefix = "bcrypt:"

// dummyBcryptHash is a valid bcrypt hash at bcrypt.DefaultCost. Its plaintext is
// irrelevant — it exists only so the login "user not found" path can perform the
// same bcrypt work as the "wrong password" path, defeating username enumeration
// via a timing side-channel.
const dummyBcryptHash = "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"

// EqualizeLoginTiming performs a throwaway bcrypt comparison so that an
// unknown-username login costs roughly the same as a known-username/wrong-password
// login. The result is intentionally discarded; call it on the not-found path.
func EqualizeLoginTiming(password string) {
	_ = bcrypt.CompareHashAndPassword([]byte(dummyBcryptHash), []byte(password))
}

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return passwordHashPrefix + string(hash), nil
}

func IsPasswordHash(password string) bool {
	return strings.HasPrefix(password, passwordHashPrefix) || strings.HasPrefix(password, "$2a$") ||
		strings.HasPrefix(password, "$2b$") || strings.HasPrefix(password, "$2y$")
}

func CheckPassword(storedPassword string, password string) (bool, bool) {
	if strings.HasPrefix(storedPassword, passwordHashPrefix) {
		hash := strings.TrimPrefix(storedPassword, passwordHashPrefix)
		return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil, false
	}
	if IsPasswordHash(storedPassword) {
		return bcrypt.CompareHashAndPassword([]byte(storedPassword), []byte(password)) == nil, true
	}
	// Legacy plaintext path (only until the stored password is migrated to
	// bcrypt). Use a constant-time compare so the match does not leak via timing.
	return subtle.ConstantTimeCompare([]byte(storedPassword), []byte(password)) == 1, true
}
