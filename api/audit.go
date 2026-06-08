package api

import (
	"github.com/deposist/s-ui-x/logger"
	"github.com/deposist/s-ui-x/service"

	"github.com/gin-gonic/gin"
)

func (a *ApiService) recordAudit(c *gin.Context, actor string, event string, resource string, severity string, details map[string]any) {
	if err := a.AuditService.Record(service.AuditEvent{
		Actor:     actor,
		Event:     event,
		Resource:  resource,
		Severity:  severity,
		IP:        getRemoteIp(c),
		UserAgent: c.Request.UserAgent(),
		Details:   details,
	}); err != nil {
		logger.Warning("audit record failed:", err)
	}
}
