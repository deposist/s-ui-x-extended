package importxui

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
)

func TestIntegrationImportXUIFullFixturePlanApply(t *testing.T) {
	if !integrationImportFixturesAvailable(t) {
		t.Skip("требует test-db/x-ui.db и test-db/s-ui.db; см. tests/baseline/env.md")
	}

	cases := []struct {
		strategy  Strategy
		adminMode AdminMode
	}{
		{strategy: StrategyMerge, adminMode: AdminModeSkip},
		{strategy: StrategyReplace, adminMode: AdminModeSkip},
		{strategy: StrategySkip, adminMode: AdminModeSkip},
		{strategy: StrategyMerge, adminMode: AdminModeNewPassword},
	}

	for _, tc := range cases {
		t.Run(fmt.Sprintf("%s_%s", tc.strategy, tc.adminMode), func(t *testing.T) {
			src, _ := setupImportTestDB(t)
			plan, err := Plan(src, PlanOptions{
				Strategy:        tc.strategy,
				IncludeSettings: true,
				AdminMode:       tc.adminMode,
				IncludeHistory:  true,
				IncludeRouting:  true,
			})
			if err != nil {
				t.Fatal(err)
			}
			report, err := Apply(src, *plan, ApplyOptions{})
			if err != nil {
				t.Fatal(err)
			}
			if report.BackupPath == "" {
				t.Fatal("Apply did not return backupPath")
			}
			if _, err := os.Stat(report.BackupPath); err != nil {
				t.Fatalf("backupPath file is not available: %v", err)
			}
			var audit model.AuditEvent
			if err := database.GetDB().Where("event = ?", "xui_import").Order("id desc").First(&audit).Error; err != nil {
				t.Fatal(err)
			}
			if audit.Resource != "database" || audit.Severity != "info" {
				t.Fatalf("unexpected xui_import audit: %#v", audit)
			}
		})
	}
}

func integrationImportFixturesAvailable(t *testing.T) bool {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"x-ui.db", "s-ui.db"} {
		if _, err := os.Stat(filepath.Join(wd, "..", "..", "test-db", name)); err != nil {
			return false
		}
	}
	return true
}
