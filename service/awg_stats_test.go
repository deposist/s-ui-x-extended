package service

import (
	"context"
	"net/netip"
	"testing"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
)

func TestAWGStatsDirectionsIdempotencyResetAndHandshake(t *testing.T) {
	live := AWGPeerSnapshot{}
	manager, _ := newAWGCreateTestManager(t, &fakeAWGProvisioner{
		snapshot: func(context.Context) (AWGPeerSnapshot, error) { return live, nil },
		add: func(_ context.Context, peer AWGPeerSpec) error {
			live[peer.PublicKey] = AWGPeerState{AllowedIPs: []netip.Prefix{peer.AllowedIP}}
			return nil
		},
		remove: func(_ context.Context, key string) error { delete(live, key); return nil },
	})
	client := createAWGEligibleClient(t)
	device, err := manager.CreateDevice(context.Background(), client.Id, "stats", "Phone", 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	live[device.PublicKey] = AWGPeerState{ReceiveBytes: 100, TransmitBytes: 200, LastHandshake: 55}
	got, err := manager.CollectStats(context.Background())
	if err != nil || got.Upload != 100 || got.Download != 200 || got.Devices != 1 {
		t.Fatalf("stats=%#v err=%v", got, err)
	}
	got, err = manager.CollectStats(context.Background())
	if err != nil || got.Upload != 0 || got.Download != 0 {
		t.Fatalf("repeated stats=%#v err=%v", got, err)
	}
	live[device.PublicKey] = AWGPeerState{ReceiveBytes: 7, TransmitBytes: 9, LastHandshake: 77}
	got, err = manager.CollectStats(context.Background())
	if err != nil || got.Upload != 7 || got.Download != 9 {
		t.Fatalf("reset stats=%#v err=%v", got, err)
	}
	var storedDevice model.AWGDevice
	var storedClient model.Client
	if err := database.GetDB().First(&storedDevice, device.ID).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().First(&storedClient, client.Id).Error; err != nil {
		t.Fatal(err)
	}
	if storedDevice.TotalRx != 107 || storedDevice.TotalTx != 209 || storedDevice.LastHandshake != 77 || storedClient.Up != 107 || storedClient.Down != 209 {
		t.Fatalf("device=%#v client up/down=%d/%d", storedDevice, storedClient.Up, storedClient.Down)
	}
}

func TestAWGStatsAggregatesTwoDevicesAndIgnoresUnknownPeer(t *testing.T) {
	live := AWGPeerSnapshot{}
	keySeed := byte(0x22)
	manager, _ := newAWGCreateTestManager(t, &fakeAWGProvisioner{
		snapshot: func(context.Context) (AWGPeerSnapshot, error) { return live, nil },
		add: func(_ context.Context, peer AWGPeerSpec) error {
			live[peer.PublicKey] = AWGPeerState{AllowedIPs: []netip.Prefix{peer.AllowedIP}}
			return nil
		},
		remove: func(_ context.Context, key string) error { delete(live, key); return nil },
	})
	manager.deps.GenerateKeys = func() (AWGGeneratedKeys, error) {
		keySeed++
		return AWGGeneratedKeys{PrivateKey: repeatAWGByte(keySeed), PublicKey: repeatAWGByte(keySeed + 20), PSK: repeatAWGByte(keySeed + 40)}, nil
	}
	client := createAWGEligibleClient(t)
	first, err := manager.CreateDevice(context.Background(), client.Id, "stats-1", "Phone", 2, 0)
	if err != nil {
		t.Fatal(err)
	}
	second, err := manager.CreateDevice(context.Background(), client.Id, "stats-2", "Laptop", 2, 0)
	if err != nil {
		t.Fatal(err)
	}
	live[first.PublicKey] = AWGPeerState{ReceiveBytes: 3, TransmitBytes: 5}
	live[second.PublicKey] = AWGPeerState{ReceiveBytes: 7, TransmitBytes: 11}
	live["unknown"] = AWGPeerState{ReceiveBytes: 10_000, TransmitBytes: 10_000}
	got, err := manager.CollectStats(context.Background())
	if err != nil || got.Devices != 2 || got.Upload != 10 || got.Download != 16 {
		t.Fatalf("stats=%#v err=%v", got, err)
	}
	var stored model.Client
	if err := database.GetDB().First(&stored, client.Id).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Up != 10 || stored.Down != 16 {
		t.Fatalf("client up/down=%d/%d", stored.Up, stored.Down)
	}
}

func TestAWGStatsRollbackKeepsBaselinesAndClientCountersTogether(t *testing.T) {
	live := AWGPeerSnapshot{}
	manager, _ := newAWGCreateTestManager(t, &fakeAWGProvisioner{
		snapshot: func(context.Context) (AWGPeerSnapshot, error) { return live, nil },
		add: func(_ context.Context, peer AWGPeerSpec) error {
			live[peer.PublicKey] = AWGPeerState{AllowedIPs: []netip.Prefix{peer.AllowedIP}}
			return nil
		},
		remove: func(_ context.Context, key string) error { delete(live, key); return nil },
	})
	client := createAWGEligibleClient(t)
	device, err := manager.CreateDevice(context.Background(), client.Id, "stats-rollback", "Phone", 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	live[device.PublicKey] = AWGPeerState{ReceiveBytes: 13, TransmitBytes: 17}
	if err := database.GetDB().Exec(`CREATE TRIGGER fail_awg_client_traffic BEFORE UPDATE OF up, down ON clients
		BEGIN SELECT RAISE(ABORT, 'forced traffic rollback'); END`).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := manager.CollectStats(context.Background()); err == nil {
		t.Fatal("expected transaction failure")
	}
	var storedDevice model.AWGDevice
	var storedClient model.Client
	if err := database.GetDB().First(&storedDevice, device.ID).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().First(&storedClient, client.Id).Error; err != nil {
		t.Fatal(err)
	}
	if storedDevice.RxBaseline != 0 || storedDevice.TxBaseline != 0 || storedDevice.TotalRx != 0 || storedDevice.TotalTx != 0 || storedClient.Up != 0 || storedClient.Down != 0 {
		t.Fatalf("rollback split state: device=%#v client up/down=%d/%d", storedDevice, storedClient.Up, storedClient.Down)
	}
}

func repeatAWGByte(value byte) []byte {
	result := make([]byte, 32)
	for i := range result {
		result[i] = value
	}
	return result
}
