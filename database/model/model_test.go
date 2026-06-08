package model

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestTokensJSONDoesNotExposePlaintextToken(t *testing.T) {
	token := Tokens{
		Id:          1,
		Desc:        "api token",
		Token:       "sensitive-secret-token",
		TokenHash:   "bcrypt-hash",
		TokenPrefix: "prefix-only",
		Scope:       "admin",
		Enabled:     true,
	}

	data, err := json.Marshal(token)
	if err != nil {
		t.Fatalf("marshal token: %v", err)
	}

	if strings.Contains(string(data), token.Token) {
		t.Fatalf("marshaled token contains plaintext token: %s", data)
	}

	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal token JSON: %v", err)
	}
	if _, ok := got["token"]; ok {
		t.Fatalf("token field should not be serialized: %s", data)
	}
	if _, ok := got["tokenHash"]; ok {
		t.Fatalf("tokenHash field should not be serialized: %s", data)
	}
	if got["tokenPrefix"] != token.TokenPrefix {
		t.Fatalf("tokenPrefix should remain available, got %v", got["tokenPrefix"])
	}
}
