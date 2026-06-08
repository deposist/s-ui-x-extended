package api

import (
	"net/http"
	"strings"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

func TestAPIHandlerRegistersLegacyActionRoutesExplicitly(t *testing.T) {
	initSessionTestDB(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := &APIHandler{}
	handler.initRouter(router.Group("/api"))

	routes := map[string]bool{}
	for _, route := range router.Routes() {
		routes[route.Method+" "+route.Path] = true
		if route.Path == "/api/:postAction" || route.Path == "/api/:getAction" {
			t.Fatalf("legacy catch-all route still registered: %s %s", route.Method, route.Path)
		}
	}

	expected := map[string][]string{
		http.MethodPost: {
			"/api/login",
			"/api/changePass",
			"/api/addAdmin",
			"/api/deleteAdmin",
			"/api/save",
			"/api/restartApp",
			"/api/restartSb",
			"/api/linkConvert",
			"/api/subConvert",
			"/api/importdb",
			"/api/import-xui",
			"/api/import-xui/plan",
			"/api/import-xui/apply",
			"/api/import-xui/rollback",
			"/api/addToken",
			"/api/deleteToken",
			"/api/setTokenEnabled",
			"/api/logoutAllAdmins",
			"/api/checkOutbounds",
			"/api/rotateSubSecret",
			"/api/telegram/test",
			"/api/telegram/backup",
			"/api/telegram/backup/run",
			"/api/ip-monitor/:client/clear",
		},
		http.MethodGet: {
			"/api/csrf",
			"/api/logout",
			"/api/load",
			"/api/inbounds",
			"/api/outbounds",
			"/api/endpoints",
			"/api/services",
			"/api/tls",
			"/api/clients",
			"/api/config",
			"/api/users",
			"/api/settings",
			"/api/stats",
			"/api/status",
			"/api/onlines",
			"/api/logs",
			"/api/changes",
			"/api/keypairs",
			"/api/getdb",
			"/api/tokens",
			"/api/singbox-config",
			"/api/checkOutbound",
			"/api/version",
			"/api/import-xui/reports",
			"/api/security/audit",
			"/api/realtime/ws-token",
			"/api/realtime/ws",
			"/api/ip-monitor/:client",
			"/api/observability/history",
			"/api/observability/core-history",
		},
	}

	for method, paths := range expected {
		for _, path := range paths {
			if !routes[method+" "+path] {
				t.Fatalf("missing explicit route %s %s", method, path)
			}
		}
	}
}

func TestImportXUIRoutesUseSharedRegistryIssue35(t *testing.T) {
	initSessionTestDB(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(sessions.Sessions("s-ui", cookie.NewStore([]byte("test-secret"))))
	apiv2 := NewAPIv2Handler(router.Group("/apiv2"))
	NewAPIHandler(router.Group("/api"), apiv2)

	routes := map[string]gin.RouteInfo{}
	for _, route := range router.Routes() {
		routes[route.Method+" "+route.Path] = route
	}

	for _, spec := range importXUIRouteSpecs {
		for _, prefix := range []string{"/api", "/apiv2"} {
			key := spec.method + " " + prefix + spec.path
			if _, ok := routes[key]; !ok {
				t.Fatalf("missing import-xui shared route %s", key)
			}
		}
	}

	route, ok := routes[http.MethodPost+" /apiv2/import-xui"]
	if !ok {
		t.Fatal("missing explicit POST /apiv2/import-xui route")
	}
	if strings.Contains(route.Handler, "postHandler") {
		t.Fatalf("POST /apiv2/import-xui is still handled by generic postHandler: %s", route.Handler)
	}
}
