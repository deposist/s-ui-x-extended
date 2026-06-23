package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/deposist/s-ui-x-extended/service"

	"github.com/gin-gonic/gin"
)

func (a *ApiService) RunDoctor(c *gin.Context) {
	if !a.requireTokenScopeAny(c, "doctor", "admin", "read", "write", "observability") {
		return
	}
	report := a.DoctorService.Run(getHostname(c))
	jsonObj(c, report, nil)
}

func (a *ApiService) DiagnoseClient(c *gin.Context) {
	if !a.requireTokenScopeAny(c, "doctor", "admin", "read", "write", "observability") {
		return
	}
	var req service.DoctorClientRequest
	if err := json.Unmarshal([]byte(c.PostForm("data")), &req); err != nil {
		if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
			c.JSON(http.StatusBadRequest, Msg{Success: false, Msg: "doctor: invalid request"})
			return
		}
	}
	// A caller-supplied diagnostic target is fetched by the panel through every
	// outbound (including the always-present direct dialer), so it is an SSRF
	// vector and must pass the same guard as the checkOutbound endpoints. An
	// empty target means "use the built-in default" and is safe.
	if strings.TrimSpace(req.Target) != "" {
		if err := validateOutboundCheckTarget(c.Request.Context(), req.Target); err != nil {
			jsonMsg(c, "doctor", err)
			return
		}
	}
	report, err := a.DoctorService.DiagnoseClient(req, getHostname(c))
	jsonObj(c, report, err)
}
