package common

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestPasswordHashAndMigrationChecks(t *testing.T) {
	hash, err := HashPassword("secret")
	if err != nil {
		t.Fatal(err)
	}
	if hash == "secret" || !IsPasswordHash(hash) {
		t.Fatalf("password was not hashed with expected marker: %q", hash)
	}
	if ok, migrate := CheckPassword(hash, "secret"); !ok || migrate {
		t.Fatalf("hashed password check = %v, migrate = %v", ok, migrate)
	}
	if ok, migrate := CheckPassword("secret", "secret"); !ok || !migrate {
		t.Fatalf("plain password check = %v, migrate = %v", ok, migrate)
	}
	if ok, _ := CheckPassword(hash, "wrong"); ok {
		t.Fatal("wrong password was accepted")
	}
}

// TestEqualizeLoginTimingUsesValidDummyHash pins S-2: the not-found login path
// must do the same bcrypt work as a wrong-password path, so the dummy hash has
// to be a valid bcrypt hash at DefaultCost.
func TestEqualizeLoginTimingUsesValidDummyHash(t *testing.T) {
	cost, err := bcrypt.Cost([]byte(dummyBcryptHash))
	if err != nil {
		t.Fatalf("dummy hash is not a valid bcrypt hash: %v", err)
	}
	if cost != bcrypt.DefaultCost {
		t.Fatalf("dummy hash cost = %d, want DefaultCost %d", cost, bcrypt.DefaultCost)
	}
	// Must not panic; the comparison result is intentionally discarded.
	EqualizeLoginTiming("any-password")
}
