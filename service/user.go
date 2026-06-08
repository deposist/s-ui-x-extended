package service

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
	"github.com/deposist/s-ui-x/logger"
	"github.com/deposist/s-ui-x/util/common"

	"gorm.io/gorm"
)

type UserService struct {
	Runtime *Runtime
}

type DeleteUserResult struct {
	User              model.User
	DeletedTokenCount int64
}

func (s *UserService) runtime() *Runtime {
	if s != nil {
		return runtimeOrDefault(s.Runtime)
	}
	return DefaultRuntime()
}

const (
	defaultAPITokenScope = "admin"
	maxAPITokenScopeLen  = len("observability")
)

var allowedAPITokenScopes = []string{
	"admin",
	"read",
	"write",
	"database",
	"telegram",
	"observability",
}

func (s *UserService) GetFirstUser() (*model.User, error) {
	db := database.GetDB()

	user := &model.User{}
	err := db.Model(model.User{}).
		First(user).
		Error
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *UserService) UpdateFirstUser(username string, password string) error {
	if username == "" {
		return common.NewError("username can not be empty")
	} else if password == "" {
		return common.NewError("password can not be empty")
	}
	db := database.GetDB()
	passwordHash, err := common.HashPassword(password)
	if err != nil {
		return err
	}
	user := &model.User{}
	err = db.Model(model.User{}).First(user).Error
	if database.IsNotFound(err) {
		user.Username = username
		user.Password = passwordHash
		user.ForcePasswordReset = false
		return db.Model(model.User{}).Create(user).Error
	} else if err != nil {
		return err
	}
	user.Username = username
	user.Password = passwordHash
	user.ForcePasswordReset = false
	return db.Save(user).Error
}

func (s *UserService) Login(username string, password string, remoteIP string) (string, error) {
	user, needsMigration := s.CheckUser(username, password, remoteIP)
	if user == nil {
		return "", common.NewError("wrong user or password! IP: ", remoteIP)
	}
	if needsMigration {
		if err := s.updatePasswordHash(user, password); err != nil {
			logger.Warning("password migration failed:", err)
		}
	}
	return user.Username, nil
}

func (s *UserService) CheckUser(username string, password string, remoteIP string) (*model.User, bool) {
	db := database.GetDB()

	user := &model.User{}
	err := db.Model(model.User{}).
		Where("username = ?", username).
		First(user).
		Error
	if database.IsNotFound(err) {
		// Equalize timing with the wrong-password path so a missing username is
		// not distinguishable by response latency (user enumeration).
		common.EqualizeLoginTiming(password)
		return nil, false
	} else if err != nil {
		logger.Warning("check user err:", err, " IP: ", remoteIP)
		return nil, false
	}
	ok, needsMigration := common.CheckPassword(user.Password, password)
	if !ok {
		return nil, false
	}

	lastLoginTxt := time.Now().Format("2006-01-02 15:04:05") + " " + remoteIP
	err = db.Model(model.User{}).
		Where("username = ?", username).
		Update("last_logins", &lastLoginTxt).Error
	if err != nil {
		logger.Warning("unable to log login data", err)
	}
	return user, needsMigration
}

func (s *UserService) GetUsers() (*[]model.User, error) {
	var users []model.User
	db := database.GetDB()
	err := db.Model(model.User{}).Select("id,username,last_logins").Scan(&users).Error
	if err != nil {
		return nil, err
	}
	return &users, nil
}

func (s *UserService) UserExists(username string) (bool, error) {
	if username == "" {
		return false, nil
	}
	var count int64
	err := database.GetDB().Model(model.User{}).Where("username = ?", username).Count(&count).Error
	return count > 0, err
}

func (s *UserService) AddUser(actorUsername string, currentPass string, newUsername string, newPassword string) (*model.User, error) {
	newUsername = strings.TrimSpace(newUsername)
	if newUsername == "" {
		return nil, common.NewError("username can not be empty")
	}
	if newPassword == "" {
		return nil, common.NewError("password can not be empty")
	}

	var created model.User
	err := database.GetDB().Transaction(func(tx *gorm.DB) error {
		if err := s.checkUserPassword(tx, actorUsername, currentPass); err != nil {
			return err
		}
		var count int64
		if err := tx.Model(model.User{}).Where("username = ?", newUsername).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return common.NewError("user already exists")
		}
		passwordHash, err := common.HashPassword(newPassword)
		if err != nil {
			return err
		}
		created = model.User{
			Username:           newUsername,
			Password:           passwordHash,
			ForcePasswordReset: false,
		}
		return tx.Create(&created).Error
	})
	if err != nil {
		return nil, err
	}
	return &created, nil
}

func (s *UserService) DeleteUser(actorUsername string, currentPass string, targetID string) (DeleteUserResult, error) {
	var result DeleteUserResult
	id, err := parseUserID(targetID)
	if err != nil {
		return result, err
	}
	err = database.GetDB().Transaction(func(tx *gorm.DB) error {
		if err := s.checkUserPassword(tx, actorUsername, currentPass); err != nil {
			return err
		}
		var target model.User
		if err := tx.Model(model.User{}).Where("id = ?", id).First(&target).Error; err != nil {
			return err
		}
		if target.Username == actorUsername {
			return common.NewError("current admin can not be deleted")
		}
		tokenDelete := tx.Where("user_id = ?", target.Id).Delete(&model.Tokens{})
		if tokenDelete.Error != nil {
			return tokenDelete.Error
		}
		if err := tx.Delete(&target).Error; err != nil {
			return err
		}
		result.User = target
		result.DeletedTokenCount = tokenDelete.RowsAffected
		return nil
	})
	return result, err
}

// ChangePass updates the credentials of the user identified by username. The
// caller passes the AUTHENTICATED session user's name (never a client-supplied
// id), so an admin can only change their own account, not another admin's.
func (s *UserService) ChangePass(username string, oldPass string, newUser string, newPass string) error {
	db := database.GetDB()
	user := &model.User{}
	err := db.Model(model.User{}).Where("username = ?", username).First(user).Error
	if err != nil || database.IsNotFound(err) {
		return err
	}
	ok, _ := common.CheckPassword(user.Password, oldPass)
	if !ok {
		return common.NewError("wrong user or password")
	}
	passwordHash, err := common.HashPassword(newPass)
	if err != nil {
		return err
	}
	user.Username = newUser
	user.Password = passwordHash
	user.ForcePasswordReset = false
	return db.Save(user).Error
}

func (s *UserService) checkUserPassword(tx *gorm.DB, username string, password string) error {
	if username == "" || password == "" {
		return common.NewError("wrong user or password")
	}
	user := &model.User{}
	err := tx.Model(model.User{}).Where("username = ?", username).First(user).Error
	if database.IsNotFound(err) {
		return common.NewError("wrong user or password")
	} else if err != nil {
		return err
	}
	ok, _ := common.CheckPassword(user.Password, password)
	if !ok {
		return common.NewError("wrong user or password")
	}
	return nil
}

func parseUserID(raw string) (uint, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, common.NewError("user id can not be empty")
	}
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || id == 0 {
		return 0, common.NewError("invalid user id")
	}
	return uint(id), nil
}

func (s *UserService) updatePasswordHash(user *model.User, password string) error {
	passwordHash, err := common.HashPassword(password)
	if err != nil {
		return err
	}
	return database.GetDB().Model(model.User{}).Where("id = ?", user.Id).Updates(map[string]any{
		"password":             passwordHash,
		"force_password_reset": false,
	}).Error
}

func (s *UserService) LoadTokens() ([]byte, error) {
	if err := s.migrateLegacyTokens(); err != nil {
		return nil, err
	}
	db := database.GetDB()
	var tokens []model.Tokens
	err := db.Model(model.Tokens{}).Preload("User").
		Where("enabled = ? AND token_hash <> '' AND (expiry = 0 OR expiry > ?)", true, time.Now().Unix()).
		Find(&tokens).Error
	if err != nil {
		return nil, err
	}
	var result []map[string]interface{}
	for _, t := range tokens {
		if t.User == nil {
			continue
		}
		result = append(result, map[string]interface{}{
			"id":          t.Id,
			"tokenHash":   t.TokenHash,
			"tokenPrefix": t.TokenPrefix,
			"scope":       normalizeTokenScope(t.Scope),
			"enabled":     t.Enabled,
			"expiry":      t.Expiry,
			"username":    t.User.Username,
		})
	}
	jsonResult, _ := json.MarshalIndent(result, "", "  ")
	return jsonResult, nil
}

func (s *UserService) GetUserTokens(username string) (*[]model.Tokens, error) {
	if err := s.migrateLegacyTokens(); err != nil {
		return nil, err
	}
	db := database.GetDB()
	var token []model.Tokens
	err := db.Model(model.Tokens{}).
		Select("id, desc, token_prefix, scope, enabled, expiry, user_id, created_at, updated_at, last_used_at, last_used_ip").
		Where("user_id = (select id from users where username = ?)", username).
		Order("id desc").
		Find(&token).Error
	if err != nil && !database.IsNotFound(err) {
		logger.Warning("get user tokens failed:", err)
		return nil, err
	}
	for i := range token {
		token[i].Token = maskedToken(token[i].TokenPrefix)
		token[i].Scope = normalizeTokenScope(token[i].Scope)
	}
	return &token, nil
}

func (s *UserService) AddToken(username string, expiry int64, desc string, scope string) (string, error) {
	db := database.GetDB()
	scope, err := validateTokenScope(scope)
	if err != nil {
		return "", err
	}
	var userId uint
	err = db.Model(model.User{}).Where("username = ?", username).Select("id").Scan(&userId).Error
	if err != nil {
		return "", err
	}
	if expiry > 0 {
		expiry = expiry*86400 + time.Now().Unix()
	}
	plainToken := common.Random(32)
	tokenHash, err := s.HashAPIToken(plainToken)
	if err != nil {
		return "", err
	}
	now := time.Now().Unix()
	token := &model.Tokens{
		Desc:        desc,
		TokenHash:   tokenHash,
		TokenPrefix: tokenPrefix(plainToken),
		Scope:       scope,
		Enabled:     true,
		Expiry:      expiry,
		CreatedAt:   now,
		UpdatedAt:   now,
		UserId:      userId,
	}
	err = db.Create(token).Error
	if err != nil {
		return "", err
	}
	return plainToken, nil
}

func (s *UserService) DeleteToken(id string) error {
	db := database.GetDB()
	return db.Model(model.Tokens{}).Where("id = ?", id).Delete(&model.Tokens{}).Error
}

func (s *UserService) SetTokenEnabled(id string, enabled bool) error {
	return database.GetDB().Model(model.Tokens{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"enabled":    enabled,
			"updated_at": time.Now().Unix(),
		}).Error
}

func (s *UserService) RecordTokenUse(id uint, ip string) error {
	debouncer := s.runtime().tokenUseDebouncer()
	if debouncer != nil {
		debouncer.Record(id, ip, time.Now().Unix())
	}
	return nil
}

func (s *UserService) HashAPIToken(token string) (string, error) {
	salt, err := (&SettingService{}).GetInstallSalt()
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	hash.Write(salt)
	hash.Write([]byte{0})
	hash.Write([]byte(token))
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func (s *UserService) migrateLegacyTokens() error {
	db := database.GetDB()
	var tokens []model.Tokens
	if err := db.Model(model.Tokens{}).Where("(token_hash = '' OR token_hash IS NULL) AND token <> ''").Find(&tokens).Error; err != nil {
		return err
	}
	for _, token := range tokens {
		tokenHash, err := s.HashAPIToken(token.Token)
		if err != nil {
			return err
		}
		now := time.Now().Unix()
		updates := map[string]interface{}{
			"token":        "",
			"token_hash":   tokenHash,
			"token_prefix": tokenPrefix(token.Token),
			"scope":        normalizeTokenScope(token.Scope),
			"updated_at":   now,
		}
		if token.CreatedAt == 0 {
			updates["created_at"] = now
		}
		if err := db.Model(model.Tokens{}).Where("id = ?", token.Id).Updates(updates).Error; err != nil {
			return err
		}
	}
	return nil
}

func normalizeTokenScope(scope string) string {
	scope = strings.TrimSpace(scope)
	if scope == "" {
		return defaultAPITokenScope
	}
	return scope
}

func validateTokenScope(scope string) (string, error) {
	scope = normalizeTokenScope(scope)
	if !apiTokenScopeAllowed(scope) {
		return "", common.NewError("invalid token scope")
	}
	return scope, nil
}

func apiTokenScopeAllowed(scope string) bool {
	if len(scope) > maxAPITokenScopeLen {
		return false
	}
	matched := 0
	for _, allowed := range allowedAPITokenScopes {
		matched |= constantTimeStringEqual(scope, allowed, maxAPITokenScopeLen)
	}
	return matched == 1
}

func constantTimeStringEqual(a string, b string, maxLen int) int {
	// Length mismatch sets diff without truncating len() into a byte (matches
	// the crypto/subtle convention of bailing on differing lengths).
	var diff byte
	if len(a) != len(b) {
		diff = 1
	}
	for i := 0; i < maxLen; i++ {
		var av byte
		var bv byte
		if i < len(a) {
			av = a[i]
		}
		if i < len(b) {
			bv = b[i]
		}
		diff |= av ^ bv
	}
	return subtle.ConstantTimeByteEq(diff, 0)
}

func tokenPrefix(token string) string {
	if len(token) <= 8 {
		return token
	}
	return token[:8]
}

func maskedToken(prefix string) string {
	if prefix == "" {
		return "****"
	}
	return "****" + prefix
}
