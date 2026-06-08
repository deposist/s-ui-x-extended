package importxui

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
	"github.com/deposist/s-ui-x/util/common"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type clusterEXUIUser struct {
	id       int64
	username string
	password string
}

func TestIssue8ResetRequiredPlanItemsCarryAdminMode(t *testing.T) {
	src := setupClusterEImportDB(t, []clusterEXUIUser{{
		id:       1,
		username: "xui-admin",
		password: "source-secret",
	}})

	plan, err := Plan(src, PlanOptions{Strategy: StrategyMerge, AdminMode: AdminModeResetRequired})
	if err != nil {
		t.Fatal(err)
	}
	item := findPlanItem(t, plan, KindAdmin, int64(1))
	if item.AdminMode != string(AdminModeResetRequired) {
		t.Fatalf("admin item AdminMode=%q, want %q", item.AdminMode, AdminModeResetRequired)
	}
	var preview map[string]any
	if err := json.Unmarshal(item.PreviewJSON, &preview); err != nil {
		t.Fatal(err)
	}
	if preview["mode"] != string(AdminModeResetRequired) {
		t.Fatalf("preview mode=%#v, want %q", preview["mode"], AdminModeResetRequired)
	}
}

func TestIssue2ResetRequiredCreatesForceResetWithoutGeneratedPassword(t *testing.T) {
	src := setupClusterEImportDB(t, []clusterEXUIUser{{
		id:       1,
		username: "xui-admin",
		password: "source-secret",
	}})
	plan, err := Plan(src, PlanOptions{Strategy: StrategyMerge, AdminMode: AdminModeResetRequired})
	if err != nil {
		t.Fatal(err)
	}

	report, err := Apply(src, *plan, ApplyOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.GeneratedAdmins) != 0 {
		t.Fatalf("reset_required must not return generated admins: %#v", report.GeneratedAdmins)
	}
	stored := clusterEUserByName(t, "xui-admin")
	if !stored.ForcePasswordReset {
		t.Fatalf("reset_required user should require reset: %#v", stored)
	}
	if ok, _ := common.CheckPassword(stored.Password, "source-secret"); !ok {
		t.Fatal("reset_required create should store the source password")
	}
}

func TestIssue2ResetRequiredMergeAndReplaceKeepUserRowsStable(t *testing.T) {
	for _, tc := range []struct {
		name           string
		strategy       Strategy
		wantLocalPass  bool
		wantSourcePass bool
	}{
		{name: "merge preserves local password", strategy: StrategyMerge, wantLocalPass: true},
		{name: "replace uses source password", strategy: StrategyReplace, wantSourcePass: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src := setupClusterEImportDB(t, []clusterEXUIUser{{
				id:       1,
				username: "admin",
				password: "source-secret",
			}})
			localHash, err := common.HashPassword("local-secret")
			if err != nil {
				t.Fatal(err)
			}
			before := clusterEUserByName(t, "admin")
			if err := database.GetDB().Model(&model.User{}).Where("id = ?", before.Id).Updates(map[string]any{
				"password":             localHash,
				"force_password_reset": false,
			}).Error; err != nil {
				t.Fatal(err)
			}
			plan, err := Plan(src, PlanOptions{Strategy: tc.strategy, AdminMode: AdminModeResetRequired})
			if err != nil {
				t.Fatal(err)
			}

			report, err := Apply(src, *plan, ApplyOptions{})
			if err != nil {
				t.Fatal(err)
			}
			if len(report.GeneratedAdmins) != 0 {
				t.Fatalf("reset_required must not return generated admins: %#v", report.GeneratedAdmins)
			}
			after := clusterEUserByName(t, "admin")
			if after.Id != before.Id {
				t.Fatalf("reset_required should update the existing user row, id %d -> %d", before.Id, after.Id)
			}
			if !after.ForcePasswordReset {
				t.Fatalf("reset_required should set force reset: %#v", after)
			}
			if ok, _ := common.CheckPassword(after.Password, "local-secret"); ok != tc.wantLocalPass {
				t.Fatalf("local password validity=%v, want %v", ok, tc.wantLocalPass)
			}
			if ok, _ := common.CheckPassword(after.Password, "source-secret"); ok != tc.wantSourcePass {
				t.Fatalf("source password validity=%v, want %v", ok, tc.wantSourcePass)
			}
		})
	}
}

func TestIssue2NewPasswordClearsForceResetAndReturnsGeneratedAdmin(t *testing.T) {
	src := setupClusterEImportDB(t, []clusterEXUIUser{{
		id:       1,
		username: "admin",
		password: "source-secret",
	}})
	before := clusterEUserByName(t, "admin")
	if err := database.GetDB().Model(&model.User{}).Where("id = ?", before.Id).Update("force_password_reset", true).Error; err != nil {
		t.Fatal(err)
	}
	plan, err := Plan(src, PlanOptions{Strategy: StrategyMerge, AdminMode: AdminModeNewPassword})
	if err != nil {
		t.Fatal(err)
	}

	report, err := Apply(src, *plan, ApplyOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.GeneratedAdmins) != 1 || report.GeneratedAdmins[0].Password == "" {
		t.Fatalf("new_password should return exactly one generated admin: %#v", report.GeneratedAdmins)
	}
	after := clusterEUserByName(t, "admin")
	if after.Id != before.Id {
		t.Fatalf("new_password should update the existing user row, id %d -> %d", before.Id, after.Id)
	}
	if after.ForcePasswordReset {
		t.Fatalf("new_password should clear force reset: %#v", after)
	}
	if ok, _ := common.CheckPassword(after.Password, report.GeneratedAdmins[0].Password); !ok {
		t.Fatal("stored password does not match generated password")
	}
}

func setupClusterEImportDB(t *testing.T, users []clusterEXUIUser) string {
	t.Helper()
	closeMainDBForImportTest(t)
	dir := makeImportXUITempDir(t)
	t.Setenv("SUI_DB_FOLDER", dir)
	dst := filepath.Join(dir, "s-ui.db")
	if err := database.InitDB(dst); err != nil {
		if strings.Contains(err.Error(), "go-sqlite3 requires cgo") {
			t.Skip(err)
		}
		t.Fatal(err)
	}
	t.Cleanup(func() {
		closeMainDBForImportTest(t)
	})
	src := filepath.Join(dir, "x-ui.db")
	createClusterEXUISource(t, src, users)
	return src
}

func createClusterEXUISource(t *testing.T, path string, users []clusterEXUIUser) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		if strings.Contains(err.Error(), "go-sqlite3 requires cgo") {
			t.Skip(err)
		}
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err == nil {
		defer sqlDB.Close()
	}
	if err := db.Exec(`CREATE TABLE inbounds (
		id integer primary key,
		user_id integer,
		up integer,
		down integer,
		total integer,
		all_time integer,
		remark text,
		enable integer,
		expiry_time integer,
		traffic_reset text,
		last_traffic_reset_time integer,
		listen text,
		port integer,
		protocol text,
		settings text,
		stream_settings text,
		tag text,
		sniffing text
	)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE client_traffics (
		id integer primary key,
		inbound_id integer,
		enable integer,
		email text,
		up integer,
		down integer,
		all_time integer,
		expiry_time integer,
		total integer,
		reset integer,
		last_online integer
	)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE users (
		id integer primary key,
		username text,
		password text
	)`).Error; err != nil {
		t.Fatal(err)
	}
	for _, user := range users {
		if err := db.Exec("INSERT INTO users(id, username, password) VALUES(?, ?, ?)", user.id, user.username, user.password).Error; err != nil {
			t.Fatal(err)
		}
	}
}

func clusterEUserByName(t *testing.T, username string) model.User {
	t.Helper()
	var user model.User
	if err := database.GetDB().Where("username = ?", username).First(&user).Error; err != nil {
		t.Fatal(err)
	}
	return user
}
