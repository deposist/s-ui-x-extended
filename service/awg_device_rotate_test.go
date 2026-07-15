package service

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
)

func newAWGRotateFixture(t *testing.T, provisioner *fakeAWGProvisioner) (*AWGManager, model.Client, AWGDeviceInfo, *recordingAWGCipher) {
	t.Helper()
	var creating bool
	originalAdd := provisioner.add
	provisioner.add = func(ctx context.Context, peer AWGPeerSpec) error {
		if creating {
			return nil
		}
		return originalAdd(ctx, peer)
	}
	manager, cipher := newAWGCreateTestManager(t, provisioner)
	client := createAWGEligibleClient(t)
	creating = true
	device, err := manager.CreateDevice(context.Background(), client.Id, "create-rotate", "Phone", 1, 0)
	creating = false
	if err != nil {
		t.Fatal(err)
	}
	manager.deps.GenerateKeys = func() (AWGGeneratedKeys, error) {
		return AWGGeneratedKeys{
			PrivateKey: bytes.Repeat([]byte{0x55}, 32),
			PublicKey:  bytes.Repeat([]byte{0x66}, 32),
			PSK:        bytes.Repeat([]byte{0x77}, 32),
		}, nil
	}
	manager.deps.NewCryptoContext = func() ([]byte, error) {
		return bytes.Repeat([]byte{0x88}, awgCryptoContextSize), nil
	}
	cipher.plaintext = nil
	return manager, client, device, cipher
}

func TestAWGRotateOwnedDevicePersistsBeforeFailClosedProvisioning(t *testing.T) {
	var order []string
	provisioner := &fakeAWGProvisioner{}
	manager, client, old, cipher := newAWGRotateFixture(t, provisioner)
	provisioner.remove = func(_ context.Context, publicKey string) error {
		var stored model.AWGDevice
		if err := database.GetDB().First(&stored, old.ID).Error; err != nil {
			t.Fatal(err)
		}
		if stored.SyncState != "pending_rotate" || stored.Provisioned || stored.PreviousPublicKey != old.PublicKey || stored.PublicKey == old.PublicKey {
			t.Fatalf("remove observed non-durable rotation state: %#v", stored)
		}
		order = append(order, "remove:"+publicKey)
		return nil
	}
	provisioner.add = func(_ context.Context, peer AWGPeerSpec) error {
		order = append(order, "add:"+peer.PublicKey)
		return nil
	}

	got, err := manager.RotateOwnedDevice(context.Background(), old.ID, client.Id, " rotate-1 ")
	if err != nil {
		t.Fatal(err)
	}
	if reflect.DeepEqual(got, old) || got.SyncState != "in_sync" || !got.Provisioned {
		t.Fatalf("unexpected rotated device: %#v", got)
	}
	if !reflect.DeepEqual(order, []string{"remove:" + old.PublicKey, "add:" + got.PublicKey}) {
		t.Fatalf("provisioning order = %#v", order)
	}
	if len(cipher.plaintext) != 2 || !bytes.Equal(cipher.plaintext[0], bytes.Repeat([]byte{0x55}, 32)) || !bytes.Equal(cipher.plaintext[1], bytes.Repeat([]byte{0x77}, 32)) {
		t.Fatalf("rotation did not encrypt new private key and PSK: %#v", cipher.plaintext)
	}
	var stored model.AWGDevice
	if err := database.GetDB().First(&stored, old.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.PreviousPublicKey != "" || stored.RotateRequestKey != "rotate-1" || stored.SyncState != "in_sync" || !stored.Provisioned || stored.RxBaseline != 0 || stored.TxBaseline != 0 {
		t.Fatalf("unexpected finalized rotation: %#v", stored)
	}
}

func TestAWGRotateOwnedDeviceRemoveFailureDoesNotAddOrReturnDevice(t *testing.T) {
	secret := "secret-uapi-payload"
	var addCalls int
	provisioner := &fakeAWGProvisioner{
		remove: func(context.Context, string) error { return errors.New(secret) },
		add:    func(context.Context, AWGPeerSpec) error { addCalls++; return nil },
	}
	manager, client, old, _ := newAWGRotateFixture(t, provisioner)

	got, err := manager.RotateOwnedDevice(context.Background(), old.ID, client.Id, "rotate-remove-fail")
	if !errors.Is(err, ErrAWGRotationFailed) || got.ID != 0 || addCalls != 0 {
		t.Fatalf("got=%#v err=%v addCalls=%d", got, err, addCalls)
	}
	var stored model.AWGDevice
	if dbErr := database.GetDB().First(&stored, old.ID).Error; dbErr != nil {
		t.Fatal(dbErr)
	}
	if stored.SyncState != "pending_rotate" || stored.Provisioned || stored.PreviousPublicKey != old.PublicKey || stored.LastError != "AWG rotation failed" || bytes.Contains([]byte(stored.LastError), []byte(secret)) {
		t.Fatalf("unexpected remove failure state: %#v", stored)
	}
}

func TestAWGRotateOwnedDeviceAddFailureLeavesOldAbsentAndReplayReusesKeys(t *testing.T) {
	var generateCalls, removeCalls, addCalls int
	provisioner := &fakeAWGProvisioner{
		remove: func(context.Context, string) error { removeCalls++; return nil },
		add: func(context.Context, AWGPeerSpec) error {
			addCalls++
			if addCalls == 1 {
				return errors.New("temporary add failure")
			}
			return nil
		},
	}
	manager, client, old, _ := newAWGRotateFixture(t, provisioner)
	originalGenerate := manager.deps.GenerateKeys
	manager.deps.GenerateKeys = func() (AWGGeneratedKeys, error) {
		generateCalls++
		return originalGenerate()
	}

	if got, err := manager.RotateOwnedDevice(context.Background(), old.ID, client.Id, "rotate-retry"); !errors.Is(err, ErrAWGRotationFailed) || got.ID != 0 {
		t.Fatalf("first rotate got=%#v err=%v", got, err)
	}
	var pending model.AWGDevice
	if err := database.GetDB().First(&pending, old.ID).Error; err != nil {
		t.Fatal(err)
	}
	if pending.SyncState != "pending_rotate" || pending.Provisioned || pending.PreviousPublicKey != old.PublicKey {
		t.Fatalf("add failure did not retain fail-closed state: %#v", pending)
	}
	got, err := manager.RotateOwnedDevice(context.Background(), old.ID, client.Id, "rotate-retry")
	if err != nil {
		t.Fatal(err)
	}
	if got.PublicKey != pending.PublicKey || generateCalls != 1 || removeCalls != 2 || addCalls != 2 {
		t.Fatalf("replay got=%#v generate=%d remove=%d add=%d", got, generateCalls, removeCalls, addCalls)
	}
	if replay, err := manager.RotateOwnedDevice(context.Background(), old.ID, client.Id, "rotate-retry"); err != nil || replay.ID != got.ID || generateCalls != 1 || removeCalls != 2 || addCalls != 2 {
		t.Fatalf("completed replay got=%#v err=%v generate=%d remove=%d add=%d", replay, err, generateCalls, removeCalls, addCalls)
	}
}

func TestAWGRotateOwnedDeviceRejectsForeignAndConcurrentRequest(t *testing.T) {
	provisioner := &fakeAWGProvisioner{
		remove: func(context.Context, string) error { return errors.New("offline") },
		add:    func(context.Context, AWGPeerSpec) error { return nil },
	}
	manager, client, old, _ := newAWGRotateFixture(t, provisioner)
	if got, err := manager.RotateOwnedDevice(context.Background(), old.ID, client.Id+1, "foreign"); !errors.Is(err, ErrAWGDeviceNotFound) || got.ID != 0 {
		t.Fatalf("foreign rotate got=%#v err=%v", got, err)
	}
	if _, err := manager.RotateOwnedDevice(context.Background(), old.ID, client.Id, "first"); !errors.Is(err, ErrAWGRotationFailed) {
		t.Fatalf("first pending rotate error=%v", err)
	}
	if got, err := manager.RotateOwnedDevice(context.Background(), old.ID, client.Id, "second"); !errors.Is(err, ErrAWGOperationInProgress) || got.ID != 0 {
		t.Fatalf("different request got=%#v err=%v", got, err)
	}
}

func TestAWGRotateOwnedDeviceNewRotationRequiresActiveClient(t *testing.T) {
	provisioner := &fakeAWGProvisioner{
		remove: func(context.Context, string) error { return nil },
		add:    func(context.Context, AWGPeerSpec) error { return nil },
	}
	manager, client, old, _ := newAWGRotateFixture(t, provisioner)
	if err := database.GetDB().Model(&model.Client{}).Where("id = ?", client.Id).Update("enable", false).Error; err != nil {
		t.Fatal(err)
	}
	if got, err := manager.RotateOwnedDevice(context.Background(), old.ID, client.Id, "inactive"); !errors.Is(err, ErrAWGClientInactive) || got.ID != 0 {
		t.Fatalf("inactive rotate got=%#v err=%v", got, err)
	}
}

func TestAWGRotateOwnedDeviceDoesNotAddAfterClientDepletesDuringRotation(t *testing.T) {
	var addCalls int
	provisioner := &fakeAWGProvisioner{
		add: func(context.Context, AWGPeerSpec) error { addCalls++; return nil },
	}
	manager, client, old, _ := newAWGRotateFixture(t, provisioner)
	provisioner.remove = func(context.Context, string) error {
		return database.GetDB().Model(&model.Client{}).Where("id = ?", client.Id).Update("enable", false).Error
	}
	if got, err := manager.RotateOwnedDevice(context.Background(), old.ID, client.Id, "deplete-during-rotate"); !errors.Is(err, ErrAWGClientInactive) || got.ID != 0 {
		t.Fatalf("rotate got=%#v err=%v", got, err)
	}
	if addCalls != 0 {
		t.Fatalf("inactive client received new peer: addCalls=%d", addCalls)
	}
	var stored model.AWGDevice
	if err := database.GetDB().First(&stored, old.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.SyncState != "pending_rotate" || stored.Provisioned || stored.PreviousPublicKey != old.PublicKey {
		t.Fatalf("rotation did not remain recoverable and fail-closed: %#v", stored)
	}
}
