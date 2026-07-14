package core

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/netip"
	"strconv"
	"strings"
)

const wireGuardKeySize = 32

// WireGuardPeerSnapshot is the non-secret runtime state returned for one peer.
type WireGuardPeerSnapshot struct {
	PublicKey     string
	AllowedIPs    []netip.Prefix
	LastHandshake int64
	ReceiveBytes  uint64
	TransmitBytes uint64
}

// WireGuardKeyBase64ToHex converts a canonical WireGuard key to its UAPI form.
func WireGuardKeyBase64ToHex(key string) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(key)
	if err != nil || len(decoded) != wireGuardKeySize || base64.StdEncoding.EncodeToString(decoded) != key {
		return "", fmt.Errorf("invalid WireGuard base64 key")
	}
	return hex.EncodeToString(decoded), nil
}

// WireGuardKeyHexToBase64 converts a UAPI key to canonical WireGuard form.
func WireGuardKeyHexToBase64(key string) (string, error) {
	decoded, err := hex.DecodeString(key)
	if err != nil || len(decoded) != wireGuardKeySize {
		return "", fmt.Errorf("invalid WireGuard hex key")
	}
	return base64.StdEncoding.EncodeToString(decoded), nil
}

// BuildWireGuardAddPeerUAPI builds a point update for one peer.
func BuildWireGuardAddPeerUAPI(publicKey, presharedKey string, allowedIP netip.Prefix) (string, error) {
	if !allowedIP.IsValid() {
		return "", fmt.Errorf("invalid WireGuard allowed IP")
	}
	publicKeyHex, err := WireGuardKeyBase64ToHex(publicKey)
	if err != nil {
		return "", fmt.Errorf("public key: %w", err)
	}
	presharedKeyHex, err := WireGuardKeyBase64ToHex(presharedKey)
	if err != nil {
		return "", fmt.Errorf("preshared key: %w", err)
	}
	return "public_key=" + publicKeyHex +
		"\npreshared_key=" + presharedKeyHex +
		"\nreplace_allowed_ips=true" +
		"\nallowed_ip=" + allowedIP.Masked().String() + "\n\n", nil
}

// BuildWireGuardRemovePeerUAPI builds an idempotent removal request for one peer.
func BuildWireGuardRemovePeerUAPI(publicKey string) (string, error) {
	publicKeyHex, err := WireGuardKeyBase64ToHex(publicKey)
	if err != nil {
		return "", fmt.Errorf("public key: %w", err)
	}
	return "public_key=" + publicKeyHex + "\nremove=true\n\n", nil
}

// ParseWireGuardPeerSnapshot parses peer state without retaining device private
// keys or peer preshared keys returned by IpcGet.
func ParseWireGuardPeerSnapshot(input string) ([]WireGuardPeerSnapshot, error) {
	var (
		peers   []WireGuardPeerSnapshot
		current *WireGuardPeerSnapshot
		seen    map[string]bool
	)
	for lineNumber, line := range strings.Split(input, "\n") {
		if line == "" {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok || key == "" {
			return nil, fmt.Errorf("invalid WireGuard UAPI line %d", lineNumber+1)
		}
		if key == "public_key" {
			publicKey, err := WireGuardKeyHexToBase64(value)
			if err != nil {
				return nil, fmt.Errorf("peer public key on line %d: %w", lineNumber+1, err)
			}
			peers = append(peers, WireGuardPeerSnapshot{PublicKey: publicKey})
			current = &peers[len(peers)-1]
			seen = make(map[string]bool)
			continue
		}
		if current == nil {
			if !isWireGuardDeviceField(key) {
				return nil, fmt.Errorf("unknown WireGuard device field %q", key)
			}
			continue
		}
		if key != "allowed_ip" {
			if seen[key] {
				return nil, fmt.Errorf("duplicate WireGuard peer field %q", key)
			}
			seen[key] = true
		}
		switch key {
		case "preshared_key", "protocol_version", "endpoint", "last_handshake_time_nsec", "persistent_keepalive_interval":
			// Known fields that are not needed by the manager. In particular, do
			// not retain the preshared key.
		case "last_handshake_time_sec":
			parsed, err := strconv.ParseInt(value, 10, 64)
			if err != nil || parsed < 0 {
				return nil, fmt.Errorf("invalid WireGuard peer field %q", key)
			}
			current.LastHandshake = parsed
		case "rx_bytes":
			parsed, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid WireGuard peer field %q", key)
			}
			current.ReceiveBytes = parsed
		case "tx_bytes":
			parsed, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid WireGuard peer field %q", key)
			}
			current.TransmitBytes = parsed
		case "allowed_ip":
			prefix, err := netip.ParsePrefix(value)
			if err != nil {
				return nil, fmt.Errorf("invalid WireGuard peer field %q", key)
			}
			current.AllowedIPs = append(current.AllowedIPs, prefix.Masked())
		default:
			return nil, fmt.Errorf("unknown WireGuard peer field %q", key)
		}
	}
	return peers, nil
}

func isWireGuardDeviceField(key string) bool {
	switch key {
	case "private_key", "listen_port", "fwmark", "jc", "jmin", "jmax", "s1", "s2", "s3", "s4",
		"h1", "h2", "h3", "h4", "i1", "i2", "i3", "i4", "i5", "j1", "j2", "j3", "itime":
		return true
	default:
		return false
	}
}
