package database

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/deposist/s-ui-x-extended/database/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestBackupPreservesFleetAssignmentsAndPaymentCharges(t *testing.T) {
	dir := makeDBTempDir(t, "sui-fleet-backup-")
	t.Setenv("SUI_DB_FOLDER", dir)
	if err := InitDB(filepath.Join(dir, "s-ui.db")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { closeMainDB(t) })
	live := GetDB()
	if err := live.AutoMigrate(&model.PaidSubPaymentCharge{}); err != nil {
		t.Fatal(err)
	}
	for _, record := range []any{
		&model.Node{UUID: "11111111-1111-4111-8111-111111111111", Name: "DE", Enabled: true},
		&model.ClientNode{ClientId: 42, NodeUUID: "11111111-1111-4111-8111-111111111111"},
		&model.ClientLimits{ClientId: 42, SpeedShared: true, Speed: 12345, Connections: 9},
		&model.FleetNodeOwnership{NodeUUID: "11111111-1111-4111-8111-111111111111", SquadID: 7},
		&model.PaidSubPaymentCharge{Provider: "test", ChargeID: "charge-42", OrderID: 42, Disposition: "granted", ConfirmedAt: 123},
	} {
		if err := live.Create(record).Error; err != nil {
			t.Fatal(err)
		}
	}
	path, cleanup, err := PrepareDbBackup("")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	backup, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := backup.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	for _, record := range []any{&model.Node{}, &model.ClientNode{}, &model.ClientLimits{}, &model.FleetNodeOwnership{}, &model.PaidSubPaymentCharge{}} {
		var count int64
		if err := backup.Model(record).Count(&count).Error; err != nil {
			t.Errorf("read backed up %T: %v", record, err)
			continue
		}
		if count != 1 {
			t.Errorf("backup contains %d %T rows, want 1", count, record)
		}
	}
	var limits model.ClientLimits
	if err := backup.First(&limits, "client_id = ?", 42).Error; err == nil && (limits.Speed != 12345 || limits.Connections != 9) {
		t.Fatalf("limits changed during backup: %+v", limits)
	}
}

func TestBackupPreservesEveryCompositeKeyPaymentCharge(t *testing.T) {
	dir := makeDBTempDir(t, "sui-charge-backup-")
	t.Setenv("SUI_DB_FOLDER", dir)
	if err := InitDB(filepath.Join(dir, "s-ui.db")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { closeMainDB(t) })
	live := GetDB()
	if err := live.AutoMigrate(&model.PaidSubPaymentCharge{}); err != nil {
		t.Fatal(err)
	}
	charges := make([]model.PaidSubPaymentCharge, 250)
	for index := range charges {
		charges[index] = model.PaidSubPaymentCharge{Provider: "test", ChargeID: fmt.Sprintf("charge-%03d", index), OrderID: uint(index + 1), Disposition: "granted"}
	}
	if err := CreateInBatchesSafe(live, &charges); err != nil {
		t.Fatal(err)
	}
	path, cleanup, err := PrepareDbBackup("")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	backup, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := backup.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	var count int64
	if err := backup.Model(&model.PaidSubPaymentCharge{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 250 {
		t.Fatalf("backup preserved %d payment charges, want all 250", count)
	}
}

func TestPrepareDbBackupReturnsCopyErrorWithoutPanicking(t *testing.T) {
	dir := makeDBTempDir(t, "sui-backup-error-")
	t.Setenv("SUI_DB_FOLDER", dir)
	if err := InitDB(filepath.Join(dir, "s-ui.db")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { closeMainDB(t) })
	if err := GetDB().Migrator().DropTable(&model.Stats{}); err != nil {
		t.Fatal(err)
	}
	_, _, err := PrepareDbBackup("")
	if err == nil {
		t.Fatal("missing required source table must return an error")
	}
}
