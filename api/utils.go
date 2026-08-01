package api

import (
	"net"
	"net/http"
	"net/netip"
	"os"
	"strings"
	"sync"

	"github.com/deposist/s-ui-x-extended/logger"
	"github.com/deposist/s-ui-x-extended/service"
	"github.com/deposist/s-ui-x-extended/util/redact"

	"github.com/gin-gonic/gin"
)

type Msg struct {
	Success bool        `json:"success"`
	Msg     string      `json:"msg"`
	Obj     interface{} `json:"obj"`
}

// getRemoteIp returns the client IP, walking the X-Forwarded-For chain from the
// transport peer outward and returning the first hop that is not in the
// configured list of trusted proxies. Without trusted proxies it always
// returns the transport peer.
func getRemoteIp(c *gin.Context) string {
	remoteIP := canonicalClientIP(splitRemoteIP(c.Request.RemoteAddr))
	if !isTrustedProxy(remoteIP) {
		return remoteIP
	}
	value := c.GetHeader("X-Forwarded-For")
	if value == "" {
		return remoteIP
	}
	parts := strings.Split(value, ",")
	// Walk right-to-left: strip trusted proxies.
	for i := len(parts) - 1; i >= 0; i-- {
		hop := canonicalClientIP(strings.TrimSpace(parts[i]))
		if hop == "" {
			continue
		}
		if !isTrustedProxy(hop) {
			return hop
		}
	}
	return remoteIP
}

func splitRemoteIP(addr string) string {
	ip, _, err := net.SplitHostPort(addr)
	if err != nil {
		return strings.Trim(addr, "[]")
	}
	return strings.Trim(ip, "[]")
}

func canonicalClientIP(value string) string {
	value = strings.TrimSpace(strings.Trim(value, "[]"))
	if value == "" || strings.Contains(value, "%") {
		return ""
	}
	addr, err := netip.ParseAddr(value)
	if err != nil || addr.Zone() != "" {
		return ""
	}
	return addr.Unmap().String()
}

// RequestIsHTTPS reports whether the request arrived over HTTPS, trusting
// X-Forwarded-Proto only when the peer is a configured trusted proxy. Exported
// so the security-headers middleware can reuse this gated check for its HSTS
// decision (a spoofed X-Forwarded-Proto from an untrusted client must not
// trigger HSTS).
func RequestIsHTTPS(c *gin.Context) bool {
	if c.Request.TLS != nil {
		return true
	}
	return isTrustedProxy(canonicalClientIP(splitRemoteIP(c.Request.RemoteAddr))) && strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https")
}

// cookieSameSiteNoneRequested reports whether the operator asked for
// SameSite=None session cookies via SUI_COOKIE_SAMESITE=none.
//
// This is needed when the panel is embedded in a cross-site iframe. CSRF
// protection remains enforced independently through the X-CSRF-Token header.
func cookieSameSiteNoneRequested() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("SUI_COOKIE_SAMESITE")), "none")
}

func resolveCookieSecure(c *gin.Context, settingService *service.SettingService) bool {
	if cookieSameSiteNoneRequested() {
		// Browsers reject SameSite=None cookies unless Secure is also set.
		return true
	}
	if settingService != nil {
		forceSecure, err := settingService.GetForceCookieSecure()
		if err != nil {
			logger.Warning("invalid forceCookieSecure setting:", err)
		} else if forceSecure {
			return true
		}

		if webURI, err := settingService.GetWebURI(); err == nil {
			if strings.HasPrefix(strings.ToLower(strings.TrimSpace(webURI)), "https://") {
				return true
			}
		} else {
			logger.Warning("unable to get webURI:", err)
		}

	}
	return RequestIsHTTPS(c)
}

// resolveCookieSameSite returns the SameSite mode for session cookies. It is
// Lax by default, Strict when the sessionSameSiteStrict setting is enabled, and
// None when SUI_COOKIE_SAMESITE=none opts into cross-site iframe embedding.
func resolveCookieSameSite(settingService *service.SettingService) http.SameSite {
	if cookieSameSiteNoneRequested() {
		return http.SameSiteNoneMode
	}
	if settingService != nil {
		strict, err := settingService.GetSessionSameSiteStrict()
		if err != nil {
			logger.Warning("invalid sessionSameSiteStrict setting:", err)
		} else if strict {
			return http.SameSiteStrictMode
		}
	}
	return http.SameSiteLaxMode
}

var (
	trustedProxiesMu     sync.Mutex
	trustedProxiesRaw    string
	trustedProxiesParsed []netip.Prefix
)

func parseTrustedProxies() []netip.Prefix {
	raw := os.Getenv("SUI_TRUSTED_PROXIES")
	trustedProxiesMu.Lock()
	defer trustedProxiesMu.Unlock()
	if raw == trustedProxiesRaw {
		return trustedProxiesParsed
	}
	trustedProxiesRaw = raw
	if raw == "" {
		trustedProxiesParsed = nil
		return nil
	}
	var parsed []netip.Prefix
	for _, item := range strings.Split(raw, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if prefix, err := netip.ParsePrefix(item); err == nil {
			parsed = append(parsed, prefix)
			continue
		}
		if itemAddr, err := netip.ParseAddr(item); err == nil {
			itemAddr = itemAddr.Unmap()
			parsed = append(parsed, netip.PrefixFrom(itemAddr, itemAddr.BitLen()))
			continue
		}
		logger.Warningf("invalid SUI_TRUSTED_PROXIES entry: %q", item)
	}
	trustedProxiesParsed = parsed
	return parsed
}

func isTrustedProxy(remoteIP string) bool {
	prefixes := parseTrustedProxies()
	if len(prefixes) == 0 {
		return false
	}
	addr, err := netip.ParseAddr(canonicalClientIP(remoteIP))
	if err != nil {
		return false
	}
	for _, prefix := range prefixes {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}

func getHostname(c *gin.Context) string {
	host := c.Request.Host
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	host = strings.Trim(host, "[]")
	if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}
	return host
}

func jsonMsg(c *gin.Context, msg string, err error) {
	jsonMsgObj(c, msg, nil, err)
}

func jsonObj(c *gin.Context, obj interface{}, err error) {
	jsonMsgObj(c, "", obj, err)
}

// jsonMsgObj writes the standard API envelope (Msg) at HTTP 200 with a success
// flag. This SPA contract is intentional: business/validation errors are
// conveyed as {success:false, msg} at HTTP 200 (the frontend reads the flag),
// not via HTTP status codes; transport-level handlers that genuinely need real
// status codes (xuiImportError, telegram backup) set them explicitly. The
// client-facing error text is redacted to avoid leaking paths/SQL/internals;
// the full error is still logged server-side.
func jsonMsgObj(c *gin.Context, msg string, obj interface{}, err error) {
	m := Msg{
		Obj: obj,
	}
	if err == nil {
		m.Success = true
		if msg != "" {
			m.Msg = msg
		}
	} else {
		m.Success = false
		m.Msg = msg + ": " + redact.String(err.Error())
		logger.Warning("failed :", err)
	}
	c.JSON(http.StatusOK, m)
}

func pureJsonMsg(c *gin.Context, success bool, msg string) {
	if success {
		c.JSON(http.StatusOK, Msg{
			Success: true,
			Msg:     msg,
		})
	} else {
		c.JSON(http.StatusOK, Msg{
			Success: false,
			Msg:     msg,
		})
	}
}

func checkLogin(c *gin.Context) {
	if SessionRequiresPasswordReset(c) {
		if passwordResetAllowedPath(c.Request.Method, c.Request.URL.Path) {
			c.Next()
			return
		}
		c.JSON(http.StatusForbidden, Msg{Success: false, Msg: service.ErrForcePasswordReset.Error(), Obj: gin.H{"forcePasswordReset": true}})
		c.Abort()
		return
	}
	if !IsLogin(c) {
		if c.GetHeader("X-Requested-With") == "XMLHttpRequest" {
			pureJsonMsg(c, false, "Invalid login")
		} else {
			c.Redirect(http.StatusTemporaryRedirect, loginRedirectPath())
		}
		c.Abort()
	} else {
		c.Next()
	}
}

func passwordResetAllowedPath(method string, path string) bool {
	webPath, err := (&service.SettingService{}).GetWebPath()
	if err != nil {
		webPath = "/"
	}
	switch path {
	case joinURL(webPath, "api/csrf"):
		return method == http.MethodGet
	case joinURL(webPath, "api/changePass"), joinURL(webPath, "api/logout"):
		return method == http.MethodPost
	default:
		return false
	}
}

func loginRedirectPath() string {
	webPath, err := (&service.SettingService{}).GetWebPath()
	if err != nil || webPath == "" {
		return "/login"
	}
	return strings.TrimRight(webPath, "/") + "/login"
}
