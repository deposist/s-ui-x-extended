package service

import (
	"encoding/json"
	"testing"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
)

func TestAssembleFailoverForCore(t *testing.T) {
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
	wantMembers := []any{"a", "b", "direct"}
	if got := m["outbounds"]; !equalAnySlice(got, wantMembers) {
		t.Fatalf("outbounds = %v, want %v (direct appended as all-down fallback)", got, wantMembers)
	}

	// No direct outbound available → no fallback member appended.
	got2, err := assembleFailoverForCore(o, "")
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
		"empty":         `{"outbounds":[]}`,
		"missing member": `{"outbounds":["m1","ghost"]}`,
		"group member":  `{"outbounds":["m1","sel"]}`,
		"self ref":      `{"outbounds":["fo"]}`,
		"duplicate":     `{"outbounds":["m1","m1"]}`,
		"bad scheme":    `{"outbounds":["m1"],"failover":{"probe_target":"ftp://x.example/"}}`,
		"tiny interval": `{"outbounds":["m1"],"failover":{"interval":"1s"}}`,
	}
	for name, opts := range rejects {
		if err := validateFailoverGroup(db, fo(opts)); err == nil {
			t.Fatalf("%s: expected rejection, got nil", name)
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
