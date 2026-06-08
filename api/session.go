package api

import (
	"encoding/gob"

	"github.com/deposist/s-ui-x/database/model"
	"github.com/deposist/s-ui-x/logger"
	"github.com/deposist/s-ui-x/service"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

const (
	loginUser              = "LOGIN_USER"
	loginSessionGeneration = "LOGIN_SESSION_GENERATION"
)

func init() {
	gob.Register(model.User{})
}

func SetLoginUser(c *gin.Context, userName string, maxAge int, sessionGeneration string) error {
	options := sessions.Options{
		Path:     "/",
		Secure:   resolveCookieSecure(c, &service.SettingService{}),
		HttpOnly: true,
		SameSite: resolveCookieSameSite(&service.SettingService{}),
	}
	if maxAge > 0 {
		options.MaxAge = maxAge * 60
	}

	s := sessions.Default(c)
	s.Set(loginUser, userName)
	if sessionGeneration != "" {
		s.Set(loginSessionGeneration, sessionGeneration)
	}
	ResetSessionCSRF(s)
	// Rotate the session ID on login so a planted pre-auth (CSRF) session cannot
	// survive authentication under an attacker-known ID (session-fixation defense).
	s.Set(service.SessionRegenerateKey, true)
	s.Options(options)

	return s.Save()
}

func GetLoginUser(c *gin.Context) string {
	s := sessions.Default(c)
	obj := s.Get(loginUser)
	if obj == nil {
		return ""
	}
	objStr, ok := obj.(string)
	if !ok {
		return ""
	}
	if !sessionGenerationValid(s) {
		return ""
	}
	if !sessionUserExists(objStr) {
		return ""
	}
	return objStr
}

func sessionUserExists(username string) bool {
	exists, err := (&service.UserService{}).UserExists(username)
	if err != nil {
		logger.Warning("unable to validate session user:", err)
		return false
	}
	return exists
}

func sessionGenerationValid(s sessions.Session) bool {
	current, err := (&service.SettingService{}).GetSessionGeneration()
	if err != nil {
		logger.Warning("unable to get session generation:", err)
		return false
	}
	if current == "" {
		return true
	}
	obj := s.Get(loginSessionGeneration)
	sessionGeneration, ok := obj.(string)
	return ok && sessionGeneration == current
}

func IsLogin(c *gin.Context) bool {
	return GetLoginUser(c) != ""
}

func ClearSession(c *gin.Context) {
	s := sessions.Default(c)
	s.Clear()
	s.Options(sessions.Options{
		Path:     "/",
		MaxAge:   -1,
		Secure:   resolveCookieSecure(c, &service.SettingService{}),
		HttpOnly: true,
		SameSite: resolveCookieSameSite(&service.SettingService{}),
	})
	if err := s.Save(); err != nil {
		logger.Warning("failed to save cleared session: ", err)
	}
}
