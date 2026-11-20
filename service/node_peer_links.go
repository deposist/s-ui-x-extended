package service

import (
	"crypto/ecdh"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/deposist/s-ui-x-extended/core/capabilities"
	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/deposist/s-ui-x-extended/util"
	singjson "github.com/sagernet/sing/common/json"
	"github.com/sagernet/sing/common/json/badoption"
	"gorm.io/gorm"
)

type peerPublicTLS struct {
	Enabled    bool               `json:"enabled"`
	ServerName string             `json:"server_name,omitempty"`
	ALPN       []string           `json:"alpn,omitempty"`
	Reality    *peerPublicReality `json:"reality,omitempty"`
}

type peerPublicReality struct {
	PublicKey   string `json:"public_key"`
	ShortID     string `json:"short_id,omitempty"`
	Fingerprint string `json:"fingerprint"`
}

// Only settings that can be represented by the existing share links are announced.
// Transport headers other than Host may contain credentials.
type peerPublicTransport struct {
	Type        string   `json:"type"`
	Host        string   `json:"host,omitempty"`
	Hosts       []string `json:"hosts,omitempty"`
	Path        string   `json:"path,omitempty"`
	ServiceName string   `json:"service_name,omitempty"`
}

type peerPublicProxyOptions struct {
	UpMbps            uint32 `json:"up_mbps,omitempty"`
	DownMbps          uint32 `json:"down_mbps,omitempty"`
	CongestionControl string `json:"congestion_control,omitempty"`
	UDPRelayMode      string `json:"udp_relay_mode,omitempty"`
}

type peerPublicLink struct {
	Port      uint16                  `json:"port"`
	TLS       *peerPublicTLS          `json:"tls,omitempty"`
	Transport *peerPublicTransport    `json:"transport,omitempty"`
	Options   *peerPublicProxyOptions `json:"options,omitempty"`
}

func (link peerPublicLink) valid(inboundType string) bool {
	if link.Port == 0 || !peerProtocolSupported(inboundType) || !link.TLS.valid() || !link.Transport.valid() {
		return false
	}
	reality := link.TLS != nil && link.TLS.Reality != nil
	if link.Transport != nil && link.Transport.Type == "quic" && (link.TLS == nil || reality) {
		return false
	}
	switch inboundType {
	case "vless", "trojan":
		return link.Options == nil
	case "vmess":
		return link.Options == nil && !reality
	case "anytls":
		return link.Transport == nil && link.Options == nil && link.TLS != nil
	case "hysteria", "hysteria2", "tuic":
		return link.Transport == nil && link.TLS != nil && !reality && link.Options.valid(inboundType)
	default:
		return false
	}
}

func (tls *peerPublicTLS) valid() bool {
	if tls == nil {
		return true
	}
	if !tls.Enabled || !validPeerHost(tls.ServerName) || len(tls.ALPN) > 16 {
		return false
	}
	for _, value := range tls.ALPN {
		if !validPeerLinkText(value, 255) || strings.Contains(value, ",") {
			return false
		}
	}
	if tls.Reality == nil {
		return true
	}
	publicKey, err := base64.RawURLEncoding.Strict().DecodeString(tls.Reality.PublicKey)
	return err == nil && len(publicKey) == 32 && tls.ServerName != "" &&
		validPeerShortID(tls.Reality.ShortID) && tls.Reality.Fingerprint != "" &&
		validPeerLinkText(tls.Reality.Fingerprint, 64)
}

func validPeerShortID(value string) bool {
	if len(value) > 16 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func validPeerLinkText(value string, maxSize int) bool {
	return len(value) <= maxSize && strings.IndexFunc(value, unicode.IsControl) == -1
}

func validPeerTransportHost(host string) bool {
	if len(host) > 261 {
		return false
	}
	if validPeerHost(host) {
		return true
	}
	if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
		return net.ParseIP(host[1:len(host)-1]) != nil
	}
	name, port, err := net.SplitHostPort(host)
	n, portErr := strconv.ParseUint(port, 10, 16)
	return err == nil && portErr == nil && n > 0 && name != "" && validPeerHost(name)
}

func (transport *peerPublicTransport) valid() bool {
	if transport == nil {
		return true
	}
	if !validPeerLinkText(transport.Path, 2048) || !validPeerLinkText(transport.ServiceName, 256) {
		return false
	}
	switch transport.Type {
	case "ws", "httpupgrade":
		return len(transport.Hosts) == 0 && transport.ServiceName == "" && validPeerTransportHost(transport.Host)
	case "http":
		if len(transport.Hosts) > 16 || transport.Host != "" || transport.ServiceName != "" {
			return false
		}
		for _, host := range transport.Hosts {
			if !validPeerTransportHost(host) {
				return false
			}
		}
		return true
	case "grpc":
		return transport.Host == "" && len(transport.Hosts) == 0 && transport.Path == ""
	case "quic":
		return transport.Host == "" && len(transport.Hosts) == 0 && transport.Path == "" && transport.ServiceName == ""
	default:
		return false
	}
}

func (options *peerPublicProxyOptions) valid(inboundType string) bool {
	if options == nil {
		return inboundType != "hysteria"
	}
	if inboundType == "tuic" {
		return options.UpMbps == 0 && options.DownMbps == 0 &&
			(options.CongestionControl == "" || options.CongestionControl == "cubic" || options.CongestionControl == "new_reno" || options.CongestionControl == "bbr") &&
			(options.UDPRelayMode == "" || options.UDPRelayMode == "native" || options.UDPRelayMode == "quic")
	}
	return options.CongestionControl == "" && options.UDPRelayMode == "" &&
		(inboundType != "hysteria" || options.UpMbps > 0 && options.DownMbps > 0)
}

func announcePeerInbounds(db *gorm.DB) ([]PeerInbound, error) {
	var rows []model.Inbound
	if err := db.Preload("Tls").Order("id").Limit(513).Find(&rows).Error; err != nil {
		return nil, err
	}
	inbounds := make([]PeerInbound, 0, len(rows))
	buildTags := capabilities.BuildTags()
	for _, row := range rows {
		entry := PeerInbound{ID: row.Id, Tag: row.Tag, Type: row.Type}
		link := publicPeerLink(row)
		if link != nil && (buildTags["with_quic"] || !link.requiresQUIC(row.Type)) &&
			(buildTags["with_utls"] || link.TLS == nil || link.TLS.Reality == nil) {
			entry.Link = link
		}
		inbounds = append(inbounds, entry)
	}
	return inbounds, nil
}

func publicPeerLink(row model.Inbound) *peerPublicLink {
	if !peerProtocolSupported(row.Type) {
		return nil
	}
	var options struct {
		Port      uint16          `json:"listen_port"`
		Transport json.RawMessage `json:"transport"`
		Obfs      json.RawMessage `json:"obfs"`
		Realm     *struct{}       `json:"realm"`
		peerPublicProxyOptions
	}
	if json.Unmarshal(row.Options, &options) != nil {
		return nil
	}
	link := &peerPublicLink{Port: options.Port}
	var ok bool
	if len(options.Transport) > 0 {
		link.Transport, ok = publicPeerTransport(options.Transport)
		if !ok {
			return nil
		}
	}
	switch row.Type {
	case "hysteria", "hysteria2":
		if obfs := string(options.Obfs); (obfs != "" && obfs != "null" && obfs != `""`) || options.Realm != nil {
			return nil
		}
		if options.UpMbps > 0 || options.DownMbps > 0 {
			link.Options = &peerPublicProxyOptions{UpMbps: options.UpMbps, DownMbps: options.DownMbps}
		}
	case "tuic":
		var client struct {
			UDPRelayMode string `json:"udp_relay_mode"`
		}
		if len(row.OutJson) > 0 && json.Unmarshal(row.OutJson, &client) != nil {
			return nil
		}
		if client.UDPRelayMode != "" {
			options.UDPRelayMode = client.UDPRelayMode
		}
		if options.CongestionControl != "" || options.UDPRelayMode != "" {
			link.Options = &peerPublicProxyOptions{CongestionControl: options.CongestionControl, UDPRelayMode: options.UDPRelayMode}
		}
	}
	if row.TlsId != 0 {
		link.TLS, ok = publicPeerTLS(row.Tls)
		if !ok {
			return nil
		}
	}
	if !link.valid(row.Type) {
		return nil
	}
	return link
}

func (link peerPublicLink) requiresQUIC(inboundType string) bool {
	return inboundType == "hysteria" || inboundType == "hysteria2" || inboundType == "tuic" ||
		link.Transport != nil && link.Transport.Type == "quic"
}

func publicPeerTransport(raw json.RawMessage) (*peerPublicTransport, bool) {
	var options struct {
		Type        string          `json:"type"`
		Host        json.RawMessage `json:"host"`
		Path        string          `json:"path"`
		ServiceName string          `json:"service_name"`
		Method      string          `json:"method"`
		Headers     struct {
			Host badoption.Listable[string] `json:"Host"`
		} `json:"headers"`
	}
	if singjson.Unmarshal(raw, &options) != nil {
		return nil, false
	}
	transport := &peerPublicTransport{Type: options.Type}
	switch options.Type {
	case "", "tcp":
		return nil, true
	case "ws":
		transport.Path = options.Path
		if len(options.Headers.Host) > 0 {
			transport.Host = options.Headers.Host[0]
		}
	case "http":
		if options.Method != "" && options.Method != "PUT" {
			return nil, false
		}
		transport.Path = options.Path
		var hosts badoption.Listable[string]
		if len(options.Host) > 0 && singjson.Unmarshal(options.Host, &hosts) != nil {
			return nil, false
		}
		transport.Hosts = hosts
	case "httpupgrade":
		transport.Path = options.Path
		if len(options.Host) > 0 && json.Unmarshal(options.Host, &transport.Host) != nil {
			return nil, false
		}
	case "grpc":
		transport.ServiceName = options.ServiceName
	case "quic":
	default:
		return nil, false
	}
	return transport, true
}

func publicPeerTLS(tls *model.Tls) (*peerPublicTLS, bool) {
	var server struct {
		Enabled                          bool                       `json:"enabled"`
		ServerName                       string                     `json:"server_name"`
		ALPN                             badoption.Listable[string] `json:"alpn"`
		ClientCertificate                badoption.Listable[string] `json:"client_certificate"`
		ClientCertificatePath            badoption.Listable[string] `json:"client_certificate_path"`
		ClientCertificatePublicKeySHA256 badoption.Listable[string] `json:"client_certificate_public_key_sha256"`
		Reality                          struct {
			Enabled    bool                       `json:"enabled"`
			PrivateKey string                     `json:"private_key"`
			ShortIDs   badoption.Listable[string] `json:"short_id"`
		} `json:"reality"`
	}
	if tls == nil || singjson.Unmarshal(tls.Server, &server) != nil {
		return nil, false
	}
	if !server.Enabled {
		return nil, !server.Reality.Enabled
	}
	if len(server.ClientCertificate) > 0 || len(server.ClientCertificatePath) > 0 || len(server.ClientCertificatePublicKeySHA256) > 0 {
		return nil, false
	}
	publicTLS := &peerPublicTLS{Enabled: true, ServerName: server.ServerName, ALPN: server.ALPN}
	if !server.Reality.Enabled {
		return publicTLS, true
	}
	key, err := base64.RawURLEncoding.Strict().DecodeString(server.Reality.PrivateKey)
	if err != nil {
		return nil, false
	}
	privateKey, err := ecdh.X25519().NewPrivateKey(key)
	if err != nil {
		return nil, false
	}
	var client struct {
		UTLS struct {
			Fingerprint string `json:"fingerprint"`
		} `json:"utls"`
	}
	publicTLS.Reality = &peerPublicReality{
		PublicKey:   base64.RawURLEncoding.EncodeToString(privateKey.PublicKey().Bytes()),
		Fingerprint: "chrome",
	}
	if len(server.Reality.ShortIDs) > 0 {
		publicTLS.Reality.ShortID = server.Reality.ShortIDs[0]
	}
	if json.Unmarshal(tls.Client, &client) == nil && client.UTLS.Fingerprint != "" {
		publicTLS.Reality.Fingerprint = client.UTLS.Fingerprint
	}
	return publicTLS, true
}

func (link peerPublicLink) subscriptionInbound(entry PeerInbound) model.Inbound {
	inbound := model.Inbound{
		Type:    entry.Type,
		Tag:     entry.Tag,
		Addrs:   json.RawMessage(`[]`),
		OutJson: json.RawMessage(`{}`),
	}
	options := struct {
		Port      uint16         `json:"listen_port"`
		Transport map[string]any `json:"transport,omitempty"`
		*peerPublicProxyOptions
	}{Port: link.Port, peerPublicProxyOptions: link.Options}
	if transport := link.Transport; transport != nil {
		options.Transport = map[string]any{"type": transport.Type}
		switch transport.Type {
		case "ws":
			options.Transport["path"] = transport.Path
			options.Transport["headers"] = map[string]string{"Host": transport.Host}
		case "http":
			options.Transport["path"] = transport.Path
			options.Transport["host"] = transport.Hosts
		case "httpupgrade":
			options.Transport["path"] = transport.Path
			options.Transport["host"] = transport.Host
		case "grpc":
			options.Transport["service_name"] = transport.ServiceName
		}
	}
	inbound.Options, _ = json.Marshal(options)
	if tls := link.TLS; tls != nil {
		server := map[string]any{"enabled": true}
		if tls.ServerName != "" {
			server["server_name"] = tls.ServerName
		}
		if len(tls.ALPN) > 0 {
			server["alpn"] = tls.ALPN
		}
		client := make(map[string]any)
		if reality := tls.Reality; reality != nil {
			server["reality"] = map[string]any{"enabled": true}
			client["reality"] = map[string]any{"public_key": reality.PublicKey, "short_id": reality.ShortID}
			client["utls"] = map[string]any{"enabled": true, "fingerprint": reality.Fingerprint}
		}
		serverTLS, _ := json.Marshal(server)
		clientTLS, _ := json.Marshal(client)
		inbound.TlsId, inbound.Tls = 1, &model.Tls{Server: serverTLS, Client: clientTLS}
	}
	return inbound
}

func peerInboundsFromInventory(inventory json.RawMessage) []PeerInbound {
	entries := DecodeNodeInventory(inventory).Inbounds
	inbounds := make([]PeerInbound, 0, len(entries))
	for _, entry := range entries {
		inbounds = append(inbounds, PeerInbound{ID: entry.ID, Type: entry.Type, Tag: entry.Tag, Link: entry.PeerLink})
	}
	return inbounds
}

func peerAssignmentLinkReason(node *model.Node, inbound PeerInbound, eligible, currentRevision bool, now time.Time) string {
	switch {
	case !node.Enabled || node.PeerDisconnect:
		return "node_disabled"
	case inbound.Link == nil || !inbound.Link.valid(inbound.Type):
		return "inbound_unavailable"
	case node.LinkHost == "":
		return "missing_address"
	case !eligible:
		return "incompatible_policy"
	case !currentRevision:
		return "pending_confirmation"
	case !nodePeerAccessActive(node, now):
		return "access_expired"
	default:
		return "available"
	}
}

func pairedClientLinks(db *gorm.DB, clientID uint) ([]map[string]string, time.Time) {
	var nodes []model.Node
	if err := db.Model(&model.Node{}).
		Where("enabled AND enrollment_digest <> '' AND peer_revision <> '' AND link_host <> '' AND NOT COALESCE(peer_disconnect, 0)").
		Where("uuid IN (?)", db.Model(&model.PeerAssignment{}).Select("node_uuid").Where("client_id = ?", clientID)).
		Order("name, id").Find(&nodes).Error; err != nil {
		return nil, time.Time{}
	}
	var links []map[string]string
	var deadline time.Time
	now := time.Now()
	for _, node := range nodes {
		if !nodePeerAccessActive(&node, now) {
			continue
		}
		inbounds := peerInboundsFromInventory(node.Inventory)
		byID := make(map[uint]PeerInbound, len(inbounds))
		for _, inbound := range inbounds {
			byID[inbound.ID] = inbound
		}
		state, err := projectedPeerState(db, &node, inbounds, now)
		if err != nil || state.Revision != node.PeerRevision {
			continue
		}
		for _, user := range state.Users {
			entry := byID[user.InboundID]
			if user.ClientID != clientID || peerAssignmentLinkReason(&node, entry, true, true, now) != "available" {
				continue
			}
			inbound := entry.Link.subscriptionInbound(entry)
			clientConfig, _ := json.Marshal(map[string]peerUser{user.Type: user}) // #nosec G117 -- The subscription includes only the requesting client's proxy credential.
			for _, uri := range util.LinkGenerator(clientConfig, &inbound, node.LinkHost) {
				links = append(links, map[string]string{"remark": node.Name + " " + entry.Tag, "type": "node", "uri": uri})
				expires := time.Unix(node.LastPeerSync, 0).Add(model.PeerAccessLease)
				if node.PeerLeaseCapable {
					expires = time.Unix(node.PeerAccessUntil, 0)
				}
				if state.nextExpiry > 0 && state.nextExpiry < expires.Unix() {
					expires = time.Unix(state.nextExpiry, 0)
				}
				if deadline.IsZero() || expires.Before(deadline) {
					deadline = expires
				}
			}
		}
	}
	return links, deadline
}
