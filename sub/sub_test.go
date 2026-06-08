package sub

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
	"github.com/deposist/s-ui-x/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var subUUIDV4Pattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func initSubTestDB(t *testing.T) {
	t.Helper()
	resetSubDisplaySettingsCacheForTest()
	tempDir := t.TempDir()
	t.Setenv("SUI_DB_FOLDER", tempDir)
	closeSubTestDB(database.GetDB())
	if err := database.InitDB(filepath.Join(tempDir, "s-ui.db")); err != nil {
		if strings.Contains(err.Error(), "go-sqlite3 requires cgo") {
			t.Skip(err)
		}
		t.Fatal(err)
	}
	testDB := database.GetDB()
	t.Cleanup(func() {
		closeSubTestDB(testDB)
	})
}

func closeSubTestDB(db *gorm.DB) {
	if db == nil {
		return
	}
	if sqlDB, err := db.DB(); err == nil {
		_ = sqlDB.Close()
		time.Sleep(25 * time.Millisecond)
	}
}

func TestGetClientBySubIdPrefersSecretAndSupportsLegacyName(t *testing.T) {
	initSubTestDB(t)
	if _, err := (&service.SettingService{}).GetAllSetting(); err != nil {
		t.Fatal(err)
	}
	client := model.Client{
		Enable:    true,
		Name:      "legacy-name",
		SubSecret: "secret-id",
		Inbounds:  []byte("[]"),
		Links:     []byte("[]"),
	}
	if err := database.GetDB().Create(&client).Error; err != nil {
		t.Fatal(err)
	}

	subService := &SubService{}
	bySecret, err := subService.getClientBySubId("secret-id")
	if err != nil {
		t.Fatal(err)
	}
	if bySecret.Name != "legacy-name" {
		t.Fatalf("unexpected secret lookup client: %#v", bySecret)
	}

	byName, err := subService.getClientBySubId("legacy-name")
	if err != nil {
		t.Fatal(err)
	}
	if byName.SubSecret != "secret-id" {
		t.Fatalf("legacy lookup did not return expected client: %#v", byName)
	}
}

func TestGetClientBySubIdCanDisableLegacyName(t *testing.T) {
	initSubTestDB(t)
	settingService := &service.SettingService{}
	if _, err := settingService.GetAllSetting(); err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Model(model.Setting{}).Where("key = ?", "subSecretRequired").Update("value", "true").Error; err != nil {
		t.Fatal(err)
	}
	client := model.Client{
		Enable:    true,
		Name:      "legacy-name",
		SubSecret: "secret-id",
		Inbounds:  []byte("[]"),
		Links:     []byte("[]"),
	}
	if err := database.GetDB().Create(&client).Error; err != nil {
		t.Fatal(err)
	}

	subService := &SubService{}
	if _, err := subService.getClientBySubId("legacy-name"); err == nil {
		t.Fatal("legacy name lookup should be disabled when subSecretRequired=true")
	}
	if _, err := subService.getClientBySubId("secret-id"); err != nil {
		t.Fatalf("secret lookup should still work: %v", err)
	}
}

func TestEnsureClientSubSecretGeneratesUUIDV4(t *testing.T) {
	initSubTestDB(t)
	if _, err := (&service.SettingService{}).GetAllSetting(); err != nil {
		t.Fatal(err)
	}
	client := model.Client{
		Enable:   true,
		Name:     "legacy-name",
		Inbounds: []byte("[]"),
		Links:    []byte("[]"),
	}
	if err := database.GetDB().Create(&client).Error; err != nil {
		t.Fatal(err)
	}

	if err := (&SubService{}).ensureClientSubSecret(database.GetDB(), &client); err != nil {
		t.Fatal(err)
	}
	if !subUUIDV4Pattern.MatchString(client.SubSecret) {
		t.Fatalf("sub secret is not uuid-v4: %q", client.SubSecret)
	}

	var stored model.Client
	if err := database.GetDB().Where("id = ?", client.Id).First(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if stored.SubSecret != client.SubSecret {
		t.Fatalf("sub secret was not persisted: %#v", stored)
	}
}

func TestSubSecretRequiredReturns404ForLegacyNameURL(t *testing.T) {
	initSubTestDB(t)
	resetRateLimitBucketsForTest()
	settingService := &service.SettingService{}
	if _, err := settingService.GetAllSetting(); err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Model(model.Setting{}).Where("key = ?", "subSecretRequired").Update("value", "true").Error; err != nil {
		t.Fatal(err)
	}
	client := model.Client{
		Enable:    true,
		Name:      "legacy-name",
		SubSecret: "secret-id",
		Inbounds:  []byte("[]"),
		Links:     []byte("[]"),
	}
	if err := database.GetDB().Create(&client).Error; err != nil {
		t.Fatal(err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewSubHandler(router.Group(""))

	legacyRecorder := httptest.NewRecorder()
	router.ServeHTTP(legacyRecorder, httptest.NewRequest(http.MethodGet, "/legacy-name", nil))
	if legacyRecorder.Code != http.StatusNotFound {
		t.Fatalf("legacy name URL should be hidden, got %d", legacyRecorder.Code)
	}

	secretRecorder := httptest.NewRecorder()
	router.ServeHTTP(secretRecorder, httptest.NewRequest(http.MethodGet, "/secret-id", nil))
	if secretRecorder.Code != http.StatusOK {
		t.Fatalf("secret URL should still work, got %d", secretRecorder.Code)
	}
}

func TestSafeSubscriptionHeadersRemovesControlCharacters(t *testing.T) {
	got := safeSubscriptionHeaders([]string{"ok\r\nInjected: bad"})[0]
	if strings.ContainsAny(got, "\r\n") {
		t.Fatalf("header was not sanitized: %q", got)
	}
}

func TestSubscriptionHeadersUseConfiguredTitleAndURLs(t *testing.T) {
	initSubTestDB(t)
	settingService := &service.SettingService{}
	if _, err := settingService.GetAllSetting(); err != nil {
		t.Fatal(err)
	}
	settings := map[string]string{
		"subTitle":      "Panel\r\nInjected: bad",
		"subSupportUrl": "https://example.com/support",
		"subProfileUrl": "https://example.com/profile",
		"subAnnounce":   "Maintenance\r\nInjected: bad",
	}
	for key, value := range settings {
		if err := database.GetDB().Model(model.Setting{}).Where("key = ?", key).Update("value", value).Error; err != nil {
			t.Fatal(err)
		}
	}

	cfg := cachedSubDisplaySettings(&service.SettingService{}, time.Now())
	headers := buildClientHeaders(&model.Client{Name: "alice"}, cfg)
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/sub/alice", nil)
	(&SubHandler{}).addHeaders(c, headers)

	if title := recorder.Header().Get("Profile-Title"); strings.ContainsAny(title, "\r\n") || !strings.Contains(title, "Panel") {
		t.Fatalf("unexpected sanitized title: %q", title)
	}
	if recorder.Header().Get("Support-Url") != "https://example.com/support" {
		t.Fatalf("support URL header missing: %#v", recorder.Header())
	}
	if recorder.Header().Get("Profile-Web-Page-Url") != "https://example.com/profile" {
		t.Fatalf("profile URL header missing: %#v", recorder.Header())
	}
	if announce := recorder.Header().Get("Profile-Announcement"); strings.ContainsAny(announce, "\r\n") || !strings.Contains(announce, "Maintenance") {
		t.Fatalf("unexpected sanitized announce: %q", announce)
	}
}

func TestSubscriptionEnableSettingsDisableFormats(t *testing.T) {
	initSubTestDB(t)
	settingService := &service.SettingService{}
	if _, err := settingService.GetAllSetting(); err != nil {
		t.Fatal(err)
	}
	for key, value := range map[string]string{
		"subLinkEnable":  "false",
		"subJsonEnable":  "false",
		"subClashEnable": "false",
	} {
		if err := database.GetDB().Model(model.Setting{}).Where("key = ?", key).Update("value", value).Error; err != nil {
			t.Fatal(err)
		}
	}

	links := json.RawMessage(`[{"type":"external","uri":"https://example.com/sub"}]`)
	if got := (&LinkService{}).GetLinks(&links, "all", ""); len(got) != 0 {
		t.Fatalf("link subscriptions should be disabled, got %#v", got)
	}
	if _, _, err := (&JsonService{}).GetJson("missing", "json"); err == nil {
		t.Fatal("json subscription should be disabled before client lookup")
	}
	if _, _, err := (&ClashService{}).GetClash("missing"); err == nil {
		t.Fatal("clash subscription should be disabled before client lookup")
	}
}

func TestSubServerServesDefaultAndCustomFormatPaths(t *testing.T) {
	initSubTestDB(t)
	resetRateLimitBucketsForTest()
	settingService := &service.SettingService{}
	if _, err := settingService.GetAllSetting(); err != nil {
		t.Fatal(err)
	}
	for key, value := range map[string]string{
		"subJsonPath":  "/sing-json/",
		"subClashPath": "/sing-clash/",
	} {
		if err := database.GetDB().Model(model.Setting{}).Where("key = ?", key).Update("value", value).Error; err != nil {
			t.Fatal(err)
		}
	}
	client := model.Client{
		Enable:    true,
		Name:      "alice",
		SubSecret: "secret-id",
		Config:    json.RawMessage(`{}`),
		Inbounds:  json.RawMessage(`[]`),
		Links:     json.RawMessage(`[]`),
	}
	if err := database.GetDB().Create(&client).Error; err != nil {
		t.Fatal(err)
	}

	server := NewServer()
	router, err := server.initRouter()
	if err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{"/json/secret-id", "/sing-json/secret-id"} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		if recorder.Code != http.StatusOK {
			t.Fatalf("%s returned %d", path, recorder.Code)
		}
		if !strings.Contains(recorder.Body.String(), `"outbounds"`) {
			t.Fatalf("%s did not return JSON subscription: %s", path, recorder.Body.String())
		}
	}

	for _, path := range []string{"/clash/secret-id", "/sing-clash/secret-id"} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		if recorder.Code != http.StatusOK {
			t.Fatalf("%s returned %d", path, recorder.Code)
		}
		if !strings.Contains(recorder.Body.String(), "proxy-groups:") {
			t.Fatalf("%s did not return Clash subscription: %s", path, recorder.Body.String())
		}
	}
}

func TestSubHandlerLinkDisableReturns404ForBaseSubscription(t *testing.T) {
	initSubTestDB(t)
	resetRateLimitBucketsForTest()
	settingService := &service.SettingService{}
	if _, err := settingService.GetAllSetting(); err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Model(model.Setting{}).Where("key = ?", "subLinkEnable").Update("value", "false").Error; err != nil {
		t.Fatal(err)
	}
	client := model.Client{
		Enable:    true,
		Name:      "alice",
		SubSecret: "secret-id",
		Config:    json.RawMessage(`{}`),
		Inbounds:  json.RawMessage(`[]`),
		Links:     json.RawMessage(`[]`),
	}
	if err := database.GetDB().Create(&client).Error; err != nil {
		t.Fatal(err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewSubHandler(router.Group("/sub"))

	for _, method := range []string{http.MethodGet, http.MethodHead} {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(method, "/sub/secret-id", nil)
		router.ServeHTTP(recorder, req)
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("%s base subscription should be hidden, got %d", method, recorder.Code)
		}
	}

	jsonRecorder := httptest.NewRecorder()
	router.ServeHTTP(jsonRecorder, httptest.NewRequest(http.MethodGet, "/sub/secret-id?format=json", nil))
	if jsonRecorder.Code != http.StatusOK {
		t.Fatalf("json format should use subJsonEnable, got %d", jsonRecorder.Code)
	}
}
