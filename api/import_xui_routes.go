package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type importXUIRouteSpec struct {
	method  string
	path    string
	handler func(*ApiService) gin.HandlerFunc
}

var importXUIRouteSpecs = []importXUIRouteSpec{
	{method: http.MethodPost, path: "/import-xui", handler: func(a *ApiService) gin.HandlerFunc { return a.ImportXui }},
	{method: http.MethodPost, path: "/import-xui/plan", handler: func(a *ApiService) gin.HandlerFunc { return a.ImportXuiPlan }},
	{method: http.MethodPost, path: "/import-xui/apply", handler: func(a *ApiService) gin.HandlerFunc { return a.ImportXuiApply }},
	{method: http.MethodPost, path: "/import-xui/rollback", handler: func(a *ApiService) gin.HandlerFunc { return a.ImportXuiRollback }},
	{method: http.MethodGet, path: "/import-xui/reports", handler: func(a *ApiService) gin.HandlerFunc { return a.ImportXuiReports }},
}

func registerImportXUIRoutes(g *gin.RouterGroup, a *ApiService) {
	for _, spec := range importXUIRouteSpecs {
		switch spec.method {
		case http.MethodGet:
			g.GET(spec.path, spec.handler(a))
		case http.MethodPost:
			g.POST(spec.path, spec.handler(a))
		default:
			panic("unsupported import-xui route method: " + spec.method)
		}
	}
}
