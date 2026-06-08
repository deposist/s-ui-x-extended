package service

import (
	"github.com/deposist/s-ui-x/logger"
)

func RecordListenFallbackAudit(component string, requestedAddr string, fallbackAddr string, bindErr error) error {
	details := map[string]any{
		"component":      component,
		"requested_addr": requestedAddr,
		"fallback_addr":  fallbackAddr,
	}
	if bindErr != nil {
		details["bind_error"] = bindErr.Error()
	}
	err := (&AuditService{}).Record(AuditEvent{
		Actor:    "system",
		Event:    "listen_fallback",
		Resource: "network",
		Severity: AuditSeverityWarn,
		Details:  details,
	})
	if err != nil {
		logger.Warning("listen fallback audit failed:", err)
	}
	return err
}
