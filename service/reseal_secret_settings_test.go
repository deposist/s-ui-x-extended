package service

import (
	"testing"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
	"github.com/deposist/s-ui-x/util/secretbox"
)

// TestResealSecretSettings pins M2: a secret sealed under the DB-derived box (the
// state of a value written before SUI_SECRETBOX_KEY was adopted) is re-sealed
// under the out-of-database env box once it is configured, becoming
// unrecoverable from the database alone. The sweep is a noop without the env key
// and idempotent.
func TestResealSecretSettings(t *testing.T) {
	t.Setenv("SUI_SECRETBOX_KEY", "")
	s := initSettingTestDB(t)
	if _, err := s.GetAllSetting(); err != nil {
		t.Fatal(err)
	}

	const key = "telegramBotToken"
	const secretValue = "123456789:pre-adoption-secret-token-value"

	// Seal under the DB-derived box (candidates[0] when no env key is set) and
	// store it, emulating a value written before SUI_SECRETBOX_KEY adoption.
	dbCandidates, err := s.getSecretboxCandidates()
	if err != nil {
		t.Fatal(err)
	}
	if dbCandidates[0].name != "settings_secretbox_v1" {
		t.Fatalf("expected DB-derived preferred box without env key, got %q", dbCandidates[0].name)
	}
	legacySealed, err := dbCandidates[0].box.EncryptString(secretValue, key)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Model(model.Setting{}).Where("key = ?", key).Update("value", legacySealed).Error; err != nil {
		t.Fatal(err)
	}

	// Without the env key the sweep is a noop (the preferred box would itself be
	// DB-derived, so re-sealing would not improve at-rest protection).
	if n, err := s.ResealSecretSettings(); err != nil || n != 0 {
		t.Fatalf("reseal without SUI_SECRETBOX_KEY must be a noop: n=%d err=%v", n, err)
	}

	// Adopt a strong out-of-DB env key and re-seal.
	t.Setenv("SUI_SECRETBOX_KEY", encodedTestSecretboxKey())
	n, err := s.ResealSecretSettings()
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("expected exactly 1 row re-sealed, got %d", n)
	}

	// The stored value changed and now opens under the preferred env box (idx 0),
	// recovering the original plaintext.
	var after model.Setting
	if err := database.GetDB().Model(model.Setting{}).Where("key = ?", key).First(&after).Error; err != nil {
		t.Fatal(err)
	}
	if after.Value == legacySealed {
		t.Fatal("value was not re-sealed")
	}
	if !secretbox.IsEncrypted(after.Value) {
		t.Fatal("re-sealed value must still be encrypted at rest")
	}
	envCandidates, err := s.getSecretboxCandidates()
	if err != nil {
		t.Fatal(err)
	}
	idx, pt, ok := decryptWithCandidate(envCandidates, key, after.Value)
	if !ok || pt != secretValue {
		t.Fatalf("re-sealed value must decrypt to the original: ok=%v pt=%q", ok, pt)
	}
	if idx != 0 {
		t.Fatalf("re-sealed value must open under the preferred env box (idx 0), got %d (%s)", idx, envCandidates[idx].name)
	}

	// At-rest hardening: a DB-only adversary (the DB-derived boxes, no env key)
	// can no longer recover it.
	if _, _, ok := decryptWithCandidate(dbCandidates, key, after.Value); ok {
		t.Fatal("re-sealed value must NOT be recoverable from the DB-derived box alone")
	}

	// Idempotent: a second sweep changes nothing.
	if n, err := s.ResealSecretSettings(); err != nil || n != 0 {
		t.Fatalf("second reseal must be a noop: n=%d err=%v", n, err)
	}
}
