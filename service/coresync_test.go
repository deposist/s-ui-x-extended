package service

import (
	"fmt"
	"testing"
)

// TestPostCommitCorePlanHasObjectChanges verifies the decision helper that
// distinguishes "nothing to apply" from "hot reload needed" and "full restart
// needed" without spinning up a core.
func TestPostCommitCorePlanHasObjectChanges(t *testing.T) {
	if (&postCommitCorePlan{}).hasObjectChanges() {
		t.Fatal("empty plan must report no object changes")
	}
	if (&postCommitCorePlan{needsCoreRestart: true}).hasObjectChanges() {
		t.Fatal("restart-only plan must not report object changes")
	}
	if !(&postCommitCorePlan{outboundIds: []uint{1}}).hasObjectChanges() {
		t.Fatal("plan with outbound IDs must report object changes")
	}
	if !(&postCommitCorePlan{removedOutboundTags: []string{"old"}}).hasObjectChanges() {
		t.Fatal("plan with removed tags must report object changes")
	}
	if !(&postCommitCorePlan{inboundIds: []uint{1}}).hasObjectChanges() {
		t.Fatal("plan with inbound IDs must report object changes")
	}
	if !(&postCommitCorePlan{endpointIds: []uint{1}}).hasObjectChanges() {
		t.Fatal("plan with endpoint IDs must report object changes")
	}
	if !(&postCommitCorePlan{serviceIds: []uint{1}}).hasObjectChanges() {
		t.Fatal("plan with service IDs must report object changes")
	}
}

// TestPostCommitCorePlanMergeRestart verifies that a restart-flagged entity
// change propagates needsCoreRestart and the reason.
func TestPostCommitCorePlanMergeRestart(t *testing.T) {
	var p postCommitCorePlan
	p.mergeOutboundChange(&entityCoreChange{
		needsRestart:  true,
		restartReason: "captured by selector",
	})
	if !p.needsCoreRestart {
		t.Fatal("mergeOutboundChange with needsRestart must set needsCoreRestart")
	}
	if p.restartReason != "captured by selector" {
		t.Fatalf("restartReason = %q, want %q", p.restartReason, "captured by selector")
	}
}

// TestPostCommitCorePlanMergeDoesNotOverwriteReason verifies that a later
// non-restart change does not clobber an earlier restart reason.
func TestPostCommitCorePlanMergeDoesNotOverwriteReason(t *testing.T) {
	var p postCommitCorePlan
	p.mergeOutboundChange(&entityCoreChange{needsRestart: true, restartReason: "first reason"})
	p.mergeInboundChange(&entityCoreChange{reloadIds: []uint{1}})
	if !p.needsCoreRestart {
		t.Fatal("needsCoreRestart must stay true after a non-restart merge")
	}
	if p.restartReason != "first reason" {
		t.Fatalf("restartReason = %q, want %q", p.restartReason, "first reason")
	}
	if len(p.inboundIds) != 1 {
		t.Fatalf("inboundIds len = %d, want 1", len(p.inboundIds))
	}
}

// TestApplyPostCommitCoreChangesNoChangesDoesNothing verifies that a plan with
// no object changes and no restart does not touch the running core.
func TestApplyPostCommitCoreChangesNoChangesDoesNothing(t *testing.T) {
	initSettingTestDB(t)
	coreInstance := startTestCore(t)

	before := coreInstance.GetInstance()
	configService := NewConfigServiceWithRuntime(NewRuntime(coreInstance))
	configService.applyPostCommitCoreChangesLocked(postCommitCorePlan{})

	if coreInstance.GetInstance() != before {
		t.Fatal("empty plan must not restart the core")
	}
}

// TestApplyPostCommitCoreChangesHotReloadOutbounds verifies that a plan with
// outbound IDs triggers a hot reload, not a full restart.
func TestApplyPostCommitCoreChangesHotReloadOutbounds(t *testing.T) {
	initSettingTestDB(t)
	coreInstance := startTestCore(t)
	outbound := createTestOutbound(t, "proxy-apply", 1080)

	recorder := &outboundOpsRecorder{}
	recorder.stubOutboundHooks(t)

	before := coreInstance.GetInstance()
	configService := NewConfigServiceWithRuntime(NewRuntime(coreInstance))
	configService.applyPostCommitCoreChangesLocked(postCommitCorePlan{
		outboundIds: []uint{outbound.Id},
	})

	if coreInstance.GetInstance() != before {
		t.Fatal("hot reload plan must not restart the core")
	}
	want := []string{fmt.Sprintf("reload-outbounds:[%d]", outbound.Id)}
	if len(recorder.ops) != 1 || recorder.ops[0] != want[0] {
		t.Fatalf("ops = %v, want %v", recorder.ops, want)
	}
}

// TestApplyPostCommitCoreChangesNeedsRestartTriggersRestart verifies that a
// plan with needsCoreRestart restarts the core even when object IDs are also
// present (restart takes priority over hot reload).
func TestApplyPostCommitCoreChangesNeedsRestartTriggersRestart(t *testing.T) {
	initSettingTestDB(t)
	coreInstance := startTestCore(t)
	outbound := createTestOutbound(t, "proxy-restart", 1080)

	recorder := &outboundOpsRecorder{}
	recorder.stubOutboundHooks(t)

	before := coreInstance.GetInstance()
	configService := NewConfigServiceWithRuntime(NewRuntime(coreInstance))
	configService.applyPostCommitCoreChangesLocked(postCommitCorePlan{
		needsCoreRestart: true,
		restartReason:    "test: captured by selector",
		outboundIds:      []uint{outbound.Id},
	})

	if len(recorder.ops) != 0 {
		t.Fatalf("restart plan must not hot-reload, got ops %v", recorder.ops)
	}
	after := coreInstance.GetInstance()
	if after == before {
		t.Fatal("restart plan must restart the core")
	}
	if after == nil {
		t.Fatal("core did not come back up after restart")
	}
}

// TestApplyPostCommitCoreChangesFailedHotReloadEscalatesToRestart verifies
// that when a partial hot reload fails, the core falls back to a full restart
// so it never keeps serving a partially updated state.
func TestApplyPostCommitCoreChangesFailedHotReloadEscalatesToRestart(t *testing.T) {
	initSettingTestDB(t)
	coreInstance := startTestCore(t)
	outbound := createTestOutbound(t, "proxy-fail", 1080)

	// Make the outbound hot-reload fail.
	prevRestart := restartOutboundsAfterSave
	restartOutboundsAfterSave = func(_ *ConfigService, _ []uint) error {
		return fmt.Errorf("simulated hot-reload failure")
	}
	t.Cleanup(func() { restartOutboundsAfterSave = prevRestart })

	before := coreInstance.GetInstance()
	configService := NewConfigServiceWithRuntime(NewRuntime(coreInstance))
	configService.applyPostCommitCoreChangesLocked(postCommitCorePlan{
		outboundIds: []uint{outbound.Id},
	})

	after := coreInstance.GetInstance()
	if after == before {
		t.Fatal("failed hot reload must escalate to a full restart")
	}
	if after == nil {
		t.Fatal("core did not come back up after escalation restart")
	}
}
