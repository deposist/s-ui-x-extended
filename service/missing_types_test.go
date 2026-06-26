package service

import (
	"encoding/json"
	"testing"

	"github.com/deposist/s-ui-x-extended/database"
	"gorm.io/gorm"
)

func getDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := database.GetDB()
	if db == nil {
		t.Skip("database not initialized")
	}
	return db
}

// TestCoreFailoverOutboundRoundTrip verifies that a core-failover outbound
// with outbounds, strategy, and delay can be saved and loaded without losing
// fields.
func TestCoreFailoverOutboundRoundTrip(t *testing.T) {
	initSettingTestDB(t)
	db := getDB(t)

	payload := json.RawMessage(`{"type":"core-failover","tag":"cf-test","outbounds":["m1"],"strategy":"sequential","delay":"5s"}`)

	// Create a member outbound so validation passes.
	createTestOutbound(t, "m1", 1081)

	configService := NewConfigServiceWithRuntime(NewRuntime(nil))
	_, err := configService.Save("outbounds", "new", payload, "", "admin", "example.com")
	if err != nil {
		t.Fatalf("save core-failover outbound: %v", err)
	}

	// Load it back.
	var result map[string]any
	if err := db.Raw(`SELECT options FROM outbounds WHERE tag = 'cf-test'`).Scan(&result).Error; err != nil {
		t.Fatal(err)
	}
	opts, _ := result["options"].(string)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(opts), &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed["strategy"] != "sequential" {
		t.Fatalf("strategy = %v, want sequential", parsed["strategy"])
	}
	if parsed["delay"] != "5s" {
		t.Fatalf("delay = %v, want 5s", parsed["delay"])
	}
}

// TestBlockOutboundRoundTrip verifies that a block outbound (no fields) can be
// saved and loaded.
func TestBlockOutboundRoundTrip(t *testing.T) {
	initSettingTestDB(t)
	db := getDB(t)

	payload := json.RawMessage(`{"type":"block","tag":"block-test"}`)

	configService := NewConfigServiceWithRuntime(NewRuntime(nil))
	_, err := configService.Save("outbounds", "new", payload, "", "admin", "example.com")
	if err != nil {
		t.Fatalf("save block outbound: %v", err)
	}

	var count int64
	db.Raw(`SELECT count(*) FROM outbounds WHERE tag = 'block-test'`).Scan(&count)
	if count != 1 {
		t.Fatalf("block outbound not saved, count = %d", count)
	}
}

// TestBondInboundRoundTrip verifies that a bond inbound with an inbounds list
// can be saved and loaded.
func TestBondInboundRoundTrip(t *testing.T) {
	initSettingTestDB(t)
	db := getDB(t)

	// Create a member inbound so validation passes.
	createTestInbound(t, "member-in")

	payload := json.RawMessage(`{"type":"bond","tag":"bond-test","listen":"0.0.0.0","listen_port":1080,"inbounds":["member-in"]}`)

	configService := NewConfigServiceWithRuntime(NewRuntime(nil))
	_, err := configService.Save("inbounds", "new", payload, "", "admin", "example.com")
	if err != nil {
		t.Fatalf("save bond inbound: %v", err)
	}

	var result map[string]any
	if err := db.Raw(`SELECT options FROM inbounds WHERE tag = 'bond-test'`).Scan(&result).Error; err != nil {
		t.Fatal(err)
	}
	opts, _ := result["options"].(string)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(opts), &parsed); err != nil {
		t.Fatal(err)
	}
	inbounds, ok := parsed["inbounds"].([]any)
	if !ok || len(inbounds) != 1 || inbounds[0] != "member-in" {
		t.Fatalf("inbounds = %v, want [member-in]", parsed["inbounds"])
	}
}

// TestCoreFailoverInboundRoundTrip verifies that a core-failover inbound with
// an inbounds list can be saved and loaded.
func TestCoreFailoverInboundRoundTrip(t *testing.T) {
	initSettingTestDB(t)
	db := getDB(t)

	createTestInbound(t, "cf-member")

	payload := json.RawMessage(`{"type":"core-failover","tag":"cf-in-test","listen":"0.0.0.0","listen_port":1080,"inbounds":["cf-member"]}`)

	configService := NewConfigServiceWithRuntime(NewRuntime(nil))
	_, err := configService.Save("inbounds", "new", payload, "", "admin", "example.com")
	if err != nil {
		t.Fatalf("save core-failover inbound: %v", err)
	}

	var result map[string]any
	if err := db.Raw(`SELECT options FROM inbounds WHERE tag = 'cf-in-test'`).Scan(&result).Error; err != nil {
		t.Fatal(err)
	}
	opts, _ := result["options"].(string)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(opts), &parsed); err != nil {
		t.Fatal(err)
	}
	inbounds, ok := parsed["inbounds"].([]any)
	if !ok || len(inbounds) != 1 || inbounds[0] != "cf-member" {
		t.Fatalf("inbounds = %v, want [cf-member]", parsed["inbounds"])
	}
}

// TestCoreFailoverOutboundValidationRejectsMissingMember verifies that
// validation rejects a core-failover outbound with a non-existent member.
func TestCoreFailoverOutboundValidationRejectsMissingMember(t *testing.T) {
	initSettingTestDB(t)

	payload := json.RawMessage(`{"type":"core-failover","tag":"cf-bad","outbounds":["ghost"],"strategy":"sequential"}`)

	configService := NewConfigServiceWithRuntime(NewRuntime(nil))
	_, err := configService.Save("outbounds", "new", payload, "", "admin", "example.com")
	if err == nil {
		t.Fatal("expected error for missing member, got nil")
	}
}

// TestBondInboundValidationRejectsMissingMember verifies that validation
// rejects a bond inbound with a non-existent member.
func TestBondInboundValidationRejectsMissingMember(t *testing.T) {
	initSettingTestDB(t)

	payload := json.RawMessage(`{"type":"bond","tag":"bond-bad","listen":"0.0.0.0","listen_port":1080,"inbounds":["ghost"]}`)

	configService := NewConfigServiceWithRuntime(NewRuntime(nil))
	_, err := configService.Save("inbounds", "new", payload, "", "admin", "example.com")
	if err == nil {
		t.Fatal("expected error for missing member, got nil")
	}
}
