package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/netip"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func TestAWGConfigGoldenRerenderAndOwnership(t *testing.T) {
	manager, _ := newAWGCreateTestManager(t, &fakeAWGProvisioner{
		snapshot: func(context.Context) (AWGPeerSnapshot, error) { return AWGPeerSnapshot{}, nil },
		add:      func(context.Context, AWGPeerSpec) error { return nil },
		remove:   func(context.Context, string) error { return nil },
	})
	serverPrivate := wgtypes.Key(bytes.Repeat([]byte{0x55}, 32))
	options, err := json.Marshal(map[string]any{
		"address": []string{"10.77.0.1/29"}, "private_key": serverPrivate.String(), "listen_port": 51820, "mtu": 1380,
		"amnezia": map[string]any{"jc": 3, "jmin": 10, "jmax": 20, "s1": 15, "s2": 18, "s3": 12, "s4": 8,
			"h1": "1000-1099", "h2": 2000, "h3": 3000, "h4": 4000, "i1": "<b 0x01020304><r 8>",
			"j1": "<b 0x11><r 4>", "j2": "<c><r 6>", "j3": "<t><r 2>", "itime": 120},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Create(&model.Endpoint{Type: "wireguard", Tag: "awg", Options: options}).Error; err != nil {
		t.Fatal(err)
	}
	manager.deps.LoadSettings = func() (AWGSettings, error) {
		return AWGSettings{Enabled: true, EndpointTag: "awg", PublicEndpoint: "vpn.example.com:51820", Subnet: netip.MustParsePrefix("10.77.0.0/29"), DNS: []netip.Addr{netip.MustParseAddr("1.1.1.1")}}, nil
	}
	client := createAWGEligibleClient(t)
	device, err := manager.CreateDevice(context.Background(), client.Id, "config", "Phone", 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	got, err := manager.RenderOwnedConfig(context.Background(), device.ID, client.Id)
	if err != nil {
		t.Fatal(err)
	}
	want := "[Interface]\n" +
		"PrivateKey = " + base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0x11}, 32)) + "\n" +
		"Address = 10.77.0.2/32\nDNS = 1.1.1.1\nMTU = 1380\nJc = 3\nJmin = 10\nJmax = 20\nS1 = 15\nS2 = 18\nS3 = 12\nS4 = 8\nH1 = 1000-1099\nH2 = 2000\nH3 = 3000\nH4 = 4000\nI1 = <b 0x01020304><r 8>\nJ1 = <b 0x11><r 4>\nJ2 = <c><r 6>\nJ3 = <t><r 2>\nItime = 120\n\n[Peer]\n" +
		"PublicKey = " + serverPrivate.PublicKey().String() + "\n" +
		"PresharedKey = " + base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0x33}, 32)) + "\n" +
		"Endpoint = vpn.example.com:51820\nAllowedIPs = 0.0.0.0/0, ::/0\nPersistentKeepalive = 25\n"
	if string(got) != want {
		t.Fatalf("config mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
	again, err := manager.RenderOwnedConfig(context.Background(), device.ID, client.Id)
	if err != nil || !bytes.Equal(got, again) {
		t.Fatalf("rerender changed config: err=%v", err)
	}
	if _, err := manager.RenderOwnedConfig(context.Background(), device.ID, client.Id+100); !errors.Is(err, ErrAWGDeviceNotFound) {
		t.Fatalf("foreign device error=%v", err)
	}
}

func TestAWGConfigUnavailableWhilePendingAndQRExactPolicy(t *testing.T) {
	manager, _ := newAWGCreateTestManager(t, &fakeAWGProvisioner{
		snapshot: func(context.Context) (AWGPeerSnapshot, error) { return AWGPeerSnapshot{}, nil },
		add:      func(context.Context, AWGPeerSpec) error { return errors.New("unavailable") },
		remove:   func(context.Context, string) error { return nil },
	})
	client := createAWGEligibleClient(t)
	if _, err := manager.CreateDevice(context.Background(), client.Id, "pending-config", "Phone", 1, 0); err == nil {
		t.Fatal("expected pending create")
	}
	var device model.AWGDevice
	if err := database.GetDB().Where("client_id = ?", client.Id).First(&device).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := manager.RenderOwnedConfig(context.Background(), device.Id, client.Id); !errors.Is(err, ErrAWGConfigUnavailable) {
		t.Fatalf("pending config error=%v", err)
	}
	payload := []byte("exact-awg-config")
	png, err := RenderAWGConfigQR(payload)
	if err != nil || len(png) == 0 {
		t.Fatalf("QR render err=%v size=%d", err, len(png))
	}
	// Empty config is unavailable, not "too large".
	if _, err := RenderAWGConfigQR(nil); !errors.Is(err, ErrAWGConfigUnavailable) {
		t.Fatalf("empty config error=%v", err)
	}
	// A full AWG 2.0 config (long I/J junk fields) must still fit at Low ECC.
	// qrcode.Low tops out near 2953 bytes; ~2200 exercises the realistic upper
	// band without tripping the codec limit.
	if _, err := RenderAWGConfigQR([]byte(strings.Repeat("x", 2200))); err != nil {
		t.Fatalf("realistic AWG 2.0 config should render: %v", err)
	}
	// Beyond the codec capacity, surface ErrAWGConfigTooLargeQR so callers fall
	// back to the .conf download.
	if _, err := RenderAWGConfigQR([]byte(strings.Repeat("x", 4000))); !errors.Is(err, ErrAWGConfigTooLargeQR) {
		t.Fatalf("oversize error=%v", err)
	}
}
