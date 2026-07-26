package service

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateConfigRuleConditions(t *testing.T) {
	cases := []struct {
		name    string
		config  string
		wantErr bool
	}{
		{"no route block", `{"dns":{"servers":[]}}`, false},
		{"no rules", `{"route":{"rules":[]}}`, false},
		{"action only rules stay allowed", `{"route":{"rules":[{"action":"sniff"},{"protocol":["dns"],"action":"hijack-dns"}]}}`, false},
		{"plain rule with condition", `{"route":{"rules":[{"domain_suffix":["example.com"],"outbound":"direct"}]}}`, false},
		{"logical rule with conditions", `{"route":{"rules":[{"type":"logical","mode":"and","rules":[{"domain":["a.com"]},{"port":[443]}],"outbound":"direct"}]}}`, false},
		{"logical rule with empty sub-rule", `{"route":{"rules":[{"type":"logical","mode":"and","rules":[{}],"outbound":"direct"}]}}`, true},
		{"logical rule without branches", `{"route":{"rules":[{"type":"logical","mode":"and","rules":[],"outbound":"direct"}]}}`, true},
		{"empty logical dns rule", `{"dns":{"rules":[{"type":"logical","mode":"and","rules":[{}],"server":"local"}]}}`, true},

		// The next two were previously expected to be rejected. They are not,
		// and the old expectation was the bug: core.ValidateConfig builds both
		// of these configs successfully, so rejecting them here blocked saves
		// the core would have accepted.
		//
		// An empty object beside a real sibling survives decoding as a valid
		// default rule, and an invert-only rule is valid because validity means
		// "not deeply equal to the zero value with Invert copied". The old
		// "meaningful field" heuristic could see neither.
		{"logical rule with one empty branch", `{"route":{"rules":[{"type":"logical","mode":"and","rules":[{"domain":["a.com"]},{}],"outbound":"direct"}]}}`, false},
		{"logical rule carrying only invert", `{"route":{"rules":[{"type":"logical","mode":"and","rules":[{"invert":true}],"outbound":"direct"}]}}`, false},
		{"valid logical dns rule", `{"dns":{"rules":[{"type":"logical","mode":"and","rules":[{"domain":["a.com"]}],"server":"local"}]}}`, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateConfigRuleConditions(json.RawMessage(tc.config))
			if tc.wantErr && err == nil {
				t.Fatalf("expected rejection for %s", tc.config)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected rejection for %s: %v", tc.config, err)
			}
			if tc.wantErr && !strings.Contains(err.Error(), "no conditions") {
				t.Fatalf("error should explain the cause, got: %v", err)
			}
		})
	}
}

// TestValidateConfigRuleConditionsAcceptsDroppedRules pins the deliberate
// asymmetry in how the save path treats the two failure modes.
//
// A lone empty DNS rule is discarded during decoding, so the core loads the
// config without complaint. Rejecting the save would therefore be stricter than
// the core, and worse, it would wedge an operator whose stored config already
// contains such a rule: every later edit would fail on a rule they did not
// touch. The loss is logged and surfaced by the doctor instead.
func TestValidateConfigRuleConditionsAcceptsDroppedRules(t *testing.T) {
	if err := validateConfigRuleConditions(json.RawMessage(`{"dns":{"rules":[{}]}}`)); err != nil {
		t.Fatalf("a discarded rule must not block the save, got: %v", err)
	}
}

func TestValidateConfigRuleConditionsRejectsUndecodableRules(t *testing.T) {
	config := json.RawMessage(`{"route":{"rules":[{"action":"nonsense"}]}}`)
	if err := validateConfigRuleConditions(config); err == nil {
		t.Fatal("an undecodable rule must block the config save")
	}
}
func TestValidateConfigRuleConditionsNamesOffendingRule(t *testing.T) {
	config := `{"route":{"rules":[{"action":"sniff"},{"domain":["a.com"]},{"type":"logical","mode":"and","rules":[{}]}]}}`
	err := validateConfigRuleConditions(json.RawMessage(config))
	if err == nil || !strings.Contains(err.Error(), "route.rules[2]") {
		t.Fatalf("error should name route.rules[2], got: %v", err)
	}
}

// TestValidateConfigRuleConditionsNamesDeepDescendant proves the reported path
// walks all the way into a nested logical node instead of stopping at the
// top-level rule index. Without the full path an operator has to hunt for the
// offending branch by hand, which is the whole reason the editor needs it.
func TestValidateConfigRuleConditionsNamesDeepDescendant(t *testing.T) {
	for _, tc := range []struct {
		name     string
		config   string
		wantPath string
	}{
		{
			name:     "route",
			config:   `{"route":{"rules":[{"type":"logical","mode":"and","rules":[{"domain":["a.com"]},{"type":"logical","mode":"or","rules":[]}],"outbound":"direct"}]}}`,
			wantPath: "route.rules[0].rules[1].rules",
		},
		{
			// A sole {} child is dropped during decoding, leaving the nested
			// logical node with an empty sub-rule list, so this reports the same
			// path as a literally empty array.
			name:     "route with a dropped sole child",
			config:   `{"route":{"rules":[{"type":"logical","mode":"and","rules":[{"domain":["a.com"]},{"type":"logical","mode":"or","rules":[{}]}],"outbound":"direct"}]}}`,
			wantPath: "route.rules[0].rules[1].rules",
		},
		{
			name:     "dns",
			config:   `{"dns":{"rules":[{"type":"logical","mode":"and","rules":[{"domain":["a.com"]},{"type":"logical","mode":"or","rules":[]}],"server":"local"}]}}`,
			wantPath: "dns.rules[0].rules[1].rules",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := validateConfigRuleConditions(json.RawMessage(tc.config))
			if err == nil {
				t.Fatalf("expected rejection for %s", tc.config)
			}
			if !strings.Contains(err.Error(), tc.wantPath) {
				t.Fatalf("error should name %s, got: %v", tc.wantPath, err)
			}
			// The valid sibling must not be blamed alongside the broken branch.
			if strings.Contains(err.Error(), tc.wantPath[:strings.LastIndex(tc.wantPath, "[")]+"[0]") {
				t.Fatalf("the valid sibling should not be reported, got: %v", err)
			}
		})
	}
}
func TestDispatchSaveRejectsRuleWithoutConditionsBeforeTouchingDB(t *testing.T) {
	s := &ConfigService{}
	config := `{"route":{"rules":[{"type":"logical","mode":"and","rules":[{}],"outbound":"direct"}]}}`
	_, _, _, err := s.dispatchSave(nil, "config", "edit", json.RawMessage(config), "", "")
	if err == nil || !strings.Contains(err.Error(), "route.rules[0]") {
		t.Fatalf("dispatchSave should reject the rule before touching the database, got: %v", err)
	}
}
