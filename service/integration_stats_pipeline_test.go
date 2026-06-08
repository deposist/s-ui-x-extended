package service

import (
	"testing"

	"github.com/deposist/s-ui-x/ipmonitor"
)

func TestIntegrationStatsPipelineNilCoreSmoke(t *testing.T) {
	initSettingTestDB(t)
	runtime := NewRuntimeWithCoreProvider(nil)
	restore := ReplaceDefaultRuntimeForTest(runtime)
	t.Cleanup(restore)
	ipmonitor.ResetCaches()

	if err := (&StatsService{Runtime: runtime}).SaveStats(true); err != nil {
		t.Fatalf("SaveStats with nil core should be a no-op, got %v", err)
	}
}

func TestIntegrationStatsPipelineRealtimeWithTestCore_XFAILPhase3(t *testing.T) {
	t.Skip("XFAIL Phase3: требуется test-core или hook для подмены core.Core/Box.StatsTracker без запуска sing-box")
}
