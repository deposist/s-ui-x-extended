package service

import (
	"strconv"
	"testing"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/deposist/s-ui-x-extended/util/common"
)

func createTokenOwnershipUser(t *testing.T, username string) model.User {
	t.Helper()
	hash, err := common.HashPassword("password")
	if err != nil {
		t.Fatal(err)
	}
	user := model.User{Username: username, Password: hash}
	if err := database.GetDB().Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	return user
}

func createTokenOwnershipToken(t *testing.T, userID uint, desc string, enabled bool) model.Tokens {
	t.Helper()
	token := model.Tokens{Desc: desc, TokenHash: desc + "-hash", TokenPrefix: desc, Scope: "admin", UserId: userID}
	if err := database.GetDB().Create(&token).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Model(&model.Tokens{}).Where("id = ?", token.Id).Update("enabled", enabled).Error; err != nil {
		t.Fatal(err)
	}
	token.Enabled = enabled
	return token
}

func TestDeleteTokenRequiresOwner(t *testing.T) {
	initSettingTestDB(t)
	userService := &UserService{}
	other := createTokenOwnershipUser(t, "other-admin")
	adminToken := createTokenOwnershipToken(t, 1, "admin-token", true)
	otherToken := createTokenOwnershipToken(t, other.Id, "other-token", true)

	if err := userService.DeleteToken("admin", strconv.Itoa(int(otherToken.Id))); err == nil {
		t.Fatal("deleting another admin's token should fail")
	}
	var count int64
	if err := database.GetDB().Model(&model.Tokens{}).Where("id = ?", otherToken.Id).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatal("admin deleted another admin's token")
	}

	if err := userService.DeleteToken("admin", strconv.Itoa(int(adminToken.Id))); err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Model(&model.Tokens{}).Where("id = ?", adminToken.Id).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("admin token was not deleted")
	}
}

func TestSetTokenEnabledRequiresOwner(t *testing.T) {
	initSettingTestDB(t)
	userService := &UserService{}
	other := createTokenOwnershipUser(t, "other-admin")
	adminToken := createTokenOwnershipToken(t, 1, "admin-token", false)
	otherToken := createTokenOwnershipToken(t, other.Id, "other-token", false)

	if err := userService.SetTokenEnabled("admin", strconv.Itoa(int(otherToken.Id)), true); err == nil {
		t.Fatal("enabling another admin's token should fail")
	}
	var stored model.Tokens
	if err := database.GetDB().Where("id = ?", otherToken.Id).First(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Enabled {
		t.Fatal("admin enabled another admin's token")
	}

	if err := userService.SetTokenEnabled("admin", strconv.Itoa(int(adminToken.Id)), true); err != nil {
		t.Fatal(err)
	}
	stored = model.Tokens{}
	if err := database.GetDB().Where("id = ?", adminToken.Id).First(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if !stored.Enabled {
		t.Fatal("admin token was not enabled")
	}
}
