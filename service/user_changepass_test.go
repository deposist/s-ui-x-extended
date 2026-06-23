package service

import (
	"testing"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/deposist/s-ui-x-extended/util/common"
)

// TestUserServiceChangePassValidatesUsername pins the L1 regression: ChangePass
// must reject an empty/whitespace new username (self-lockout) and a name already
// taken by a different admin (ambiguous login), mirroring AddUser's validation.
func TestUserServiceChangePassValidatesUsername(t *testing.T) {
	initSettingTestDB(t)
	us := &UserService{}
	if err := us.UpdateFirstUser("admin", "correct-password"); err != nil {
		t.Fatal(err)
	}
	if _, err := us.AddUser("admin", "correct-password", "bob", "bob-password"); err != nil {
		t.Fatal(err)
	}

	if err := us.ChangePass("admin", "correct-password", "", "new-password"); err == nil {
		t.Fatal("empty new username must be rejected")
	}
	if err := us.ChangePass("admin", "correct-password", "   ", "new-password"); err == nil {
		t.Fatal("whitespace-only new username must be rejected")
	}
	if err := us.ChangePass("admin", "correct-password", "bob", "new-password"); err == nil {
		t.Fatal("renaming to an existing different user's name must be rejected")
	}
	if err := us.ChangePass("admin", "wrong-password", "admin2", "new-password"); err == nil {
		t.Fatal("wrong old password must be rejected")
	}

	if err := us.ChangePass("admin", "correct-password", "admin2", "new-password"); err != nil {
		t.Fatalf("valid change should succeed: %v", err)
	}
	var user model.User
	if err := database.GetDB().Where("username = ?", "admin2").First(&user).Error; err != nil {
		t.Fatalf("renamed user not found: %v", err)
	}
	if ok, _ := common.CheckPassword(user.Password, "new-password"); !ok {
		t.Fatal("new password was not applied after rename")
	}
}
