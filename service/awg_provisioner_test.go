package service

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/netip"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x-extended/core"
)

type fakeAWGIPC struct {
	get      string
	getErr   error
	setErr   error
	getCalls int
	sets     []string
	afterSet func(string)
}

func (f *fakeAWGIPC) IpcGet() (string, error) {
	f.getCalls++
	return f.get, f.getErr
}

func (f *fakeAWGIPC) IpcSet(payload string) error {
	f.sets = append(f.sets, payload)
	if f.setErr != nil {
		return f.setErr
	}
	if f.afterSet != nil {
		f.afterSet(payload)
	}
	return nil
}

func testAWGKey(fill byte) string {
	return base64.StdEncoding.EncodeToString(bytesOf(fill, 32))
}

func testAWGKeyHex(fill byte) string {
	return hex.EncodeToString(bytesOf(fill, 32))
}

func bytesOf(fill byte, size int) []byte {
	value := make([]byte, size)
	for index := range value {
		value[index] = fill
	}
	return value
}

func newTestAWGProvisioner(ipc core.WireGuardIPC, calls *int) AWGProvisioner {
	return newAWGProvisioner("managed-awg", func(_ context.Context, tag string, fn func(core.WireGuardIPC) error) error {
		*calls++
		if tag != "managed-awg" {
			return errors.New("unexpected endpoint tag")
		}
		return fn(ipc)
	})
}

func TestAWGProvisionerSnapshot(t *testing.T) {
	publicKey := testAWGKey(1)
	ipc := &fakeAWGIPC{get: "private_key=" + testAWGKeyHex(9) + "\n" +
		"public_key=" + testAWGKeyHex(1) + "\n" +
		"allowed_ip=10.20.0.2/32\n" +
		"last_handshake_time_sec=123\nrx_bytes=456\ntx_bytes=789\n"}
	calls := 0
	provisioner := newTestAWGProvisioner(ipc, &calls)

	snapshot, err := provisioner.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("Snapshot() error = %v", err)
	}
	if calls != 1 {
		t.Fatalf("WithIPC calls = %d, want 1", calls)
	}
	peer, ok := snapshot[publicKey]
	if !ok {
		t.Fatalf("Snapshot() missing canonical base64 public key %q", publicKey)
	}
	if len(peer.AllowedIPs) != 1 || peer.AllowedIPs[0] != netip.MustParsePrefix("10.20.0.2/32") {
		t.Fatalf("AllowedIPs = %v", peer.AllowedIPs)
	}
	if peer.LastHandshake != 123 || peer.ReceiveBytes != 456 || peer.TransmitBytes != 789 {
		t.Fatalf("peer counters = %+v", peer)
	}
}

func TestAWGProvisionerSnapshotRejectsDuplicatePublicKey(t *testing.T) {
	ipc := &fakeAWGIPC{get: "public_key=" + testAWGKeyHex(1) + "\n" +
		"allowed_ip=10.20.0.2/32\n" +
		"public_key=" + testAWGKeyHex(1) + "\n" +
		"allowed_ip=10.20.0.3/32\n"}
	calls := 0
	provisioner := newTestAWGProvisioner(ipc, &calls)

	_, err := provisioner.Snapshot(context.Background())
	if err == nil || strings.Contains(err.Error(), testAWGKey(1)) || strings.Contains(err.Error(), testAWGKeyHex(1)) {
		t.Fatalf("Snapshot() error = %q, want sanitized duplicate-key error", err)
	}
}

func TestAWGProvisionerAddUsesPointUpdateAndVerifies(t *testing.T) {
	publicKey := testAWGKey(1)
	presharedKey := testAWGKey(2)
	allowedIP := netip.MustParsePrefix("10.20.0.2/32")
	ipc := &fakeAWGIPC{}
	ipc.afterSet = func(string) {
		ipc.get = "public_key=" + testAWGKeyHex(1) + "\nallowed_ip=10.20.0.2/32\n"
	}
	calls := 0
	provisioner := newTestAWGProvisioner(ipc, &calls)

	err := provisioner.Add(context.Background(), AWGPeerSpec{
		PublicKey:    publicKey,
		PresharedKey: presharedKey,
		AllowedIP:    allowedIP,
	})
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if calls != 1 {
		t.Fatalf("WithIPC calls = %d, want mutation and verification in one callback", calls)
	}
	if len(ipc.sets) != 1 {
		t.Fatalf("IpcSet calls = %d, want 1", len(ipc.sets))
	}
	payload := ipc.sets[0]
	for _, expected := range []string{
		"public_key=" + testAWGKeyHex(1),
		"preshared_key=" + testAWGKeyHex(2),
		"replace_allowed_ips=true",
		"allowed_ip=10.20.0.2/32",
	} {
		if !strings.Contains(payload, expected) {
			t.Errorf("IpcSet payload missing %q", expected)
		}
	}
	if strings.Contains(payload, publicKey) || strings.Contains(payload, presharedKey) {
		t.Error("IpcSet payload contains domain base64 key")
	}
}

func TestAWGProvisionerAddDetectsFalseIpcSetSuccess(t *testing.T) {
	ipc := &fakeAWGIPC{}
	calls := 0
	provisioner := newTestAWGProvisioner(ipc, &calls)

	err := provisioner.Add(context.Background(), AWGPeerSpec{
		PublicKey:    testAWGKey(1),
		PresharedKey: testAWGKey(2),
		AllowedIP:    netip.MustParsePrefix("10.20.0.2/32"),
	})
	if err == nil || !strings.Contains(err.Error(), "verification") {
		t.Fatalf("Add() error = %v, want verification failure", err)
	}
}

func TestAWGProvisionerAddVerifiesAfterCancellationDuringIpcSet(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	ipc := &fakeAWGIPC{}
	ipc.afterSet = func(string) {
		cancel()
		ipc.get = "public_key=" + testAWGKeyHex(1) + "\nallowed_ip=10.20.0.2/32\n"
	}
	calls := 0
	provisioner := newTestAWGProvisioner(ipc, &calls)

	err := provisioner.Add(ctx, AWGPeerSpec{
		PublicKey:    testAWGKey(1),
		PresharedKey: testAWGKey(2),
		AllowedIP:    netip.MustParsePrefix("10.20.0.2/32"),
	})
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if ipc.getCalls != 1 {
		t.Fatalf("IpcGet calls = %d, want verification after completed mutation", ipc.getCalls)
	}
}

func TestAWGProvisionerAddRejectsNonHostIPv4AllowedIP(t *testing.T) {
	for _, allowedIP := range []netip.Prefix{
		netip.MustParsePrefix("10.20.0.0/24"),
		netip.MustParsePrefix("2001:db8::2/128"),
	} {
		t.Run(allowedIP.String(), func(t *testing.T) {
			ipc := &fakeAWGIPC{}
			calls := 0
			provisioner := newTestAWGProvisioner(ipc, &calls)

			err := provisioner.Add(context.Background(), AWGPeerSpec{
				PublicKey:    testAWGKey(1),
				PresharedKey: testAWGKey(2),
				AllowedIP:    allowedIP,
			})
			if err == nil {
				t.Fatal("Add() returned nil error")
			}
			if calls != 0 || len(ipc.sets) != 0 {
				t.Fatalf("invalid address reached IPC: WithIPC calls = %d, IpcSet calls = %d", calls, len(ipc.sets))
			}
		})
	}
}

func TestAWGProvisionerRemoveIsIdempotentAndVerifies(t *testing.T) {
	publicKey := testAWGKey(1)
	ipc := &fakeAWGIPC{}
	calls := 0
	provisioner := newTestAWGProvisioner(ipc, &calls)

	if err := provisioner.Remove(context.Background(), publicKey); err != nil {
		t.Fatalf("Remove() absent peer error = %v", err)
	}
	if calls != 1 || len(ipc.sets) != 1 {
		t.Fatalf("WithIPC calls = %d, IpcSet calls = %d; want 1, 1", calls, len(ipc.sets))
	}
	if !strings.Contains(ipc.sets[0], "public_key="+testAWGKeyHex(1)+"\nremove=true") {
		t.Fatalf("remove payload = %q", ipc.sets[0])
	}

	ipc.get = "public_key=" + testAWGKeyHex(1) + "\nallowed_ip=10.20.0.2/32\n"
	if err := provisioner.Remove(context.Background(), publicKey); err == nil || !strings.Contains(err.Error(), "verification") {
		t.Fatalf("Remove() false success error = %v, want verification failure", err)
	}
}

func TestAWGProvisionerRemoveVerifiesAfterCancellationDuringIpcSet(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	ipc := &fakeAWGIPC{get: "public_key=" + testAWGKeyHex(1) + "\nallowed_ip=10.20.0.2/32\n"}
	ipc.afterSet = func(string) {
		cancel()
		ipc.get = ""
	}
	calls := 0
	provisioner := newTestAWGProvisioner(ipc, &calls)

	if err := provisioner.Remove(ctx, testAWGKey(1)); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	if ipc.getCalls != 1 {
		t.Fatalf("IpcGet calls = %d, want verification after completed mutation", ipc.getCalls)
	}
}

// An endpoint-scoped provisioner is constructed with the managed endpoint's
// own tag (awg_endpoint_manager.go). The legacy global AWG settings must never
// replace it: when the legacy scheme is disabled its awgEndpointTag is empty,
// which previously sent every scoped reconcile to "AWG endpoint is
// unavailable" and collapsed into "AWG reconcile failed".
func TestAWGProvisionerEndpointTagWinsOverGlobalSettings(t *testing.T) {
	initSettingTestDB(t)
	runtime := NewRuntime(nil)
	provisioner := NewAWGProvisioner(runtime, "ep-scoped")
	var gotTag string
	provisioner.(*awgProvisioner).withIPC = func(_ context.Context, tag string, fn func(core.WireGuardIPC) error) error {
		gotTag = tag
		return fn(&fakeAWGIPC{})
	}
	if _, err := provisioner.Snapshot(context.Background()); err != nil {
		t.Fatalf("Snapshot() error = %v", err)
	}
	if gotTag != "ep-scoped" {
		t.Fatalf("IPC endpoint tag = %q, want %q (legacy global AWG settings must not override an endpoint-scoped tag)", gotTag, "ep-scoped")
	}
}

// An endpoint-scoped provisioner always carries its endpoint tag; an empty tag
// can only mean a misconstructed manager and must fail closed.
func TestAWGProvisionerEmptyTagFailsClosed(t *testing.T) {
	provisioner := NewAWGProvisioner(NewRuntime(nil), "")
	if _, err := provisioner.Snapshot(context.Background()); err == nil {
		t.Fatal("empty endpoint tag must not reach the IPC layer")
	}
}

func TestAWGProvisionerSanitizesIPCErrors(t *testing.T) {
	secret := "private-and-preshared-secret"
	for _, testCase := range []struct {
		name string
		ipc  *fakeAWGIPC
		run  func(AWGProvisioner) error
	}{
		{
			name: "get",
			ipc:  &fakeAWGIPC{getErr: errors.New("IpcGet leaked " + secret)},
			run: func(provisioner AWGProvisioner) error {
				_, err := provisioner.Snapshot(context.Background())
				return err
			},
		},
		{
			name: "set",
			ipc:  &fakeAWGIPC{setErr: errors.New("IpcSet leaked " + secret)},
			run: func(provisioner AWGProvisioner) error {
				return provisioner.Add(context.Background(), AWGPeerSpec{
					PublicKey:    testAWGKey(1),
					PresharedKey: testAWGKey(2),
					AllowedIP:    netip.MustParsePrefix("10.20.0.2/32"),
				})
			},
		},
		{
			name: "wrapped context error",
			ipc:  &fakeAWGIPC{getErr: fmt.Errorf("%s: %w", secret, context.Canceled)},
			run: func(provisioner AWGProvisioner) error {
				_, err := provisioner.Snapshot(context.Background())
				return err
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			calls := 0
			err := testCase.run(newTestAWGProvisioner(testCase.ipc, &calls))
			if err == nil {
				t.Fatal("operation returned nil error")
			}
			if strings.Contains(err.Error(), secret) || strings.Contains(err.Error(), "IpcGet leaked") || strings.Contains(err.Error(), "IpcSet leaked") {
				t.Fatalf("operation leaked raw IPC error: %q", err)
			}
		})
	}
}
