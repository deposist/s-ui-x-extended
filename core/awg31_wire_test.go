//go:build integration

// Run with:
//
//	go test -tags="integration with_wireguard with_gvisor" ./core -run TestAWG31RandomTrailersHandshakeWire
//
// This is a RED regression test pinning a known wire-format bug in the pinned
// engine dependency shtorm-7/wireguard-go v0.0.5-extended-1.6.1 (used by
// sing-box-extended v1.14.0-extended-2.7.5). It is expected to FAIL until the
// engine pin is fixed; it is never run by CI (integration tag).
//
// Bug: with random_trailers=true, SendHandshakeInitiation/SendHandshakeResponse
// pass a slice of len(MessageSize+trailerLen) into msg.marshal, which strictly
// requires len == MessageSize and returns errMessageLengthMismatch. The fork
// discards that error (`_ = msg.marshal(packet)`), so the handshake message
// body is never written: the wire packet carries padding + ZEROS + random
// trailer, MAC1/MAC2 land inside the trailer region, and the peer cannot even
// classify the packet (type field is 0, outside every H1-H4 range). Handshakes
// fail silently in both directions -> clients hang in "connecting" forever.
// The upstream amnezia-vpn/amneziawg-go client engine (AmneziaVPN >= 5.0.1.5)
// implements random trailers correctly, so the mismatch is server-side only.
package core

import (
	"encoding/binary"
	"encoding/json"
	"net"
	"testing"
	"time"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func TestAWG31RandomTrailersHandshakeWire(t *testing.T) {
	for _, trailers := range []bool{false, true} {
		name := "trailers_off"
		if trailers {
			name = "trailers_on"
		}
		t.Run(name, func(t *testing.T) {
			listener, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			priv, err := wgtypes.GeneratePrivateKey()
			if err != nil {
				t.Fatal(err)
			}
			remote, err := wgtypes.GeneratePrivateKey()
			if err != nil {
				t.Fatal(err)
			}
			peer := map[string]any{
				"address": "127.0.0.1", "port": listener.LocalAddr().(*net.UDPAddr).Port,
				"public_key": remote.PublicKey().String(), "allowed_ips": []string{"0.0.0.0/0"},
				"persistent_keepalive_interval": 1,
			}
			config := awgCoreConfig(t, "10.77.0.2/32", priv.String(), 0, []map[string]any{peer})
			var raw map[string]any
			if err := json.Unmarshal(config, &raw); err != nil {
				t.Fatal(err)
			}
			amnezia := raw["endpoints"].([]any)[0].(map[string]any)["amnezia"].(map[string]any)
			amnezia["random_trailers"] = trailers
			amnezia["disable_cookies"] = true
			config, err = json.Marshal(raw)
			if err != nil {
				t.Fatal(err)
			}
			startAWGCore(t, config)
			if err := listener.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
				t.Fatal(err)
			}
			buf := make([]byte, 65535)
			for {
				n, _, err := listener.ReadFromUDP(buf)
				if err != nil {
					t.Fatal(err)
				}
				// A real initiation from this config: 15 bytes prefix/padding
				// then the 148-byte message whose first 4 bytes are the H1 type.
				if n < 15+148 {
					continue // junk packet
				}
				if trailers && n == 15+148 {
					continue // exact-size packet: trailer length happened to be 0
				}
				header := binary.LittleEndian.Uint32(buf[15:19])
				valid := header >= 1000 && header <= 1099
				t.Logf("UDP payload=%d, initiation type=%d, type_valid=%v, random_trailers=%v, disable_cookies=true", n, header, valid, trailers)
				if !valid {
					t.Fatal("handshake packet body was never marshaled: known engine bug (see file comment)")
				}
				break
			}
		})
	}
}
