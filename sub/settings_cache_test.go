package sub

import (
	"testing"
	"time"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
	"github.com/deposist/s-ui-x/service"
)

func setSubTitle(t *testing.T, value string) {
	t.Helper()
	if err := database.GetDB().Model(model.Setting{}).Where("key = ?", "subTitle").Update("value", value).Error; err != nil {
		t.Fatal(err)
	}
}

// TestCachedSubDisplaySettingsReusesSnapshotWithinTTL proves the O2 cache serves
// a snapshot for subDisplaySettingsTTL before re-reading the database, so the hot
// subscription path does not issue the display-setting SELECTs on every request.
func TestCachedSubDisplaySettingsReusesSnapshotWithinTTL(t *testing.T) {
	initSubTestDB(t)
	ss := &service.SettingService{}
	if _, err := ss.GetAllSetting(); err != nil {
		t.Fatal(err)
	}

	setSubTitle(t, "first")
	base := time.Unix(1_700_000_000, 0)
	if got := cachedSubDisplaySettings(ss, base); got.title != "first" {
		t.Fatalf("cold read title = %q, want %q", got.title, "first")
	}

	// A change within the TTL window must not be observed: the snapshot is reused.
	setSubTitle(t, "second")
	if got := cachedSubDisplaySettings(ss, base.Add(subDisplaySettingsTTL-time.Second)); got.title != "first" {
		t.Fatalf("within-TTL read title = %q, want cached %q", got.title, "first")
	}

	// Once the TTL elapses the database is read again and the new value appears.
	if got := cachedSubDisplaySettings(ss, base.Add(subDisplaySettingsTTL+time.Second)); got.title != "second" {
		t.Fatalf("post-TTL read title = %q, want refreshed %q", got.title, "second")
	}
}

// TestResetSubDisplaySettingsCacheForcesReread guards the test-isolation hook: a
// reset must discard the snapshot so a later read re-queries the database even
// within the TTL window (this is what initSubTestDB relies on between tests).
func TestResetSubDisplaySettingsCacheForcesReread(t *testing.T) {
	initSubTestDB(t)
	ss := &service.SettingService{}
	if _, err := ss.GetAllSetting(); err != nil {
		t.Fatal(err)
	}

	setSubTitle(t, "alpha")
	base := time.Unix(1_700_000_000, 0)
	if got := cachedSubDisplaySettings(ss, base); got.title != "alpha" {
		t.Fatalf("cold read title = %q, want %q", got.title, "alpha")
	}

	setSubTitle(t, "beta")
	resetSubDisplaySettingsCacheForTest()
	if got := cachedSubDisplaySettings(ss, base); got.title != "beta" {
		t.Fatalf("after reset title = %q, want re-read %q", got.title, "beta")
	}
}
