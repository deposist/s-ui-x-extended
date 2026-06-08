package api

import (
	"strconv"

	"github.com/deposist/s-ui-x/ipmonitor"
	"github.com/deposist/s-ui-x/service"

	"github.com/gin-gonic/gin"
)

func (a *ApiService) GetClientIPHistory(c *gin.Context) {
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "100"))
	if err != nil {
		limit = 100
	}
	rows, err := ipmonitor.History(c.Param("client"), limit)
	jsonObj(c, rows, err)
}

func (a *ApiService) ClearClientIPHistory(c *gin.Context) {
	clientName := c.Param("client")
	err := ipmonitor.Clear(clientName)
	if err == nil {
		a.recordAudit(c, GetLoginUser(c), "client_ip_history_cleared", "client", service.AuditSeverityWarn, map[string]any{
			"client": clientName,
		})
	}
	jsonMsg(c, "save", err)
}
