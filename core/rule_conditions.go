package core

import (
	"context"
	"encoding/json"
	"strconv"

	E "github.com/sagernet/sing/common/exceptions"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
)

// Rule kinds accepted by SingleRuleConditionIssues and reported on every
// issue, so a caller can tell a route problem from a DNS one without parsing
// the path.
const (
	RuleKindRoute = "route"
	RuleKindDNS   = "dns"
)

// Issue codes. missing-conditions is the only one sing-box itself can produce
// through IsValid; invalid-rule exists so an unexpected shape can never be
// reported as "no problems found".
const (
	RuleConditionCodeMissingConditions = "missing-conditions"
	RuleConditionCodeInvalidRule       = "invalid-rule"
	// RuleConditionCodeDroppedRule reports a rule that the decoder discards
	// instead of rejecting. See appendDroppedRuleIssues for why this is a
	// separate code rather than a validity failure.
	RuleConditionCodeDroppedRule = "dropped-rule"
)

const (
	// Both validity messages contain the phrase "no conditions" deliberately.
	// It is the wording the core itself uses ("missing conditions"), and both
	// the save path and the doctor key their operator-facing copy off it, so a
	// logical rule with an empty branch list must not be described only as
	// "no sub-rules".
	ruleConditionMessageMissing = "logical rule has no conditions: its sub-rule list is empty"
	ruleConditionMessageInvalid = "rule has no conditions"
	ruleConditionMessageDropped = "empty rules were silently discarded while decoding"
)

const (
	maxRuleTreeDepth = 64
	maxRuleTreeNodes = 4096
)

// RuleConditionIssue locates one rule that sing-box would reject as having no
// conditions. Path is the exact JSON path within the assembled config, so the
// UI can anchor the message on the offending node rather than the whole rule.
type RuleConditionIssue struct {
	Kind    string `json:"kind"`
	Path    string `json:"path"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ruleArrays is the condition-relevant slice of a config: the two rule arrays
// and nothing else.
type ruleArrays struct {
	Route json.RawMessage
	DNS   json.RawMessage
}

// RuleConditionIssues reports every route and DNS rule in the supplied config
// that sing-box would treat as condition-less.
//
// The check deliberately delegates validity to the pinned fork's own
// option.Rule.IsValid / option.DNSRule.IsValid rather than re-deriving
// "meaningful field" heuristics in the panel. Those heuristics are impossible
// to keep correct: validity is defined as "not deeply equal to the zero value
// with Invert copied", and JSON decoding defaults an omitted action to route,
// which means a decoded empty object is valid while a hand-built zero-value
// struct is not. Only the real decoder reproduces that.
//
// An error means the rules could not be typed at all; it is not a "no issues"
// result and callers must treat it as a rejection.
func RuleConditionIssues(configJSON []byte) ([]RuleConditionIssue, error) {
	arrays, err := extractRuleArrays(configJSON)
	if err != nil {
		return nil, err
	}
	return ruleConditionIssues(arrays)
}

// SingleRuleConditionIssues validates one rule in isolation, for modal-level
// validation before a rule is merged into the config.
//
// The rule is wrapped as index zero of the matching array so the reported
// paths are identical in shape to the whole-config ones: a caller can reuse the
// same path-to-node mapping for both.
func SingleRuleConditionIssues(kind string, ruleJSON []byte) ([]RuleConditionIssue, error) {
	if len(ruleJSON) == 0 {
		return nil, E.New("empty rule")
	}
	wrapped, err := json.Marshal([]json.RawMessage{json.RawMessage(ruleJSON)})
	if err != nil {
		return nil, err
	}
	switch kind {
	case RuleKindRoute:
		return ruleConditionIssues(ruleArrays{Route: wrapped})
	case RuleKindDNS:
		return ruleConditionIssues(ruleArrays{DNS: wrapped})
	default:
		return nil, E.New("unknown rule kind: " + kind)
	}
}

// extractRuleArrays pulls out only route.rules and dns.rules. Everything else
// in the config is irrelevant to condition validity, and excluding it keeps an
// unrelated inbound or outbound problem from being reported by this check.
func extractRuleArrays(configJSON []byte) (ruleArrays, error) {
	if len(configJSON) == 0 {
		return ruleArrays{}, nil
	}
	var top struct {
		Route struct {
			Rules json.RawMessage `json:"rules"`
		} `json:"route"`
		DNS struct {
			Rules json.RawMessage `json:"rules"`
		} `json:"dns"`
	}
	if err := json.Unmarshal(configJSON, &top); err != nil {
		return ruleArrays{}, E.Cause(err, "decode config rules")
	}
	return ruleArrays{Route: top.Route.Rules, DNS: top.DNS.Rules}, nil
}

// ruleConditionIssues decodes the two rule arrays through the real sing-box
// decoder and walks the decoded values.
func ruleConditionIssues(arrays ruleArrays) ([]RuleConditionIssue, error) {
	if len(arrays.Route) == 0 && len(arrays.DNS) == 0 {
		return nil, nil
	}
	if err := validateRawRuleTreeLimits(arrays); err != nil {
		return nil, err
	}
	document, err := buildRuleDocument(arrays)
	if err != nil {
		return nil, err
	}
	var opt option.Options
	if err := opt.UnmarshalJSONContext(registryContext(context.Background()), document); err != nil {
		return nil, E.Cause(err, "decode rules")
	}

	var issues []RuleConditionIssue
	if opt.Route != nil {
		issues = appendRouteRuleIssues(issues, "route.rules", opt.Route.Rules)
	}
	if opt.DNS != nil {
		issues = appendDNSRuleIssues(issues, "dns.rules", opt.DNS.Rules)
	}
	decodedRoute := 0
	if opt.Route != nil {
		decodedRoute = len(opt.Route.Rules)
	}
	decodedDNS := 0
	if opt.DNS != nil {
		decodedDNS = len(opt.DNS.Rules)
	}
	issues, err = appendDroppedRuleIssues(issues, RuleKindRoute, "route.rules", arrays.Route, decodedRoute)
	if err != nil {
		return nil, err
	}
	issues, err = appendDroppedRuleIssues(issues, RuleKindDNS, "dns.rules", arrays.DNS, decodedDNS)
	if err != nil {
		return nil, err
	}
	return issues, nil
}

// appendDroppedRuleIssues reports rules that the decoder silently discards
// instead of rejecting.
//
// This is not a validity failure and deliberately does not go through IsValid.
// sing-box routes these arrays through badjson, which rebuilds the surrounding
// object and can drop an empty JSON object on the way. `{"dns":{"rules":[{}]}}`
// therefore decodes to zero rules and the config is *accepted* — the operator's
// rule is simply gone. Reporting that as invalid would make the panel stricter
// than the core, which the parity tests forbid, but staying silent would let a
// rule vanish without a word, so it gets its own code and callers choose how
// loud to be.
//
// The issue is raised against the array rather than a specific index on
// purpose. Survival is a property of the whole array, not of one element:
// `[{}]` collapses to zero rules while `[{},{"domain":[...]}]` keeps both. So
// probing elements in isolation would confidently blame the wrong index, and
// only the totals can be compared honestly.
func appendDroppedRuleIssues(issues []RuleConditionIssue, kind, path string, rawRules []byte, decodedCount int) ([]RuleConditionIssue, error) {
	if len(rawRules) == 0 {
		return issues, nil
	}
	var elements []json.RawMessage
	if err := json.Unmarshal(rawRules, &elements); err != nil {
		return nil, E.Cause(err, "decode "+kind+" rules")
	}
	if len(elements) <= decodedCount {
		return issues, nil
	}
	return append(issues, RuleConditionIssue{
		Kind: kind,
		Path: path,
		Code: RuleConditionCodeDroppedRule,
		Message: ruleConditionMessageDropped + ": " +
			strconv.Itoa(len(elements)) + " submitted, " + strconv.Itoa(decodedCount) + " kept",
	}), nil
}

func validateRawRuleTreeLimits(arrays ruleArrays) error {
	for _, rules := range []struct {
		kind string
		raw  json.RawMessage
	}{{RuleKindRoute, arrays.Route}, {RuleKindDNS, arrays.DNS}} {
		if len(rules.raw) == 0 {
			continue
		}
		var roots []json.RawMessage
		if err := json.Unmarshal(rules.raw, &roots); err != nil {
			return E.Cause(err, "decode "+rules.kind+" rules")
		}
		nodes := 0
		for _, root := range roots {
			if err := validateRawRuleNode(root, 1, &nodes); err != nil {
				return E.Cause(err, rules.kind+" rule tree")
			}
		}
	}
	return nil
}

func validateRawRuleNode(raw json.RawMessage, depth int, nodes *int) error {
	if depth > maxRuleTreeDepth {
		return E.New("exceeds maximum depth of ", maxRuleTreeDepth)
	}
	*nodes = *nodes + 1
	if *nodes > maxRuleTreeNodes {
		return E.New("exceeds maximum node count of ", maxRuleTreeNodes)
	}
	var object struct {
		Rules []json.RawMessage `json:"rules"`
	}
	if err := json.Unmarshal(raw, &object); err != nil {
		return err
	}
	for _, child := range object.Rules {
		if err := validateRawRuleNode(child, depth+1, nodes); err != nil {
			return err
		}
	}
	return nil
}

// buildRuleDocument wraps the raw arrays in the smallest options document that
// still decodes them with full type information.
func buildRuleDocument(arrays ruleArrays) ([]byte, error) {
	document := make(map[string]any, 2)
	if len(arrays.Route) > 0 {
		document["route"] = map[string]any{"rules": arrays.Route}
	}
	if len(arrays.DNS) > 0 {
		document["dns"] = map[string]any{"rules": arrays.DNS}
	}
	return json.Marshal(document)
}

func appendRouteRuleIssues(issues []RuleConditionIssue, prefix string, rules []option.Rule) []RuleConditionIssue {
	for i, rule := range rules {
		path := indexedPath(prefix, i)
		if rule.Type != C.RuleTypeLogical {
			if !rule.DefaultOptions.IsValid() {
				issues = append(issues, invalidRuleIssue(RuleKindRoute, path))
			}
			continue
		}
		logical := rule.LogicalOptions
		if len(logical.Rules) == 0 {
			issues = append(issues, missingConditionsIssue(RuleKindRoute, path+".rules"))
			continue
		}
		before := len(issues)
		issues = appendRouteRuleIssues(issues, path+".rules", logical.Rules)
		// A logical rule is valid exactly when it has children and every child is
		// valid. Since child issues were computed bottom-up, no second recursive
		// IsValid scan is needed here.
		if len(issues) == before {
			continue
		}
	}
	return issues
}

func appendDNSRuleIssues(issues []RuleConditionIssue, prefix string, rules []option.DNSRule) []RuleConditionIssue {
	for i, rule := range rules {
		path := indexedPath(prefix, i)
		if rule.Type != C.RuleTypeLogical {
			if !rule.DefaultOptions.IsValid() {
				issues = append(issues, invalidRuleIssue(RuleKindDNS, path))
			}
			continue
		}
		logical := rule.LogicalOptions
		if len(logical.Rules) == 0 {
			issues = append(issues, missingConditionsIssue(RuleKindDNS, path+".rules"))
			continue
		}
		before := len(issues)
		issues = appendDNSRuleIssues(issues, path+".rules", logical.Rules)
		if len(issues) == before {
			continue
		}
	}
	return issues
}

func indexedPath(prefix string, index int) string {
	return prefix + "[" + strconv.Itoa(index) + "]"
}

func missingConditionsIssue(kind, path string) RuleConditionIssue {
	return RuleConditionIssue{Kind: kind, Path: path, Code: RuleConditionCodeMissingConditions, Message: ruleConditionMessageMissing}
}

func invalidRuleIssue(kind, path string) RuleConditionIssue {
	return RuleConditionIssue{Kind: kind, Path: path, Code: RuleConditionCodeInvalidRule, Message: ruleConditionMessageInvalid}
}

// FormatRuleConditionIssues renders issues as "<path>: <message>" lines, the
// form Doctor and the save path both report.
func FormatRuleConditionIssues(issues []RuleConditionIssue) []string {
	if len(issues) == 0 {
		return nil
	}
	formatted := make([]string, 0, len(issues))
	for _, issue := range issues {
		formatted = append(formatted, issue.Path+": "+issue.Message)
	}
	return formatted
}
