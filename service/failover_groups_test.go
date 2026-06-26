package service

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
)

func TestAssembleFailoverForCore(t *testing.T) {
	// Default policy (hold_current) → direct is NOT appended.
	o := model.Outbound{
		Type:    FailoverType,
		Tag:     "g",
		Options: json.RawMessage(`{"outbounds":["a","b"],"failover":{"probe_target":"https://x.example/","interval":"30s","hysteresis":2}}`),
	}

	got, err := assembleFailoverForCore(o, "direct")
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(got, &m); err != nil {
		t.Fatal(err)
	}
	if m["type"] != "selector" {
		t.Fatalf("assembled type = %v, want selector", m["type"])
	}
	if _, leaked := m["failover"]; leaked {
		t.Fatal("assembled selector must not carry the failover metadata key (sing-box rejects unknown keys)")
	}
	if m["default"] != "a" {
		t.Fatalf("default = %v, want a", m["default"])
	}
	// hold_current default → no direct appended.
	wantMembers := []any{"a", "b"}
	if got := m["outbounds"]; !equalAnySlice(got, wantMembers) {
		t.Fatalf("outbounds = %v, want %v (hold_current default must not append direct)", got, wantMembers)
	}

	// Explicit direct policy → direct IS appended.
	oDirect := model.Outbound{
		Type:    FailoverType,
		Tag:     "g",
		Options: json.RawMessage(`{"outbounds":["a","b"],"failover":{"probe_target":"https://x.example/","interval":"30s","hysteresis":2,"all_down_policy":"direct"}}`),
	}
	gotD, err := assembleFailoverForCore(oDirect, "direct")
	if err != nil {
		t.Fatal(err)
	}
	var mD map[string]any
	_ = json.Unmarshal(gotD, &mD)
	wantDirect := []any{"a", "b", "direct"}
	if got := mD["outbounds"]; !equalAnySlice(got, wantDirect) {
		t.Fatalf("outbounds with direct policy = %v, want %v", got, wantDirect)
	}

	// No direct outbound available → no fallback member appended even with direct policy.
	got2, err := assembleFailoverForCore(oDirect, "")
	if err != nil {
		t.Fatal(err)
	}
	var m2 map[string]any
	_ = json.Unmarshal(got2, &m2)
	if got := m2["outbounds"]; !equalAnySlice(got, []any{"a", "b"}) {
		t.Fatalf("outbounds without direct = %v, want [a b]", got)
	}
}

func equalAnySlice(got any, want []any) bool {
	s, ok := got.([]any)
	if !ok || len(s) != len(want) {
		return false
	}
	for i := range want {
		if s[i] != want[i] {
			return false
		}
	}
	return true
}

func TestValidateFailoverGroup(t *testing.T) {
	initSettingTestDB(t)
	db := database.GetDB()
	createTestOutbound(t, "m1", 1081)
	createTestOutbound(t, "m2", 1082)
	if err := db.Create(&model.Outbound{Type: "selector", Tag: "sel", Options: json.RawMessage(`{"outbounds":["m1"]}`)}).Error; err != nil {
		t.Fatal(err)
	}

	fo := func(opts string) model.Outbound {
		return model.Outbound{Type: FailoverType, Tag: "fo", Options: json.RawMessage(opts)}
	}

	if err := validateFailoverGroup(db, fo(`{"outbounds":["m1","m2"]}`)); err != nil {
		t.Fatalf("valid multi-member group rejected: %v", err)
	}
	if err := validateFailoverGroup(db, fo(`{"outbounds":["m1"]}`)); err != nil {
		t.Fatalf("valid single-member group rejected: %v", err)
	}

	rejects := map[string]string{
		"empty":          `{"outbounds":[]}`,
		"missing member": `{"outbounds":["m1","ghost"]}`,
		"group member":   `{"outbounds":["m1","sel"]}`,
		"self ref":       `{"outbounds":["fo"]}`,
		"duplicate":      `{"outbounds":["m1","m1"]}`,
		"bad scheme":     `{"outbounds":["m1"],"failover":{"probe_target":"ftp://x.example/"}}`,
		"tiny interval":  `{"outbounds":["m1"],"failover":{"interval":"1s"}}`,
		"bad policy":     `{"outbounds":["m1"],"failover":{"all_down_policy":"unknown"}}`,
	}
	for name, opts := range rejects {
		if err := validateFailoverGroup(db, fo(opts)); err == nil {
			t.Fatalf("%s: expected rejection, got nil", name)
		}
	}

	// Valid all-down policies must be accepted.
	for _, policy := range []string{"hold_current", "block", "direct"} {
		opts := fmt.Sprintf(`{"outbounds":["m1"],"failover":{"all_down_policy":%q}}`, policy)
		if err := validateFailoverGroup(db, fo(opts)); err != nil {
			t.Fatalf("valid all_down_policy %q rejected: %v", policy, err)
		}
	}
}

// A failover group captures its member adapters at selector construction, so
// editing a member must escalate to a full core restart (not a hot reload).
func TestConfigSaveFailoverGroupMemberRestartsCore(t *testing.T) {
	initSettingTestDB(t)
	coreInstance := startTestCore(t)
	member := createTestOutbound(t, "fo-member", 1080)
	group := model.Outbound{
		Type:    FailoverType,
		Tag:     "auto-fo",
		Options: json.RawMessage(`{"outbounds":["fo-member","direct"],"failover":{"interval":"30s"}}`),
	}
	if err := database.GetDB().Create(&group).Error; err != nil {
		t.Fatal(err)
	}
	seedConfigBlob(t, json.RawMessage(`{"log":{"disabled":true}}`))

	recorder := &outboundOpsRecorder{}
	recorder.stubOutboundHooks(t)

	before := coreInstance.GetInstance()
	configService := NewConfigServiceWithRuntime(NewRuntime(coreInstance))
	if _, err := configService.Save("outbounds", "edit", socksPayload(member.Id, "fo-member", 1081), "", "admin", "example.com"); err != nil {
		t.Fatal(err)
	}

	if len(recorder.ops) != 0 {
		t.Fatalf("failover-member edit must not hot-reload, got ops %v", recorder.ops)
	}
	if coreInstance.GetInstance() == before {
		t.Fatal("failover-member edit must restart the core")
	}
}
