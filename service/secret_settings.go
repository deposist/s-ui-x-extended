package service

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
	"github.com/deposist/s-ui-x/logger"
	"github.com/deposist/s-ui-x/util/common"
	"github.com/deposist/s-ui-x/util/secretbox"
	"golang.org/x/crypto/hkdf"
)

var (
	secretboxFallbackWarning sync.Once
	secretboxInvalidWarning  sync.Once
	cookieKeyFallbackWarning sync.Once
	cookieKeyInvalidWarning  sync.Once

	encryptedSettingKeys = map[string]struct{}{
		"telegramBackupPassphrase": {},
		"telegramBotToken":         {},
		"telegramProxyPassword":    {},
		"telegramProxyURL":         {},
		"telegramProxyUsername":    {},
		"paidSubBotToken":          {},
		"paidSubYooKassaToken":     {},
		"paidSubStripeToken":       {},
		"paidSubPayMasterToken":    {},
		"paidSubCryptoBotToken":    {},
		"paidSubProxyURL":          {},
		"paidSubProxyUsername":     {},
		"paidSubProxyPassword":     {},
	}
)

// #nosec G101 -- UI placeholder text shown in place of a stored secret, not a credential.
const StoredSecretMarker = "••• stored •••"

var (
	cookieKeyHKDFInfo            = []byte("sui:cookie:v1")
	settingsSecretboxKeyHKDFInfo = []byte("sui:settings-secretbox:v1")
	legacyCookieKeyHKDFSalt      = []byte("s-ui session cookie v1")
	legacyCookieKeyHKDFInfo      = []byte("cookie signing key")
)

type secretboxCandidate struct {
	name string
	box  *secretbox.Box
}

func isEncryptedSettingKey(key string) bool {
	_, ok := encryptedSettingKeys[key]
	return ok
}

func writeSecretSettingMarker(settings map[string]string, key string, value string) {
	settings[key+"HasSecret"] = strconv.FormatBool(value != "")
	if key == "telegramBackupPassphrase" {
		if value == "" {
			settings[key] = ""
		} else {
			settings[key] = StoredSecretMarker
		}
	}
}

func (s *SettingService) getSecretbox() (*secretbox.Box, error) {
	candidates, err := s.getSecretboxCandidates()
	if err != nil {
		return nil, err
	}
	if len(candidates) == 0 {
		return nil, common.NewError("no secretbox key candidates")
	}
	return candidates[0].box, nil
}

func (s *SettingService) getSecretboxCandidates() ([]secretboxCandidate, error) {
	if key := strings.TrimSpace(os.Getenv("SUI_SECRETBOX_KEY")); key != "" {
		parsed, err := parseEnvKeyMaterial(key, 32)
		if err == nil {
			box, err := secretbox.NewRawKey(parsed)
			if err != nil {
				return nil, err
			}
			legacyEnvBox, err := secretbox.New(parsed)
			if err != nil {
				return nil, err
			}
			candidates := []secretboxCandidate{
				{name: "env_raw", box: box},
				{name: "legacy_env_hkdf", box: legacyEnvBox},
			}
			settingsSecretCandidates, err := s.settingsSecretboxCandidates()
			if err != nil {
				return nil, err
			}
			return append(candidates, settingsSecretCandidates...), nil
		}
		secretboxInvalidWarning.Do(func() {
			logger.Warning("SUI_SECRETBOX_KEY is invalid:", err, "; encrypted settings use HKDF-derived settings.secret key")
		})
	}
	secretboxFallbackWarning.Do(func() {
		logger.Warning("SUI_SECRETBOX_KEY is not set or invalid; encrypted settings use HKDF-derived settings.secret key")
	})
	return s.settingsSecretboxCandidates()
}

func (s *SettingService) settingsSecretboxCandidates() ([]secretboxCandidate, error) {
	secret, err := s.GetSecret()
	if err != nil {
		return nil, err
	}
	primaryKey, err := deriveHKDFKey(secret, nil, settingsSecretboxKeyHKDFInfo)
	if err != nil {
		return nil, err
	}
	primaryBox, err := secretbox.NewRawKey(primaryKey)
	zeroBytes(primaryKey)
	if err != nil {
		return nil, err
	}
	legacyBox, err := secretbox.New(secret)
	if err != nil {
		return nil, err
	}
	return []secretboxCandidate{
		{name: "settings_secretbox_v1", box: primaryBox},
		{name: "legacy_settings_secret", box: legacyBox},
	}, nil
}

func (s *SettingService) GetCookieKeys() ([][]byte, error) {
	if raw := strings.TrimSpace(os.Getenv("SUI_COOKIE_KEY")); raw != "" {
		keys, err := parseEnvKeyList(raw, 32)
		if err == nil {
			return keys, nil
		}
		cookieKeyInvalidWarning.Do(func() {
			logger.Warning("SUI_COOKIE_KEY is invalid:", err, "; using HKDF-derived compatibility key from settings.secret")
		})
	} else {
		cookieKeyFallbackWarning.Do(func() {
			logger.Warning("SUI_COOKIE_KEY is not set; using HKDF-derived compatibility key from settings.secret")
		})
	}

	secret, err := s.GetSecret()
	if err != nil {
		return nil, err
	}
	cookieKey, err := deriveHKDFKey(secret, nil, cookieKeyHKDFInfo)
	if err != nil {
		return nil, err
	}
	legacyCookieKey, err := deriveHKDFKey(secret, legacyCookieKeyHKDFSalt, legacyCookieKeyHKDFInfo)
	if err != nil {
		zeroBytes(cookieKey)
		return nil, err
	}
	keys := appendUniqueKey(nil, cookieKey)
	keys = appendUniqueKey(keys, legacyCookieKey)
	keys = appendUniqueKey(keys, secret)
	return keys, nil
}

func parseEnvKeyList(raw string, minLen int) ([][]byte, error) {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n' || r == '\r'
	})
	keys := make([][]byte, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		key, err := parseEnvKeyMaterial(part, minLen)
		if err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	if len(keys) == 0 {
		return nil, common.NewError("empty key list")
	}
	return keys, nil
}

func parseEnvKeyMaterial(value string, minLen int) ([]byte, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, common.NewError("empty key material")
	}
	if decoded, ok := decodeKeyMaterial(value); ok {
		if len(decoded) < minLen {
			return nil, common.NewErrorf("decoded key length %d is smaller than %d bytes", len(decoded), minLen)
		}
		return decoded, nil
	}
	return nil, common.NewError("key material must be base64-encoded")
}

func decodeKeyMaterial(value string) ([]byte, bool) {
	if decoded, err := base64.StdEncoding.DecodeString(value); err == nil && len(decoded) > 0 {
		return decoded, true
	}
	if decoded, err := base64.RawStdEncoding.DecodeString(value); err == nil && len(decoded) > 0 {
		return decoded, true
	}
	if decoded, err := base64.RawURLEncoding.DecodeString(value); err == nil && len(decoded) > 0 {
		return decoded, true
	}
	return nil, false
}

// hkdfDerivedKeyLen is the byte length of every HKDF-derived key in this
// package (secretbox and cookie keys are all 32-byte / AES-256 sized).
const hkdfDerivedKeyLen = 32

func deriveHKDFKey(masterKey []byte, salt []byte, info []byte) ([]byte, error) {
	if len(masterKey) == 0 {
		return nil, common.NewError("empty master key")
	}
	key := make([]byte, hkdfDerivedKeyLen)
	reader := hkdf.New(sha256.New, masterKey, salt, info)
	if _, err := io.ReadFull(reader, key); err != nil {
		return nil, err
	}
	return key, nil
}

func appendUniqueKey(keys [][]byte, key []byte) [][]byte {
	if len(key) == 0 {
		return keys
	}
	for _, existing := range keys {
		if bytes.Equal(existing, key) {
			return keys
		}
	}
	return append(keys, key)
}

func (s *SettingService) encryptSettingValue(key string, value string) (string, error) {
	if value == "" || secretbox.IsEncrypted(value) {
		return value, nil
	}
	box, err := s.getSecretbox()
	if err != nil {
		return "", err
	}
	return box.EncryptString(value, key)
}

func (s *SettingService) decryptSettingValue(key string, value string) (string, error) {
	if value == "" || !secretbox.IsEncrypted(value) {
		return value, nil
	}
	candidates, err := s.getSecretboxCandidates()
	if err != nil {
		return "", err
	}
	for i, candidate := range candidates {
		plaintext, err := candidate.box.DecryptString(value, key)
		if err == nil {
			if i > 0 {
				s.recordSecretboxFallback(key, candidate.name)
			}
			return plaintext, nil
		}
	}
	return "", common.NewError("secret setting decrypt failed")
}

func (s *SettingService) decryptSettingBytes(key string, value string) ([]byte, error) {
	if value == "" {
		return nil, nil
	}
	if !secretbox.IsEncrypted(value) {
		return []byte(value), nil
	}
	candidates, err := s.getSecretboxCandidates()
	if err != nil {
		return nil, err
	}
	for i, candidate := range candidates {
		plaintext, err := candidate.box.DecryptBytes(value, key)
		if err == nil {
			if i > 0 {
				s.recordSecretboxFallback(key, candidate.name)
			}
			return plaintext, nil
		}
	}
	return nil, common.NewError("secret setting decrypt failed")
}

func (s *SettingService) recordSecretboxFallback(key string, candidate string) {
	if database.GetDB() == nil {
		return
	}
	if !secretboxFallbackAuditContext() {
		return
	}
	if err := (&AuditService{}).Record(AuditEvent{
		Event:    "settings_secretbox_key_fallback",
		Resource: "settings",
		Severity: AuditSeverityWarn,
		Details: map[string]any{
			"key":       key,
			"candidate": candidate,
		},
	}); err != nil {
		logger.Warning("secretbox fallback audit failed:", err)
	}
}

func secretboxFallbackAuditContext() bool {
	var pcs [8]uintptr
	n := runtime.Callers(3, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])
	for {
		frame, more := frames.Next()
		if strings.HasSuffix(frame.Function, "(*SettingService).getString") {
			return true
		}
		if !more {
			return false
		}
	}
}

func zeroBytes(buf []byte) {
	for i := range buf {
		buf[i] = 0
	}
}

// decryptWithCandidate returns the index of the first secretbox candidate that
// opens value, the recovered plaintext, and whether any candidate succeeded.
func decryptWithCandidate(candidates []secretboxCandidate, key, value string) (int, string, bool) {
	for i, candidate := range candidates {
		if plaintext, err := candidate.box.DecryptString(value, key); err == nil {
			return i, plaintext, true
		}
	}
	return -1, "", false
}

// ResealSecretSettings re-encrypts every encrypted secret setting that still
// opens under a non-preferred (DB-derived) box so it becomes recoverable only
// with the out-of-database SUI_SECRETBOX_KEY. This closes the gap where a value
// written before SUI_SECRETBOX_KEY was adopted stays decryptable from the
// database alone (its key is derived from the plaintext settings.secret row).
//
// It is:
//   - a NO-OP when SUI_SECRETBOX_KEY is unset (candidates[0] would itself be the
//     DB-derived box, so re-sealing would not improve at-rest protection);
//   - idempotent (a row already sealed under candidates[0] is skipped);
//   - fail-safe per row (a row that decrypts under no candidate is left
//     untouched, never corrupted).
//
// A row is only rewritten after it successfully decrypts, and it is re-sealed
// with that exact recovered plaintext, so the round-trip cannot lose data. Once
// re-sealed under SUI_SECRETBOX_KEY a value can no longer be recovered from the
// database alone — that is the intended hardening. Returns the rows re-sealed.
func (s *SettingService) ResealSecretSettings() (int, error) {
	if strings.TrimSpace(os.Getenv("SUI_SECRETBOX_KEY")) == "" {
		return 0, nil
	}
	db := database.GetDB()
	if db == nil {
		return 0, nil
	}
	candidates, err := s.getSecretboxCandidates()
	if err != nil {
		return 0, err
	}
	if len(candidates) == 0 {
		return 0, nil
	}
	resealed := 0
	for key := range encryptedSettingKeys {
		var setting model.Setting
		if err := db.Model(model.Setting{}).Where("key = ?", key).First(&setting).Error; err != nil {
			if !database.IsNotFound(err) {
				logger.Warning("reseal secret setting: read failed for", key, ":", err)
			}
			continue
		}
		if setting.Value == "" || !secretbox.IsEncrypted(setting.Value) {
			continue
		}
		idx, plaintext, ok := decryptWithCandidate(candidates, key, setting.Value)
		if !ok {
			logger.Warning("reseal secret setting: no candidate decrypts", key, "; leaving as-is")
			continue
		}
		if idx == 0 {
			continue // already under the preferred out-of-DB box
		}
		sealed, err := candidates[0].box.EncryptString(plaintext, key)
		if err != nil {
			logger.Warning("reseal secret setting: re-encrypt failed for", key, ":", err)
			continue
		}
		if err := db.Model(model.Setting{}).Where("key = ?", key).Update("value", sealed).Error; err != nil {
			logger.Warning("reseal secret setting: persist failed for", key, ":", err)
			continue
		}
		resealed++
	}
	// The caller (app.Init) logs the re-sealed count; avoid a duplicate line here.
	return resealed, nil
}
