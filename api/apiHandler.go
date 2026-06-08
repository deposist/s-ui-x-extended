package api

import (
	"strings"

	"github.com/deposist/s-ui-x/paidsub"

	"github.com/gin-gonic/gin"
)

type APIHandler struct {
	ApiService
	apiv2         *APIv2Handler
	csrfLoginPath string
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
	g.Use(func(c *gin.Context) {
		path := c.Request.URL.Path
		if !strings.HasSuffix(path, "login") && !strings.HasSuffix(path, "logout") {
			checkLogin(c)
		}
	})
	g.Use(a.csrfMiddleware)
	a.registerGroupedRoutes(g)
}

func (a *APIHandler) registerGroupedRoutes(g *gin.RouterGroup) {
	g.POST("/login", a.ApiService.Login)
	g.POST("/changePass", a.ApiService.ChangePass)
	g.POST("/addAdmin", a.ApiService.AddAdmin)
	g.POST("/deleteAdmin", a.reloadTokensAfter(a.ApiService.DeleteAdmin))
	g.POST("/save", a.save)
	g.POST("/restartApp", a.ApiService.RestartApp)
	g.POST("/restartSb", a.ApiService.RestartSb)
	g.POST("/linkConvert", a.ApiService.LinkConvert)
	g.POST("/subConvert", a.ApiService.SubConvert)
	g.POST("/importdb", a.ApiService.ImportDb)
	registerImportXUIRoutes(g, &a.ApiService)
	g.POST("/addToken", a.reloadTokensAfter(a.ApiService.AddToken))
	g.POST("/deleteToken", a.reloadTokensAfter(a.ApiService.DeleteToken))
	g.POST("/setTokenEnabled", a.reloadTokensAfter(a.ApiService.SetTokenEnabled))
	g.POST("/logoutAllAdmins", a.ApiService.LogoutAllAdmins)

	g.GET("/csrf", a.ApiService.GetCSRF)
	g.GET("/logout", a.ApiService.Logout)
	g.GET("/load", a.ApiService.LoadData)
	for _, action := range []string{"inbounds", "outbounds", "endpoints", "providers", "services", "tls", "clients", "config"} {
		action := action
		g.GET("/"+action, a.loadPartialData(action))
	}
	g.GET("/users", a.ApiService.GetUsers)
	g.GET("/settings", a.ApiService.GetSettings)
	g.GET("/stats", a.ApiService.GetStats)
	g.GET("/status", a.ApiService.GetStatus)
	g.GET("/onlines", a.ApiService.GetOnlines)
	g.GET("/logs", a.ApiService.GetLogs)
	g.GET("/changes", a.ApiService.CheckChanges)
	g.GET("/keypairs", a.ApiService.GetKeypairs)
	g.GET("/getdb", a.ApiService.GetDb)
	g.GET("/tokens", a.ApiService.GetTokens)
	g.GET("/singbox-config", a.ApiService.GetSingboxConfig)
	g.GET("/checkOutbound", a.ApiService.GetCheckOutbound)
	g.GET("/version", a.ApiService.GetVersionInfo)
	g.POST("/checkOutbounds", a.ApiService.CheckOutbounds)
	g.POST("/rotateSubSecret", a.ApiService.RotateSubSecret)

	security := g.Group("/security")
	security.GET("/audit", a.ApiService.GetSecurityAudit)

	telegram := g.Group("/telegram")
	telegram.POST("/test", a.ApiService.TestTelegram)
	telegram.POST("/backup", a.ApiService.BackupToTelegram)
	telegram.POST("/backup/run", a.ApiService.RunTelegramBackup)

	realtime := g.Group("/realtime")
	realtime.GET("/ws-token", a.ApiService.IssueWSToken)
	realtime.GET("/ws", a.ApiService.RealtimeWS)

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

func (a *APIHandler) reloadTokensAfter(handler gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		handler(c)
		if a.apiv2 != nil {
			a.apiv2.ReloadTokens()
		}
	}
}
