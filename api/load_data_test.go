package api

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/deposist/s-ui-x-extended/core"
	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/deposist/s-ui-x-extended/service"

	"github.com/gin-gonic/gin"
)

func TestLoadDataIncludesSubscriptionURIOverrides(t *testing.T) {
	settingService := initSessionTestDB(t)
	restoreRuntime := service.ReplaceDefaultRuntimeForTest(service.NewRuntime(core.NewCore()))
	t.Cleanup(restoreRuntime)
	if _, err := settingService.GetAllSetting(); err != nil {
		t.Fatal(err)
	}
	for key, value := range map[string]string{
		"subJsonURI":  "https://json.example/sub/",
		"subClashURI": "https://clash.example/sub/",
	} {
		if err := database.GetDB().Model(model.Setting{}).Where("key = ?", key).Update("value", value).Error; err != nil {
			t.Fatal(err)
		}
	}

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "http://panel.example/api/load", nil)

	data, err := (&ApiService{}).getData(c)
	if err != nil {
		t.Fatal(err)
	}
	payload, ok := data.(map[string]interface{})
	if !ok {
		t.Fatalf("unexpected load payload: %#v", data)
	}
	if payload["subJsonURI"] != "https://json.example/sub/" {
		t.Fatalf("subJsonURI override missing: %#v", payload)
	}
	if payload["subClashURI"] != "https://clash.example/sub/" {
		t.Fatalf("subClashURI override missing: %#v", payload)
	}
}

func TestLoadDataReturnsAuthoritativeRevisionForFullAndUnchangedResponses(t *testing.T) {
	initSessionTestDB(t)
	runtime := service.NewRuntime(core.NewCore())
	restoreRuntime := service.ReplaceDefaultRuntimeForTest(runtime)
	t.Cleanup(restoreRuntime)
	apiService := NewApiService(WithRuntime(runtime))

	gin.SetMode(gin.TestMode)
	load := func(rawURL string) map[string]interface{} {
		t.Helper()
		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		c.Request = httptest.NewRequest(http.MethodGet, rawURL, nil)
		data, err := apiService.getData(c)
		if err != nil {
			t.Fatal(err)
		}
		payload, ok := data.(map[string]interface{})
		if !ok {
			t.Fatalf("unexpected load payload: %#v", data)
		}
		return payload
	}

	full := load("http://panel.example/api/load")
	revision, ok := full["revision"].(int64)
	if !ok || revision <= 0 {
		t.Fatalf("expected positive authoritative revision, got %#v", full["revision"])
	}
	if _, ok := full["config"]; !ok {
		t.Fatalf("expected initial full snapshot, got %#v", full)
	}

	unchanged := load("http://panel.example/api/load?lu=" + strconv.FormatInt(revision, 10))
	if unchanged["revision"] != revision {
		t.Fatalf("revision changed without a server update: got %#v want %d", unchanged["revision"], revision)
	}
	if _, ok := unchanged["config"]; ok {
		t.Fatalf("unchanged response unexpectedly included config: %#v", unchanged)
	}
}
