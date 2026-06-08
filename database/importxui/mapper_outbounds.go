package importxui

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/deposist/s-ui-x/database/model"
)

// dnsHijackTarget and rejectTarget are sentinel routing targets for Xray system
// outbounds that map to a route-rule *action* instead of a real outbound: the
// migration creates no `dns`/`block` outbound and sing-box no longer auto-provides
// one, so a rule that resolves to one becomes action "hijack-dns" / "reject" (see
// MapXrayRouting). Their
// values are deliberately NOT valid outbound tags ("__…__") so they cannot
// collide with a user outbound: a proxy legitimately tagged "dns"/"block"/
// "blocked" maps to its own tag in the targets map and keeps routing to itself,
// rather than being silently turned into an action.
const (
	dnsHijackTarget = "__hijack_dns__"
	rejectTarget    = "__reject__"
)

// directOutboundTag is the tag of the built-in direct outbound that freedom
// outbounds (and remote rule-set downloads) route through. sing-box 1.11+ only
// auto-creates a fallback direct outbound when there are zero outbounds and no
// route.final, so a migrated config that has any other outbound must carry an
// explicit direct outbound or it fails at route time with
// "outbound not found: direct" (see ensureDirectOutbound).
const directOutboundTag = "direct"

// xrayProxySettings is the `settings` block of an Xray *outbound*. vmess/vless
// use `vnext`; trojan/shadowsocks/socks/http use `servers`. Only the first
// server is migrated (sing-box outbounds target a single server).
type xrayProxySettings struct {
	Vnext   []xrayProxyServer `json:"vnext"`
	Servers []xrayProxyServer `json:"servers"`
}

type xrayProxyServer struct {
	Address  string          `json:"address"`
	Port     int             `json:"port"`
	Password string          `json:"password"` // trojan, shadowsocks
	Method   string          `json:"method"`   // shadowsocks
	Users    []xrayProxyUser `json:"users"`
}

type xrayProxyUser struct {
	ID         string `json:"id"`         // vmess, vless
	AlterID    int    `json:"alterId"`    // vmess
	Security   string `json:"security"`   // vmess
	Flow       string `json:"flow"`       // vless
	Encryption string `json:"encryption"` // vless (none)
	User       string `json:"user"`       // socks, http
	Pass       string `json:"pass"`       // socks, http
}

// xuiTLSSetting is the `tlsSettings` block of an Xray stream (client/outbound
// side). Only the fields s-ui can carry over are decoded.
type xuiTLSSetting struct {
	ServerName    string           `json:"serverName"`
	AllowInsecure bool             `json:"allowInsecure"`
	Fingerprint   string           `json:"fingerprint"`
	ALPN          []string         `json:"alpn"`
	Certificates  []xuiCertificate `json:"certificates"`
}

// outboundsFromXray converts a proxy Xray outbound (vmess/vless/trojan/
// shadowsocks/socks/http) into one or more s-ui (sing-box) outbounds. A single
// server yields one outbound carrying the source tag; multiple servers yield
// one member outbound per server plus a urltest group that carries the source
// tag (so routing rules keep resolving to it). It returns an empty slice with a
// warning for protocols with no mapping or malformed settings, so the caller
// surfaces the loss instead of dropping it silently.
func outboundsFromXray(ob xrayOutbound) ([]model.Outbound, []string) {
	proto := strings.ToLower(strings.TrimSpace(ob.Protocol))
	tag := strings.TrimSpace(ob.Tag)
	var settings xrayProxySettings
	if err := decodeJSON(ob.Settings, &settings); err != nil {
		return nil, []string{fmt.Sprintf("outbound %s: invalid %s settings: %v; skipped", tag, proto, err)}
	}

	servers := settings.Servers
	if proto == "vmess" || proto == "vless" {
		servers = settings.Vnext
	}
	if len(servers) == 0 {
		return nil, []string{fmt.Sprintf("outbound %s: %s has no server; skipped", tag, proto)}
	}

	stream := parseOutboundStream(ob)
	carriesStream := proto == "vmess" || proto == "vless" || proto == "trojan" || proto == "http"
	var warnings []string

	var tlsBlock map[string]any
	var transport map[string]any
	if carriesStream {
		var w []string
		tlsBlock, w = mapOutboundClientTLS(tag, stream)
		warnings = append(warnings, w...)
		transport, w = mapTransport("outbound", tag, stream)
		warnings = append(warnings, w...)
	}

	// packet_encoding (xudp) is the one mux-adjacent setting that interoperates
	// across Xray and sing-box; Xray mux itself is not wire-compatible, so it is
	// reported rather than enabled.
	packetEncoding, muxWarnings := outboundProxyExtras(ob, tag)
	warnings = append(warnings, muxWarnings...)

	build := func(srv xrayProxyServer, outTag string) (*model.Outbound, []string) {
		opts := map[string]any{}
		server := strings.TrimSpace(srv.Address)
		if server == "" {
			return nil, []string{fmt.Sprintf("outbound %s: missing server address; skipped", outTag)}
		}
		opts["server"] = server
		opts["server_port"] = srv.Port
		switch proto {
		case "vmess":
			user := firstProxyUser(srv.Users)
			opts["uuid"] = strings.TrimSpace(user.ID)
			opts["security"] = firstNonEmpty(user.Security, "auto")
			opts["alter_id"] = user.AlterID
		case "vless":
			user := firstProxyUser(srv.Users)
			opts["uuid"] = strings.TrimSpace(user.ID)
			if flow := strings.TrimSpace(user.Flow); flow != "" {
				opts["flow"] = flow
			}
		case "trojan":
			opts["password"] = srv.Password
		case "shadowsocks":
			opts["method"] = firstNonEmpty(srv.Method, "none")
			opts["password"] = srv.Password
		case "socks":
			opts["version"] = "5"
			if user := firstProxyUser(srv.Users); strings.TrimSpace(user.User) != "" {
				opts["username"] = strings.TrimSpace(user.User)
				opts["password"] = user.Pass
			}
		case "http":
			if user := firstProxyUser(srv.Users); strings.TrimSpace(user.User) != "" {
				opts["username"] = strings.TrimSpace(user.User)
				opts["password"] = user.Pass
			}
		}
		if carriesStream {
			if tlsBlock != nil {
				opts["tls"] = tlsBlock
			}
			if transport != nil {
				opts["transport"] = transport
			}
		}
		if packetEncoding != "" && (proto == "vless" || proto == "vmess") {
			opts["packet_encoding"] = packetEncoding
		}
		optionsJSON, err := marshalJSON(opts)
		if err != nil {
			return nil, []string{fmt.Sprintf("outbound %s: %v", outTag, err)}
		}
		return &model.Outbound{Type: proto, Tag: outTag, Options: optionsJSON}, nil
	}

	switch proto {
	case "vmess", "vless", "trojan", "shadowsocks", "socks", "http":
		// supported below
	default:
		return nil, []string{fmt.Sprintf("outbound %s: protocol %q has no automatic s-ui mapping; recreate it manually", tag, proto)}
	}

	if len(servers) == 1 {
		out, w := build(servers[0], tag)
		warnings = append(warnings, w...)
		if out == nil {
			return nil, warnings
		}
		return []model.Outbound{*out}, warnings
	}

	// Multiple servers: one member per server + a urltest group carrying the tag.
	members := make([]model.Outbound, 0, len(servers))
	memberTags := make([]string, 0, len(servers))
	for i := range servers {
		memberTag := fmt.Sprintf("%s-%d", tag, i)
		out, w := build(servers[i], memberTag)
		warnings = append(warnings, w...)
		if out != nil {
			members = append(members, *out)
			memberTags = append(memberTags, memberTag)
		}
	}
	if len(members) == 0 {
		return nil, warnings
	}
	groupJSON, err := marshalJSON(map[string]any{"outbounds": memberTags})
	if err != nil {
		return nil, append(warnings, fmt.Sprintf("outbound %s: %v", tag, err))
	}
	warnings = append(warnings, fmt.Sprintf("outbound %s had %d servers; migrated as a urltest group over %v", tag, len(members), memberTags))
	return append(members, model.Outbound{Type: "urltest", Tag: tag, Options: groupJSON}), warnings
}

// outboundProxyExtras inspects an Xray outbound's mux block. It returns the
// sing-box packet_encoding to set ("xudp" when the source used XUDP) and a
// warning when Xray mux was enabled — sing-box multiplex uses a different,
// non-interoperable wire protocol, so enabling it automatically would break an
// otherwise-working outbound.
func outboundProxyExtras(ob xrayOutbound, tag string) (string, []string) {
	if len(ob.Mux) == 0 {
		return "", nil
	}
	var mux struct {
		Enabled         bool `json:"enabled"`
		Concurrency     int  `json:"concurrency"`
		XudpConcurrency int  `json:"xudpConcurrency"`
	}
	if err := json.Unmarshal(ob.Mux, &mux); err != nil {
		return "", nil
	}
	var packetEncoding string
	if mux.XudpConcurrency > 0 {
		packetEncoding = "xudp"
	}
	var warnings []string
	if mux.Enabled {
		warnings = append(warnings, fmt.Sprintf("outbound %s had Xray mux enabled (concurrency %d); sing-box multiplex is not wire-compatible with Xray mux, so it was left disabled — enable multiplex manually only if the remote also speaks sing-box mux", tag, mux.Concurrency))
	}
	return packetEncoding, warnings
}

func firstProxyUser(users []xrayProxyUser) xrayProxyUser {
	if len(users) == 0 {
		return xrayProxyUser{}
	}
	return users[0]
}

// parseOutboundStream decodes an Xray outbound's streamSettings into the shared
// xuiStreamSettings shape so mapTransport and mapOutboundClientTLS can reuse the
// inbound helpers. An absent/invalid block yields a zero (tcp/none) stream.
func parseOutboundStream(ob xrayOutbound) xuiStreamSettings {
	var stream xuiStreamSettings
	if len(ob.StreamSettings) == 0 {
		return stream
	}
	if err := json.Unmarshal(ob.StreamSettings, &stream); err != nil {
		return xuiStreamSettings{}
	}
	stream.Network = strings.ToLower(strings.TrimSpace(stream.Network))
	stream.Security = strings.ToLower(strings.TrimSpace(stream.Security))
	return stream
}

// mapOutboundClientTLS builds the sing-box outbound `tls` block from an Xray
// outbound's streamSettings. For reality it uses the peer public key/short id
// that the Xray outbound stores at the top level of realitySettings (unlike an
// inbound, which stores the private key). Returns nil when TLS is disabled.
func mapOutboundClientTLS(tag string, stream xuiStreamSettings) (map[string]any, []string) {
	switch stream.Security {
	case "", "none":
		return nil, nil
	case "tls":
		tls := map[string]any{"enabled": true}
		if sni := strings.TrimSpace(stream.TLSSettings.ServerName); sni != "" {
			tls["server_name"] = sni
		}
		if stream.TLSSettings.AllowInsecure {
			tls["insecure"] = true
		}
		if len(stream.TLSSettings.ALPN) > 0 {
			tls["alpn"] = stream.TLSSettings.ALPN
		}
		if fp := strings.TrimSpace(stream.TLSSettings.Fingerprint); fp != "" {
			tls["utls"] = map[string]any{"enabled": true, "fingerprint": fp}
		}
		return tls, nil
	case "reality":
		r := stream.RealitySettings
		serverName := firstNonEmpty(r.ServerName, firstString(r.ServerNames))
		fingerprint := firstNonEmpty(r.Fingerprint, "chrome")
		var warnings []string
		if strings.TrimSpace(r.PublicKey) == "" {
			warnings = append(warnings, fmt.Sprintf("outbound %s: reality publicKey is empty; verify the outbound TLS settings", tag))
		}
		tls := map[string]any{
			"enabled":     true,
			"server_name": serverName,
			"utls": map[string]any{
				"enabled":     true,
				"fingerprint": fingerprint,
			},
			"reality": map[string]any{
				"enabled":    true,
				"public_key": strings.TrimSpace(r.PublicKey),
				"short_id":   firstNonEmpty(r.ShortID, firstString(r.ShortIDs)),
			},
		}
		return tls, warnings
	default:
		return nil, []string{fmt.Sprintf("outbound %s: TLS security %q requires manual review", tag, stream.Security)}
	}
}
