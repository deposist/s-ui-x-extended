package core

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/sagernet/sing-box/option"
)

// decodeRouteRule decodes a single route rule exactly the way a config would,
// so a test can assert against the fork's own IsValid rather than against our
// helper's opinion of it.
func decodeRouteRules(t *testing.T, ruleJSON string) []option.Rule {
	t.Helper()
	var opt option.Options
	document := `{"route":{"rules":[` + ruleJSON + `]}}`
	if err := opt.UnmarshalJSONContext(registryContext(context.Background()), []byte(document)); err != nil {
		t.Fatalf("decode route rule %s: %v", ruleJSON, err)
	}
	if opt.Route == nil {
		return nil
	}
	return opt.Route.Rules
}

func decodeRouteRule(t *testing.T, ruleJSON string) option.Rule {
	t.Helper()
	rules := decodeRouteRules(t, ruleJSON)
	if len(rules) != 1 {
		t.Fatalf("expected exactly one decoded route rule from %s, got %d", ruleJSON, len(rules))
	}
	return rules[0]
}

// decodeDNSRules returns whatever survives decoding, which is not always one
// rule: an empty object submitted on its own is discarded by badjson before
// the typed decode ever sees it. Callers that need the rule itself assert on
// the length.
func decodeDNSRules(t *testing.T, ruleJSON string) []option.DNSRule {
	t.Helper()
	var opt option.Options
	document := `{"dns":{"rules":[` + ruleJSON + `]}}`
	if err := opt.UnmarshalJSONContext(registryContext(context.Background()), []byte(document)); err != nil {
		t.Fatalf("decode dns rule %s: %v", ruleJSON, err)
	}
	if opt.DNS == nil {
		return nil
	}
	return opt.DNS.Rules
}

func decodeDNSRule(t *testing.T, ruleJSON string) option.DNSRule {
	t.Helper()
	rules := decodeDNSRules(t, ruleJSON)
	if len(rules) != 1 {
		t.Fatalf("expected exactly one decoded dns rule from %s, got %d", ruleJSON, len(rules))
	}
	return rules[0]
}

func issuePaths(issues []RuleConditionIssue) []string {
	paths := make([]string, 0, len(issues))
	for _, issue := range issues {
		paths = append(paths, issue.Path)
	}
	return paths
}

func requirePaths(t *testing.T, issues []RuleConditionIssue, want ...string) {
	t.Helper()
	got := issuePaths(issues)
	if len(got) != len(want) {
		t.Fatalf("expected paths %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected paths %v, got %v", want, got)
		}
	}
}

// TestRuleConditionsDecodedThinShapesAreValid pins the most counter-intuitive
// part of the contract: an action-only branch and an invert-only branch both
// satisfy the core, because decoding defaults the action to "route" and
// validity is "not deeply equal to the zero value with Invert copied". A
// panel-side heuristic that looks for a "meaningful field" rejects both.
func TestRuleConditionsDecodedThinShapesAreValid(t *testing.T) {
	for _, shape := range []string{`{"action":"reject"}`, `{"invert":true}`} {
		t.Run("route"+shape, func(t *testing.T) {
			if decoded := decodeRouteRule(t, shape); !decoded.IsValid() {
				t.Fatalf("fork reports decoded route %s invalid; contract changed", shape)
			}
			issues, err := RuleConditionIssues([]byte(`{"route":{"rules":[{"type":"logical","mode":"and","rules":[` + shape + `]}]}}`))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(issues) != 0 {
				t.Fatalf("expected no issues for nested %s, got %v", shape, issuePaths(issues))
			}
		})
		t.Run("dns"+shape, func(t *testing.T) {
			if decoded := decodeDNSRule(t, shape); !decoded.IsValid() {
				t.Fatalf("fork reports decoded dns %s invalid; contract changed", shape)
			}
			issues, err := RuleConditionIssues([]byte(`{"dns":{"rules":[{"type":"logical","mode":"and","rules":[` + shape + `]}]}}`))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(issues) != 0 {
				t.Fatalf("expected no issues for nested %s, got %v", shape, issuePaths(issues))
			}
		})
	}
}

// TestRuleConditionsEmptyObjectIsDiscarded pins the behaviour that invalidates
// the obvious mental model of "an empty rule is an invalid rule". An empty JSON
// object is neither valid nor rejected: badjson rebuilds the enclosing object
// during decoding and the empty element disappears, so the core accepts the
// config with the rule missing.
//
// The consequences are asymmetric and worth stating explicitly, because every
// one of them shapes what the helper may report:
//
//   - alone in route.rules the object survives and is valid, since the route
//     array is not rebuilt the same way;
//   - alone in dns.rules it is discarded, leaving zero rules;
//   - alone in a logical rule's sub-array it is discarded for route and DNS
//     alike, which leaves the parent with no sub-rules and therefore invalid;
//   - alongside a real sibling it survives even in dns.rules, which is why
//     dropped rules can only be reported per array and never per index.
func TestRuleConditionsEmptyObjectIsDiscarded(t *testing.T) {
	t.Run("survives alone in route.rules", func(t *testing.T) {
		if !decodeRouteRule(t, `{}`).IsValid() {
			t.Fatal("expected a lone empty route rule to survive and be valid")
		}
		issues, err := RuleConditionIssues([]byte(`{"route":{"rules":[{}]}}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(issues) != 0 {
			t.Fatalf("expected no issues, got %v", issuePaths(issues))
		}
	})

	t.Run("is discarded alone in dns.rules", func(t *testing.T) {
		if rules := decodeDNSRules(t, `{}`); len(rules) != 0 {
			t.Fatalf("expected a lone empty dns rule to be discarded, got %d rules", len(rules))
		}
		issues, err := RuleConditionIssues([]byte(`{"dns":{"rules":[{}]}}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		requirePaths(t, issues, "dns.rules")
		if issues[0].Code != RuleConditionCodeDroppedRule {
			t.Fatalf("expected a dropped-rule issue, got %q", issues[0].Code)
		}
	})

	t.Run("survives beside a sibling in dns.rules", func(t *testing.T) {
		if rules := decodeDNSRules(t, `{},{"domain":["a.example"]}`); len(rules) != 2 {
			t.Fatalf("expected both dns rules to survive, got %d", len(rules))
		}
		issues, err := RuleConditionIssues([]byte(`{"dns":{"rules":[{},{"domain":["a.example"]}]}}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(issues) != 0 {
			t.Fatalf("expected no issues, got %v", issuePaths(issues))
		}
	})

	t.Run("empties a logical rule", func(t *testing.T) {
		for _, kind := range []string{RuleKindRoute, RuleKindDNS} {
			document := `{"` + kind + `":{"rules":[{"type":"logical","mode":"and","rules":[{}]}]}}`
			issues, err := RuleConditionIssues([]byte(document))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			requirePaths(t, issues, kind+".rules[0].rules")
			if issues[0].Code != RuleConditionCodeMissingConditions {
				t.Fatalf("expected missing-conditions for %s, got %q", kind, issues[0].Code)
			}
		}
	})
}

// TestRuleConditionsZeroValueStructIsInvalid proves the decoded-vs-constructed
// distinction is real and is not conflated by the helper: the same "empty"
// rule is valid when decoded and invalid when built as a Go zero value.
func TestRuleConditionsZeroValueStructIsInvalid(t *testing.T) {
	var routeRule option.Rule
	routeRule.Type = "default"
	if routeRule.IsValid() {
		t.Fatal("zero-value route rule should be invalid")
	}
	if decodeRouteRule(t, `{}`).IsValid() == routeRule.IsValid() {
		t.Fatal("decoded and zero-value rules must not agree; the default action is what separates them")
	}

	var dnsRule option.DNSRule
	dnsRule.Type = "default"
	if dnsRule.IsValid() {
		t.Fatal("zero-value dns rule should be invalid")
	}
}

func TestRuleConditionsEmptyLogicalIsRejected(t *testing.T) {
	t.Run("route", func(t *testing.T) {
		issues, err := RuleConditionIssues([]byte(`{"route":{"rules":[{"type":"logical","mode":"and","rules":[]}]}}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		requirePaths(t, issues, "route.rules[0].rules")
		if issues[0].Kind != RuleKindRoute || issues[0].Code != RuleConditionCodeMissingConditions {
			t.Fatalf("unexpected issue metadata: %+v", issues[0])
		}
	})
	t.Run("dns", func(t *testing.T) {
		issues, err := RuleConditionIssues([]byte(`{"dns":{"rules":[{"type":"logical","mode":"and","rules":[]}]}}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		requirePaths(t, issues, "dns.rules[0].rules")
		if issues[0].Kind != RuleKindDNS {
			t.Fatalf("unexpected kind: %+v", issues[0])
		}
	})
	t.Run("missing rules key", func(t *testing.T) {
		issues, err := RuleConditionIssues([]byte(`{"route":{"rules":[{"type":"logical","mode":"and"}]}}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		requirePaths(t, issues, "route.rules[0].rules")
	})
}

// TestRuleConditionsReportsOnlyInvalidBranch guards against the previous
// behaviour, which flagged whole rules and valid siblings alike.
func TestRuleConditionsReportsOnlyInvalidBranch(t *testing.T) {
	config := `{"route":{"rules":[
		{"type":"logical","mode":"and","rules":[{"domain":["a.example"]}]},
		{"domain":["b.example"]},
		{"type":"logical","mode":"or","rules":[{"domain":["c.example"]},{"type":"logical","mode":"and","rules":[]}]}
	]}}`
	issues, err := RuleConditionIssues([]byte(config))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	requirePaths(t, issues, "route.rules[2].rules[1].rules")
}

// TestRuleConditionsReportsExactNestedPath exercises more than two levels and
// both arrays at once, and pins deterministic ordering: route before DNS,
// array order within each.
func TestRuleConditionsReportsExactNestedPath(t *testing.T) {
	config := `{
		"route":{"rules":[
			{"domain":["keep.example"]},
			{"type":"logical","mode":"and","rules":[
				{"type":"logical","mode":"or","rules":[
					{"domain":["deep.example"]},
					{"type":"logical","mode":"and","rules":[]}
				]}
			]}
		]},
		"dns":{"rules":[
			{"type":"logical","mode":"and","rules":[{"type":"logical","mode":"and","rules":[]}]}
		]}
	}`
	issues, err := RuleConditionIssues([]byte(config))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	requirePaths(t, issues,
		"route.rules[1].rules[0].rules[1].rules",
		"dns.rules[0].rules[0].rules",
	)
	if issues[0].Kind != RuleKindRoute || issues[1].Kind != RuleKindDNS {
		t.Fatalf("expected route issue before dns issue, got %+v", issues)
	}
}

// TestRuleConditionsRejectsMalformedInput proves a decode failure is an error
// rather than an empty "looks fine" result. Silently accepting undecodable
// rules is what would let a panel save a config the core cannot start.
func TestRuleConditionsRejectsMalformedInput(t *testing.T) {
	for name, config := range map[string]string{
		"broken json":        `{"route":{"rules":[`,
		"unknown rule type":  `{"route":{"rules":[{"type":"nonsense"}]}}`,
		"unknown action":     `{"route":{"rules":[{"action":"nonsense"}]}}`,
		"unknown field":      `{"route":{"rules":[{"not_a_field":true}]}}`,
		"rule is not object": `{"route":{"rules":["nope"]}}`,
		"rules is not array": `{"route":{"rules":{"a":1}}}`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := RuleConditionIssues([]byte(config)); err == nil {
				t.Fatalf("expected a decode error for %s", name)
			}
		})
	}
}

func TestRuleConditionsIgnoresUnrelatedConfig(t *testing.T) {
	// An inbound the panel cannot decode must not turn a rule-condition check
	// into a failure: this guard is scoped to rules on purpose.
	issues, err := RuleConditionIssues([]byte(`{"inbounds":[{"type":"totally-unknown"}],"route":{"rules":[{"domain":["a.example"]}]}}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(issues) != 0 {
		t.Fatalf("expected no issues, got %v", issuePaths(issues))
	}
}

func TestRuleConditionsEmptyConfig(t *testing.T) {
	for name, config := range map[string]string{
		"empty object": `{}`,
		"no rules":     `{"route":{}}`,
		"empty arrays": `{"route":{"rules":[]},"dns":{"rules":[]}}`,
	} {
		t.Run(name, func(t *testing.T) {
			issues, err := RuleConditionIssues([]byte(config))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(issues) != 0 {
				t.Fatalf("expected no issues, got %v", issuePaths(issues))
			}
		})
	}
}

// TestRuleConditionsDistinctFromValidateConfig documents the division of
// labour: this helper answers only "does every logical rule have conditions",
// while ValidateConfig remains the authority on construction. Conflating the
// two is what makes error messages unactionable.
func TestRuleConditionsDistinctFromValidateConfig(t *testing.T) {
	// Condition-valid, but unbuildable for a reason this helper knows nothing
	// about: two outbounds share a tag. A construction fault has to stay
	// ValidateConfig's to report, or the two checks start duplicating each
	// other's messages.
	//
	// The fixture is deliberately a duplicate tag rather than the more obvious
	// "no outbound at all" or "rule points at a missing outbound": the core
	// accepts both of those, so neither can stand in for a build failure.
	conditionValid := `{
		"log":{"disabled":true},
		"outbounds":[{"type":"direct","tag":"same"},{"type":"direct","tag":"same"}],
		"route":{"rules":[{"domain":["a.example"]}]}
	}`
	issues, err := RuleConditionIssues([]byte(conditionValid))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(issues) != 0 {
		t.Fatalf("expected no condition issues, got %v", issuePaths(issues))
	}
	if err := ValidateConfig([]byte(conditionValid)); err == nil {
		t.Fatal("expected ValidateConfig to reject a duplicate outbound tag")
	}

	// Complete config whose logical rule wraps an invert-only sub-rule: valid on
	// both checks, even though the sub-rule matches nothing in particular. This
	// is the shape a "meaningful field" heuristic would wrongly reject.
	complete := `{
		"log":{"disabled":true},
		"outbounds":[{"type":"direct","tag":"direct"}],
		"route":{"rules":[{"type":"logical","mode":"and","rules":[{"invert":true}],"outbound":"direct"}]}
	}`
	issues, err = RuleConditionIssues([]byte(complete))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(issues) != 0 {
		t.Fatalf("expected no condition issues, got %v", issuePaths(issues))
	}
	if err := ValidateConfig([]byte(complete)); err != nil {
		t.Fatalf("expected ValidateConfig to accept the complete config: %v", err)
	}

	// And the empty logical rule the helper rejects is rejected by the core
	// too, which is the whole reason the guard is fail-closed.
	empty := `{
		"log":{"disabled":true},
		"outbounds":[{"type":"direct","tag":"direct"}],
		"route":{"rules":[{"type":"logical","mode":"and","rules":[],"outbound":"direct"}]}
	}`
	issues, err = RuleConditionIssues([]byte(empty))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	requirePaths(t, issues, "route.rules[0].rules")
	if err := ValidateConfig([]byte(empty)); err == nil {
		t.Fatal("expected ValidateConfig to reject an empty logical rule; the helper would then be stricter than the core")
	}
}

func TestSingleRuleConditionIssues(t *testing.T) {
	t.Run("route paths start at index zero", func(t *testing.T) {
		issues, err := SingleRuleConditionIssues(RuleKindRoute, []byte(`{"type":"logical","mode":"and","rules":[{"type":"logical","mode":"and","rules":[]}]}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		requirePaths(t, issues, "route.rules[0].rules[0].rules")
	})
	t.Run("dns paths start at index zero", func(t *testing.T) {
		issues, err := SingleRuleConditionIssues(RuleKindDNS, []byte(`{"type":"logical","mode":"and","rules":[]}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		requirePaths(t, issues, "dns.rules[0].rules")
	})
	t.Run("valid rule", func(t *testing.T) {
		issues, err := SingleRuleConditionIssues(RuleKindRoute, []byte(`{"domain":["a.example"]}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(issues) != 0 {
			t.Fatalf("expected no issues, got %v", issuePaths(issues))
		}
	})
	// The modal's practical payoff: a lone empty DNS rule cannot be reported as
	// invalid, because the core accepts the config, but it must not be reported
	// as fine either, because the saved rule would not exist.
	t.Run("empty dns rule is reported as dropped", func(t *testing.T) {
		issues, err := SingleRuleConditionIssues(RuleKindDNS, []byte(`{}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		requirePaths(t, issues, "dns.rules")
		if issues[0].Kind != RuleKindDNS || issues[0].Code != RuleConditionCodeDroppedRule {
			t.Fatalf("expected a dns dropped-rule issue, got %+v", issues[0])
		}
	})
	t.Run("empty route rule is accepted", func(t *testing.T) {
		issues, err := SingleRuleConditionIssues(RuleKindRoute, []byte(`{}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(issues) != 0 {
			t.Fatalf("expected no issues, got %v", issuePaths(issues))
		}
	})
	t.Run("rejects unknown kind", func(t *testing.T) {
		if _, err := SingleRuleConditionIssues("outbound", []byte(`{}`)); err == nil {
			t.Fatal("expected an error for an unknown kind")
		}
	})
	t.Run("rejects empty rule", func(t *testing.T) {
		if _, err := SingleRuleConditionIssues(RuleKindRoute, nil); err == nil {
			t.Fatal("expected an error for an empty rule")
		}
	})
	t.Run("rejects malformed rule", func(t *testing.T) {
		if _, err := SingleRuleConditionIssues(RuleKindRoute, []byte(`{"type":"nonsense"}`)); err == nil {
			t.Fatal("expected an error for a malformed rule")
		}
	})
}

// validityIssues drops dropped-rule reports, which describe a rule that the
// core discarded rather than one it considered invalid. Only the remainder is
// comparable with IsValid.
func validityIssues(issues []RuleConditionIssue) []RuleConditionIssue {
	kept := make([]RuleConditionIssue, 0, len(issues))
	for _, issue := range issues {
		if issue.Code != RuleConditionCodeDroppedRule {
			kept = append(kept, issue)
		}
	}
	return kept
}

// hasDroppedIssue reports whether the helper flagged silent rule loss.
func hasDroppedIssue(issues []RuleConditionIssue) bool {
	return len(validityIssues(issues)) != len(issues)
}

// TestRuleConditionIssuesMatchForkIsValid is the parity assertion: for every
// fixture, our verdict and the fork's own IsValid must agree. If a future fork
// bump changes the semantics, this fails instead of silently drifting.
//
// The oracle is "every rule that survived decoding is IsValid", not "the single
// decoded rule is IsValid", because a rule can vanish during decoding. A
// vanished rule leaves the core with nothing to object to, so the config is
// accepted and parity requires no validity issue from us — only the separate
// dropped-rule report, which is asserted alongside rather than mixed in.
func TestRuleConditionIssuesMatchForkIsValid(t *testing.T) {
	fixtures := []string{
		`{}`,
		`{"invert":true}`,
		`{"action":"reject"}`,
		`{"domain":["a.example"]}`,
		`{"type":"logical","mode":"and","rules":[]}`,
		`{"type":"logical","mode":"and","rules":[{}]}`,
		`{"type":"logical","mode":"or","rules":[{"domain":["a.example"]},{"type":"logical","mode":"and","rules":[]}]}`,
		`{"type":"logical","mode":"and","rules":[{"type":"logical","mode":"or","rules":[{"invert":true}]}]}`,
	}
	for _, fixture := range fixtures {
		t.Run(fixture, func(t *testing.T) {
			routeRules := decodeRouteRules(t, fixture)
			routeAccepted := true
			for _, rule := range routeRules {
				if !rule.IsValid() {
					routeAccepted = false
				}
			}
			routeIssues, err := SingleRuleConditionIssues(RuleKindRoute, []byte(fixture))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if routeAccepted != (len(validityIssues(routeIssues)) == 0) {
				t.Fatalf("route parity broken for %s: fork accepts=%v issues=%v", fixture, routeAccepted, issuePaths(routeIssues))
			}
			if (len(routeRules) == 0) != hasDroppedIssue(routeIssues) {
				t.Fatalf("route drop reporting broken for %s: kept=%d issues=%v", fixture, len(routeRules), issuePaths(routeIssues))
			}

			dnsRules := decodeDNSRules(t, fixture)
			dnsAccepted := true
			for _, rule := range dnsRules {
				if !rule.IsValid() {
					dnsAccepted = false
				}
			}
			dnsIssues, err := SingleRuleConditionIssues(RuleKindDNS, []byte(fixture))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if dnsAccepted != (len(validityIssues(dnsIssues)) == 0) {
				t.Fatalf("dns parity broken for %s: fork accepts=%v issues=%v", fixture, dnsAccepted, issuePaths(dnsIssues))
			}
			if (len(dnsRules) == 0) != hasDroppedIssue(dnsIssues) {
				t.Fatalf("dns drop reporting broken for %s: kept=%d issues=%v", fixture, len(dnsRules), issuePaths(dnsIssues))
			}
		})
	}
}

func TestRuleConditionIssuesRejectsExcessiveDepth(t *testing.T) {
	rule := `{"domain":["leaf.example"]}`
	for range maxRuleTreeDepth {
		rule = `{"type":"logical","mode":"and","rules":[` + rule + `]}`
	}
	_, err := SingleRuleConditionIssues(RuleKindRoute, []byte(rule))
	if err == nil || !strings.Contains(err.Error(), "maximum depth") {
		t.Fatalf("expected depth-limit error, got %v", err)
	}
}

func TestFormatRuleConditionIssues(t *testing.T) {
	if FormatRuleConditionIssues(nil) != nil {
		t.Fatal("expected nil for no issues")
	}
	formatted := FormatRuleConditionIssues([]RuleConditionIssue{
		{Kind: RuleKindRoute, Path: "route.rules[0].rules", Code: RuleConditionCodeMissingConditions, Message: "logical rule has no sub-rules"},
	})
	if len(formatted) != 1 || formatted[0] != "route.rules[0].rules: logical rule has no sub-rules" {
		t.Fatalf("unexpected formatting: %v", formatted)
	}
}

// TestRuleConditionIssueJSONShape pins the wire contract the modal depends on.
func TestRuleConditionIssueJSONShape(t *testing.T) {
	encoded, err := json.Marshal(RuleConditionIssue{Kind: RuleKindDNS, Path: "dns.rules[0].rules", Code: RuleConditionCodeMissingConditions, Message: "logical rule has no sub-rules"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, key := range []string{`"kind"`, `"path"`, `"code"`, `"message"`} {
		if !strings.Contains(string(encoded), key) {
			t.Fatalf("expected %s in %s", key, encoded)
		}
	}
}
