package migration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/deposist/s-ui-x-extended/database/model"
)

func TestMoveJSONToDBOptionalSections(t *testing.T) {
	for _, configJSON := range []string{`{}`, `{"inbounds":null,"outbounds":null,"experimental":null}`, `{"inbounds":[],"outbounds":[]}`} {
		t.Run(configJSON, func(t *testing.T) {
			db := openMigrationTestDB(t)
			if err := db.AutoMigrate(&model.Setting{}); err != nil {
				t.Fatal(err)
			}
			writeLegacyMigrationConfig(t, configJSON)
			if err := moveJsonToDb(db); err != nil {
				t.Fatalf("valid optional sections: %v", err)
			}
		})
	}
}

func TestMoveJSONToDBRejectsInvalidSectionsBeforeReplacingTables(t *testing.T) {
	for _, configJSON := range []string{
		`null`, `{"inbounds":{}}`, `{"inbounds":[null]}`, `{"outbounds":{}}`, `{"outbounds":[1]}`,
		`{"experimental":[]}`, `{"route":[]}`, `{"route":{"rules":{}}}`, `{"route":{"rules":[null]}}`,
	} {
		t.Run(configJSON, func(t *testing.T) {
			db := openMigrationTestDB(t)
			if err := db.AutoMigrate(&model.Inbound{}); err != nil {
				t.Fatal(err)
			}
			if err := db.Create(&model.Inbound{Tag: "keep", Type: "vless"}).Error; err != nil {
				t.Fatal(err)
			}
			writeLegacyMigrationConfig(t, configJSON)
			if err := moveJsonToDb(db); err == nil {
				t.Fatal("invalid legacy config was accepted")
			}
			var count int64
			if err := db.Model(&model.Inbound{}).Where("tag = ?", "keep").Count(&count).Error; err != nil || count != 1 {
				t.Fatalf("existing inbound was lost: count=%d err=%v", count, err)
			}
		})
	}
}

func writeLegacyMigrationConfig(t *testing.T, configJSON string) {
	t.Helper()
	binDir := t.TempDir()
	t.Setenv("SUI_BIN_FOLDER", binDir)
	// moveJsonToDb resolves the configured folder relative to the executable.
	oldArgs := os.Args
	os.Args = []string{filepath.Join(filepath.Dir(binDir), "sui")}
	t.Cleanup(func() { os.Args = oldArgs })
	t.Setenv("SUI_BIN_FOLDER", filepath.Base(binDir))
	if err := os.WriteFile(filepath.Join(binDir, "config.json"), []byte(configJSON), 0o600); err != nil {
		t.Fatal(err)
	}
}
