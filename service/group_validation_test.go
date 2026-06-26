package service

import (
	"encoding/json"
	"testing"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
)

func TestValidateFallbackGroup(t *testing.T) {
	initSettingTestDB(t)
	db := database.GetDB()
	createTestOutbound(t, "m1", 1081)
	createTestOutbound(t, "m2", 1082)

	fb := func(opts string) model.Outbound {
		return model.Outbound{Type: "fallback", Tag: "fb", Options: json.RawMessage(opts)}
	}

	if err := validateFallbackGroup(db, fb(`{"outbounds":["m1","m2"]}`)); err != nil {
		t.Fatalf("valid fallback group rejected: %v", err)
	}
	if err := validateFallbackGroup(db, fb(`{"outbounds":["m1"]}`)); err != nil {
		t.Fatalf("valid single-member fallback rejected: %v", err)
	}

	rejects := map[string]string{
		"empty":          `{"outbounds":[]}`,
		"missing member": `{"outbounds":["m1","ghost"]}`,
		"self ref":       `{"outbounds":["fb"]}`,
		"duplicate":      `{"outbounds":["m1","m1"]}`,
	}
	for name, opts := range rejects {
		if err := validateFallbackGroup(db, fb(opts)); err == nil {
			t.Fatalf("%s: expected rejection, got nil", name)
		}
	}
}

func TestValidateSelectorURLTestGroupWithProviders(t *testing.T) {
	initSettingTestDB(t)
	db := database.GetDB()
	createTestOutbound(t, "m1", 1081)

	if err := db.Create(&model.Provider{Type: "remote", Tag: "prov-a", Options: json.RawMessage(`{"url":"https://example.com/sub"}`)}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.Provider{Type: "remote", Tag: "prov-b", Options: json.RawMessage(`{"url":"https://example.com/sub2"}`)}).Error; err != nil {
		t.Fatal(err)
	}

	sel := func(opts string) model.Outbound {
		return model.Outbound{Type: "selector", Tag: "grp", Options: json.RawMessage(opts)}
	}

	// Valid: static members + providers.
	if err := validateSelectorURLTestGroup(db, sel(`{"outbounds":["m1"],"providers":["prov-a"]}`)); err != nil {
		t.Fatalf("valid selector with providers rejected: %v", err)
	}

	// Valid: only providers, no static members.
	if err := validateSelectorURLTestGroup(db, sel(`{"outbounds":[],"providers":["prov-a"]}`)); err != nil {
		t.Fatalf("valid selector with only providers rejected: %v", err)
	}

	// Valid: use_all_providers without listing any.
	if err := validateSelectorURLTestGroup(db, sel(`{"outbounds":[],"use_all_providers":true}`)); err != nil {
		t.Fatalf("valid selector with use_all_providers rejected: %v", err)
	}

	// Invalid: no members and no providers.
	if err := validateSelectorURLTestGroup(db, sel(`{"outbounds":[]}`)); err == nil {
		t.Fatal("selector with no members or providers must be rejected")
	}

	// Invalid: missing provider.
	if err := validateSelectorURLTestGroup(db, sel(`{"outbounds":["m1"],"providers":["ghost"]}`)); err == nil {
		t.Fatal("selector with missing provider must be rejected")
	}

	// Invalid: duplicate provider.
	if err := validateSelectorURLTestGroup(db, sel(`{"outbounds":["m1"],"providers":["prov-a","prov-a"]}`)); err == nil {
		t.Fatal("selector with duplicate provider must be rejected")
	}

	// Invalid: group member is itself a group (cycle prevention).
	if err := db.Create(&model.Outbound{Type: "selector", Tag: "other-grp", Options: json.RawMessage(`{"outbounds":["m1"]}`)}).Error; err != nil {
		t.Fatal(err)
	}
	if err := validateSelectorURLTestGroup(db, sel(`{"outbounds":["other-grp"]}`)); err == nil {
		t.Fatal("selector with group member must be rejected to prevent cycles")
	}
}

// Provider deletion must be blocked when a group references it.
func TestProviderDeleteBlockedByGroupReference(t *testing.T) {
	initSettingTestDB(t)
	db := database.GetDB()
	createTestOutbound(t, "m1", 1081)
	if err := db.Create(&model.Provider{Type: "remote", Tag: "prov-a", Options: json.RawMessage(`{"url":"https://example.com/sub"}`)}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.Outbound{Type: "selector", Tag: "grp", Options: json.RawMessage(`{"outbounds":["m1"],"providers":["prov-a"]}`)}).Error; err != nil {
		t.Fatal(err)
	}

	var outbounds []model.Outbound
	if err := db.Model(model.Outbound{}).Find(&outbounds).Error; err != nil {
		t.Fatal(err)
	}
	refs := scanOutboundRowsForProviderTag(outbounds, "prov-a", 0)
	if len(refs) != 1 || refs[0].Locator != `selector "grp" (providers list)` {
		t.Fatalf("provider ref scan = %+v, want grp", refs)
	}
}
