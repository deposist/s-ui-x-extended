package migration

import (
	"encoding/json"
	"testing"

	"github.com/deposist/s-ui-x-extended/database/model"
)

func TestTo13AcceptsNullClientConfig(t *testing.T) {
	db := openMigrationTestDB(t)
	if err := db.AutoMigrate(&model.Setting{}, &model.Client{}, &model.Outbound{}); err != nil {
		t.Fatal(err)
	}
	client := model.Client{Name: "null-config", Config: json.RawMessage(`null`)}
	if err := db.Create(&client).Error; err != nil {
		t.Fatal(err)
	}
	if err := to1_3(db); err != nil {
		t.Fatalf("1.3 migration rejected null client config: %v", err)
	}
	var migrated model.Client
	if err := db.First(&migrated, client.Id).Error; err != nil {
		t.Fatal(err)
	}
	var config map[string]json.RawMessage
	if err := json.Unmarshal(migrated.Config, &config); err != nil || config == nil {
		t.Fatalf("migrated client config is not an object: %s (%v)", migrated.Config, err)
	}
}
