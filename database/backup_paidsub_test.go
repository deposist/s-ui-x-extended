package database

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x-extended/database/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestBackupIncludesPaidSubAndAWGState(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "s-ui.db")
	t.Setenv("SUI_DB_FOLDER", filepath.Dir(dbPath))
	if err := InitDB(dbPath); err != nil {
		if strings.Contains(err.Error(), "go-sqlite3 requires cgo") {
			t.Skip(err)
		}
		t.Fatal(err)
	}
	t.Cleanup(func() {
		closeMainDB(t)
		cleanupBackupSidecars(dbPath)
	})

	db := GetDB()
	models := []any{&model.PaidSubBinding{}, &model.PaidSubTariff{}, &model.PaidSubPaymentOrder{}, &model.AWGDevice{}}
	if err := db.AutoMigrate(models...); err != nil {
		t.Fatal(err)
	}
	binding := model.PaidSubBinding{ClientId: 11, TgUserId: 22}
	tariff := model.PaidSubTariff{Name: "AWG", Currency: "RUB", MaxAWGDevices: 4}
	order := model.PaidSubPaymentOrder{ClientId: 11, TariffId: 1, Provider: "test", Currency: "RUB", Status: "paid", IdempotencyKey: "backup-order", GrantedAWGDevices: 4}
	device := model.AWGDevice{
		ClientId: 11, Name: "phone", CreateRequestKey: "telegram-update-100", RotateRequestKey: "telegram-update-200", CryptoContext: []byte("context"), PublicKey: "public",
		PrivateKeyEnc: []byte("encrypted-private"), PSKEnc: []byte("encrypted-psk"),
		IPv4Address: "10.77.0.2", DesiredEnabled: true, SyncState: "in_sync", CreatedAt: 1, UpdatedAt: 1,
	}
	for _, value := range []any{&binding, &tariff, &order, &device} {
		if err := db.Create(value).Error; err != nil {
			t.Fatal(err)
		}
	}

	backupPath, cleanup, err := PrepareDbBackup("")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	backupDB, err := gorm.Open(sqlite.Open(backupPath), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if sqlDB, dbErr := backupDB.DB(); dbErr == nil {
			_ = sqlDB.Close()
		}
	})

	for _, value := range []any{&model.PaidSubBinding{}, &model.PaidSubTariff{}, &model.PaidSubPaymentOrder{}, &model.AWGDevice{}} {
		var count int64
		if err := backupDB.Model(value).Count(&count).Error; err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatalf("backup count for %T = %d; want 1", value, count)
		}
	}
	var restored model.AWGDevice
	if err := backupDB.First(&restored).Error; err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(restored.PrivateKeyEnc, device.PrivateKeyEnc) || !bytes.Equal(restored.PSKEnc, device.PSKEnc) {
		t.Fatal("encrypted AWG key material was not preserved")
	}
	if restored.CreateRequestKey != device.CreateRequestKey {
		t.Fatalf("AWG create request key = %q; want %q", restored.CreateRequestKey, device.CreateRequestKey)
	}
	if restored.RotateRequestKey != device.RotateRequestKey {
		t.Fatalf("AWG rotate request key = %q; want %q", restored.RotateRequestKey, device.RotateRequestKey)
	}
}

func TestBackupCreatesEmptyOptionalPaidSubTables(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "s-ui.db")
	t.Setenv("SUI_DB_FOLDER", filepath.Dir(dbPath))
	if err := InitDB(dbPath); err != nil {
		if strings.Contains(err.Error(), "go-sqlite3 requires cgo") {
			t.Skip(err)
		}
		t.Fatal(err)
	}
	t.Cleanup(func() {
		closeMainDB(t)
		cleanupBackupSidecars(dbPath)
	})

	backupPath, cleanup, err := PrepareDbBackup("")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	backupDB, err := gorm.Open(sqlite.Open(backupPath), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"paidsub_bindings", "tariffs", "payment_orders", "awg_devices"} {
		if !backupDB.Migrator().HasTable(table) {
			t.Fatalf("optional backup table %q is missing", table)
		}
	}
	if _, err := os.Stat(backupPath); err != nil {
		t.Fatal(err)
	}
}
