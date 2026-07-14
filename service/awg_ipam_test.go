package service

import (
	"errors"
	"net/netip"
	"testing"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"

	"gorm.io/gorm"
)

func initAWGIPAMTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	initSettingTestDB(t)
	db := database.GetDB()
	if err := db.AutoMigrate(&model.AWGDevice{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_awg_devices_active_ipv4 ON awg_devices(ipv4_address) WHERE desired_enabled = 1`).Error; err != nil {
		t.Fatal(err)
	}
	return db
}

func testAWGDevice(address string, reusableAfter int64) model.AWGDevice {
	return model.AWGDevice{
		ClientId: 1, Name: address, CryptoContext: []byte{1}, PublicKey: "key-" + address,
		PrivateKeyEnc: []byte{1}, PSKEnc: []byte{1}, IPv4Address: address,
		DesiredEnabled: true, SyncState: "in_sync", CreatedAt: 1, UpdatedAt: 1,
		IPReusableAfter: reusableAfter,
	}
}

func TestAllocateAWGIPv4SkipsReservedActiveAndQuarantined(t *testing.T) {
	db := initAWGIPAMTestDB(t)
	for _, device := range []model.AWGDevice{
		testAWGDevice("10.77.0.2", 0),
		testAWGDevice("10.77.0.3", 200),
		testAWGDevice("10.77.0.4", 50),
	} {
		if err := db.Create(&device).Error; err != nil {
			t.Fatal(err)
		}
		if device.IPReusableAfter > 0 {
			if err := db.Model(&device).Update("desired_enabled", false).Error; err != nil {
				t.Fatal(err)
			}
		}
	}

	err := db.Transaction(func(tx *gorm.DB) error {
		address, err := AllocateAWGIPv4(tx, netip.MustParsePrefix("10.77.0.0/29"), netip.MustParseAddr("10.77.0.1"), 100)
		if err != nil {
			return err
		}
		if address.String() != "10.77.0.4" {
			t.Fatalf("allocated %s; want reusable 10.77.0.4", address)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestAllocateAWGIPv4ReservesPeerPendingVerifiedRemoval(t *testing.T) {
	db := initAWGIPAMTestDB(t)
	device := testAWGDevice("10.77.0.2", 0)
	device.SyncState = "pending_remove"
	device.Provisioned = true
	if err := db.Create(&device).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&device).Update("desired_enabled", false).Error; err != nil {
		t.Fatal(err)
	}

	err := db.Transaction(func(tx *gorm.DB) error {
		address, err := AllocateAWGIPv4(tx, netip.MustParsePrefix("10.77.0.0/29"), netip.MustParseAddr("10.77.0.1"), 100)
		if err != nil {
			return err
		}
		if address.String() != "10.77.0.3" {
			t.Fatalf("allocated %s; want pending-remove peer address 10.77.0.2 reserved", address)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestAllocateAWGIPv4TransactionAndUniqueIndex(t *testing.T) {
	db := initAWGIPAMTestDB(t)
	allocated := make([]string, 0, 2)
	for index := 0; index < 2; index++ {
		err := db.Transaction(func(tx *gorm.DB) error {
			address, err := AllocateAWGIPv4(tx, netip.MustParsePrefix("10.77.0.0/29"), netip.MustParseAddr("10.77.0.1"), 100)
			if err != nil {
				return err
			}
			device := testAWGDevice(address.String(), 0)
			device.PublicKey = address.String()
			if err := tx.Create(&device).Error; err != nil {
				return err
			}
			allocated = append(allocated, address.String())
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if allocated[0] != "10.77.0.2" || allocated[1] != "10.77.0.3" {
		t.Fatalf("allocated addresses = %v", allocated)
	}
	duplicate := testAWGDevice("10.77.0.2", 0)
	duplicate.PublicKey = "different-key"
	if err := db.Create(&duplicate).Error; err == nil {
		t.Fatal("partial unique index accepted duplicate active address")
	}
}

func TestAllocateAWGIPv4Exhaustion(t *testing.T) {
	db := initAWGIPAMTestDB(t)
	device := testAWGDevice("10.77.0.2", 0)
	if err := db.Create(&device).Error; err != nil {
		t.Fatal(err)
	}
	err := db.Transaction(func(tx *gorm.DB) error {
		_, err := AllocateAWGIPv4(tx, netip.MustParsePrefix("10.77.0.0/30"), netip.MustParseAddr("10.77.0.1"), 100)
		return err
	})
	if !errors.Is(err, ErrAWGAddressPoolExhausted) {
		t.Fatalf("error = %v; want ErrAWGAddressPoolExhausted", err)
	}
}

func TestAllocateAWGIPv4RejectsInvalidPersistedAddress(t *testing.T) {
	db := initAWGIPAMTestDB(t)
	device := testAWGDevice("not-an-ip", 0)
	if err := db.Create(&device).Error; err != nil {
		t.Fatal(err)
	}
	err := db.Transaction(func(tx *gorm.DB) error {
		_, err := AllocateAWGIPv4(tx, netip.MustParsePrefix("10.77.0.0/29"), netip.MustParseAddr("10.77.0.1"), 100)
		return err
	})
	if err == nil || err.Error() != "invalid persisted AWG IPv4 address" {
		t.Fatalf("unexpected error: %v", err)
	}
}
