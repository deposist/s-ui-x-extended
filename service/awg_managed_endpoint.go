package service

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/netip"
	"sort"
	"time"

	"github.com/deposist/s-ui-x-extended/database/model"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	"gorm.io/gorm"
)

type AWGManagedEndpoint struct {
	ServerAddress   netip.Addr
	ServerPublicKey string
}

type awgManagedEndpointOptions struct {
	Address    []string `json:"address"`
	PrivateKey string   `json:"private_key"`
	ListenPort uint16   `json:"listen_port"`
	MTU        uint32   `json:"mtu"`
	Peers      []struct {
		Address                     string   `json:"address,omitempty"`
		Port                        uint16   `json:"port,omitempty"`
		PublicKey                   string   `json:"public_key,omitempty"`
		PresharedKey                string   `json:"pre_shared_key,omitempty"`
		AllowedIPs                  []string `json:"allowed_ips,omitempty"`
		PersistentKeepaliveInterval uint32   `json:"persistent_keepalive_interval,omitempty"`
	} `json:"peers"`
	Amnezia *struct {
		JC   int    `json:"jc"`
		JMin int    `json:"jmin"`
		JMax int    `json:"jmax"`
		S1   int    `json:"s1"`
		S2   int    `json:"s2"`
		S3   int    `json:"s3"`
		S4   int    `json:"s4"`
		H1   any    `json:"h1"`
		H2   any    `json:"h2"`
		H3   any    `json:"h3"`
		H4   any    `json:"h4"`
		I1   string `json:"i1"`
		I2   string `json:"i2"`
		I3   string `json:"i3"`
		I4   string `json:"i4"`
		I5   string `json:"i5"`

		HeaderProtectionKey    string `json:"header_protection_key"`
		ContentPaddingAddition any    `json:"content_padding_addition"`
		RekeyAfterTime         any    `json:"rekey_after_time"`
		RekeyTimeout           any    `json:"rekey_timeout"`
		RejectAfterTime        any    `json:"reject_after_time"`
		KeepaliveTimeout       any    `json:"keepalive_timeout"`
		MaxHandshakeAttempts   any    `json:"max_handshake_attempts"`
	} `json:"amnezia"`
}

// AWGPersistedPeer is the restart-safe peer representation stored in the
// managed endpoint options. It is deliberately unexported through API models.
type AWGPersistedPeer struct {
	PublicKey                   string
	AllowedIPs                  []string
	PersistentKeepaliveInterval uint32
}

// SyncAWGManagedEndpointPeers updates only peers and uses the original options
// bytes as an optimistic concurrency guard against a simultaneous admin edit.
func SyncAWGManagedEndpointPeers(db *gorm.DB, settings AWGSettings, peers []AWGPersistedPeer) error {
	if db == nil {
		return fmt.Errorf("managed AWG endpoint database is unavailable")
	}
	var endpoint model.Endpoint
	if err := db.Where("tag = ?", settings.EndpointTag).First(&endpoint).Error; err != nil {
		return fmt.Errorf("load managed AWG endpoint")
	}
	original := append([]byte(nil), endpoint.Options...)
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(original, &raw); err != nil {
		return fmt.Errorf("managed AWG endpoint options are invalid")
	}
	type persistedPeer struct {
		PublicKey                   string   `json:"public_key"`
		AllowedIPs                  []string `json:"allowed_ips"`
		PersistentKeepaliveInterval uint32   `json:"persistent_keepalive_interval,omitempty"`
	}
	encodedPeers := make([]persistedPeer, 0, len(peers))
	for _, peer := range peers {
		encodedPeers = append(encodedPeers, persistedPeer{
			PublicKey:                   peer.PublicKey,
			AllowedIPs:                  append([]string(nil), peer.AllowedIPs...),
			PersistentKeepaliveInterval: peer.PersistentKeepaliveInterval,
		})
	}
	peersJSON, err := json.Marshal(encodedPeers)
	if err != nil {
		return fmt.Errorf("encode managed AWG peers")
	}
	raw["peers"] = peersJSON
	next, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return fmt.Errorf("encode managed AWG endpoint")
	}
	if bytes.Equal(bytes.TrimSpace(original), bytes.TrimSpace(next)) {
		return nil
	}
	result := db.Model(&model.Endpoint{}).Where("id = ? AND CAST(options AS BLOB) = ?", endpoint.Id, original).Update("options", json.RawMessage(next))
	if result.Error != nil {
		return fmt.Errorf("persist managed AWG peers")
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("managed AWG endpoint changed during peer sync")
	}
	return nil
}

func injectAWGManagedEndpointPeers(db *gorm.DB, settings AWGSettings, endpoints []json.RawMessage) ([]json.RawMessage, error) {
	return injectAWGManagedEndpointPeersForEndpoint(db, settings, 0, endpoints)
}

func injectAWGManagedEndpointPeersForEndpoint(db *gorm.DB, settings AWGSettings, endpointID uint, endpoints []json.RawMessage) ([]json.RawMessage, error) {
	if !settings.Enabled {
		return endpoints, nil
	}
	type corePeer struct {
		PublicKey    string   `json:"public_key"`
		PresharedKey string   `json:"pre_shared_key"`
		AllowedIPs   []string `json:"allowed_ips"`
	}
	peers := []corePeer{}
	var buildErr error
	cipher, err := NewAWGCipherFromEnv()
	if err != nil {
		buildErr = ErrAWGEncryptionUnavailable
	} else {
		var devices []model.AWGDevice
		deviceQuery := db.Where("desired_enabled = ?", true)
		if endpointID > 0 {
			deviceQuery = deviceQuery.Where("endpoint_id = ?", endpointID)
		}
		if err := deviceQuery.Find(&devices).Error; err != nil {
			buildErr = fmt.Errorf("load managed AWG peers")
		} else {
			for _, device := range devices {
				var client model.Client
				now := time.Now().Unix()
				if err := db.First(&client, device.ClientId).Error; err != nil || !clientIsActiveAt(client, now) || deviceExpiredAt(device, now) {
					continue
				}
				psk, err := cipher.Decrypt(device.ClientId, device.CryptoContext, device.PSKEnc)
				if err != nil || len(psk) != 32 {
					clear(psk)
					buildErr = ErrAWGEncryptionUnavailable
					peers = nil
					break
				}
				peers = append(peers, corePeer{PublicKey: device.PublicKey, PresharedKey: base64.StdEncoding.EncodeToString(psk), AllowedIPs: []string{device.IPv4Address + "/32"}})
				clear(psk)
			}
		}
	}
	sort.Slice(peers, func(i, j int) bool { return peers[i].PublicKey < peers[j].PublicKey })
	for i, rawEndpoint := range endpoints {
		var endpoint map[string]json.RawMessage
		if err := json.Unmarshal(rawEndpoint, &endpoint); err != nil {
			continue
		}
		var tag string
		_ = json.Unmarshal(endpoint["tag"], &tag)
		if tag != settings.EndpointTag {
			continue
		}
		encodedPeers, err := json.Marshal(peers)
		if err != nil {
			return endpoints, fmt.Errorf("encode managed AWG peers")
		}
		endpoint["peers"] = encodedPeers
		encodedEndpoint, err := json.Marshal(endpoint)
		if err != nil {
			return endpoints, fmt.Errorf("encode managed AWG endpoint")
		}
		endpoints[i] = encodedEndpoint
	}
	return endpoints, buildErr
}

func LoadAWGManagedEndpoint(db *gorm.DB, settings AWGSettings) (AWGManagedEndpoint, error) {
	managed, _, err := loadAWGManagedEndpoint(db, settings)
	return managed, err
}

func loadAWGManagedEndpoint(db *gorm.DB, settings AWGSettings) (AWGManagedEndpoint, awgManagedEndpointOptions, error) {
	var managed AWGManagedEndpoint
	var options awgManagedEndpointOptions
	if db == nil {
		return managed, options, fmt.Errorf("managed AWG endpoint database is unavailable")
	}
	if settings.EndpointTag == "" {
		return managed, options, fmt.Errorf("managed AWG endpoint tag is empty")
	}
	if !settings.Subnet.IsValid() || !settings.Subnet.Addr().Is4() {
		return managed, options, fmt.Errorf("managed AWG endpoint subnet is invalid")
	}

	var endpoint model.Endpoint
	if err := db.Where("tag = ?", settings.EndpointTag).First(&endpoint).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return managed, options, fmt.Errorf("managed AWG endpoint not found")
		}
		return managed, options, fmt.Errorf("load managed AWG endpoint")
	}
	if endpoint.Type != "wireguard" {
		return managed, options, fmt.Errorf("managed AWG endpoint must use WireGuard")
	}
	if err := json.Unmarshal(endpoint.Options, &options); err != nil {
		return managed, options, fmt.Errorf("managed AWG endpoint options are invalid")
	}

	privateKey, err := wgtypes.ParseKey(options.PrivateKey)
	if err != nil {
		return managed, options, fmt.Errorf("managed AWG endpoint private key is invalid")
	}
	if len(options.Address) == 0 {
		return managed, options, fmt.Errorf("managed AWG endpoint requires an IPv4 address")
	}
	serverPrefix, err := netip.ParsePrefix(options.Address[0])
	if err != nil || !serverPrefix.Addr().Is4() {
		return managed, options, fmt.Errorf("managed AWG endpoint address does not match awgSubnet")
	}
	subnet := settings.Subnet.Masked()
	serverAddress := serverPrefix.Addr()
	broadcastAddress := uint32IPv4(ipv4Uint32(subnet.Addr()) | (^uint32(0) >> subnet.Bits()))
	if !subnet.Contains(serverAddress) || serverAddress == subnet.Addr() || serverAddress == broadcastAddress {
		return managed, options, fmt.Errorf("managed AWG endpoint address does not match awgSubnet")
	}

	managed.ServerAddress = serverAddress
	managed.ServerPublicKey = privateKey.PublicKey().String()
	return managed, options, nil
}
