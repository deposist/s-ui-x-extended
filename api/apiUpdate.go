package api

import (
	"sync"
	"time"

	"github.com/deposist/s-ui-x-extended/config"
	"github.com/deposist/s-ui-x-extended/service"
	"github.com/deposist/s-ui-x-extended/util/common"

	"github.com/gin-gonic/gin"
)

// updateStatusResponse is the GET /api/update/status (and check) payload: the
// channel version info plus the current self-update job.
type updateStatusResponse struct {
	service.VersionInfo
	Job service.UpdateJob `json:"job"`
}

const updateCheckMinInterval = 5 * time.Second

var (
	updateCheckMu     sync.Mutex
	updateCheckLastAt time.Time
)

// allowForcedUpdateCheck rate-limits forced (network) version checks to bound
// abuse and external-API usage (SR-009). When denied, callers serve the cached
// result instead of hitting GitHub again.
func allowForcedUpdateCheck(now time.Time) bool {
	updateCheckMu.Lock()
	defer updateCheckMu.Unlock()
	if !updateCheckLastAt.IsZero() && now.Sub(updateCheckLastAt) < updateCheckMinInterval {
		return false
	}
	updateCheckLastAt = now
	return true
}

func (a *ApiService) buildUpdateStatus(info service.VersionInfo) updateStatusResponse {
	return updateStatusResponse{VersionInfo: info, Job: a.PanelUpdateService.Status()}
}

// UpdateStatus reports current/available versions and job state without a forced
// network call (uses the cache). GET /api/update/status.
func (a *ApiService) UpdateStatus(c *gin.Context) {
	channel := a.SettingService.GetUpdateChannel()
	info := a.VersionService.CheckForChannel(channel, false)
	jsonObj(c, a.buildUpdateStatus(info), nil)
}

// UpdateCheck performs an explicit, rate-limited check for the selected channel
// and persists the channel choice. POST /api/update/check.
func (a *ApiService) UpdateCheck(c *gin.Context) {
	channel := config.NormalizeUpdateChannel(c.DefaultPostForm("channel", a.SettingService.GetUpdateChannel()))
	if err := a.SettingService.SetUpdateChannel(channel); err != nil {
		jsonMsg(c, "update", err)
		return
	}
	force := allowForcedUpdateCheck(time.Now())
	info := a.VersionService.CheckForChannel(channel, force)
	a.recordAudit(c, GetLoginUser(c), "panel_update_check", "update", service.AuditSeverityInfo, map[string]any{
		"channel": channel, "latest": info.Latest, "rateLimited": !force,
	})
	jsonObj(c, a.buildUpdateStatus(info), nil)
}

// UpdateApply starts an update to the channel's available version. It enforces
// step-up re-authentication (SR-010) and a confirmed target version (FR-016).
// POST /api/update/apply.
func (a *ApiService) UpdateApply(c *gin.Context) {
	user := GetLoginUser(c)
	remoteIP := getRemoteIp(c)
	channel := config.NormalizeUpdateChannel(c.DefaultPostForm("channel", a.SettingService.GetUpdateChannel()))
	password := c.PostForm("password")
	targetVersion := c.PostForm("targetVersion")

	// SR-010 step-up. The password is validated against the existing credential
	// check and is NEVER written to the audit log or details (CHK025).
	if password == "" {
		a.auditApply(c, user, channel, "", "reauth_required")
		jsonMsg(c, "update", common.NewError("re-authentication required"))
		return
	}
	// Reuse the login brute-force throttle so the step-up password cannot be
	// hammered from an authenticated session (defense-in-depth for SR-010).
	userKey := loginRateLimitUserKey(user)
	if err := checkLoginRateLimit(remoteIP); err != nil {
		a.auditApply(c, user, channel, "", "rate_limited")
		jsonMsg(c, "update", err)
		return
	}
	if u, _ := a.UserService.CheckUser(user, password, remoteIP); u == nil {
		recordLoginFailure(remoteIP)
		recordLoginFailure(userKey)
		a.auditApply(c, user, channel, "", "reauth_failed")
		jsonMsg(c, "update", common.NewError("re-authentication failed"))
		return
	}
	resetLoginFailures(remoteIP)
	resetLoginFailures(userKey)

	target, err := a.VersionService.ResolveTarget(channel)
	if err != nil {
		a.auditApply(c, user, channel, "", "no_target")
		jsonMsg(c, "update", err)
		return
	}

	// FR-016: refuse if the available version changed since the admin checked.
	if !targetVersionMatches(targetVersion, target) {
		a.auditApply(c, user, channel, target.Version, "version_changed")
		jsonMsg(c, "update", common.NewError("available version changed; re-run Check updates"))
		return
	}

	if err := a.PanelUpdateService.Apply(target, user); err != nil {
		a.auditApply(c, user, channel, target.Version, "rejected")
		jsonMsg(c, "update", err)
		return
	}
	a.auditApply(c, user, channel, target.Version, "started")
	jsonObj(c, a.buildUpdateStatus(a.VersionService.CheckForChannel(channel, false)), nil)
}

func (a *ApiService) auditApply(c *gin.Context, user string, channel string, to string, result string) {
	severity := service.AuditSeverityInfo
	if result != "started" {
		severity = service.AuditSeverityWarn
	}
	a.recordAudit(c, user, "panel_update_apply", "update", severity, map[string]any{
		"channel": channel, "from": config.GetVersion(), "to": to, "result": result,
	})
}

// targetVersionMatches enforces FR-016: a non-empty requested version must match
// the resolved target (by tag or normalized semver).
func targetVersionMatches(requested string, target service.ReleaseTarget) bool {
	if requested == "" {
		return false
	}
	if requested == target.Tag || requested == target.Version {
		return true
	}
	return config.NormalizeVersion(requested) == target.Version
}
