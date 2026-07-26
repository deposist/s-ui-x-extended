package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/deposist/s-ui-x-extended/core"

	"github.com/gin-gonic/gin"
)

// A rule editor payload is small in normal use. Bounding form parsing prevents
// the decoder endpoint from buffering arbitrarily large attacker input.
const maxRuleConditionRequestBytes int64 = 1 << 20

// ruleConditionRequest is the payload for POST /config/rule-conditions.
//
// Rule stays a json.RawMessage so the panel never models the rule schema: the
// core's own decoder is the only thing that interprets it, which is the whole
// point of this endpoint.
type ruleConditionRequest struct {
	Kind string          `json:"kind"`
	Rule json.RawMessage `json:"rule"`
}

// ruleConditionResponse carries the verdict for one candidate rule. An empty
// list means the rule is accepted as-is.
type ruleConditionResponse struct {
	Issues []core.RuleConditionIssue `json:"issues"`
}

// ValidateRuleConditions validates a single route or DNS rule before it is
// merged into the config.
//
// This exists so the rule editor can refuse a condition-less rule while the
// operator is still looking at it, instead of letting the whole-config save fail
// afterwards. It returns the same core-derived verdict the save path and the
// doctor use, so the three can never disagree.
//
// Write scope is required even though nothing is mutated. The check is part of
// the config-editing surface, and a read-only token has no rule to validate;
// this also keeps the endpoint from becoming a decoder that read tokens can
// drive with arbitrary input.
func (a *ApiService) ValidateRuleConditions(c *gin.Context) {
	if !a.requireTokenScopeAny(c, "config-rule-conditions", "admin", "write") {
		return
	}

	// Only the JSON-in-form-field `data` shape is accepted. There is
	// deliberately no fallback to flattened form keys or a raw JSON body: a
	// caller that got the envelope wrong should be told so, not silently
	// half-understood.
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxRuleConditionRequestBytes)
	if err := c.Request.ParseForm(); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			c.JSON(http.StatusRequestEntityTooLarge, Msg{Success: false, Msg: "rule-conditions: request body is too large"})
			return
		}
		ruleConditionBadRequest(c, "cannot parse form body")
		return
	}

	raw, ok := c.GetPostForm("data")
	if !ok {
		ruleConditionBadRequest(c, "missing form field \"data\"")
		return
	}
	var req ruleConditionRequest
	if err := json.Unmarshal([]byte(raw), &req); err != nil {
		ruleConditionBadRequest(c, "form field \"data\" is not valid JSON")
		return
	}
	if !isJSONObject(req.Rule) {
		ruleConditionBadRequest(c, "\"rule\" must be a JSON object")
		return
	}

	issues, err := core.SingleRuleConditionIssues(req.Kind, req.Rule)
	if err != nil {
		// An undecodable rule is a failed check rather than a server fault: the
		// operator handed us a shape the core would also refuse, and the
		// decoder's message names the offending field. The submitted rule is
		// never echoed back or logged.
		ruleConditionBadRequest(c, err.Error())
		return
	}
	if issues == nil {
		// Always answer with an array so the caller never has to distinguish
		// "no issues" from a missing field.
		issues = []core.RuleConditionIssue{}
	}
	jsonObj(c, ruleConditionResponse{Issues: issues}, nil)
}

func ruleConditionBadRequest(c *gin.Context, reason string) {
	c.JSON(http.StatusBadRequest, Msg{Success: false, Msg: "rule-conditions: " + reason})
}

// isJSONObject reports whether raw is a JSON object, rejecting the null, array
// and scalar shapes that would otherwise decode into a zero-value rule and be
// judged as though the operator had submitted an empty object.
func isJSONObject(raw json.RawMessage) bool {
	trimmed := bytes.TrimSpace(raw)
	return len(trimmed) > 0 && trimmed[0] == '{'
}
