package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x-extended/core"

	"github.com/gin-gonic/gin"
)

// postRuleConditions drives the handler through the exact envelope the frontend
// uses: a single form field named "data" holding the whole JSON payload.
func postRuleConditions(t *testing.T, payload string, scope string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	form := url.Values{}
	form.Set("data", payload)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/config/rule-conditions", strings.NewReader(form.Encode()))
	c.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if scope != "" {
		c.Set(apiTokenScopeKey, scope)
	}
	(&ApiService{}).ValidateRuleConditions(c)
	return recorder
}

// decodeIssues unwraps the success envelope and returns the reported issues.
func decodeIssues(t *testing.T, body string) []core.RuleConditionIssue {
	t.Helper()
	var envelope struct {
		Success bool `json:"success"`
		Obj     struct {
			Issues []core.RuleConditionIssue `json:"issues"`
		} `json:"obj"`
	}
	if err := json.Unmarshal([]byte(body), &envelope); err != nil {
		t.Fatalf("decode envelope: %v (body %s)", err, body)
	}
	if !envelope.Success {
		t.Fatalf("expected a success envelope, got %s", body)
	}
	return envelope.Obj.Issues
}

func TestValidateRuleConditionsReportsIssues(t *testing.T) {
	t.Run("logical rule without branches is reported with its exact path", func(t *testing.T) {
		recorder := postRuleConditions(t, `{"kind":"route","rule":{"type":"logical","mode":"and","rules":[],"outbound":"direct"}}`, "")
		if recorder.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
		}
		issues := decodeIssues(t, recorder.Body.String())
		if len(issues) != 1 {
			t.Fatalf("expected one issue, got %d: %s", len(issues), recorder.Body.String())
		}
		if issues[0].Path != "route.rules[0].rules" {
			t.Fatalf("unexpected path %q", issues[0].Path)
		}
		if issues[0].Code != core.RuleConditionCodeMissingConditions {
			t.Fatalf("unexpected code %q", issues[0].Code)
		}
	})

	// A nested empty branch beside a real sibling survives decoding and is
	// accepted by the core, so the endpoint must not invent an issue for it.
	// This is exactly the shape the replaced heuristic used to block.
	t.Run("nested empty branch beside a sibling is accepted", func(t *testing.T) {
		recorder := postRuleConditions(t, `{"kind":"route","rule":{"type":"logical","mode":"and","rules":[{"domain":["a.example"]},{}],"outbound":"direct"}}`, "")
		if recorder.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
		}
		if issues := decodeIssues(t, recorder.Body.String()); len(issues) != 0 {
			t.Fatalf("expected no issues, got %+v", issues)
		}
	})

	// An action-only rule is a legitimate catch-all that the old "meaningful
	// field" heuristic rejected.
	t.Run("action-only rule is accepted", func(t *testing.T) {
		recorder := postRuleConditions(t, `{"kind":"route","rule":{"action":"sniff"}}`, "")
		if issues := decodeIssues(t, recorder.Body.String()); len(issues) != 0 {
			t.Fatalf("expected no issues, got %+v", issues)
		}
	})

	// A lone empty DNS rule is discarded during decoding rather than rejected,
	// so the operator must be told the rule would not exist even though the
	// config would load.
	t.Run("empty dns rule is reported as discarded", func(t *testing.T) {
		recorder := postRuleConditions(t, `{"kind":"dns","rule":{}}`, "")
		if recorder.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
		}
		issues := decodeIssues(t, recorder.Body.String())
		if len(issues) != 1 || issues[0].Code != core.RuleConditionCodeDroppedRule {
			t.Fatalf("expected one dropped-rule issue, got %+v", issues)
		}
	})

	// The response always carries an array so the caller never has to tell "no
	// issues" apart from a missing field.
	t.Run("valid rule returns an explicit empty array", func(t *testing.T) {
		recorder := postRuleConditions(t, `{"kind":"dns","rule":{"domain":["a.example"]}}`, "")
		if body := recorder.Body.String(); !strings.Contains(body, `"issues":[]`) {
			t.Fatalf("expected an explicit empty array, got %s", body)
		}
	})
}

func TestValidateRuleConditionsRejectsMalformedPayloads(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		payload string
	}{
		{"not JSON", `{`},
		{"unknown kind", `{"kind":"tls","rule":{}}`},
		{"missing rule", `{"kind":"route"}`},
		{"null rule", `{"kind":"route","rule":null}`},
		{"array rule", `{"kind":"route","rule":[]}`},
		{"scalar rule", `{"kind":"route","rule":7}`},
		{"undecodable rule", `{"kind":"route","rule":{"type":"logical","mode":5}}`},
		{"unknown rule field", `{"kind":"route","rule":{"nope":true}}`},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			recorder := postRuleConditions(t, testCase.payload, "")
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d: %s", recorder.Code, recorder.Body.String())
			}
			if !strings.Contains(recorder.Body.String(), `"success":false`) {
				t.Fatalf("expected a failure envelope, got %s", recorder.Body.String())
			}
		})
	}

	// Only the `data` field is read; there is no fallback to flattened form keys.
	t.Run("flattened form keys are not accepted", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		form := url.Values{}
		form.Set("kind", "route")
		form.Set("rule", `{"domain":["a.example"]}`)
		c.Request = httptest.NewRequest(http.MethodPost, "/api/config/rule-conditions", strings.NewReader(form.Encode()))
		c.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		(&ApiService{}).ValidateRuleConditions(c)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", recorder.Code, recorder.Body.String())
		}
	})

	// The submitted rule must never be echoed back, so a rejected payload cannot
	// leak config fragments into logs or error surfaces.
	t.Run("rejection does not echo the rule", func(t *testing.T) {
		recorder := postRuleConditions(t, `{"kind":"route","rule":{"nope":"s3cret-marker"}}`, "")
		if strings.Contains(recorder.Body.String(), "s3cret-marker") {
			t.Fatalf("response echoed the submitted rule: %s", recorder.Body.String())
		}
	})
}

func TestValidateRuleConditionsRejectsOversizedPayload(t *testing.T) {
	gigantic := strings.Repeat("x", int(maxRuleConditionRequestBytes)+1)
	recorder := postRuleConditions(t, gigantic, "")
	if recorder.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "request body is too large") {
		t.Fatalf("unexpected response: %s", recorder.Body.String())
	}
}

// TestValidateRuleConditionsScopes covers the bearer-token surface. Denial is
// audited, and the audit write needs a real session-backed router, so these run
// through the shared helpers rather than a bare test context.
func TestValidateRuleConditionsScopes(t *testing.T) {
	payload := `{"kind":"route","rule":{"domain":["a.example"]}}`

	for _, tt := range []struct {
		name       string
		scope      string
		hasScope   bool
		wantStatus int
	}{
		// A browser cookie session carries no token scope at all and is let
		// through by requireTokenScopeAny.
		{name: "cookie session without scope", wantStatus: http.StatusOK},
		{name: "admin bearer scope", scope: "admin", hasScope: true, wantStatus: http.StatusOK},
		{name: "write bearer scope", scope: "write", hasScope: true, wantStatus: http.StatusOK},
		{name: "read bearer scope", scope: "read", hasScope: true, wantStatus: http.StatusForbidden},
		{name: "observability bearer scope", scope: "observability", hasScope: true, wantStatus: http.StatusForbidden},
		{name: "unknown bearer scope", scope: "unknown", hasScope: true, wantStatus: http.StatusForbidden},
	} {
		t.Run(tt.name, func(t *testing.T) {
			settingService := initSessionTestDB(t)
			handler := (&ApiService{}).ValidateRuleConditions
			if tt.hasScope {
				handler = withTestTokenScope("api-user", tt.scope, handler)
			}
			router, cookies := newAuthenticatedTestRouter(t, settingService, func(router *gin.Engine) {
				router.POST("/api/config/rule-conditions", handler)
			})

			form := url.Values{}
			form.Set("data", payload)
			request := httptest.NewRequest(http.MethodPost, "/api/config/rule-conditions", strings.NewReader(form.Encode()))
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			recorder := performAuthenticatedTestRequest(router, request, cookies...)

			if recorder.Code != tt.wantStatus {
				t.Fatalf("expected %d, got %d: %s", tt.wantStatus, recorder.Code, recorder.Body.String())
			}
			if tt.wantStatus == http.StatusForbidden && !strings.Contains(recorder.Body.String(), "insufficient scope") {
				t.Fatalf("expected a scope denial, got %s", recorder.Body.String())
			}
		})
	}
}

// TestValidateRuleConditionsRoutesRegistered proves the endpoint is reachable on
// both surfaces. The apiv2 case is the load-bearing one: /:postAction matches a
// single segment, so this two-segment path exists only because it is declared
// explicitly ahead of the dispatcher.
func TestValidateRuleConditionsRoutesRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)

	for _, surface := range []struct {
		name  string
		path  string
		build func(*gin.Engine)
	}{
		{
			name: "api",
			path: "/api/config/rule-conditions",
			build: func(engine *gin.Engine) {
				(&APIHandler{}).registerGroupedRoutes(engine.Group("/api"))
			},
		},
		{
			name: "apiv2",
			path: "/apiv2/config/rule-conditions",
			build: func(engine *gin.Engine) {
				(&APIv2Handler{tokens: map[string]TokenInMemory{}}).initRouter(engine.Group("/apiv2"))
			},
		},
	} {
		t.Run(surface.name, func(t *testing.T) {
			engine := gin.New()
			surface.build(engine)
			found := false
			for _, route := range engine.Routes() {
				if route.Method == http.MethodPost && route.Path == surface.path {
					found = true
				}
			}
			if !found {
				t.Fatalf("POST %s is not registered", surface.path)
			}
		})
	}
}
