package util

import (
	"encoding/json"
	"testing"
)

// TestGroupConfigsDoNotLeakSecrets verifies that panel-assembled group
// configurations (selector, urltest, fallback, failover-as-selector) do not
// contain any server-only secret fields. Groups only carry tag references and
// policy metadata, never credentials or private keys.
func TestGroupConfigsDoNotLeakSecrets(t *testing.T) {
	groupConfigs := map[string]string{
		"selector":           `{"type":"selector","tag":"g","outbounds":["a","b"],"default":"a","interrupt_exist_connections":true}`,
		"urltest":            `{"type":"urltest","tag":"g","outbounds":["a","b"],"url":"https://www.gstatic.com/generate_204","interval":"1m","tolerance":50}`,
		"fallback":           `{"type":"fallback","tag":"g","outbounds":["a","b"]}`,
		"failover_assembled": `{"type":"selector","tag":"auto-us","outbounds":["us-1","us-2","direct"],"default":"us-1","interrupt_exist_connections":true}`,
	}

	forbiddenKeys := []string{
		"private_key", "host_key", "host_key_path", "key_path",
		"password", "secret", "token", "api_key",
		"username", "user",
	}

	for name, config := range groupConfigs {
		t.Run(name, func(t *testing.T) {
			var decoded map[string]any
			if err := json.Unmarshal([]byte(config), &decoded); err != nil {
				t.Fatalf("unmarshal %s: %v", name, err)
			}
			for _, forbidden := range forbiddenKeys {
				if _, found := decoded[forbidden]; found {
					t.Errorf("%s config leaked forbidden key %q: %v", name, forbidden, decoded)
				}
			}
			// Also check nested failover metadata if present.
			if failover, ok := decoded["failover"].(map[string]any); ok {
				for _, forbidden := range forbiddenKeys {
					if _, found := failover[forbidden]; found {
						t.Errorf("%s failover metadata leaked forbidden key %q: %v", name, forbidden, failover)
					}
				}
			}
		})
	}
}
