package core

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/sagernet/sing-box/common/srs"
	"github.com/sagernet/sing-box/option"
)

func writeTestRuleSet(t *testing.T, path string) {
	t.Helper()
	var buf bytes.Buffer
	ruleSet := option.PlainRuleSet{
		Rules: []option.HeadlessRule{
			{
				Type: "default",
				DefaultOptions: option.DefaultHeadlessRule{
					DomainSuffix: []string{"example.com"},
				},
			},
		},
	}
	if err := srs.Write(&buf, ruleSet, 3); err != nil {
		t.Fatalf("srs.Write: %v", err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
}

func localRuleSetConfigJSON(path string) []byte {
	path = filepath.ToSlash(path)
	return []byte(`{
		"log": {"level": "error"},
		"outbounds": [{"type": "direct", "tag": "direct"}],
		"route": {
			"rule_set": [
				{"type": "local", "tag": "geosite-test", "format": "binary", "path": "` + path + `"}
			],
			"rules": [
				{"rule_set": "geosite-test", "outbound": "direct"}
			]
		}
	}`)
}

// This is the load-bearing assumption of the whole feature: the core accepts a
// local rule-set, so the panel can materialize files itself instead of letting
// the core fetch them at startup.
func TestValidateConfigAcceptsLocalRuleSet(t *testing.T) {
	path := filepath.Join(t.TempDir(), "geosite-test.srs")
	writeTestRuleSet(t, path)
	if err := ValidateConfig(localRuleSetConfigJSON(path)); err != nil {
		t.Fatalf("a local rule-set with a valid file must validate: %v", err)
	}
}

// And the reason the panel must download before saving: a local rule-set whose
// file is absent is fatal, not a warning.
func TestValidateConfigRejectsMissingLocalRuleSetFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "absent.srs")
	if err := ValidateConfig(localRuleSetConfigJSON(path)); err == nil {
		t.Fatal("a local rule-set with a missing file must not validate")
	}
}

// The DNS schema changed across sing-box versions, so the exact shape the preset
// writes is pinned here: a DoH server as dns.final next to the regional UDP
// resolver that stays scoped to its own rule.
func TestValidateConfigAcceptsDohFinalWithRegionalResolver(t *testing.T) {
	path := filepath.Join(t.TempDir(), "geosite-test.srs")
	writeTestRuleSet(t, path)
	path = filepath.ToSlash(path)
	config := []byte(`{
		"log": {"level": "error"},
		"outbounds": [{"type": "direct", "tag": "direct"}],
		"dns": {
			"servers": [
				{"type": "udp", "tag": "preset-ru-dns-direct", "server": "77.88.8.8", "server_port": 53},
				{"type": "https", "tag": "preset-dns-proxy", "server": "1.1.1.1"}
			],
			"rules": [
				{"action": "route", "rule_set": ["geosite-test"], "server": "preset-ru-dns-direct"}
			],
			"final": "preset-dns-proxy"
		},
		"route": {
			"rule_set": [
				{"type": "local", "tag": "geosite-test", "format": "binary", "path": "` + path + `"}
			],
			"rules": [{"rule_set": "geosite-test", "outbound": "direct"}]
		}
	}`)
	if err := ValidateConfig(config); err != nil {
		t.Fatalf("preset DNS shape must validate: %v", err)
	}
}

// Documents why the preset only emits a detour it has verified: config
// validation happily accepts an unknown detour tag, but the core then fails at
// startup with "outbound detour not found". Nothing upstream of the preset will
// catch this mistake.
func TestValidateConfigDoesNotCatchUnknownDnsDetour(t *testing.T) {
	config := []byte(`{
		"log": {"level": "error"},
		"outbounds": [{"type": "direct", "tag": "direct"}],
		"dns": {
			"servers": [{"type": "https", "tag": "preset-dns-proxy", "server": "1.1.1.1", "detour": "missing-outbound"}],
			"final": "preset-dns-proxy"
		},
		"route": {"rules": []}
	}`)
	if err := ValidateConfig(config); err != nil {
		t.Skipf("validation now rejects unknown detours, the preset guard may be redundant: %v", err)
	}
}
