package importxui

import (
	"errors"
	"strings"
	"testing"
)

func TestImport_CheckpointFailureIsReportedAsWarningAfterCommit(t *testing.T) {
	src, _ := setupImportTestDB(t)
	old := walCheckpoint
	walCheckpoint = func() error { return errors.New("checkpoint unavailable") }
	t.Cleanup(func() { walCheckpoint = old })

	report, err := Import(src, Options{Strategy: StrategyMerge})
	if err != nil {
		t.Fatalf("committed import must not fail because checkpoint failed: %v", err)
	}
	if !reportContainsWarning(report.Warnings, "wal checkpoint") {
		t.Fatalf("checkpoint failure was not reported: %v", report.Warnings)
	}
}

func TestApply_CheckpointFailureIsReportedAsWarningAfterCommit(t *testing.T) {
	src, _ := setupImportTestDB(t)
	plan, err := Plan(src, PlanOptions{Strategy: StrategyMerge})
	if err != nil {
		t.Fatal(err)
	}
	old := walCheckpoint
	walCheckpoint = func() error { return errors.New("checkpoint unavailable") }
	t.Cleanup(func() { walCheckpoint = old })

	report, err := Apply(src, *plan, ApplyOptions{})
	if err != nil {
		t.Fatalf("committed plan apply must not fail because checkpoint failed: %v", err)
	}
	if !reportContainsWarning(report.Warnings, "wal checkpoint") {
		t.Fatalf("checkpoint failure was not reported: %v", report.Warnings)
	}
}

func reportContainsWarning(warnings []string, needle string) bool {
	for _, warning := range warnings {
		if strings.Contains(warning, needle) {
			return true
		}
	}
	return false
}
