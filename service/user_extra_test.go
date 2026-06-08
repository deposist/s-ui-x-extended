package service

import (
	"strconv"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
	"github.com/deposist/s-ui-x/util/common"
)

func TestUserServiceLoginHappyWrongAndLastLogin(t *testing.T) {
	initSettingTestDB(t)
	userService := &UserService{}
	if err := userService.UpdateFirstUser("admin", "correct-password"); err != nil {
		t.Fatal(err)
	}

	username, err := userService.Login("admin", "correct-password", "203.0.113.10")
	if err != nil {
		t.Fatal(err)
	}
	if username != "admin" {
		t.Fatalf("unexpected login username %q", username)
	}
	if _, err := userService.Login("admin", "wrong-password", "203.0.113.11"); err == nil {
		t.Fatal("wrong password should be rejected")
	}

	var user model.User
	if err := database.GetDB().Where("username = ?", "admin").First(&user).Error; err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(user.LastLogins, "203.0.113.10") {
		t.Fatalf("last login IP was not recorded: %q", user.LastLogins)
	}
}

func TestUserServiceLoginLockedDocumentedAtAPILayer(t *testing.T) {
	t.Skip("Login lockout is enforced by api checkLoginRateLimit, not UserService.Login; keep service unit boundary unchanged")
}

func TestUserServiceAddTokenScopePersistenceAndInvalidScope(t *testing.T) {
	initSettingTestDB(t)
	userService := &UserService{}

	for _, scope := range []string{"read", "write", "database", "telegram", "observability"} {
		if _, err := userService.AddToken("admin", 0, "scope "+scope, scope); err != nil {
			t.Fatalf("scope %q should be accepted: %v", scope, err)
		}
	}
	if _, err := userService.AddToken("admin", 0, "bad", "admin:all"); err == nil {
		t.Fatal("invalid scope should be rejected")
	}

	var stored []model.Tokens
	if err := database.GetDB().Order("id asc").Find(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if len(stored) != 5 {
		t.Fatalf("expected five stored tokens, got %d", len(stored))
	}
	for _, token := range stored {
		if token.TokenHash == "" || token.TokenPrefix == "" || !token.Enabled {
			t.Fatalf("stored token missing secure fields: %#v", token)
		}
		if !apiTokenScopeAllowed(token.Scope) {
			t.Fatalf("stored invalid scope: %#v", token)
		}
	}
}

func TestUserServiceHashAPITokenDeterministicWithStableInstallSalt(t *testing.T) {
	initSettingTestDB(t)
	settingService := &SettingService{}
	if _, err := settingService.GetInstallSalt(); err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Model(model.Setting{}).Where("key = ?", "installSalt").Update("value", "phase2-stable-salt").Error; err != nil {
		t.Fatal(err)
	}

	userService := &UserService{}
	first, err := userService.HashAPIToken("plain-token")
	if err != nil {
		t.Fatal(err)
	}
	second, err := userService.HashAPIToken("plain-token")
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("hash changed with stable installSalt: %q != %q", first, second)
	}

	if err := database.GetDB().Model(model.Setting{}).Where("key = ?", "installSalt").Update("value", "phase2-other-salt").Error; err != nil {
		t.Fatal(err)
	}
	third, err := userService.HashAPIToken("plain-token")
	if err != nil {
		t.Fatal(err)
	}
	if third == first {
		t.Fatal("hash should change when installSalt changes")
	}
}

func TestUserServiceMigrateLegacyTokensKeepsDisabledIssue27(t *testing.T) {
	initSettingTestDB(t)
	userService := &UserService{}
	legacy := model.Tokens{
		Desc:   "disabled legacy",
		Token:  "legacy-disabled-token",
		UserId: 1,
	}
	if err := database.GetDB().Create(&legacy).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Model(&model.Tokens{}).Where("id = ?", legacy.Id).Update("enabled", false).Error; err != nil {
		t.Fatal(err)
	}
	var before model.Tokens
	if err := database.GetDB().Where("id = ?", legacy.Id).First(&before).Error; err != nil {
		t.Fatal(err)
	}
	if before.Enabled {
		t.Fatalf("disabled fixture was not disabled before migration: %#v", before)
	}
	if err := userService.migrateLegacyTokens(); err != nil {
		t.Fatal(err)
	}
	var stored model.Tokens
	if err := database.GetDB().Where("id = ?", legacy.Id).First(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Enabled {
		t.Fatalf("disabled legacy token was re-enabled: %#v", stored)
	}
	if stored.Token != "" {
		t.Fatalf("legacy plaintext token was not cleared: %q", stored.Token)
	}
	if stored.TokenHash == "" || stored.TokenPrefix == "" {
		t.Fatalf("legacy token hash/prefix not populated: %#v", stored)
	}
	if stored.Scope != defaultAPITokenScope {
		t.Fatalf("legacy token scope not normalized: %#v", stored)
	}
}

func TestIssue9UpdateFirstUserClearsForcePasswordReset(t *testing.T) {
	initSettingTestDB(t)
	userService := &UserService{}
	var admin model.User
	if err := database.GetDB().Where("username = ?", "admin").First(&admin).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Model(&model.User{}).Where("id = ?", admin.Id).Update("force_password_reset", true).Error; err != nil {
		t.Fatal(err)
	}

	if err := userService.UpdateFirstUser("admin", "updated-password"); err != nil {
		t.Fatal(err)
	}

	var stored model.User
	if err := database.GetDB().Where("username = ?", "admin").First(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if stored.ForcePasswordReset {
		t.Fatalf("UpdateFirstUser should clear force reset: %#v", stored)
	}
	if ok, _ := common.CheckPassword(stored.Password, "updated-password"); !ok {
		t.Fatal("updated password does not validate")
	}
}

func TestIssue9ChangePassClearsForcePasswordReset(t *testing.T) {
	initSettingTestDB(t)
	userService := &UserService{}
	if err := userService.UpdateFirstUser("admin", "old-password"); err != nil {
		t.Fatal(err)
	}
	var admin model.User
	if err := database.GetDB().Where("username = ?", "admin").First(&admin).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Model(&model.User{}).Where("id = ?", admin.Id).Update("force_password_reset", true).Error; err != nil {
		t.Fatal(err)
	}

	if err := userService.ChangePass("admin", "old-password", "admin-renamed", "new-password"); err != nil {
		t.Fatal(err)
	}

	var stored model.User
	if err := database.GetDB().Where("id = ?", admin.Id).First(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Username != "admin-renamed" {
		t.Fatalf("username was not changed: %#v", stored)
	}
	if stored.ForcePasswordReset {
		t.Fatalf("ChangePass should clear force reset: %#v", stored)
	}
	if ok, _ := common.CheckPassword(stored.Password, "new-password"); !ok {
		t.Fatal("new password does not validate")
	}
}

func TestIssue9LoginPasswordHashMigrationClearsForcePasswordReset(t *testing.T) {
	initSettingTestDB(t)
	userService := &UserService{}
	prefixedHash, err := common.HashPassword("legacy-password")
	if err != nil {
		t.Fatal(err)
	}
	rawBcryptHash := strings.TrimPrefix(prefixedHash, "bcrypt:")
	if rawBcryptHash == prefixedHash {
		t.Fatal("test hash did not use bcrypt prefix")
	}
	var admin model.User
	if err := database.GetDB().Where("username = ?", "admin").First(&admin).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Model(&model.User{}).Where("id = ?", admin.Id).Updates(map[string]any{
		"password":             rawBcryptHash,
		"force_password_reset": true,
	}).Error; err != nil {
		t.Fatal(err)
	}

	if _, err := userService.Login("admin", "legacy-password", "203.0.113.20"); err != nil {
		t.Fatal(err)
	}

	var stored model.User
	if err := database.GetDB().Where("id = ?", admin.Id).First(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if stored.ForcePasswordReset {
		t.Fatalf("password hash migration should clear force reset: %#v", stored)
	}
	if !strings.HasPrefix(stored.Password, "bcrypt:") {
		t.Fatalf("password was not migrated to canonical hash: %q", stored.Password)
	}
}

func TestUserServiceAddUserRequiresCurrentPasswordAndStoresHash(t *testing.T) {
	initSettingTestDB(t)
	userService := &UserService{}
	if err := userService.UpdateFirstUser("admin", "current-password"); err != nil {
		t.Fatal(err)
	}

	created, err := userService.AddUser("admin", "current-password", " new-admin ", "new-password")
	if err != nil {
		t.Fatal(err)
	}
	if created.Username != "new-admin" {
		t.Fatalf("username was not normalized: %#v", created)
	}
	if created.Password == "new-password" || !common.IsPasswordHash(created.Password) {
		t.Fatalf("password must be stored as hash only: %q", created.Password)
	}
	if ok, _ := common.CheckPassword(created.Password, "new-password"); !ok {
		t.Fatal("stored password hash does not validate")
	}
	if created.ForcePasswordReset {
		t.Fatalf("new admin should not require reset: %#v", created)
	}
	if _, err := userService.AddUser("admin", "wrong-password", "denied-admin", "new-password"); err == nil {
		t.Fatal("wrong current password should be rejected")
	}
	exists, err := userService.UserExists("denied-admin")
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Fatal("denied admin should not be created")
	}
	if _, err := userService.AddUser("admin", "current-password", "new-admin", "another-password"); err == nil {
		t.Fatal("duplicate username should be rejected")
	}
}

func TestUserServiceDeleteUserRejectsSelfAndDeletesTokens(t *testing.T) {
	initSettingTestDB(t)
	userService := &UserService{}
	if err := userService.UpdateFirstUser("admin", "current-password"); err != nil {
		t.Fatal(err)
	}
	target, err := userService.AddUser("admin", "current-password", "delete-me", "target-password")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := userService.AddToken("delete-me", 0, "target token", "admin"); err != nil {
		t.Fatal(err)
	}

	var admin model.User
	if err := database.GetDB().Where("username = ?", "admin").First(&admin).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := userService.DeleteUser("admin", "current-password", strconv.FormatUint(uint64(admin.Id), 10)); err == nil {
		t.Fatal("current admin should not be deleted")
	}
	if _, err := userService.DeleteUser("admin", "wrong-password", strconv.FormatUint(uint64(target.Id), 10)); err == nil {
		t.Fatal("wrong current password should be rejected")
	}
	exists, err := userService.UserExists("delete-me")
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatal("wrong-password delete should not remove target")
	}

	result, err := userService.DeleteUser("admin", "current-password", strconv.FormatUint(uint64(target.Id), 10))
	if err != nil {
		t.Fatal(err)
	}
	if result.User.Username != "delete-me" || result.DeletedTokenCount != 1 {
		t.Fatalf("unexpected delete result: %#v", result)
	}
	exists, err = userService.UserExists("delete-me")
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Fatal("deleted admin should not exist")
	}
	var tokenCount int64
	if err := database.GetDB().Model(model.Tokens{}).Where("user_id = ?", target.Id).Count(&tokenCount).Error; err != nil {
		t.Fatal(err)
	}
	if tokenCount != 0 {
		t.Fatalf("deleted admin tokens should be removed, got %d", tokenCount)
	}
}
