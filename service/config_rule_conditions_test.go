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
		{"logical rule with one empty branch", `{"route":{"rules":[{"type":"logical","mode":"and","rules":[{"domain":["a.com"]},{}],"outbound":"direct"}]}}`, true},
		{"logical rule without branches", `{"route":{"rules":[{"type":"logical","mode":"and","rules":[],"outbound":"direct"}]}}`, true},
		{"logical rule carrying only invert", `{"route":{"rules":[{"type":"logical","mode":"and","rules":[{"invert":true}],"outbound":"direct"}]}}`, true},
		{"empty logical dns rule", `{"dns":{"rules":[{"type":"logical","mode":"and","rules":[{}],"server":"local"}]}}`, true},
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

func TestValidateConfigRuleConditionsNamesOffendingRule(t *testing.T) {
	config := `{"route":{"rules":[{"action":"sniff"},{"domain":["a.com"]},{"type":"logical","mode":"and","rules":[{}]}]}}`
	err := validateConfigRuleConditions(json.RawMessage(config))
	if err == nil || !strings.Contains(err.Error(), "route.rules[2]") {
		t.Fatalf("error should name route.rules[2], got: %v", err)
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
