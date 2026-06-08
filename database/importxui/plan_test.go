package importxui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
)

func TestPlan_ProducesPreviewJSON(t *testing.T) {
	src, _ := setupImportTestDB(t)
	plan, err := Plan(src, PlanOptions{Strategy: StrategyMerge, IncludeSettings: true, AdminMode: AdminModeNewPassword})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Items) == 0 {
		t.Fatal("plan has no items")
	}
	for _, item := range plan.Items {
		if item.PreviewJSON == nil {
			t.Fatalf("item has no preview: %#v", item)
		}
		var parsed any
		if err := json.Unmarshal(item.PreviewJSON, &parsed); err != nil {
			t.Fatalf("preview for %s/%v is invalid JSON: %v", item.Kind, item.SrcID, err)
		}
	}
	if plan.Source.Hash == "" {
		t.Fatal("plan source hash is empty")
	}
}

func TestPlan_ConflictDetection(t *testing.T) {
	src, _ := setupImportTestDB(t)
	placeholderInbound(t, "inbound-12223")
	plan, err := Plan(src, PlanOptions{Strategy: StrategyMerge})
	if err != nil {
		t.Fatal(err)
	}
	item := findPlanItem(t, plan, KindInbound, float64(8))
	if !item.Conflict || item.Action != ActionMerge {
		t.Fatalf("expected merge conflict for inbound-12223, got %#v", item)
	}
}

func TestApply_HashMismatch(t *testing.T) {
	src, _ := setupImportTestDB(t)
	plan, err := Plan(src, PlanOptions{Strategy: StrategyMerge})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(src, append(readFixtureFile(t, "x-ui.db"), []byte("changed")...), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(src, *plan, ApplyOptions{}); err == nil {
		t.Fatal("Apply should reject stale plan")
	} else if !contains(err.Error(), "plan_stale") {
		t.Fatalf("expected plan_stale error, got %v", err)
	}
}

func TestApply_RespectsPerObjectAction(t *testing.T) {
	src, _ := setupImportTestDB(t)
	plan, err := Plan(src, PlanOptions{Strategy: StrategyMerge})
	if err != nil {
		t.Fatal(err)
	}
	for i := range plan.Items {
		if plan.Items[i].Kind == KindInbound && plan.Items[i].SrcTag == "inbound-12223" {
			plan.Items[i].Action = ActionSkip
		}
	}
	if _, err := Apply(src, *plan, ApplyOptions{}); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := database.GetDB().Model(model.Inbound{}).Where("tag = ?", "inbound-12223").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("skipped inbound was imported")
	}
}

func TestApply_RespectsRenamedTag(t *testing.T) {
	src, _ := setupImportTestDB(t)
	plan, err := Plan(src, PlanOptions{Strategy: StrategyMerge})
	if err != nil {
		t.Fatal(err)
	}
	for i := range plan.Items {
		if plan.Items[i].Kind == KindInbound && plan.Items[i].SrcTag == "inbound-12223" {
			plan.Items[i].DstTag = "renamed-trojan"
		}
	}
	if _, err := Apply(src, *plan, ApplyOptions{}); err != nil {
		t.Fatal(err)
	}
	if inboundByTag(t, "renamed-trojan").Type != "trojan" {
		t.Fatal("renamed inbound was not imported")
	}
}

func TestApply_EmitsProgressAndRecordsRollbackPath(t *testing.T) {
	src, dbPath := setupImportTestDB(t)
	plan, err := Plan(src, PlanOptions{Strategy: StrategyMerge})
	if err != nil {
		t.Fatal(err)
	}
	var events []Progress
	report, err := Apply(src, *plan, ApplyOptions{
		OnProgress: func(progress Progress) {
			events = append(events, progress)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) == 0 {
		t.Fatal("Apply did not emit progress")
	}
	if report.BackupPath == "" {
		t.Fatal("backup path was not recorded")
	}
	if _, err := os.Stat(report.BackupPath); err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(report.BackupPath) != filepath.Dir(dbPath) {
		t.Fatalf("backup path should be next to s-ui db: %s", report.BackupPath)
	}
}

func TestApply_ImportsSettingsAndNewPasswordAdmins(t *testing.T) {
	src, _ := setupImportTestDB(t)
	plan, err := Plan(src, PlanOptions{
		Strategy:        StrategyMerge,
		IncludeSettings: true,
		AdminMode:       AdminModeNewPassword,
	})
	if err != nil {
		t.Fatal(err)
	}
	var settingItem *PlanItem
	var adminItem *PlanItem
	for i := range plan.Items {
		switch plan.Items[i].Kind {
		case KindSetting:
			if plan.Items[i].Action != ActionSkip && settingItem == nil {
				settingItem = &plan.Items[i]
			}
		case KindAdmin:
			if plan.Items[i].Action != ActionSkip && adminItem == nil {
				adminItem = &plan.Items[i]
			}
		}
	}
	if settingItem == nil {
		t.Fatal("plan has no runnable setting item")
	}
	if adminItem == nil {
		t.Fatal("plan has no runnable admin item")
	}
	report, err := Apply(src, *plan, ApplyOptions{})
	if err != nil {
		t.Fatal(err)
	}
	var setting model.Setting
	if err := database.GetDB().Where("key = ?", settingItem.DstTag).First(&setting).Error; err != nil {
		t.Fatalf("setting %s was not imported: %v", settingItem.DstTag, err)
	}
	if len(report.GeneratedAdmins) == 0 || report.GeneratedAdmins[0].Password == "" {
		t.Fatalf("new-password admin was not returned once in report: %#v", report.GeneratedAdmins)
	}
	var admin model.User
	if err := database.GetDB().Where("username = ?", report.GeneratedAdmins[0].Username).First(&admin).Error; err != nil {
		t.Fatal(err)
	}
	if admin.ForcePasswordReset {
		t.Fatalf("new-password admin should not require reset: %#v", admin)
	}
}

func findPlanItem(t *testing.T, plan *MigrationPlan, kind string, srcID any) PlanItem {
	t.Helper()
	key := planKey(kind, srcID)
	for _, item := range plan.Items {
		if planKey(item.Kind, item.SrcID) == key {
			return item
		}
	}
	t.Fatalf("missing plan item %s", key)
	return PlanItem{}
}

func readFixtureFile(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(fixturePath(t, name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func contains(haystack string, needle string) bool {
	return strings.Contains(haystack, needle)
}
