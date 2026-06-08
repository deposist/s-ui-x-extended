package api

import (
	"net/http"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x/database/importxui"

	"github.com/gin-gonic/gin"
)

func TestIssue37ImportXuiApplyAcceptsLargePlan(t *testing.T) {
	settingService, src := setupXuiAPITestDB(t)
	router, cookies := newAuthenticatedTestRouter(t, settingService, func(router *gin.Engine) {
		router.POST("/api/import-xui/plan", withTestTokenScope("admin", "admin", (&ApiService{}).ImportXuiPlan))
		router.POST("/api/import-xui/apply", withTestTokenScope("admin", "admin", (&ApiService{}).ImportXuiApply))
	})

	planRecorder := performAuthenticatedTestRequest(router, newXuiImportRequest(t, "/api/import-xui/plan", readFile(t, src), "1"), cookies...)
	if planRecorder.Code != http.StatusOK {
		t.Fatalf("plan status=%d body=%s", planRecorder.Code, planRecorder.Body.String())
	}
	plan := decodePlanResponse(t, planRecorder.Body.Bytes())
	if len(plan.Items) == 0 {
		t.Fatal("test plan has no items to pad")
	}

	const renamedTag = "issue37-streamed-trojan"
	renamed := false
	for i := range plan.Items {
		if plan.Items[i].Kind == importxui.KindInbound && plan.Items[i].SrcTag == "inbound-12223" {
			plan.Items[i].DstTag = renamedTag
			renamed = true
			break
		}
	}
	if !renamed {
		t.Fatal("test plan did not include inbound-12223")
	}
	plan.Items[0].Warnings = []string{strings.Repeat("a", maxXUIFieldBytes+1024)}

	applyRecorder := performAuthenticatedTestRequest(router, newXuiApplyRequest(t, readFile(t, src), plan), cookies...)
	if applyRecorder.Code != http.StatusOK {
		t.Fatalf("large plan should be accepted, status=%d body=%s", applyRecorder.Code, applyRecorder.Body.String())
	}
	if inboundByTagForAPI(t, renamedTag).Type != "trojan" {
		t.Fatal("large streamed plan was not applied")
	}
}
