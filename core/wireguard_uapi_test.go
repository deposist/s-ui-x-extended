package core

import (
	"encoding/base64"
	"net/netip"
	"strings"
	"testing"
)

func testWireGuardKey(fill byte) string {
	return base64.StdEncoding.EncodeToString(bytesOf(fill, wireGuardKeySize))
}

func bytesOf(value byte, count int) []byte {
	result := make([]byte, count)
	for index := range result {
		result[index] = value
	}
	return result
}

func TestWireGuardKeyConversions(t *testing.T) {
	base64Key := testWireGuardKey(0xab)
	hexKey := strings.Repeat("ab", wireGuardKeySize)

	gotHex, err := WireGuardKeyBase64ToHex(base64Key)
	if err != nil || gotHex != hexKey {
		t.Fatalf("WireGuardKeyBase64ToHex() = %q, %v", gotHex, err)
	}
	gotBase64, err := WireGuardKeyHexToBase64(hexKey)
	if err != nil || gotBase64 != base64Key {
		t.Fatalf("WireGuardKeyHexToBase64() = %q, %v", gotBase64, err)
	}
}

func TestWireGuardKeyConversionsRejectInvalidKeysWithoutEcho(t *testing.T) {
	for name, convert := range map[string]func(string) (string, error){
		"base64": WireGuardKeyBase64ToHex,
		"hex":    WireGuardKeyHexToBase64,
	} {
		t.Run(name, func(t *testing.T) {
			secret := "not-a-valid-secret-key" // #nosec G101 -- malformed test input, not a credential.
			_, err := convert(secret)
			if err == nil {
				t.Fatal("expected invalid key error")
			}
			if strings.Contains(err.Error(), secret) {
				t.Fatalf("error leaked input key: %v", err)
			}
		})
	}
}

func TestBuildWireGuardAddPeerUAPI(t *testing.T) {
	publicKey := testWireGuardKey(0x11)
	presharedKey := testWireGuardKey(0x22)
	payload, err := BuildWireGuardAddPeerUAPI(publicKey, presharedKey, netip.MustParsePrefix("10.77.0.2/32"))
	if err != nil {
		t.Fatal(err)
	}
	want := "public_key=" + strings.Repeat("11", wireGuardKeySize) +
		"\npreshared_key=" + strings.Repeat("22", wireGuardKeySize) +
		"\nreplace_allowed_ips=true\nallowed_ip=10.77.0.2/32\n\n"
	if payload != want {
		t.Fatalf("payload = %q; want %q", payload, want)
	}
}

func TestBuildWireGuardRemovePeerUAPI(t *testing.T) {
	payload, err := BuildWireGuardRemovePeerUAPI(testWireGuardKey(0x33))
	if err != nil {
		t.Fatal(err)
	}
	want := "public_key=" + strings.Repeat("33", wireGuardKeySize) + "\nremove=true\n\n"
	if payload != want {
		t.Fatalf("payload = %q; want %q", payload, want)
	}
}

func TestParseWireGuardPeerSnapshot(t *testing.T) {
	publicKeyHex := strings.Repeat("44", wireGuardKeySize)
	input := "private_key=" + strings.Repeat("aa", wireGuardKeySize) + "\n" +
		"listen_port=51820\njc=5\ni1=<b 0x01>\n" +
		"header_protection_key=" + strings.Repeat("cc", wireGuardKeySize) + "\n" +
		"content_padding_addition=0\nrekey_after_time=120-180\nrekey_timeout=1-5\n" +
		"reject_after_time=90-120\nkeepalive_timeout=5-10\nmax_handshake_attempts=20-30\n" +
		"public_key=" + publicKeyHex + "\n" +
		"preshared_key=" + strings.Repeat("bb", wireGuardKeySize) + "\n" +
		"protocol_version=1\nlast_handshake_time_sec=123\nlast_handshake_time_nsec=7\n" +
		"tx_bytes=456\nrx_bytes=789\npersistent_keepalive_interval=25\n" +
		"allowed_ip=10.77.0.2/32\nallowed_ip=10.77.1.0/24\n"

	peers, err := ParseWireGuardPeerSnapshot(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(peers) != 1 {
		t.Fatalf("peer count = %d; want 1", len(peers))
	}
	peer := peers[0]
	if peer.PublicKey != testWireGuardKey(0x44) || peer.LastHandshake != 123 || peer.ReceiveBytes != 789 || peer.TransmitBytes != 456 {
		t.Fatalf("unexpected peer snapshot: %+v", peer)
	}
	if len(peer.AllowedIPs) != 2 || peer.AllowedIPs[0].String() != "10.77.0.2/32" || peer.AllowedIPs[1].String() != "10.77.1.0/24" {
		t.Fatalf("unexpected allowed IPs: %v", peer.AllowedIPs)
	}
}

// The AWG 2.0 device fields j1/j2/j3/itime no longer exist in wireguard-go
// v0.0.4 (removed upstream), so IpcGet output containing them must be
// rejected as unknown device fields instead of silently accepted.
func TestParseWireGuardPeerSnapshotRejectsAWG20DeviceFields(t *testing.T) {
	for _, key := range []string{"j1", "j2", "j3", "itime"} {
		input := "private_key=" + strings.Repeat("aa", wireGuardKeySize) + "\n" +
			key + "=something\npublic_key=" + strings.Repeat("44", wireGuardKeySize) + "\n"
		if _, err := ParseWireGuardPeerSnapshot(input); err == nil {
			t.Fatalf("ParseWireGuardPeerSnapshot accepted removed AWG 2.0 device field %q", key)
		}
	}
}

func TestWireGuardAllowedIPsRejectHostBits(t *testing.T) {
	_, err := BuildWireGuardAddPeerUAPI(testWireGuardKey(0x11), testWireGuardKey(0x22), netip.MustParsePrefix("192.168.1.5/24"))
	if err == nil {
		t.Fatal("BuildWireGuardAddPeerUAPI accepted a prefix with host bits")
	}

	input := "public_key=" + strings.Repeat("44", wireGuardKeySize) + "\nallowed_ip=192.168.1.5/24\n"
	if _, err = ParseWireGuardPeerSnapshot(input); err == nil {
		t.Fatal("ParseWireGuardPeerSnapshot accepted a prefix with host bits")
	}
}

func TestParseWireGuardPeerSnapshotRejectsMalformedInputWithoutEcho(t *testing.T) {
	secret := "private-material-that-must-not-leak"
	inputs := []string{
		"malformed-line\n",
		"unknown_device_field=" + secret + "\n",
		"public_key=" + strings.Repeat("55", wireGuardKeySize) + "\nrx_bytes=" + secret + "\n",
		"public_key=" + strings.Repeat("55", wireGuardKeySize) + "\nrx_bytes=1\nrx_bytes=2\n",
	}
	for _, input := range inputs {
		_, err := ParseWireGuardPeerSnapshot(input)
		if err == nil {
			t.Fatalf("ParseWireGuardPeerSnapshot(%q) succeeded", input)
		}
		if strings.Contains(err.Error(), secret) {
			t.Fatalf("error leaked UAPI value: %v", err)
		}
	}
}
