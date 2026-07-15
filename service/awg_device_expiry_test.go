package service

import (
	"bytes"
	"context"
	"errors"
	"net/netip"
	"testing"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"

	"gorm.io/gorm"
)

// The fixed test clock from newAWGCreateTestManager.
const awgExpiryTestNow = int64(1_700_000_000)

func TestDeviceExpiredAt(t *testing.T) {
	cases := []struct {
		name      string
		expiresAt int64
		now       int64
		want      bool
	}{
		{"zero never expires", 0, awgExpiryTestNow, false},
		{"future not expired", awgExpiryTestNow + 1, awgExpiryTestNow, false},
		{"exact boundary is expired (exclusive)", awgExpiryTestNow, awgExpiryTestNow, true},
		{"past expired", awgExpiryTestNow - 1, awgExpiryTestNow, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			device := model.AWGDevice{ExpiresAt: tc.expiresAt}
			if got := deviceExpiredAt(device, tc.now); got != tc.want {
				t.Fatalf("deviceExpiredAt(%d, %d) = %v, want %v", tc.expiresAt, tc.now, got, tc.want)
			}
		})
	}
}

func TestValidateAWGDeviceExpiry(t *testing.T) {
	now := awgExpiryTestNow
	cases := []struct {
		name      string
		expiresAt int64
		wantErr   bool
	}{
		{"zero is never", 0, false},
		{"future ok", now + 3600, false},
		{"ten years ok", now + awgMaxDeviceExpirySeconds, false},
		{"now rejected (exclusive)", now, true},
		{"past rejected", now - 1, true},
		{"beyond ten years rejected", now + awgMaxDeviceExpirySeconds + 1, true},
		{"negative rejected", -5, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateAWGDeviceExpiry(tc.expiresAt, now)
			if tc.wantErr && !errors.Is(err, ErrAWGInvalidExpiry) {
				t.Fatalf("expected ErrAWGInvalidExpiry, got %v", err)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestCreateDeviceRejectsInvalidExpiry(t *testing.T) {
	manager, _ := newAWGCreateTestManager(t, &fakeAWGProvisioner{
		snapshot: func(context.Context) (AWGPeerSnapshot, error) { return AWGPeerSnapshot{}, nil },
		add:      func(context.Context, AWGPeerSpec) error { return nil },
	})
	client := createAWGEligibleClient(t)
	if _, err := manager.CreateDevice(context.Background(), client.Id, "expiry-past", "Phone", 1, awgExpiryTestNow-10); !errors.Is(err, ErrAWGInvalidExpiry) {
		t.Fatalf("past expiry error=%v", err)
	}
	if _, err := manager.CreateDevice(context.Background(), client.Id, "expiry-far", "Phone", 1, awgExpiryTestNow+awgMaxDeviceExpirySeconds+100); !errors.Is(err, ErrAWGInvalidExpiry) {
		t.Fatalf("too-distant expiry error=%v", err)
	}
}

func TestCreateDevicePersistsExpiry(t *testing.T) {
	manager, _ := newAWGCreateTestManager(t, &fakeAWGProvisioner{
		snapshot: func(context.Context) (AWGPeerSnapshot, error) { return AWGPeerSnapshot{}, nil },
		add:      func(context.Context, AWGPeerSpec) error { return nil },
	})
	client := createAWGEligibleClient(t)
	expiresAt := awgExpiryTestNow + 7*24*3600
	info, err := manager.CreateDevice(context.Background(), client.Id, "expiry-set", "Phone", 1, expiresAt)
	if err != nil {
		t.Fatal(err)
	}
	if info.ExpiresAt != expiresAt {
		t.Fatalf("info.ExpiresAt = %d, want %d", info.ExpiresAt, expiresAt)
	}
	var device model.AWGDevice
	if err := database.GetDB().First(&device, info.ID).Error; err != nil {
		t.Fatal(err)
	}
	if device.ExpiresAt != expiresAt {
		t.Fatalf("persisted ExpiresAt = %d, want %d", device.ExpiresAt, expiresAt)
	}
}

// The reconciler must deprovision an expired device (peer removed from the
// interface) while keeping the row and its desired intent - the slot stays
// occupied until manual deletion (owner decision 3).
func TestAWGReconcileRemovesExpiredDevicePeerAndKeepsRow(t *testing.T) {
	live := AWGPeerSnapshot{}
	provisioner := &fakeAWGProvisioner{
		snapshot: func(context.Context) (AWGPeerSnapshot, error) { return live, nil },
		add: func(_ context.Context, peer AWGPeerSpec) error {
			live[peer.PublicKey] = AWGPeerState{AllowedIPs: []netip.Prefix{peer.AllowedIP}}
			return nil
		},
		remove: func(_ context.Context, key string) error { delete(live, key); return nil },
	}
	manager, _ := newAWGCreateTestManager(t, provisioner)
	manager.deps.SyncEndpointPeers = func(*gorm.DB, AWGSettings, []AWGPersistedPeer) error { return nil }
	client := createAWGEligibleClient(t)
	created, err := manager.CreateDevice(context.Background(), client.Id, "expire-peer", "Phone", 1, awgExpiryTestNow+3600)
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := live[created.PublicKey]; !exists {
		t.Fatal("peer was not provisioned")
	}
	// Simulate the passage of time: shift the stored expiry into the past
	// (the manager's clock is fixed).
	if err := database.GetDB().Model(&model.AWGDevice{}).Where("id = ?", created.ID).Update("expires_at", awgExpiryTestNow-1).Error; err != nil {
		t.Fatal(err)
	}
	result, err := manager.Reconcile(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.Removed != 1 || result.Desired != 0 {
		t.Fatalf("result=%#v", result)
	}
	if _, exists := live[created.PublicKey]; exists {
		t.Fatal("expired peer still on interface")
	}
	got, err := manager.GetOwnedDevice(created.ID, client.Id)
	if err != nil || !got.DesiredEnabled || got.Provisioned {
		t.Fatalf("device=%#v err=%v", got, err)
	}
}

// Rotation of an expired device must be rejected.
func TestAWGRotateExpiredDeviceRejected(t *testing.T) {
	manager, _ := newAWGCreateTestManager(t, &fakeAWGProvisioner{
		snapshot: func(context.Context) (AWGPeerSnapshot, error) { return AWGPeerSnapshot{}, nil },
		add:      func(context.Context, AWGPeerSpec) error { return nil },
		remove:   func(context.Context, string) error { return nil },
	})
	client := createAWGEligibleClient(t)
	created, err := manager.CreateDevice(context.Background(), client.Id, "expire-rotate", "Phone", 1, awgExpiryTestNow+3600)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Model(&model.AWGDevice{}).Where("id = ?", created.ID).Update("expires_at", awgExpiryTestNow-1).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := manager.RotateOwnedDevice(context.Background(), created.ID, client.Id, "rotate-expired"); !errors.Is(err, ErrAWGDeviceExpired) {
		t.Fatalf("rotate expired device error=%v", err)
	}
}

// An expired device still occupies its device-limit slot (owner decision 3):
// with limit=1 and one expired device, creating another device must fail.
func TestExpiredDeviceStillOccupiesLimitSlot(t *testing.T) {
	manager, _ := newAWGCreateTestManager(t, &fakeAWGProvisioner{
		snapshot: func(context.Context) (AWGPeerSnapshot, error) { return AWGPeerSnapshot{}, nil },
		add:      func(context.Context, AWGPeerSpec) error { return nil },
		remove:   func(context.Context, string) error { return nil },
	})
	// The shared helper returns fixed key bytes; a revoked device keeps its
	// row (and unique public_key), so each CreateDevice here needs unique keys
	// like the real generator produces.
	var keySeq byte
	manager.deps.GenerateKeys = func() (AWGGeneratedKeys, error) {
		keySeq++
		return AWGGeneratedKeys{
			PrivateKey: bytes.Repeat([]byte{keySeq}, 32),
			PublicKey:  bytes.Repeat([]byte{0x80 + keySeq}, 32),
			PSK:        bytes.Repeat([]byte{0x40 + keySeq}, 32),
		}, nil
	}
	client := createAWGEligibleClient(t)
	created, err := manager.CreateDevice(context.Background(), client.Id, "slot-1", "Phone", 1, awgExpiryTestNow+3600)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Model(&model.AWGDevice{}).Where("id = ?", created.ID).Update("expires_at", awgExpiryTestNow-1).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := manager.CreateDevice(context.Background(), client.Id, "slot-2", "Laptop", 1, 0); !errors.Is(err, ErrAWGDeviceLimitReached) {
		t.Fatalf("expected limit reached while slot held by expired device, got %v", err)
	}
	// Revoking the expired device frees the slot.
	if err := manager.RevokeOwnedDevice(context.Background(), created.ID, client.Id); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.CreateDevice(context.Background(), client.Id, "slot-3", "Tablet", 1, 0); err != nil {
		t.Fatalf("create after revoke failed: %v", err)
	}
}

// Config rendering for an expired (deprovisioned) device is unavailable, same
// as for any deprovisioned device: renderOwnedConfigInWorker requires
// provisioned + in_sync.
func TestRenderConfigUnavailableForExpiredDeprovisionedDevice(t *testing.T) {
	live := AWGPeerSnapshot{}
	provisioner := &fakeAWGProvisioner{
		snapshot: func(context.Context) (AWGPeerSnapshot, error) { return live, nil },
		add: func(_ context.Context, peer AWGPeerSpec) error {
			live[peer.PublicKey] = AWGPeerState{AllowedIPs: []netip.Prefix{peer.AllowedIP}}
			return nil
		},
		remove: func(_ context.Context, key string) error { delete(live, key); return nil },
	}
	manager, _ := newAWGCreateTestManager(t, provisioner)
	manager.deps.SyncEndpointPeers = func(*gorm.DB, AWGSettings, []AWGPersistedPeer) error { return nil }
	client := createAWGEligibleClient(t)
	created, err := manager.CreateDevice(context.Background(), client.Id, "expire-render", "Phone", 1, awgExpiryTestNow+3600)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Model(&model.AWGDevice{}).Where("id = ?", created.ID).Update("expires_at", awgExpiryTestNow-1).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.RenderOwnedConfig(context.Background(), created.ID, client.Id); !errors.Is(err, ErrAWGConfigUnavailable) {
		t.Fatalf("render expired config error=%v", err)
	}
}

// Extending the expiry (update + reconcile) restores the peer.
func TestExtendingExpiryRestoresPeerOnReconcile(t *testing.T) {
	live := AWGPeerSnapshot{}
	provisioner := &fakeAWGProvisioner{
		snapshot: func(context.Context) (AWGPeerSnapshot, error) { return live, nil },
		add: func(_ context.Context, peer AWGPeerSpec) error {
			live[peer.PublicKey] = AWGPeerState{AllowedIPs: []netip.Prefix{peer.AllowedIP}}
			return nil
		},
		remove: func(_ context.Context, key string) error { delete(live, key); return nil },
	}
	manager, _ := newAWGCreateTestManager(t, provisioner)
	manager.deps.SyncEndpointPeers = func(*gorm.DB, AWGSettings, []AWGPersistedPeer) error { return nil }
	client := createAWGEligibleClient(t)
	created, err := manager.CreateDevice(context.Background(), client.Id, "expire-extend", "Phone", 1, awgExpiryTestNow+3600)
	if err != nil {
		t.Fatal(err)
	}
	db := database.GetDB()
	if err := db.Model(&model.AWGDevice{}).Where("id = ?", created.ID).Update("expires_at", awgExpiryTestNow-1).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, exists := live[created.PublicKey]; exists {
		t.Fatal("expired peer still present after reconcile")
	}
	// Renewal: push the boundary into the future again.
	if err := db.Model(&model.AWGDevice{}).Where("id = ?", created.ID).Update("expires_at", awgExpiryTestNow+7200).Error; err != nil {
		t.Fatal(err)
	}
	result, err := manager.Reconcile(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.Added != 1 {
		t.Fatalf("result=%#v", result)
	}
	if _, exists := live[created.PublicKey]; !exists {
		t.Fatal("renewed peer was not restored")
	}
}
