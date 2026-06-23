package migration

import (
	"errors"
	"testing"

	"github.com/deposist/s-ui-x-extended/database/model"

	"gorm.io/gorm"
)

// TestMigrateDnsNoConfigRowIsNoOp pins the M1 regression: a 1.2-era database that
// has no `config` settings row (managed purely via the entity UIs) must not abort
// the 1.2->1.3 migration. Previously migrate_dns used First() and returned
// gorm.ErrRecordNotFound, rolling back the whole migration/restore.
func TestMigrateDnsNoConfigRowIsNoOp(t *testing.T) {
	db := openMigrationTestDB(t)
	if err := db.AutoMigrate(&model.Setting{}); err != nil {
		t.Fatal(err)
	}
	// Only a version row exists; deliberately NO `config` row.
	if err := db.Create(&model.Setting{Key: "version", Value: "1.2"}).Error; err != nil {
		t.Fatal(err)
	}

	if err := migrate_dns(db); err != nil {
		t.Fatalf("migrate_dns aborted on a config-less DB: %v (want nil)", err)
	}
	if err := migrate_dns(db); err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("migrate_dns still surfaces ErrRecordNotFound on a missing config row")
	}
}

// TestTo13ConfigLessDatabaseSucceeds runs the full 1.2->1.3 step against a
// config-less database to ensure the missing-row no-op holds end-to-end.
func TestTo13ConfigLessDatabaseSucceeds(t *testing.T) {
	db := openMigrationTestDB(t)
	if err := db.AutoMigrate(&model.Setting{}, &model.Client{}, &model.Outbound{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.Setting{Key: "version", Value: "1.2"}).Error; err != nil {
		t.Fatal(err)
	}

	if err := to1_3(db); err != nil {
		t.Fatalf("to1_3 aborted on a config-less DB: %v (want nil)", err)
	}
}
