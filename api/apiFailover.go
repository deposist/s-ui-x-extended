package api

import "github.com/gin-gonic/gin"

// GetFailoverStatus returns every failover group's live active member plus
// per-member health, for the panel's active-member display.
func (a *ApiService) GetFailoverStatus(c *gin.Context) {
	status, err := a.ConfigService.FailoverStatus()
	jsonObj(c, status, err)
}
