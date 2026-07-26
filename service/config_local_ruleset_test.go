package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func localRuleSetConfig(path string) string {
	return `{"route":{"rule_set":[{"type":"local","tag":"geosite-ru","format":"binary","path":` +
		mustJSONString(path) + `}]}}`
}

func mustJSONString(value string) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return string(encoded)
}

// A config naming a local rule-set file that does not exist must be refused at
// save time. Accepting it would produce a saved config that cannot start the
// core, which is the exact regression this feature exists to prevent.
func TestValidateConfigLocalRuleSetsRejectsMissingFile(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "geosite-ru.srs")
	err := validateConfigLocalRuleSets(json.RawMessage(localRuleSetConfig(missing)))
	if err == nil {
		t.Fatal("a missing local rule-set file must be rejected")
	}
	if !strings.Contains(err.Error(), "geosite-ru") {
		t.Fatalf("error should name the offending tag, got: %v", err)
	}
}

// A file that exists but is not a rule-set (a downloaded HTML error page, a
// truncated transfer) is just as fatal to the core as a missing one.
func TestValidateConfigLocalRuleSetsRejectsCorruptFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "geosite-ru.srs")
	if err := os.WriteFile(path, []byte("<html>not a rule set</html>"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := validateConfigLocalRuleSets(json.RawMessage(localRuleSetConfig(path))); err == nil {
		t.Fatal("a corrupt local rule-set file must be rejected")
	}
}

func TestValidateConfigLocalRuleSetsRejectsEmptyPath(t *testing.T) {
	config := `{"route":{"rule_set":[{"type":"local","tag":"geosite-ru","format":"binary"}]}}`
	if err := validateConfigLocalRuleSets(json.RawMessage(config)); err == nil {
		t.Fatal("a local rule-set without a path must be rejected")
	}
}

func TestValidateConfigLocalRuleSetsAcceptsMaterializedFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "geosite-ru.srs")
	if err := os.WriteFile(path, validRuleSetBytes(t, "example.com"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := validateConfigLocalRuleSets(json.RawMessage(localRuleSetConfig(path))); err != nil {
		t.Fatalf("a materialized local rule-set must be accepted: %v", err)
	}
}

// Remote rule-sets are untouched by this validator; they have no local file to
// check, and rejecting them would break every existing config.
func TestValidateConfigLocalRuleSetsIgnoresRemote(t *testing.T) {
	config := `{"route":{"rule_set":[{"type":"remote","tag":"geosite-ru","format":"binary","url":"https://example.com/a.srs"}]}}`
	if err := validateConfigLocalRuleSets(json.RawMessage(config)); err != nil {
		t.Fatalf("remote rule-sets must not be affected: %v", err)
	}
}
