package api

import (
	"encoding/json"
	"net/http"

	"github.com/deposist/s-ui-x-extended/service"

	"github.com/gin-gonic/gin"
)

// ruleSetMaterializeRequest is the payload for POST /rulesets/materialize.
type ruleSetMaterializeRequest struct {
	Sources []service.RuleSetSource `json:"sources"`
}

// maxRuleSetSourcesPerRequest bounds the work a single call can schedule.
const maxRuleSetSourcesPerRequest = 32

// MaterializeRuleSets downloads the requested rule-set files and returns their
// on-disk paths. The caller is expected to write a config referencing those
// paths only after this succeeds, which is what keeps a missing local file (a
// fatal startup error for the core) out of the saved config.
func (a *ApiService) MaterializeRuleSets(c *gin.Context) {
	if !a.requireTokenScopeAny(c, "rulesets", "admin", "write") {
		return
	}
	var req ruleSetMaterializeRequest
	if err := json.Unmarshal([]byte(c.PostForm("data")), &req); err != nil {
		if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
			c.JSON(http.StatusBadRequest, Msg{Success: false, Msg: "rulesets: invalid request"})
			return
		}
	}
	if len(req.Sources) == 0 {
		c.JSON(http.StatusBadRequest, Msg{Success: false, Msg: "rulesets: no sources provided"})
		return
	}
	if len(req.Sources) > maxRuleSetSourcesPerRequest {
		c.JSON(http.StatusBadRequest, Msg{Success: false, Msg: "rulesets: too many sources"})
		return
	}

	assets, err := a.RuleSetAssetService.MaterializeContext(c.Request.Context(), req.Sources)
	jsonObj(c, assets, err)
}
