package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
	"github.com/deposist/s-ui-x/service"
	"github.com/gin-gonic/gin"
)

func TestGetObservabilityHistoryFiltersMetricBucketAndSince(t *testing.T) {
	settingService := initSessionTestDB(t)
	base := time.Now().Unix() + 100000
	observabilityService := &service.ObservabilityService{}
	if err := observabilityService.RecordObservabilitySample(service.ObservabilityBucket30s, service.ObservabilitySample{
		DateTime: base,
		CPU:      1,
		Memory:   map[string]interface{}{"current": uint64(10)},
		Network:  map[string]interface{}{"recv": uint64(100), "sent": uint64(200)},
	}); err != nil {
		t.Fatal(err)
	}
	if err := observabilityService.RecordObservabilitySample(service.ObservabilityBucket30s, service.ObservabilitySample{
		DateTime: base + 10,
		CPU:      3,
		Memory:   map[string]interface{}{"current": uint64(30)},
		Network:  map[string]interface{}{"recv": uint64(300), "sent": uint64(400)},
	}); err != nil {
		t.Fatal(err)
	}

	router, cookies := newAuthenticatedTestRouter(t, settingService, func(router *gin.Engine) {
		router.GET("/api/observability/history", withTestTokenScope("observer", "observability", (&ApiService{}).GetObservabilityHistory))
	})
	recorder := performAuthenticatedTestRequest(router, httptest.NewRequest(http.MethodGet, "/api/observability/history?metric=net_in&bucket=30s&since="+formatUnix(base), nil), cookies...)
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", recorder.Code)
	}
	var msg Msg
	if err := json.Unmarshal(recorder.Body.Bytes(), &msg); err != nil {
		t.Fatal(err)
	}
	if !msg.Success {
		t.Fatalf("observability request failed: %s", msg.Msg)
	}
	payload := msg.Obj.(map[string]interface{})
	if payload["bucket"] != "30s" || payload["metric"] != "net_in" {
		t.Fatalf("unexpected payload metadata: %#v", payload)
	}
	samples := payload["samples"].([]interface{})
	if len(samples) != 1 {
		t.Fatalf("expected one sample after since filter, got %#v", samples)
	}
	sample := samples[0].(map[string]interface{})
	if sample["dateTime"].(float64) != float64(base+10) || sample["value"].(float64) != 300 {
		t.Fatalf("unexpected metric sample: %#v", sample)
	}
}

func TestGetObservabilityHistoryRejectsInvalidInputs(t *testing.T) {
	settingService := initSessionTestDB(t)
	router, cookies := newAuthenticatedTestRouter(t, settingService, func(router *gin.Engine) {
		router.GET("/api/observability/history", withTestTokenScope("observer", "observability", (&ApiService{}).GetObservabilityHistory))
	})
	for _, target := range []string{
		"/api/observability/history?metric=net_in&bucket=10s",
		"/api/observability/history?metric=load&bucket=2s",
		"/api/observability/history?metric=cpu&bucket=2s&since=-1",
	} {
		recorder := performAuthenticatedTestRequest(router, httptest.NewRequest(http.MethodGet, target, nil), cookies...)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("%s returned status %d", target, recorder.Code)
		}
	}
}

func TestGetObservabilityHistoryRequiresObservabilityScope(t *testing.T) {
	settingService := initSessionTestDB(t)
	router, cookies := newAuthenticatedTestRouter(t, settingService, func(router *gin.Engine) {
		router.GET("/api/observability/history", withTestTokenScope("api-user", "read", (&ApiService{}).GetObservabilityHistory))
	})
	recorder := performAuthenticatedTestRequest(router, httptest.NewRequest(http.MethodGet, "/api/observability/history", nil), cookies...)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("unexpected status: %d", recorder.Code)
	}
	var event model.AuditEvent
	if err := database.GetDB().Where("event = ?", "scope_denied").First(&event).Error; err != nil {
		t.Fatal(err)
	}
	if event.Actor != "api-user" || event.Resource != "observability" {
		t.Fatalf("unexpected audit event: %#v", event)
	}
}

func formatUnix(value int64) string {
	return strconv.FormatInt(value, 10)
}
