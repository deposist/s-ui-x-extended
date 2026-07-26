package api

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/deposist/s-ui-x-extended/service"
	"github.com/deposist/s-ui-x-extended/util/common"
	"github.com/deposist/s-ui-x-extended/util/ssrf"

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
	if !a.requireTokenScopeAny(c, "admin", "write") {
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

	directMode, err := a.ruleSetDownloadIsDirect()
	if err != nil {
		jsonMsg(c, "rulesets", err)
		return
	}
	for _, src := range req.Sources {
		if err := validateRuleSetTargetURL(c, src.URL, directMode); err != nil {
			jsonMsg(c, "rulesets", err)
			return
		}
	}

	assets, err := a.RuleSetAssetService.Materialize(req.Sources)
	jsonObj(c, assets, err)
}

func (a *ApiService) ruleSetDownloadIsDirect() (bool, error) {
	mode, err := a.SettingService.GetRuleSetDownloadMode()
	if err != nil {
		return false, err
	}
	return mode != "outbound", nil
}

// validateRuleSetTargetURL guards a caller-supplied download target.
//
// The scheme and userinfo checks always apply. The full SSRF check (which
// resolves the hostname and rejects private addresses) only applies in direct
// mode, and deliberately so: it resolves through the panel's own resolver, and
// on a censored server that resolver is exactly what fails. Enforcing it in
// outbound mode would reject the download that the outbound could complete
// perfectly well, defeating the purpose of choosing an outbound. In outbound
// mode the connection is dialed by the core through the selected proxy rather
// than from the panel's own stack, so the panel is not the confused deputy the
// SSRF check protects against.
func validateRuleSetTargetURL(c *gin.Context, rawURL string, directMode bool) error {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return common.NewErrorf("invalid rule-set url: %v", err)
	}
	if parsed.Scheme != "https" || parsed.Hostname() == "" {
		return common.NewError("rule-set url must be an HTTPS URL")
	}
	if parsed.User != nil {
		return common.NewError("rule-set url must not include userinfo")
	}
	if !directMode {
		return nil
	}
	return ssrf.ValidateOutboundURL(c.Request.Context(), rawURL, "https")
}
