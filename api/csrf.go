package api

import (
	"crypto/subtle"
	"net/http"
	"strings"
	"time"

	"github.com/deposist/s-ui-x/util/common"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

const (
	// #nosec G101 -- session storage key name, not a credential.
	csrfTokenKey   = "CSRF_TOKEN"
	csrfExpiresKey = "CSRF_EXPIRES"
	csrfHeader     = "X-CSRF-Token"
	csrfTTL        = 2 * time.Hour
)

func (a *ApiService) IssueCSRFToken(c *gin.Context) {
	token := common.Random(32)
	expiresAt := time.Now().Add(csrfTTL).Unix()

	session := sessions.Default(c)
	session.Set(csrfTokenKey, token)
	session.Set(csrfExpiresKey, expiresAt)
	options := sessions.Options{
		Path:     "/",
		Secure:   resolveCookieSecure(c, &a.SettingService),
		HttpOnly: true,
		SameSite: resolveCookieSameSite(&a.SettingService),
	}
	if maxAge, err := a.SettingService.GetSessionMaxAge(); err == nil && maxAge > 0 {
		options.MaxAge = maxAge * 60
	}
	session.Options(options)
	if err := session.Save(); err != nil {
		jsonMsg(c, "csrf", err)
		return
	}
	jsonObj(c, gin.H{
		"token":     token,
		"expiresAt": expiresAt,
	}, nil)
}

func ResetSessionCSRF(s sessions.Session) {
	s.Delete(csrfTokenKey)
	s.Delete(csrfExpiresKey)
}

func (a *APIHandler) csrfMiddleware(c *gin.Context) {
	if !csrfProtectedMethod(c.Request.Method) || csrfExemptPath(c.Request.URL.Path, a.csrfLoginPath) {
		c.Next()
		return
	}
	session := sessions.Default(c)
	expected, ok := session.Get(csrfTokenKey).(string)
	if !ok || expected == "" {
		csrfForbidden(c, "missing csrf session")
		return
	}
	expiresAt, ok := session.Get(csrfExpiresKey).(int64)
	if !ok || expiresAt < time.Now().Unix() {
		csrfForbidden(c, "expired csrf token")
		return
	}
	actual := c.GetHeader(csrfHeader)
	if actual == "" || subtle.ConstantTimeCompare([]byte(actual), []byte(expected)) != 1 {
		csrfForbidden(c, "invalid csrf token")
		return
	}
	c.Next()
}

func csrfProtectedMethod(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func (a *APIHandler) cachedCSRFLoginPath() string {
	webPath, err := a.SettingService.GetWebPath()
	if err != nil {
		webPath = "/"
	}
	return csrfLoginPathForBase(webPath)
}

func csrfLoginPathForBase(basePath string) string {
	return joinURL(basePath, "api/login")
}

func csrfExemptPath(path string, loginPath string) bool {
	return path != "" && path == loginPath
}

func joinURL(base string, child string) string {
	base = strings.TrimSpace(base)
	child = strings.TrimLeft(strings.TrimSpace(child), "/")
	if base == "" {
		base = "/"
	}
	if !strings.HasPrefix(base, "/") {
		base = "/" + base
	}
	if !strings.HasSuffix(base, "/") {
		base += "/"
	}
	return base + child
}

func csrfForbidden(c *gin.Context, reason string) {
	c.AbortWithStatusJSON(http.StatusForbidden, Msg{
		Success: false,
		Msg:     "Invalid CSRF token",
		Obj: gin.H{
			"reason": reason,
		},
	})
}
