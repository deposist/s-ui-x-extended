package importxui

import (
	"fmt"

	"github.com/deposist/s-ui-x/database/model"
)

type xuiWireguardSettings struct {
	MTU       int                `json:"mtu"`
	SecretKey string             `json:"secretKey"`
	Peers     []xuiWireguardPeer `json:"peers"`
}

type xuiWireguardPeer struct {
	PublicKey    string   `json:"publicKey"`
	PreSharedKey string   `json:"preSharedKey"`
	AllowedIPs   []string `json:"allowedIPs"`
	KeepAlive    int      `json:"keepAlive"`
}

func mapWireguardEndpoint(row xuiInboundRow) (*model.Endpoint, []string, error) {
	var settings xuiWireguardSettings
	if err := decodeJSON(row.Settings, &settings); err != nil {
		return nil, nil, fmt.Errorf("wireguard %s settings: %w", row.Tag, err)
	}
	if len(settings.Peers) == 0 {
		return nil, []string{fmt.Sprintf("inbound %s: wireguard has no peers; skipped", row.Tag)}, nil
	}
	mtu := settings.MTU
	if mtu == 0 {
		mtu = 1420
	}
	peers := make([]map[string]any, 0, len(settings.Peers))
	for _, peer := range settings.Peers {
		item := map[string]any{
			"address":     "",
			"port":        0,
			"public_key":  peer.PublicKey,
			"allowed_ips": peer.AllowedIPs,
		}
		if peer.PreSharedKey != "" {
			item["pre_shared_key"] = peer.PreSharedKey
		}
		if peer.KeepAlive > 0 {
			item["persistent_keepalive_interval"] = peer.KeepAlive
		}
		peers = append(peers, item)
	}
	options := map[string]any{
		"address":     []string{},
		"listen_port": row.Port,
		"mtu":         mtu,
		"private_key": settings.SecretKey,
		"peers":       peers,
	}
	optionsJSON, err := marshalJSON(options)
	if err != nil {
		return nil, nil, err
	}
	return &model.Endpoint{
		Type:    "wireguard",
		Tag:     row.Tag,
		Options: optionsJSON,
		Ext:     nil,
	}, nil, nil
}
