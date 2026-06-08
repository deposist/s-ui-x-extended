package database

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x/database/model"
	"github.com/deposist/s-ui-x/util/common"
)

// TestAdaptRehashesLegacyPlaintextPassword simulates an imported legacy backup
// where the admin password is still plaintext, runs the post-migration
// adaptation, and asserts that the stored password is now a bcrypt hash that
// still validates the original plaintext.
func TestAdaptRehashesLegacyPlaintextPassword(t *testing.T) {
	dbDir := makeDBTempDir(t, "s-ui-adapt-test-")
	t.Setenv("SUI_DB_FOLDER", dbDir)
	if err := InitDB(filepath.Join(dbDir, "s-ui.db")); err != nil {
		if strings.Contains(err.Error(), "go-sqlite3 requires cgo") {
			t.Skip(err)
		}
		t.Fatal(err)
	}
	t.Cleanup(func() { closeMainDB(t) })

	// Replace the auto-generated admin password with a plaintext value, as
	// it would appear in a backup made against an older S-UI version.
	d := GetDB()
	if err := d.Model(&model.User{}).Where("username = ?", "admin").Update("password", "legacy-plaintext").Error; err != nil {
		t.Fatal(err)
	}

	if err := AdaptToCurrentVersion(); err != nil {
		t.Fatal(err)
	}

	var stored string
	if err := d.Model(&model.User{}).Select("password").Where("username = ?", "admin").Scan(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if !common.IsPasswordHash(stored) {
		t.Fatalf("password was not migrated, still plaintext: %q", stored)
	}
	ok, _ := common.CheckPassword(stored, "legacy-plaintext")
	if !ok {
		t.Fatal("rehashed password no longer validates the original plaintext")
	}
	// A second adapt run must be a no-op and must not double-hash.
	if err := AdaptToCurrentVersion(); err != nil {
		t.Fatal(err)
	}
	var second string
	if err := d.Model(&model.User{}).Select("password").Where("username = ?", "admin").Scan(&second).Error; err != nil {
		t.Fatal(err)
	}
	if second != stored {
		t.Fatal("AdaptToCurrentVersion is not idempotent; password changed on second run")
	}
}

// TestAdaptBumpsVersionSetting asserts the settings.version row is upgraded
// to the current build version regardless of whether it was missing or stale.
func TestAdaptBumpsVersionSetting(t *testing.T) {
	dbDir := makeDBTempDir(t, "s-ui-adapt-test-")
	t.Setenv("SUI_DB_FOLDER", dbDir)
	if err := InitDB(filepath.Join(dbDir, "s-ui.db")); err != nil {
		if strings.Contains(err.Error(), "go-sqlite3 requires cgo") {
			t.Skip(err)
		}
		t.Fatal(err)
	}
	t.Cleanup(func() { closeMainDB(t) })

	d := GetDB()
	// Pretend the imported backup carries a stale version pin.
	if err := d.Model(&model.Setting{}).Where("key = ?", "version").Update("value", "1.0.0").Error; err != nil {
		t.Fatal(err)
	}
	if err := AdaptToCurrentVersion(); err != nil {
		t.Fatal(err)
	}
	var version string
	if err := d.Model(&model.Setting{}).Select("value").Where("key = ?", "version").Scan(&version).Error; err != nil {
		t.Fatal(err)
	}
	if version == "1.0.0" || version == "" {
		t.Fatalf("version was not bumped: %q", version)
	}
}

func TestAdaptDoesNotDowngradeFutureVersionSetting(t *testing.T) {
	dbDir := makeDBTempDir(t, "s-ui-adapt-test-")
	t.Setenv("SUI_DB_FOLDER", dbDir)
	if err := InitDB(filepath.Join(dbDir, "s-ui.db")); err != nil {
		if strings.Contains(err.Error(), "go-sqlite3 requires cgo") {
			t.Skip(err)
		}
		t.Fatal(err)
	}
	t.Cleanup(func() { closeMainDB(t) })

	d := GetDB()
	const futureVersion = "99.0.0"
	if err := d.Model(&model.Setting{}).Where("key = ?", "version").Update("value", futureVersion).Error; err != nil {
		t.Fatal(err)
	}
	if err := AdaptToCurrentVersion(); err != nil {
		t.Fatal(err)
	}
	var version string
	if err := d.Model(&model.Setting{}).Select("value").Where("key = ?", "version").Scan(&version).Error; err != nil {
		t.Fatal(err)
	}
	if version != futureVersion {
		t.Fatalf("future version was downgraded: got %q want %q", version, futureVersion)
	}
}

func TestCompareVersionUsesSharedSemverPolicy(t *testing.T) {
	cases := []struct {
		left  string
		right string
		want  int
	}{
		{left: "v1.5.2-beta-hotfix2", right: "1.5.2", want: -1},
		{left: "1.6.0", right: "1.5.9", want: 1},
		{left: "1.4.9", right: "1.5.0", want: -1},
	}
	for _, tc := range cases {
		got, ok := compareVersion(tc.left, tc.right)
		if !ok || got != tc.want {
			t.Fatalf("compareVersion(%q, %q) = %d, %v; want %d, true", tc.left, tc.right, got, ok, tc.want)
		}
	}
}
