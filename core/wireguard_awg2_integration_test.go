//go:build integration

// Run with:
//
//	go test -tags="integration with_wireguard with_gvisor" ./core -run TestAWG2ServerEndpoint
//
// The test uses userspace network stacks and loopback UDP, so it does not need
// root privileges or a system WireGuard interface.
package core

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/netip"
	"testing"
	"time"

	M "github.com/sagernet/sing/common/metadata"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

const awgEndpointTag = "awg"

func TestAWG2ServerEndpoint(t *testing.T) {
	serverPrivateKey, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		t.Fatal(err)
	}
	clientPrivateKey, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		t.Fatal(err)
	}
	presharedKey, err := wgtypes.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}

	serverPort := freeUDPPort(t)
	server := startAWGCore(t, awgCoreConfig(t, "10.77.0.1/24", serverPrivateKey.String(), serverPort, nil))

	addPayload, err := BuildWireGuardAddPeerUAPI(
		clientPrivateKey.PublicKey().String(),
		presharedKey.String(),
		netip.MustParsePrefix("10.77.0.2/32"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := server.WithWireGuardIPC(awgEndpointTag, func(ipc WireGuardIPC) error {
		return ipc.IpcSet(addPayload)
	}); err != nil {
		t.Fatalf("add dynamic peer: %v", err)
	}

	clientPeer := map[string]any{
		"address":                       "127.0.0.1",
		"port":                          serverPort,
		"public_key":                    serverPrivateKey.PublicKey().String(),
		"pre_shared_key":                presharedKey.String(),
		"allowed_ips":                   []string{"0.0.0.0/0"},
		"persistent_keepalive_interval": 1,
	}
	client := startAWGCore(t, awgCoreConfig(t, "10.77.0.2/32", clientPrivateKey.String(), 0, []map[string]any{clientPeer}))

	tcpAddress := startTCPEchoServer(t)
	udpAddress := startUDPEchoServer(t)
	serverAddress := netip.MustParseAddr("10.77.0.1")
	tcpDestination := netip.AddrPortFrom(serverAddress, tcpAddress.Port())
	assertAWGTCP(t, client, tcpDestination)
	assertAWGUDP(t, client, netip.AddrPortFrom(serverAddress, udpAddress.Port()))

	stableConnection := dialAWGTCP(t, client, tcpDestination)
	defer stableConnection.Close()
	assertEcho(t, stableConnection, []byte("before-peer-change"))

	secondPrivateKey, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		t.Fatal(err)
	}
	secondPresharedKey, err := wgtypes.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	secondAddPayload, err := BuildWireGuardAddPeerUAPI(
		secondPrivateKey.PublicKey().String(),
		secondPresharedKey.String(),
		netip.MustParsePrefix("10.77.0.3/32"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := server.WithWireGuardIPC(awgEndpointTag, func(ipc WireGuardIPC) error {
		return ipc.IpcSet(secondAddPayload)
	}); err != nil {
		t.Fatalf("add second dynamic peer: %v", err)
	}

	secondPeer := map[string]any{
		"address":                       "127.0.0.1",
		"port":                          serverPort,
		"public_key":                    serverPrivateKey.PublicKey().String(),
		"pre_shared_key":                secondPresharedKey.String(),
		"allowed_ips":                   []string{"0.0.0.0/0"},
		"persistent_keepalive_interval": 1,
	}
	secondClient := startAWGCore(t, awgCoreConfig(t, "10.77.0.3/32", secondPrivateKey.String(), 0, []map[string]any{secondPeer}))
	assertAWGTCP(t, secondClient, tcpDestination)
	assertEcho(t, stableConnection, []byte("after-peer-add"))

	secondRemovePayload, err := BuildWireGuardRemovePeerUAPI(secondPrivateKey.PublicKey().String())
	if err != nil {
		t.Fatal(err)
	}
	if err := server.WithWireGuardIPC(awgEndpointTag, func(ipc WireGuardIPC) error {
		return ipc.IpcSet(secondRemovePayload)
	}); err != nil {
		t.Fatalf("remove second dynamic peer: %v", err)
	}
	assertEcho(t, stableConnection, []byte("after-peer-remove"))

	deadline := time.Now().Add(5 * time.Second)
	for {
		var peers []WireGuardPeerSnapshot
		err = server.WithWireGuardIPC(awgEndpointTag, func(ipc WireGuardIPC) error {
			snapshot, getErr := ipc.IpcGet()
			if getErr != nil {
				return getErr
			}
			peers, getErr = ParseWireGuardPeerSnapshot(snapshot)
			return getErr
		})
		if err == nil && len(peers) == 1 && peers[0].LastHandshake > 0 && peers[0].ReceiveBytes > 0 && peers[0].TransmitBytes > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("AWG peer counters not observed: peer_count=%d err=%v", len(peers), err)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func awgCoreConfig(t *testing.T, address, privateKey string, listenPort uint16, peers []map[string]any) []byte {
	t.Helper()
	return awgCoreConfigWith(t, address, privateKey, listenPort, peers, false)
}

// awgCoreConfigWith renders the AWG endpoint config; awg30 adds the AWG 3.0
// timing/header-protection fields so the integration test proves the new
// wireguard-go UAPI keys are accepted end to end.
func awgCoreConfigWith(t *testing.T, address, privateKey string, listenPort uint16, peers []map[string]any, awg30 bool) []byte {
	t.Helper()
	amnezia := map[string]any{
		"jc": 3, "jmin": 10, "jmax": 20,
		"s1": 15, "s2": 18, "s3": 12, "s4": 8,
		"h1": "1000-1099", "h2": "2000-2099", "h3": "3000-3099", "h4": "4000-4099",
		"i1": "<b 0x01020304><r 8>",
	}
	if awg30 {
		amnezia["content_padding_addition"] = "0"
		amnezia["rekey_after_time"] = "120-180"
		amnezia["rekey_timeout"] = "1-5"
		amnezia["reject_after_time"] = "90-120"
		amnezia["keepalive_timeout"] = "5-10"
		amnezia["max_handshake_attempts"] = "20-30"
	}
	endpoint := map[string]any{
		"type":        "wireguard",
		"tag":         awgEndpointTag,
		"address":     []string{address},
		"private_key": privateKey,
		"peers":       peers,
		"amnezia":     amnezia,
	}
	if listenPort != 0 {
		endpoint["listen_port"] = listenPort
	}
	config := map[string]any{
		"log":       map[string]any{"disabled": true},
		"endpoints": []any{endpoint},
		"outbounds": []any{map[string]any{"type": "direct", "tag": "direct"}},
		"route":     map[string]any{"final": "direct"},
	}
	encoded, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

// TestAWG30ServerFieldsAccepted proves the AWG 3.0 timing fields are accepted
// by wireguard-go v0.0.4 via the sing-box UAPI setup path, and that a
// 2.0-style client peer (no new fields) still completes a handshake against a
// 3.0-configured server.
func TestAWG30ServerFieldsAccepted(t *testing.T) {
	serverPrivateKey, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		t.Fatal(err)
	}
	clientPrivateKey, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		t.Fatal(err)
	}
	presharedKey, err := wgtypes.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}

	serverPort := freeUDPPort(t)
	server := startAWGCore(t, awgCoreConfigWith(t, "10.77.0.1/24", serverPrivateKey.String(), serverPort, nil, true))

	addPayload, err := BuildWireGuardAddPeerUAPI(
		clientPrivateKey.PublicKey().String(),
		presharedKey.String(),
		netip.MustParsePrefix("10.77.0.2/32"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := server.WithWireGuardIPC(awgEndpointTag, func(ipc WireGuardIPC) error {
		return ipc.IpcSet(addPayload)
	}); err != nil {
		t.Fatalf("add dynamic peer: %v", err)
	}

	clientPeer := map[string]any{
		"address":                       "127.0.0.1",
		"port":                          serverPort,
		"public_key":                    serverPrivateKey.PublicKey().String(),
		"pre_shared_key":                presharedKey.String(),
		"allowed_ips":                   []string{"0.0.0.0/0"},
		"persistent_keepalive_interval": 1,
	}
	client := startAWGCore(t, awgCoreConfigWith(t, "10.77.0.2/32", clientPrivateKey.String(), 0, []map[string]any{clientPeer}, false))

	tcpAddress := startTCPEchoServer(t)
	serverAddress := netip.MustParseAddr("10.77.0.1")
	assertAWGTCP(t, client, netip.AddrPortFrom(serverAddress, tcpAddress.Port()))

	var peers []WireGuardPeerSnapshot
	deadline := time.Now().Add(5 * time.Second)
	for {
		err = server.WithWireGuardIPC(awgEndpointTag, func(ipc WireGuardIPC) error {
			snapshot, getErr := ipc.IpcGet()
			if getErr != nil {
				return getErr
			}
			peers, getErr = ParseWireGuardPeerSnapshot(snapshot)
			return getErr
		})
		if err == nil && len(peers) == 1 && peers[0].LastHandshake > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("AWG 3.0 handshake not observed: peer_count=%d err=%v", len(peers), err)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func startAWGCore(t *testing.T, config []byte) *Core {
	t.Helper()
	instance := NewCore()
	if err := instance.Start(config); err != nil {
		t.Fatalf("start AWG core: %v", err)
	}
	t.Cleanup(func() {
		if err := instance.Stop(); err != nil {
			t.Errorf("stop AWG core: %v", err)
		}
	})
	return instance
}

func awgSnapshot(t *testing.T, instance *Core) []WireGuardPeerSnapshot {
	t.Helper()
	var peers []WireGuardPeerSnapshot
	if err := instance.WithWireGuardIPC(awgEndpointTag, func(ipc WireGuardIPC) error {
		raw, err := ipc.IpcGet()
		if err != nil {
			return err
		}
		peers, err = ParseWireGuardPeerSnapshot(raw)
		return err
	}); err != nil {
		t.Fatalf("read AWG snapshot: %v", err)
	}
	return peers
}

func awgEndpoint(t *testing.T, instance *Core) interface {
	DialContext(context.Context, string, M.Socksaddr) (net.Conn, error)
} {
	t.Helper()
	runtime, ok := instance.runtime()
	if !ok {
		t.Fatal("core runtime is unavailable")
	}
	endpoint, ok := runtime.endpointManager.Get(awgEndpointTag)
	if !ok {
		t.Fatal("AWG endpoint is unavailable")
	}
	return endpoint
}

func assertAWGTCP(t *testing.T, client *Core, destination netip.AddrPort) {
	t.Helper()
	connection := dialAWGTCP(t, client, destination)
	defer connection.Close()
	assertEcho(t, connection, []byte("awg2-tcp"))
}

func dialAWGTCP(t *testing.T, client *Core, destination netip.AddrPort) net.Conn {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	connection, err := awgEndpoint(t, client).DialContext(ctx, "tcp", M.SocksaddrFrom(destination.Addr(), destination.Port()))
	if err != nil {
		t.Fatalf("dial TCP through AWG: %v", err)
	}
	if err := connection.SetDeadline(time.Now().Add(10 * time.Second)); err != nil {
		connection.Close()
		t.Fatal(err)
	}
	return connection
}

func assertAWGUDP(t *testing.T, client *Core, destination netip.AddrPort) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	connection, err := awgEndpoint(t, client).DialContext(ctx, "udp", M.SocksaddrFrom(destination.Addr(), destination.Port()))
	if err != nil {
		t.Fatalf("dial UDP through AWG: %v", err)
	}
	defer connection.Close()
	if err := connection.SetDeadline(time.Now().Add(10 * time.Second)); err != nil {
		t.Fatal(err)
	}
	assertEcho(t, connection, []byte("awg2-udp"))
}

func assertEcho(t *testing.T, connection net.Conn, payload []byte) {
	t.Helper()
	if _, err := connection.Write(payload); err != nil {
		t.Fatalf("write echo payload: %v", err)
	}
	response := make([]byte, len(payload))
	if _, err := io.ReadFull(connection, response); err != nil {
		t.Fatalf("read echo payload: %v", err)
	}
	if string(response) != string(payload) {
		t.Fatalf("echo response = %q; want %q", response, payload)
	}
}

func startTCPEchoServer(t *testing.T) netip.AddrPort {
	t.Helper()
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	go func() {
		for {
			connection, acceptErr := listener.Accept()
			if acceptErr != nil {
				return
			}
			go func() {
				defer connection.Close()
				_, _ = io.Copy(connection, connection)
			}()
		}
	}()
	return listener.Addr().(*net.TCPAddr).AddrPort()
}

func startUDPEchoServer(t *testing.T) netip.AddrPort {
	t.Helper()
	connection, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = connection.Close() })
	go func() {
		buffer := make([]byte, 64*1024)
		for {
			count, source, readErr := connection.ReadFromUDP(buffer)
			if readErr != nil {
				return
			}
			_, _ = connection.WriteToUDP(buffer[:count], source)
		}
	}()
	return connection.LocalAddr().(*net.UDPAddr).AddrPort()
}

func freeUDPPort(t *testing.T) uint16 {
	t.Helper()
	connection, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	port := connection.LocalAddr().(*net.UDPAddr).Port
	if err := connection.Close(); err != nil {
		t.Fatal(err)
	}
	if port < 1 || port > 65535 {
		t.Fatal(fmt.Errorf("invalid allocated UDP port %d", port))
	}
	return uint16(port)
}
