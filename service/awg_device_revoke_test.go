package service

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
)

func createAWGRevokeTestDevice(t *testing.T, clientID uint, publicKey, previousKey, state string) model.AWGDevice {
	t.Helper()
	device := model.AWGDevice{
		ClientId: clientID, Name: "Phone", CreateRequestKey: "create-revoke", CryptoContext: []byte{1},
		PublicKey: publicKey, PreviousPublicKey: previousKey, PrivateKeyEnc: []byte{2}, PSKEnc: []byte{3},
		IPv4Address: "10.77.0.2", DesiredEnabled: true, SyncState: state, Provisioned: true,
		CreatedAt: 1_699_999_000, UpdatedAt: 1_699_999_000,
	}
	if err := database.GetDB().Create(&device).Error; err != nil {
		t.Fatal(err)
	}
	return device
}

func TestAWGDeviceOwnershipReadsDoNotExposeForeignDevice(t *testing.T) {
	manager, _ := newAWGCreateTestManager(t, &fakeAWGProvisioner{})
	owner := createAWGEligibleClient(t)
	foreign := createAWGEligibleClient(t)
	device := createAWGRevokeTestDevice(t, owner.Id, "owner-public", "", "in_sync")

	listed, err := manager.ListDevices(owner.Id)
	if err != nil || len(listed) != 1 || listed[0].ID != device.Id {
		t.Fatalf("owner list = %#v, %v", listed, err)
	}
	foreignList, err := manager.ListDevices(foreign.Id)
	if err != nil || len(foreignList) != 0 {
		t.Fatalf("foreign list = %#v, %v", foreignList, err)
	}
	if _, err := manager.GetOwnedDevice(device.Id, owner.Id); err != nil {
		t.Fatalf("owner get: %v", err)
	}
	foreignErr := func() error { _, err := manager.GetOwnedDevice(device.Id, foreign.Id); return err }()
	missingErr := func() error { _, err := manager.GetOwnedDevice(device.Id+1000, owner.Id); return err }()
	if !errors.Is(foreignErr, ErrAWGDeviceNotFound) || !errors.Is(missingErr, ErrAWGDeviceNotFound) || foreignErr.Error() != missingErr.Error() {
		t.Fatalf("foreign error = %v, missing error = %v", foreignErr, missingErr)
	}
}

func TestAWGRevokePersistsIntentBeforeRemovingBothRotationKeys(t *testing.T) {
	var removed []string
	manager, _ := newAWGCreateTestManager(t, &fakeAWGProvisioner{
		remove: func(_ context.Context, key string) error {
			var stored model.AWGDevice
			if err := database.GetDB().Where("public_key = ?", "new-public").First(&stored).Error; err != nil {
				return err
			}
			if stored.DesiredEnabled || stored.SyncState != "pending_remove" {
				t.Fatalf("remove observed non-durable intent: %#v", stored)
			}
			removed = append(removed, key)
			return nil
		},
	})
	client := createAWGEligibleClient(t)
	device := createAWGRevokeTestDevice(t, client.Id, "new-public", "old-public", "pending_rotate")

	if err := manager.RevokeOwnedDevice(context.Background(), device.Id, client.Id); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(removed, []string{"old-public", "new-public"}) {
		t.Fatalf("removed keys = %#v", removed)
	}
	var stored model.AWGDevice
	if err := database.GetDB().First(&stored, device.Id).Error; err != nil {
		t.Fatal(err)
	}
	if stored.DesiredEnabled || stored.Provisioned || stored.SyncState != "in_sync" || stored.PreviousPublicKey != "" ||
		stored.RevokedAt != 1_700_000_000 || stored.IPReusableAfter != 1_700_086_400 || stored.LastError != "" {
		t.Fatalf("unexpected revoked state: %#v", stored)
	}
}

func TestAWGRevokeFailureRetainsPendingStateAndReservedIP(t *testing.T) {
	manager, _ := newAWGCreateTestManager(t, &fakeAWGProvisioner{
		remove: func(context.Context, string) error { return errors.New("uapi secret detail") },
	})
	client := createAWGEligibleClient(t)
	device := createAWGRevokeTestDevice(t, client.Id, "current-public", "", "in_sync")

	if err := manager.RevokeOwnedDevice(context.Background(), device.Id, client.Id); !errors.Is(err, ErrAWGRevocationFailed) {
		t.Fatalf("revoke error = %v", err)
	}
	var stored model.AWGDevice
	if err := database.GetDB().First(&stored, device.Id).Error; err != nil {
		t.Fatal(err)
	}
	if stored.DesiredEnabled || !stored.Provisioned || stored.SyncState != "pending_remove" ||
		stored.RevokedAt != 0 || stored.IPReusableAfter != 0 || stored.LastError != "AWG revocation failed" {
		t.Fatalf("unexpected pending removal: %#v", stored)
	}
	settings, _ := manager.deps.LoadSettings()
	endpoint, _ := manager.deps.LoadEndpoint(database.GetDB(), settings)
	address, err := AllocateAWGIPv4(database.GetDB(), settings.Subnet, endpoint.ServerAddress, 1_700_000_000)
	if err != nil || address.String() == device.IPv4Address {
		t.Fatalf("failed revoke IP was reused: address=%v err=%v", address, err)
	}
}

func TestAWGRevokeIsOwnedAndIdempotentWithoutExtendingQuarantine(t *testing.T) {
	var removeCalls int
	manager, _ := newAWGCreateTestManager(t, &fakeAWGProvisioner{
		remove: func(context.Context, string) error { removeCalls++; return nil },
	})
	owner := createAWGEligibleClient(t)
	foreign := createAWGEligibleClient(t)
	device := createAWGRevokeTestDevice(t, owner.Id, "owned-public", "", "in_sync")

	foreignErr := manager.RevokeOwnedDevice(context.Background(), device.Id, foreign.Id)
	missingErr := manager.RevokeOwnedDevice(context.Background(), device.Id+1000, owner.Id)
	if !errors.Is(foreignErr, ErrAWGDeviceNotFound) || !errors.Is(missingErr, ErrAWGDeviceNotFound) || foreignErr.Error() != missingErr.Error() {
		t.Fatalf("foreign error = %v, missing error = %v", foreignErr, missingErr)
	}
	if err := manager.RevokeOwnedDevice(context.Background(), device.Id, owner.Id); err != nil {
		t.Fatal(err)
	}
	manager.deps.Now = func() int64 { return 1_800_000_000 }
	if err := manager.RevokeOwnedDevice(context.Background(), device.Id, owner.Id); err != nil {
		t.Fatal(err)
	}
	var stored model.AWGDevice
	if err := database.GetDB().First(&stored, device.Id).Error; err != nil {
		t.Fatal(err)
	}
	if removeCalls != 1 || stored.RevokedAt != 1_700_000_000 || stored.IPReusableAfter != 1_700_086_400 {
		t.Fatalf("idempotent revoke extended state: calls=%d device=%#v", removeCalls, stored)
	}
}

func TestAWGRevokeDeduplicatesRotationKeys(t *testing.T) {
	var removed []string
	manager, _ := newAWGCreateTestManager(t, &fakeAWGProvisioner{
		remove: func(_ context.Context, key string) error { removed = append(removed, key); return nil },
	})
	client := createAWGEligibleClient(t)
	device := createAWGRevokeTestDevice(t, client.Id, "same-public", "same-public", "pending_rotate")

	if err := manager.RevokeOwnedDevice(context.Background(), device.Id, client.Id); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(removed, []string{"same-public"}) {
		t.Fatalf("removed keys = %#v", removed)
	}
}
