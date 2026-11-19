package service

import (
	"encoding/json"

	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/deposist/s-ui-x-extended/util"
	"gorm.io/gorm"
)

type peerPublicTLS struct {
	Enabled    bool     `json:"enabled"`
	ServerName string   `json:"server_name,omitempty"`
	ALPN       []string `json:"alpn,omitempty"`
}

type peerPublicLink struct {
	Port uint16         `json:"port"`
	TLS  *peerPublicTLS `json:"tls,omitempty"`
}

func (link peerPublicLink) valid() bool {
	return link.Port > 0 && (link.TLS == nil || link.TLS.Enabled && validPeerHost(link.TLS.ServerName) && len(link.TLS.ALPN) <= 16)
}

func announcePeerInbounds(db *gorm.DB) ([]PeerInbound, error) {
	var rows []model.Inbound
	if err := db.Preload("Tls").Order("id").Limit(513).Find(&rows).Error; err != nil {
		return nil, err
	}
	inbounds := make([]PeerInbound, 0, len(rows))
	for _, row := range rows {
		entry := PeerInbound{ID: row.Id, Tag: row.Tag, Type: row.Type}
		entry.Link = publicPeerLink(row)
		inbounds = append(inbounds, entry)
	}
	return inbounds, nil
}

func publicPeerLink(row model.Inbound) *peerPublicLink {
	if row.Type != "vless" && row.Type != "vmess" && row.Type != "trojan" {
		return nil
	}
	var options struct {
		Port      uint16 `json:"listen_port"`
		Transport struct {
			Type string `json:"type"`
		} `json:"transport"`
	}
	if json.Unmarshal(row.Options, &options) != nil || (options.Transport.Type != "" && options.Transport.Type != "tcp") {
		return nil
	}
	link := &peerPublicLink{Port: options.Port}
	if row.TlsId != 0 {
		var server struct {
			peerPublicTLS
			Reality struct {
				Enabled bool `json:"enabled"`
			} `json:"reality"`
		}
		if row.Tls == nil || json.Unmarshal(row.Tls.Server, &server) != nil || server.Reality.Enabled {
			return nil
		}
		if server.Enabled {
			link.TLS = &server.peerPublicTLS
		}
	}
	if !link.valid() {
		return nil
	}
	return link
}

func pairedClientLinks(db *gorm.DB, clientID uint) []map[string]string {
	var nodes []model.Node
	if err := db.Model(&model.Node{}).
		Where("enabled AND enrollment_digest <> '' AND peer_revision <> '' AND link_host <> ''").
		Where("uuid IN (?)", db.Model(&model.PeerAssignment{}).Select("node_uuid").Where("client_id = ?", clientID)).
		Order("name, id").Find(&nodes).Error; err != nil {
		return nil
	}
	var links []map[string]string
	for _, node := range nodes {
		entries := DecodeNodeInventory(node.Inventory).Inbounds
		inbounds := make([]PeerInbound, 0, len(entries))
		byID := make(map[uint]PeerInbound, len(entries))
		for _, entry := range entries {
			inbound := PeerInbound{ID: entry.ID, Type: entry.Type, Tag: entry.Tag, Link: entry.PeerLink}
			inbounds = append(inbounds, inbound)
			byID[entry.ID] = inbound
		}
		state, err := desiredPeerUsers(db, node.UUID, inbounds)
		if err != nil || state.bindInventory(node.Inventory, node.LinkHost) != nil || state.Revision != node.PeerRevision {
			continue
		}
		for _, user := range state.Users {
			entry := byID[user.InboundID]
			if user.ClientID != clientID || entry.Link == nil {
				continue
			}
			inbound := model.Inbound{Type: entry.Type, Tag: entry.Tag, Addrs: json.RawMessage(`[]`)}
			inbound.Options, _ = json.Marshal(map[string]uint16{"listen_port": entry.Link.Port})
			if entry.Link.TLS != nil {
				tls, _ := json.Marshal(entry.Link.TLS)
				inbound.TlsId, inbound.Tls = 1, &model.Tls{Server: tls, Client: json.RawMessage(`{}`)}
			}
			clientConfig, _ := json.Marshal(map[string]peerUser{user.Type: user}) // #nosec G117 -- The subscription includes only the requesting client's proxy credential.
			for _, uri := range util.LinkGenerator(clientConfig, &inbound, node.LinkHost) {
				links = append(links, map[string]string{"remark": node.Name + " " + entry.Tag, "type": "node", "uri": uri})
			}
		}
	}
	return links
}
