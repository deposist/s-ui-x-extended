package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/netip"
	"slices"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"

	"gorm.io/gorm"
)

func TestAWGReconcileRecoversPendingAddAndPersistsPeer(t *testing.T) {
	live := AWGPeerSnapshot{}
	failAdd := true
	provisioner := &fakeAWGProvisioner{
		snapshot: func(context.Context) (AWGPeerSnapshot, error) { return live, nil },
		add: func(_ context.Context, peer AWGPeerSpec) error {
			if failAdd {
				return errors.New("core unavailable")
			}
			live[peer.PublicKey] = AWGPeerState{AllowedIPs: []netip.Prefix{peer.AllowedIP}}
			return nil
		},
		remove: func(_ context.Context, key string) error { delete(live, key); return nil },
	}
	manager, _ := newAWGCreateTestManager(t, provisioner)
	client := createAWGEligibleClient(t)
	if _, err := manager.CreateDevice(context.Background(), client.Id, "crash-before-ipc", "Phone", 1, 0); !errors.Is(err, ErrAWGProvisioningFailed) {
		t.Fatalf("create error = %v", err)
	}

	var persisted []AWGPersistedPeer
	manager.deps.SyncEndpointPeers = func(_ *gorm.DB, _ AWGSettings, peers []AWGPersistedPeer) error {
		persisted = append([]AWGPersistedPeer(nil), peers...)
		return nil
	}
	failAdd = false
	result, err := manager.Reconcile(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.Added != 1 || result.Desired != 1 || len(persisted) != 1 {
		t.Fatalf("result=%#v persisted=%#v", result, persisted)
	}
	var device model.AWGDevice
	if err := database.GetDB().Where("client_id = ?", client.Id).First(&device).Error; err != nil {
		t.Fatal(err)
	}
	if device.SyncState != "in_sync" || !device.Provisioned || device.LastError != "" || persisted[0].PublicKey != device.PublicKey {
		t.Fatalf("device=%#v persisted=%#v", device, persisted)
	}
}

func TestAWGReconcileFinalizesIpcSuccessAfterCrashWithoutSecondAdd(t *testing.T) {
	var addCalls int
	live := AWGPeerSnapshot{}
	provisioner := &fakeAWGProvisioner{
		snapshot: func(context.Context) (AWGPeerSnapshot, error) { return live, nil },
		add: func(_ context.Context, peer AWGPeerSpec) error {
			addCalls++
			live[peer.PublicKey] = AWGPeerState{AllowedIPs: []netip.Prefix{peer.AllowedIP}}
			return nil
		},
		remove: func(_ context.Context, key string) error { delete(live, key); return nil },
	}
	manager, _ := newAWGCreateTestManager(t, provisioner)
	manager.deps.SyncEndpointPeers = func(*gorm.DB, AWGSettings, []AWGPersistedPeer) error { return nil }
	client := createAWGEligibleClient(t)
	created, err := manager.CreateDevice(context.Background(), client.Id, "ipc-success-crash", "Phone", 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Model(&model.AWGDevice{}).Where("id = ?", created.ID).
		Updates(map[string]any{"sync_state": "pending_add", "provisioned": false}).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if addCalls != 1 {
		t.Fatalf("Add calls=%d, want original create only", addCalls)
	}
	got, err := manager.GetOwnedDevice(created.ID, client.Id)
	if err != nil || got.SyncState != "in_sync" || !got.Provisioned {
		t.Fatalf("device=%#v err=%v", got, err)
	}
}

func TestAWGReconcileRepairsMissingAndOrphanPeers(t *testing.T) {
	orphan := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0x77}, 32))
	live := AWGPeerSnapshot{orphan: {AllowedIPs: []netip.Prefix{netip.MustParsePrefix("10.77.0.7/32")}}}
	var removed []string
	provisioner := &fakeAWGProvisioner{
		snapshot: func(context.Context) (AWGPeerSnapshot, error) { return live, nil },
		add: func(_ context.Context, peer AWGPeerSpec) error {
			live[peer.PublicKey] = AWGPeerState{AllowedIPs: []netip.Prefix{peer.AllowedIP}}
			return nil
		},
		remove: func(_ context.Context, key string) error {
			removed = append(removed, key)
			delete(live, key)
			return nil
		},
	}
	manager, _ := newAWGCreateTestManager(t, provisioner)
	manager.deps.SyncEndpointPeers = func(*gorm.DB, AWGSettings, []AWGPersistedPeer) error { return nil }
	client := createAWGEligibleClient(t)
	created, err := manager.CreateDevice(context.Background(), client.Id, "drift", "Phone", 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	delete(live, created.PublicKey)

	result, err := manager.Reconcile(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.Added != 1 || result.Removed != 1 || !slices.Equal(removed, []string{orphan}) {
		t.Fatalf("result=%#v removed=%v", result, removed)
	}
	if _, exists := live[created.PublicKey]; !exists {
		t.Fatal("missing desired peer was not restored")
	}
}

func TestAWGReconcileRemovesInactiveClientPeerAndRetainsDesiredIntent(t *testing.T) {
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
	created, err := manager.CreateDevice(context.Background(), client.Id, "deplete", "Phone", 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Model(&model.Client{}).Where("id = ?", client.Id).Update("enable", false).Error; err != nil {
		t.Fatal(err)
	}
	result, err := manager.Reconcile(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.Removed != 1 || result.Desired != 0 {
		t.Fatalf("result=%#v", result)
	}
	got, err := manager.GetOwnedDevice(created.ID, client.Id)
	if err != nil || !got.DesiredEnabled || got.Provisioned || got.SyncState != "in_sync" {
		t.Fatalf("device=%#v err=%v", got, err)
	}
}

func TestAWGReconcileEndpointUnavailableKeepsDurablePendingState(t *testing.T) {
	manager, _ := newAWGCreateTestManager(t, &fakeAWGProvisioner{
		snapshot: func(context.Context) (AWGPeerSnapshot, error) { return nil, errors.New("unavailable") },
		add:      func(context.Context, AWGPeerSpec) error { return errors.New("unavailable") },
		remove:   func(context.Context, string) error { return errors.New("unavailable") },
	})
	client := createAWGEligibleClient(t)
	if _, err := manager.CreateDevice(context.Background(), client.Id, "unavailable", "Phone", 1, 0); err == nil {
		t.Fatal("expected create failure")
	}
	if _, err := manager.Reconcile(context.Background()); !errors.Is(err, ErrAWGReconcileFailed) {
		t.Fatalf("reconcile error=%v", err)
	}
	var device model.AWGDevice
	if err := database.GetDB().Where("client_id = ?", client.Id).First(&device).Error; err != nil {
		t.Fatal(err)
	}
	if device.SyncState != "pending_add" || device.Provisioned {
		t.Fatalf("durable pending state changed: %#v", device)
	}
}

func TestAWGReconcileSurfacesSanitizedCauseWithSentinelIdentity(t *testing.T) {
	manager, _ := newAWGCreateTestManager(t, &fakeAWGProvisioner{
		snapshot: func(context.Context) (AWGPeerSnapshot, error) { return nil, errors.New("unavailable") },
		add:      func(context.Context, AWGPeerSpec) error { return errors.New("unavailable") },
		remove:   func(context.Context, string) error { return errors.New("unavailable") },
	})
	_, err := manager.Reconcile(context.Background())
	if !errors.Is(err, ErrAWGReconcileFailed) {
		t.Fatalf("reconcile error = %v, want ErrAWGReconcileFailed", err)
	}
	if !strings.Contains(err.Error(), "unavailable") {
		t.Fatalf("reconcile error %q hides the sanitized underlying cause", err)
	}
}

func TestSyncAWGManagedEndpointPeersPreservesUnrelatedOptions(t *testing.T) {
	initSettingTestDB(t)
	db := database.GetDB()
	original := json.RawMessage(`{"address":["10.77.0.1/16"],"private_key":"server-secret","listen_port":51820,"mtu":1420,"custom":{"keep":true},"peers":[{"public_key":"stale"}]}`)
	endpoint := model.Endpoint{Type: "wireguard", Tag: "awg", Options: original}
	if err := db.Create(&endpoint).Error; err != nil {
		t.Fatal(err)
	}
	peers := []AWGPersistedPeer{{PublicKey: "new", AllowedIPs: []string{"10.77.0.2/32"}}}
	if err := SyncAWGManagedEndpointPeers(db, AWGSettings{EndpointTag: "awg"}, peers); err != nil {
		t.Fatal(err)
	}
	var stored model.Endpoint
	if err := db.First(&stored, endpoint.Id).Error; err != nil {
		t.Fatal(err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(stored.Options, &raw); err != nil {
		t.Fatal(err)
	}
	var custom map[string]bool
	var privateKey string
	if err := json.Unmarshal(raw["custom"], &custom); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw["private_key"], &privateKey); err != nil {
		t.Fatal(err)
	}
	if !custom["keep"] || privateKey != "server-secret" {
		t.Fatalf("unrelated options changed: %s", stored.Options)
	}
	var got []map[string]any
	if err := json.Unmarshal(raw["peers"], &got); err != nil || len(got) != 1 || got[0]["public_key"] != "new" {
		t.Fatalf("peers=%s err=%v", raw["peers"], err)
	}
	if _, exposed := got[0]["pre_shared_key"]; exposed {
		t.Fatalf("persisted endpoint exposed PSK: %s", raw["peers"])
	}
}
