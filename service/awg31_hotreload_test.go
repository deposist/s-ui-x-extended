//go:build integration

package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/deposist/s-ui-x-extended/core"
	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

var clientPeerPort uint16

// TestAWG31ManagedHotReloadKeepsHandshakes pins the hot-reload regression:
// saving a managed AWG endpoint must not leave peers without preshared keys
// or without the AmneziaWG 3.1 device flags in the live core. The full-start
// path (config build + InjectManagedAWGEndpointPeers) and the hot-reload path
// (RestartEndpoints marshaling stored options) differ, and the reconciler
// skips peers whose public key and allowed IP already match, so a broken
// reload is invisible to the health snapshot. This test completes a real
// handshake after a hot reload, which no DB-state check can prove.
func TestAWG31ManagedHotReloadKeepsHandshakes(t *testing.T) {
	initSettingTestDB(t)
	db := database.GetDB()
	t.Setenv(awgEncryptionKeyEnv, base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0x91}, 32)))
	cipher, err := NewAWGCipherFromEnv()
	if err != nil {
		t.Fatal(err)
	}

	client := model.Client{Enable: true, Name: "hot-reload", Expiry: time.Now().Add(time.Hour).Unix()}
	if err := db.Create(&client).Error; err != nil {
		t.Fatal(err)
	}
	serverKey, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		t.Fatal(err)
	}
	deviceKey, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		t.Fatal(err)
	}
	psk, err := wgtypes.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}

	// A production managed endpoint always carries a fixed listen_port
	// (validateManagedAWGEndpointOptions rejects 0), so a hot reload rebinds
	// the same port and clients keep targeting it.
	freePort := freeUDPPortForAWGTest(t)
	endpoint := model.Endpoint{
		Type: "wireguard", Tag: "managed-awg",
		Ext: json.RawMessage(fmt.Sprintf(`{"managed":true,"publicEndpoint":"127.0.0.1:%d","dns":["1.1.1.1"],"defaultDeviceLimit":3}`, freePort)),
		Options: json.RawMessage(fmt.Sprintf(
			`{"system":false,"address":["10.77.0.1/24"],"private_key":%q,"listen_port":%d,"amnezia":{"jc":3,"jmin":10,"jmax":20,"s1":15,"s2":18,"s3":12,"s4":8,"h1":"1000-1099","h2":"2000-2099","h3":"3000-3099","h4":"4000-4099","i1":"<b 0x01020304><r 8>","random_trailers":true,"disable_cookies":true},"peers":[]}`,
			serverKey.String(), freePort)),
	}
	if err := db.Create(&endpoint).Error; err != nil {
		t.Fatal(err)
	}

	cryptoContext := bytes.Repeat([]byte{0x44}, awgCryptoContextSize)
	pskEnc, err := cipher.Encrypt(client.Id, cryptoContext, psk[:])
	if err != nil {
		t.Fatal(err)
	}
	device := model.AWGDevice{
		ClientId: client.Id, EndpointId: endpoint.Id, Name: "phone",
		CryptoContext: cryptoContext, PublicKey: deviceKey.PublicKey().String(),
		PrivateKeyEnc: []byte{1}, PSKEnc: pskEnc, IPv4Address: "10.77.0.2",
		DesiredEnabled: true, SyncState: "in_sync", Provisioned: true, CreatedAt: 1, UpdatedAt: 1,
	}
	if err := db.Create(&device).Error; err != nil {
		t.Fatal(err)
	}

	// Full start with injected peers: the production core start path.
	raw, err := endpoint.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	injected, err := InjectManagedAWGEndpointPeers(db, []json.RawMessage{raw})
	if err != nil {
		t.Fatal(err)
	}
	config := []byte(`{"log":{"disabled":true},"endpoints":[` + string(injected[0]) + `],"outbounds":[{"type":"direct","tag":"direct"}]}`)
	server := core.NewCore()
	if err := server.Start(config); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.Stop() })

	handshake := func(stage string) {
		t.Helper()
		clientPeer := map[string]any{
			"address": "127.0.0.1", "port": clientPeerPort, "public_key": serverKey.PublicKey().String(),
			"pre_shared_key": psk.String(), "allowed_ips": []string{"0.0.0.0/0"},
			"persistent_keepalive_interval": 1,
		}
		clientConfig, err := json.Marshal(map[string]any{
			"log": map[string]any{"disabled": true},
			"endpoints": []any{map[string]any{
				"type": "wireguard", "tag": "client", "address": []string{"10.77.0.2/32"},
				"private_key": deviceKey.String(), "peers": []any{clientPeer},
				"amnezia": map[string]any{
					"jc": 3, "jmin": 10, "jmax": 20,
					"s1": 15, "s2": 18, "s3": 12, "s4": 8,
					"h1": "1000-1099", "h2": "2000-2099", "h3": "3000-3099", "h4": "4000-4099",
					"i1":              "<b 0x01020304><r 8>",
					"random_trailers": true, "disable_cookies": true,
				},
			}},
			"outbounds": []any{map[string]any{"type": "direct", "tag": "direct"}},
		})
		if err != nil {
			t.Fatal(err)
		}
		clientCore := core.NewCore()
		if err := clientCore.Start(clientConfig); err != nil {
			t.Fatalf("%s: start client core: %v", stage, err)
		}
		t.Cleanup(func() { _ = clientCore.Stop() })

		deadline := time.Now().Add(10 * time.Second)
		for {
			var snapshots []core.WireGuardPeerSnapshot
			if err := server.WithWireGuardIPC(endpoint.Tag, func(ipc core.WireGuardIPC) error {
				state, err := ipc.IpcGet()
				if err != nil {
					return err
				}
				snapshots, err = core.ParseWireGuardPeerSnapshot(state)
				return err
			}); err != nil {
				t.Fatalf("%s: snapshot: %v", stage, err)
			}
			for _, snap := range snapshots {
				if snap.PublicKey == deviceKey.PublicKey().String() && snap.LastHandshake > 0 {
					t.Logf("%s: handshake completed (rx=%d tx=%d)", stage, snap.ReceiveBytes, snap.TransmitBytes)
					return
				}
			}
			if time.Now().After(deadline) {
				t.Fatalf("%s: no handshake after hot reload; snapshots=%+v", stage, snapshots)
			}
			time.Sleep(100 * time.Millisecond)
		}
	}

	// The client needs the server's real listen port: read it from the live device.
	var serverPort uint16
	if err := server.WithWireGuardIPC(endpoint.Tag, func(ipc core.WireGuardIPC) error {
		state, err := ipc.IpcGet()
		if err != nil {
			return err
		}
		for _, line := range strings.Split(state, "\n") {
			if after, ok := strings.CutPrefix(line, "listen_port="); ok {
				value, err := strconv.ParseUint(after, 10, 16)
				if err != nil {
					return err
				}
				serverPort = uint16(value)
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if serverPort == 0 {
		t.Fatal("server listen_port not found in UAPI state")
	}
	clientPeerPort = serverPort

	handshake("full start")

	serviceRuntime := NewRuntime(server)
	manager := NewAWGManagerWithDeps(serviceRuntime, NewAWGProvisioner(serviceRuntime, endpoint.Tag), 4, AWGManagerDeps{
		DB:         db,
		EndpointID: endpoint.Id,
		LoadSettings: func() (AWGSettings, error) {
			var row model.Endpoint
			if err := db.Where("id = ?", endpoint.Id).First(&row).Error; err != nil {
				return AWGSettings{}, err
			}
			return AWGSettingsForEndpoint(row)
		},
		LoadEndpoint:      LoadAWGManagedEndpoint,
		SyncEndpointPeers: SyncAWGManagedEndpointPeers,
		GenerateKeys:      generateAWGKeys,
		NewCryptoContext:  NewAWGCryptoContext,
		CipherFactory: func() (awgKeyCipher, error) {
			return NewAWGCipherFromEnv()
		},
		Now: func() int64 { return time.Now().Unix() },
	})
	if err := manager.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Stop(context.Background()) })

	// Hot-reload path: restart the endpoint from stored options, then
	// reconcile exactly like the production post-save hook does.
	if err := (&EndpointService{Runtime: serviceRuntime}).RestartEndpoints(db, []uint{endpoint.Id}); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Reconcile(context.Background()); err != nil {
		t.Fatalf("reconcile after reload: %v", err)
	}

	handshake("after hot reload")

	// The live device must still carry both 3.1 flags after the reload.
	if err := server.WithWireGuardIPC(endpoint.Tag, func(ipc core.WireGuardIPC) error {
		state, err := ipc.IpcGet()
		if err != nil {
			return err
		}
		for _, want := range []string{"random_trailers=1", "disable_cookies=1"} {
			if !bytes.Contains([]byte(state), []byte(want)) {
				t.Errorf("live device lost %s after hot reload", want)
			}
		}
		// And the peer must still have its preshared key.
		if !bytes.Contains([]byte(state), []byte("preshared_key="+hex.EncodeToString(psk[:]))) {
			t.Error("live device lost the peer preshared key after hot reload")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

// freeUDPPortForAWGTest asks the OS for a free UDP port and releases it so the
// core can immediately rebind it. (The core package has the same helper, but
// test helpers are not importable across packages.)
func freeUDPPortForAWGTest(t *testing.T) uint16 {
	t.Helper()
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	return uint16(conn.LocalAddr().(*net.UDPAddr).Port)
}
