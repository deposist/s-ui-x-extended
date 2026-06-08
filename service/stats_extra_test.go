package service

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/deposist/s-ui-x/core"
	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
	"github.com/deposist/s-ui-x/realtime"
	"gorm.io/gorm"
)

func TestStatsServiceSaveStatsWithEmptyStats(t *testing.T) {
	coreInstance := core.NewCore()
	if err := coreInstance.Start([]byte(`{"log":{"disabled":true},"inbounds":[],"outbounds":[{"type":"direct","tag":"direct"}]}`)); err != nil {
		t.Skipf("minimal core start unavailable for empty-stats regression: %v", err)
	}
	t.Cleanup(func() {
		_ = coreInstance.Stop()
	})

	onlineResourcesMu.Lock()
	onlineResources = &onlines{User: []string{"stale-user"}}
	onlineResourcesMu.Unlock()

	statsService := &StatsService{Runtime: NewRuntime(coreInstance)}
	if err := statsService.SaveStats(true); err != nil {
		t.Fatal(err)
	}
	current, err := statsService.GetOnlines()
	if err != nil {
		t.Fatal(err)
	}
	if len(current.User) != 0 || len(current.Inbound) != 0 || len(current.Outbound) != 0 {
		t.Fatalf("empty stats should clear online resources: %#v", current)
	}
}

func TestStatsServiceSaveStatsCommitFailureAuditsAndReturnsIssue26(t *testing.T) {
	initSettingTestDB(t)
	seedStatsBenchClients(t, 1)

	prevAuditSync := AuditSyncForTest
	AuditSyncForTest = true
	t.Cleanup(func() { AuditSyncForTest = prevAuditSync })

	realtime.CloseAll("issue26_reset")
	t.Cleanup(func() { realtime.CloseAll("issue26_done") })
	ch := make(chan realtime.Event, 4)
	unregister := realtime.Register(&realtime.ClientHandle{
		User:   "admin",
		Scope:  realtime.ScopeAdmin,
		SendCh: ch,
	})
	defer unregister()

	commitErr := errors.New("issue26 sentinel commit failure")
	prevCommit := commitStatsTransaction
	commitStatsTransaction = func(tx *gorm.DB) error {
		_ = tx.Rollback().Error
		return commitErr
	}
	t.Cleanup(func() { commitStatsTransaction = prevCommit })

	tracker := core.NewStatsTracker()
	seedSyntheticUserStatsForBench(t, tracker, 1)
	statsService := &StatsService{Runtime: NewRuntime(syntheticStatsCoreForBench(t, tracker))}

	if err := statsService.SaveStats(true); !errors.Is(err, commitErr) {
		t.Fatalf("SaveStats error=%v, want %v", err, commitErr)
	}

	var audit model.AuditEvent
	if err := database.GetDB().Where("event = ?", "stats_commit_failed").First(&audit).Error; err != nil {
		t.Fatal(err)
	}
	if audit.Actor != "system" || audit.Resource != "stats" || audit.Severity != AuditSeverityWarn {
		t.Fatalf("unexpected audit event: %#v", audit)
	}
	var details map[string]any
	if err := json.Unmarshal(audit.Details, &details); err != nil {
		t.Fatal(err)
	}
	if details["error"] != commitErr.Error() {
		t.Fatalf("unexpected audit details: %#v", details)
	}

	expectStatsCommitFailedWarningIssue26(t, ch)
	expectNoStatsRealtimeEventsIssue26(t, ch)

	var statsRows int64
	if err := database.GetDB().Model(model.Stats{}).Count(&statsRows).Error; err != nil {
		t.Fatal(err)
	}
	if statsRows != 0 {
		t.Fatalf("stats rows committed after failed commit: %d", statsRows)
	}
}

func TestStatsServiceDownsampleStatsBucketsExtra(t *testing.T) {
	statsService := &StatsService{}
	input := []model.Stats{
		{DateTime: 100, Resource: "user", Tag: "alice", Direction: false, Traffic: 10},
		{DateTime: 101, Resource: "user", Tag: "alice", Direction: false, Traffic: 30},
		{DateTime: 102, Resource: "user", Tag: "alice", Direction: true, Traffic: 40},
		{DateTime: 110, Resource: "user", Tag: "alice", Direction: false, Traffic: 90},
		{DateTime: 111, Resource: "user", Tag: "alice", Direction: true, Traffic: 100},
		{DateTime: 112, Resource: "user", Tag: "alice", Direction: true, Traffic: 140},
	}

	got := statsService.downsampleStats(input, 4)
	if len(got) != 4 {
		t.Fatalf("expected 4 downsampled rows, got %d: %#v", len(got), got)
	}
	if got[0].Direction || got[0].Traffic != 20 {
		t.Fatalf("unexpected first bucket down row: %#v", got[0])
	}
	if !got[1].Direction || got[1].Traffic != 40 {
		t.Fatalf("unexpected first bucket up row: %#v", got[1])
	}
	if got[2].Direction || got[2].Traffic != 90 {
		t.Fatalf("unexpected second bucket down row: %#v", got[2])
	}
	if !got[3].Direction || got[3].Traffic != 120 {
		t.Fatalf("unexpected second bucket up row: %#v", got[3])
	}
	if got[0].Resource != "user" || got[0].Tag != "alice" {
		t.Fatalf("resource/tag not preserved: %#v", got[0])
	}
	if got[0].DateTime > got[2].DateTime {
		t.Fatalf("bucket order regressed: %#v", got)
	}
}

func expectStatsCommitFailedWarningIssue26(t *testing.T, ch <-chan realtime.Event) {
	t.Helper()
	select {
	case event := <-ch:
		if event.Type != realtime.TopicCoreState {
			t.Fatalf("expected %s, got %s", realtime.TopicCoreState, event.Type)
		}
		payload, ok := event.Payload.(map[string]any)
		if !ok || payload["warning"] != "stats_commit_failed" {
			t.Fatalf("unexpected warning payload: %#v", event.Payload)
		}
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for %s warning", realtime.TopicCoreState)
	}
}

func expectNoStatsRealtimeEventsIssue26(t *testing.T, ch <-chan realtime.Event) {
	t.Helper()
	select {
	case event := <-ch:
		if event.Type == realtime.TopicOnlines || event.Type == realtime.TopicTrafficDelta {
			t.Fatalf("unexpected normal stats realtime event after failed commit: %#v", event)
		}
		t.Fatalf("unexpected realtime event after failed commit: %#v", event)
	case <-time.After(50 * time.Millisecond):
	}
}
