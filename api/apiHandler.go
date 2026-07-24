package api

import (
	"net/http"
	"strings"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/paidsub"

	"github.com/gin-gonic/gin"
)

type APIHandler struct {
	ApiService
	apiv2           *APIv2Handler
	csrfLoginPath   string
	authExemptPaths map[string]struct{}
}

// databaseOperationMiddleware keeps an API request's DB handle alive for its
// entire lifetime. During ImportDB, new requests wait while the restore drains
// existing work and atomically swaps the SQLite database.
func IsRestoreRequestPath(path string) bool {
	return strings.HasSuffix(path, "/importdb") || strings.HasSuffix(path, "/import-xui/rollback")
}

func databaseOperationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// The restore endpoint must acquire the exclusive barrier itself; taking a
		// shared request lease here would otherwise deadlock behind this request.
		if IsRestoreRequestPath(c.Request.URL.Path) {
			c.Next()
			return
		}
		leave := database.EnterDBOperation()
		defer leave()
		c.Next()
	}
}

func NewAPIHandler(g *gin.RouterGroup, a2 *APIv2Handler, options ...Option) {
	a := &APIHandler{
		ApiService: NewApiService(options...),
		apiv2:      a2,
	}
	a.initRouter(g)
}

func (a *APIHandler) initRouter(g *gin.RouterGroup) {
	a.csrfLoginPath = a.cachedCSRFLoginPath()
	a.authExemptPaths = a.cachedAuthExemptPaths()
	g.Use(func(c *gin.Context) {
		// Exempt by EXACT path (not a "login"/"logout" suffix), so a future route
		// like /api/sociallogin can never accidentally inherit the auth bypass.
		if _, exempt := a.authExemptPaths[c.Request.URL.Path]; !exempt {
			checkLogin(c)
		}
	})
	g.Use(a.csrfMiddleware)
	a.registerGroupedRoutes(g)
}

// cachedAuthExemptPaths returns the exact full paths exempt from checkLogin:
// /login (no session yet) and /logout (terminates a session, CSRF-protected as a
// POST instead).
func (a *APIHandler) cachedAuthExemptPaths() map[string]struct{} {
	webPath, err := a.SettingService.GetWebPath()
	if err != nil {
		webPath = "/"
	}
	return map[string]struct{}{
		joinURL(webPath, "api/login"):  {},
		joinURL(webPath, "api/logout"): {},
	}
}

func (a *APIHandler) registerGroupedRoutes(g *gin.RouterGroup) {
	g.POST("/login", a.ApiService.Login)
	g.POST("/changePass", a.reloadTokensAfter(a.ApiService.ChangePass))
	g.POST("/addAdmin", a.ApiService.AddAdmin)
	g.POST("/deleteAdmin", a.reloadTokensAfter(a.ApiService.DeleteAdmin))
	g.POST("/save", a.save)
	g.POST("/restartApp", a.ApiService.RestartApp)
	g.POST("/restartSb", a.ApiService.RestartSb)
	g.POST("/regenerateClientLinks", a.ApiService.RegenerateClientLinks)
	g.POST("/linkConvert", a.ApiService.LinkConvert)
	g.POST("/subConvert", a.ApiService.SubConvert)
	g.POST("/importdb", a.ApiService.ImportDb)
	a.registerBrowserImportXUIRoutes(g)
	g.POST("/addToken", a.reloadTokensAfter(a.ApiService.AddToken))
	g.POST("/deleteToken", a.reloadTokensAfter(a.ApiService.DeleteToken))
	g.POST("/setTokenEnabled", a.reloadTokensAfter(a.ApiService.SetTokenEnabled))
	g.POST("/logoutAllAdmins", a.ApiService.LogoutAllAdmins)

	g.GET("/csrf", a.ApiService.GetCSRF)
	// Logout is a POST so the CSRF middleware protects it (a GET carries no CSRF
	// token), preventing cross-site forced logout. It stays exempt from
	// checkLogin via authExemptPaths.
	g.POST("/logout", a.ApiService.Logout)
	g.GET("/load", a.ApiService.LoadData)
	for _, action := range []string{"inbounds", "outbounds", "endpoints", "providers", "services", "tls", "clients", "config"} {
		action := action
		g.GET("/"+action, a.loadPartialData(action))
	}
	g.GET("/users", a.ApiService.GetUsers)
	g.GET("/settings", a.ApiService.GetSettings)
	g.GET("/stats", a.ApiService.GetStats)
	g.GET("/stats/traffic", a.ApiService.GetTrafficStats)
	g.GET("/status", a.ApiService.GetStatus)
	g.GET("/onlines", a.ApiService.GetOnlines)
	g.GET("/logs", a.ApiService.GetLogs)
	g.GET("/capabilities", a.ApiService.GetCapabilities)
	g.GET("/changes", a.ApiService.CheckChanges)
	g.GET("/keypairs", a.ApiService.GetKeypairs)
	g.GET("/getdb", a.ApiService.GetDb)
	g.GET("/tokens", a.ApiService.GetTokens)
	g.GET("/singbox-config", a.ApiService.GetSingboxConfig)
	g.GET("/checkOutbound", a.ApiService.GetCheckOutbound)
	g.GET("/groupPreview", a.ApiService.GetGroupPreview)
	g.GET("/failover-status", a.ApiService.GetFailoverStatus)
	g.GET("/version", a.ApiService.GetVersionInfo)
	g.POST("/checkOutbounds", a.ApiService.CheckOutbounds)
	g.POST("/rotateSubSecret", a.ApiService.RotateSubSecret)

	doctor := g.Group("/doctor")
	doctor.POST("/run", a.ApiService.RunDoctor)
	doctor.POST("/client", a.ApiService.DiagnoseClient)

	security := g.Group("/security")
	security.GET("/audit", a.ApiService.GetSecurityAudit)

	// Panel self-update (Maintenance). Inherits the group's session + CSRF auth
	// (SR-001/SR-008); apply additionally requires step-up re-auth (SR-010).
	update := g.Group("/update")
	update.GET("/status", a.ApiService.UpdateStatus)
	update.POST("/check", a.ApiService.UpdateCheck)
	update.POST("/apply", a.ApiService.UpdateApply)

	telegram := g.Group("/telegram")
	telegram.POST("/test", a.ApiService.TestTelegram)
	telegram.POST("/detect-chat", a.ApiService.DetectTelegramChat)
	telegram.POST("/backup", a.ApiService.BackupToTelegram)
	telegram.POST("/backup/run", a.ApiService.RunTelegramBackup)

	realtime := g.Group("/realtime")
	realtime.GET("/ws-token", a.ApiService.IssueWSToken)
	realtime.GET("/ws", a.ApiService.RealtimeWS)

	a.registerAWGRoutes(g)

	ipMonitor := g.Group("/ip-monitor")
	ipMonitor.GET("/:client", a.ApiService.GetClientIPHistory)
	ipMonitor.POST("/:client/clear", a.ApiService.ClearClientIPHistory)

	observability := g.Group("/observability")
	observability.GET("/history", a.ApiService.GetObservabilityHistory)
	observability.GET("/core-history", a.ApiService.GetCoreHistory)

	// Experimental Paid Subscriptions module owns its own routes; mount them on
	// the already-authenticated (session + CSRF) browser group.
	paidsub.RegisterRoutes(g, paidsub.Deps{
		LoginUser: GetLoginUser,
		Audit:     a.ApiService.recordAudit,
	})
}

func (a *APIHandler) save(c *gin.Context) {
	a.ApiService.Save(c, GetLoginUser(c))
}

func (a *APIHandler) loadPartialData(action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		err := a.ApiService.LoadPartialData(c, []string{action})
		if err != nil {
			jsonMsg(c, action, err)
		}
	}
}

func (a *APIHandler) registerBrowserImportXUIRoutes(g *gin.RouterGroup) {
	for _, spec := range importXUIRouteSpecs {
		handler := spec.handler(&a.ApiService)
		switch spec.method {
		case http.MethodGet:
			g.GET(spec.path, handler)
		case http.MethodPost:
			g.POST(spec.path, a.reloadTokensAfter(handler))
		default:
			panic("unsupported import-xui route method: " + spec.method)
		}
	}
}

func (a *APIHandler) reloadTokensAfter(handler gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		handler(c)
		if a.apiv2 != nil {
			a.apiv2.ReloadTokens()
		}
	}
}
