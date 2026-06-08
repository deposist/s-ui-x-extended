package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/deposist/s-ui-x/core"
	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
	"github.com/deposist/s-ui-x/service"

	"github.com/gin-gonic/gin"
)

func TestLoadDataIncludesSubscriptionURIOverrides(t *testing.T) {
	settingService := initSessionTestDB(t)
	service.NewConfigService(core.NewCore())
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
