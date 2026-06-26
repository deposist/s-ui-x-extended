package service

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
)

func TestOutboundHealthSnapshotSetAndGet(t *testing.T) {
	t.Cleanup(ResetHealthSnapshots)

	SetOutboundHealth("proxy-a", true, 95, "")
	s, ok := OutboundHealthSnapshotFor("proxy-a")
	if !ok {
		t.Fatal("expected snapshot for proxy-a")
	}
	if s.Status != "healthy" || s.DelayMs != 95 {
		t.Fatalf("snapshot = %+v, want healthy/95ms", s)
	}

	SetOutboundHealth("proxy-b", false, 0, "connection refused")
	s2, _ := OutboundHealthSnapshotFor("proxy-b")
	if s2.Status != "down" || s2.Error != "connection refused" {
		t.Fatalf("snapshot = %+v, want down/connection refused", s2)
	}
}

func TestOutboundHealthSnapshotPrune(t *testing.T) {
	t.Cleanup(ResetHealthSnapshots)

	SetOutboundHealth("keep", true, 10, "")
	SetOutboundHealth("drop", true, 20, "")
	PruneOutboundHealth([]string{"keep"})

	if _, ok := OutboundHealthSnapshotFor("drop"); ok {
		t.Fatal("pruned snapshot should not be found")
	}
	if _, ok := OutboundHealthSnapshotFor("keep"); !ok {
		t.Fatal("kept snapshot should still be found")
	}
}

func TestOutboundHealthSnapshotRetention(t *testing.T) {
	t.Cleanup(ResetHealthSnapshots)

	// Insert a snapshot and then manipulate its timestamp to be old.
	SetOutboundHealth("old-proxy", true, 50, "")
	outboundHealthMu.Lock()
	s := outboundHealthMap["old-proxy"]
	s.CheckedAt = time.Now().Add(-outboundHealthMaxAge - time.Minute).Unix()
	outboundHealthMap["old-proxy"] = s
	outboundHealthMu.Unlock()

	all := AllOutboundHealthSnapshots()
	if _, found := all["old-proxy"]; found {
		t.Fatal("expired snapshot should be excluded from AllOutboundHealthSnapshots")
	}
}

func TestOutboundHealthErrorIsTruncated(t *testing.T) {
	t.Cleanup(ResetHealthSnapshots)

	longErr := ""
	for i := 0; i < 300; i++ {
		longErr += "x"
	}
	SetOutboundHealth("verbose", false, 0, longErr)
	s, _ := OutboundHealthSnapshotFor("verbose")
	if len(s.Error) > 200 {
		t.Fatalf("error not truncated: len=%d", len(s.Error))
	}
}

func TestOutboundHealthStoreDeletesExpiredEntries(t *testing.T) {
	t.Cleanup(ResetHealthSnapshots)

	old := time.Now().Add(-outboundHealthMaxAge - time.Minute)
	RecordOutboundHealth("old", true, 1, "", old)
	RecordOutboundHealth("fresh", true, 2, "", time.Now())

	all := AllOutboundHealthSnapshots()
	if _, found := all["old"]; found {
		t.Fatal("expired outbound snapshot should be removed from returned map")
	}
	outboundHealthMu.RLock()
	_, stillStored := outboundHealthMap["old"]
	outboundHealthMu.RUnlock()
	if stillStored {
		t.Fatal("expired outbound snapshot should be physically deleted")
	}
}

func TestOutboundHealthStoreCapsEntries(t *testing.T) {
	t.Cleanup(ResetHealthSnapshots)

	base := time.Now()
	for i := 0; i < outboundHealthMaxEntries+25; i++ {
		RecordOutboundHealth(fmt.Sprintf("proxy-%04d", i), true, 1, "", base)
	}
	all := AllOutboundHealthSnapshots()
	if len(all) > outboundHealthMaxEntries {
		t.Fatalf("outbound health store has %d entries, want <= %d", len(all), outboundHealthMaxEntries)
	}
	if _, found := all["proxy-0000"]; found {
		t.Fatal("oldest outbound snapshot should be evicted when cap is exceeded")
	}
}

func TestProviderHealthSnapshotSetAndGet(t *testing.T) {
	t.Cleanup(ResetHealthSnapshots)

	SetProviderHealth(ProviderHealthSnapshot{
		Tag:           "prov-a",
		Status:        "degraded",
		UpdatedAt:     time.Now().Unix(),
		OutboundCount: 5,
		HealthyCount:  3,
	})
	all := AllProviderHealthSnapshots()
	if len(all) != 1 {
		t.Fatalf("expected 1 provider snapshot, got %d", len(all))
	}
	if all["prov-a"].Status != "degraded" || all["prov-a"].HealthyCount != 3 {
		t.Fatalf("snapshot = %+v, want degraded/3", all["prov-a"])
	}
}

func TestProviderHealthSnapshotPrune(t *testing.T) {
	t.Cleanup(ResetHealthSnapshots)

	SetProviderHealth(ProviderHealthSnapshot{Tag: "keep", UpdatedAt: time.Now().Unix()})
	SetProviderHealth(ProviderHealthSnapshot{Tag: "drop", UpdatedAt: time.Now().Unix()})
	PruneProviderHealth([]string{"keep"})

	all := AllProviderHealthSnapshots()
	if _, found := all["drop"]; found {
		t.Fatal("pruned provider snapshot should not be found")
	}
	if _, found := all["keep"]; !found {
		t.Fatal("kept provider snapshot should still be found")
	}
}

func TestProviderHealthSnapshotJSONContract(t *testing.T) {
	payload, err := json.Marshal(ProviderHealthSnapshot{
		Tag:           "prov-a",
		Status:        "degraded",
		UpdatedAt:     123,
		OutboundCount: 5,
		HealthyCount:  3,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(payload) {
		t.Fatalf("invalid json: %s", payload)
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["outbounds"] != float64(5) || decoded["healthyOutbounds"] != float64(3) {
		t.Fatalf("provider health json = %s, want outbounds/healthyOutbounds", payload)
	}
	if _, legacy := decoded["outboundCount"]; legacy {
		t.Fatalf("provider health json leaked legacy outboundCount key: %s", payload)
	}
}

func TestProviderHealthAggregateFromOutboundSnapshots(t *testing.T) {
	initSettingTestDB(t)
	t.Cleanup(ResetHealthSnapshots)
	db := database.GetDB()
	if err := db.Create(&model.Provider{Type: "inline", Tag: "prov-inline", Options: json.RawMessage(`{"outbounds":[{"type":"direct","tag":"ok"},{"type":"direct","tag":"bad"},{"type":"direct","tag":"missing"}]}`)}).Error; err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	RecordOutboundHealth("ok", true, 8, "", now)
	RecordOutboundHealth("bad", false, 0, "dial timeout", now.Add(time.Second))
	RefreshProviderHealth(now.Add(2 * time.Second))

	all := AllProviderHealthSnapshots()
	snapshot, found := all["prov-inline"]
	if !found {
		t.Fatalf("provider snapshot missing: %+v", all)
	}
	if snapshot.Status != "degraded" || snapshot.OutboundCount != 3 || snapshot.HealthyCount != 1 {
		t.Fatalf("provider snapshot = %+v, want degraded/3/1", snapshot)
	}
	if snapshot.LastError == "" {
		t.Fatalf("degraded provider should explain failed/missing members: %+v", snapshot)
	}
}

func TestProviderHealthAggregateIgnoresMalformedProviderMembers(t *testing.T) {
	initSettingTestDB(t)
	t.Cleanup(ResetHealthSnapshots)
	db := database.GetDB()
	if err := db.Create(&model.Provider{Type: "inline", Tag: "prov-malformed", Options: json.RawMessage(`{"outbounds":[123,{"type":"direct"},{"type":"direct","tag":"ok"}]}`)}).Error; err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	RecordOutboundHealth("ok", true, 4, "", now)
	snapshots := (&ProviderService{}).AggregateProviderHealth(now.Add(time.Second))
	if len(snapshots) != 1 {
		t.Fatalf("snapshots = %+v, want one provider", snapshots)
	}
	if snapshots[0].Status != "healthy" || snapshots[0].OutboundCount != 1 || snapshots[0].HealthyCount != 1 {
		t.Fatalf("snapshot = %+v, want healthy single valid member", snapshots[0])
	}
}

func TestProviderHealthAggregateCapsMemberPayloadButCountsAll(t *testing.T) {
	initSettingTestDB(t)
	t.Cleanup(ResetHealthSnapshots)
	db := database.GetDB()
	members := make([]map[string]string, 0, providerHealthMaxMembers+10)
	now := time.Now()
	for i := 0; i < providerHealthMaxMembers+10; i++ {
		tag := fmt.Sprintf("m-%03d", i)
		members = append(members, map[string]string{"type": "direct", "tag": tag})
		RecordOutboundHealth(tag, true, 1, "", now)
	}
	options, err := json.Marshal(map[string]any{"outbounds": members})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.Provider{Type: "inline", Tag: "prov-big", Options: options}).Error; err != nil {
		t.Fatal(err)
	}

	snapshots := (&ProviderService{}).AggregateProviderHealth(now.Add(time.Second))
	if len(snapshots) != 1 {
		t.Fatalf("snapshots = %+v, want one provider", snapshots)
	}
	if snapshots[0].Status != "healthy" {
		t.Fatalf("snapshot status = %s, want healthy", snapshots[0].Status)
	}
	if snapshots[0].OutboundCount != providerHealthMaxMembers+10 || snapshots[0].HealthyCount != providerHealthMaxMembers+10 {
		t.Fatalf("snapshot counts = %+v, want all members counted", snapshots[0])
	}
	if len(snapshots[0].Members) != providerHealthMaxMembers {
		t.Fatalf("member payload len = %d, want cap %d", len(snapshots[0].Members), providerHealthMaxMembers)
	}
}

func TestHealthErrorTruncationIsUTF8SafeAndByteBounded(t *testing.T) {
	longErr := "ошибка-" + strings.Repeat("界", 120)
	truncated := truncateForHealth(longErr, 200)
	if len(truncated) > 200 {
		t.Fatalf("truncated error is %d bytes, want <= 200", len(truncated))
	}
	if !utf8.ValidString(truncated) {
		t.Fatalf("truncated error is not valid UTF-8: %q", truncated)
	}
}

func TestResetHealthSnapshotsClearsAll(t *testing.T) {
	SetOutboundHealth("a", true, 1, "")
	SetProviderHealth(ProviderHealthSnapshot{Tag: "b", UpdatedAt: time.Now().Unix()})
	ResetHealthSnapshots()

	if s := AllOutboundHealthSnapshots(); len(s) > 0 {
		t.Fatal("outbound health not cleared")
	}
	if s := AllProviderHealthSnapshots(); len(s) > 0 {
		t.Fatal("provider health not cleared")
	}
}

// TestOnlinesIncludesHealthSnapshots verifies that GetOnlines merges health
// data into the onlines payload.
func TestOnlinesIncludesHealthSnapshots(t *testing.T) {
	t.Cleanup(ResetHealthSnapshots)

	SetOutboundHealth("proxy-x", true, 42, "")
	SetProviderHealth(ProviderHealthSnapshot{Tag: "prov-x", Status: "healthy", UpdatedAt: time.Now().Unix()})

	on, err := (&StatsService{}).GetOnlines()
	if err != nil {
		t.Fatal(err)
	}
	if len(on.OutboundHealth) != 1 || on.OutboundHealth["proxy-x"].DelayMs != 42 {
		t.Fatalf("outbound health not in onlines: %+v", on.OutboundHealth)
	}
	if len(on.ProviderHealth) != 1 || on.ProviderHealth["prov-x"].Status != "healthy" {
		t.Fatalf("provider health not in onlines: %+v", on.ProviderHealth)
	}
}
