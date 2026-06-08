package importxui

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/deposist/s-ui-x/database/model"
)

type xuiInboundSettings struct {
	Clients    []xuiClientSetting `json:"clients"`
	Accounts   []xuiAccount       `json:"accounts"`
	Method     string             `json:"method"`
	Password   string             `json:"password"`
	Network    string             `json:"network"`
	Encryption string             `json:"encryption"`
}

type xuiClientSetting struct {
	Comment    string `json:"comment"`
	Email      string `json:"email"`
	Enable     *bool  `json:"enable"`
	ExpiryTime int64  `json:"expiryTime"`
	Flow       string `json:"flow"`
	ID         string `json:"id"`
	LimitIP    int    `json:"limitIp"`
	SubID      string `json:"subId"`
	TgID       any    `json:"tgId"`
	TotalGB    int64  `json:"totalGB"`
	Password   string `json:"password"`
	Security   string `json:"security"`
}

type xuiAccount struct {
	User     string `json:"user"`
	Pass     string `json:"pass"`
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
}

type ClientRef struct {
	SrcInboundID int64
	DstInboundID uint
	SrcTag       string
	Protocol     string
	Email        string
	UUID         string
	Password     string
	Flow         string
	Comment      string
	Enable       bool
	HasEnable    bool
	ExpiryTime   int64
	TotalGB      int64
	LimitIP      int
	SubID        string
	TgID         string
}

type mappedInbound struct {
	Inbound    model.Inbound
	ClientRefs []ClientRef
	Warnings   []string
}

func mapInbound(row xuiInboundRow, tlsID uint, reality *realitySpec, server string) (*mappedInbound, error) {
	var settings xuiInboundSettings
	if err := decodeJSON(row.Settings, &settings); err != nil {
		return nil, fmt.Errorf("inbound %s settings: %w", row.Tag, err)
	}
	stream, err := parseStreamSettings(row)
	if err != nil {
		return nil, err
	}
	switch stream.Network {
	case "kcp", "mkcp", "quic":
		return &mappedInbound{Warnings: []string{fmt.Sprintf("inbound %s: transport %q is unsupported by phase 2 importer; skipped", row.Tag, stream.Network)}}, nil
	}

	inType := inboundType(row.Protocol)
	if inType == "" {
		return &mappedInbound{Warnings: []string{fmt.Sprintf("inbound %s: unsupported protocol %q; skipped", row.Tag, row.Protocol)}}, nil
	}
	if inType == "http" && len(settings.Accounts) == 0 {
		return &mappedInbound{Warnings: []string{fmt.Sprintf("inbound %s: http has no accounts; skipped", row.Tag)}}, nil
	}

	transport, transportWarnings := mapTransport("inbound", row.Tag, stream)
	tlsBlock, tlsWarnings := mapOutboundTLSBlock(stream, reality)
	warnings := append(transportWarnings, tlsWarnings...)
	if w := listenAddressWarning(row); w != "" {
		warnings = append(warnings, w)
	}

	options := map[string]any{
		"listen":      listenAddress(row.Listen),
		"listen_port": row.Port,
	}
	if transport != nil {
		options["transport"] = transport
	}
	flow := firstClientFlow(settings.Clients)
	switch inType {
	case "shadowsocks":
		method := firstNonEmpty(settings.Method, "none")
		options["method"] = method
		if settings.Password != "" {
			options["password"] = settings.Password
		}
	case "vmess":
		options["security"] = "auto"
	}

	optionsJSON, err := marshalJSON(options)
	if err != nil {
		return nil, err
	}
	outJSON, err := buildOutJson(inType, row.Tag, server, row.Port, tlsBlock, transport, flow)
	if err != nil {
		return nil, err
	}
	refs := clientRefsFromSettings(row, inType, settings)
	return &mappedInbound{
		Inbound: model.Inbound{
			Type:    inType,
			Tag:     row.Tag,
			TlsId:   tlsID,
			Addrs:   buildAddrs(),
			OutJson: outJSON,
			Options: optionsJSON,
		},
		ClientRefs: refs,
		Warnings:   warnings,
	}, nil
}

func inboundType(protocol string) string {
	switch strings.ToLower(strings.TrimSpace(protocol)) {
	case "vless":
		return "vless"
	case "vmess":
		return "vmess"
	case "trojan":
		return "trojan"
	case "shadowsocks":
		return "shadowsocks"
	case "http":
		return "http"
	case "socks":
		return "socks"
	default:
		return ""
	}
}

func listenAddress(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "0.0.0.0"
	}
	return value
}

// listenAddressWarning flags inbounds bound to a concrete source-server address.
// Such an address (a specific NIC IP from the old host) usually does not exist
// on the destination server, so the migrated inbound would fail to start there.
// Wildcard binds ("", 0.0.0.0, ::) are host-independent and need no warning.
func listenAddressWarning(row xuiInboundRow) string {
	switch strings.TrimSpace(row.Listen) {
	case "", "0.0.0.0", "::", "[::]":
		return ""
	default:
		return fmt.Sprintf("inbound %s: binds to source listen address %q which may not exist on this host; verify or clear it, otherwise the inbound will fail to start", row.Tag, strings.TrimSpace(row.Listen))
	}
}

func firstClientFlow(clients []xuiClientSetting) string {
	for _, client := range clients {
		if strings.TrimSpace(client.Flow) != "" {
			return strings.TrimSpace(client.Flow)
		}
	}
	return ""
}

func clientRefsFromSettings(row xuiInboundRow, protocol string, settings xuiInboundSettings) []ClientRef {
	refs := make([]ClientRef, 0, len(settings.Clients)+len(settings.Accounts))
	for _, client := range settings.Clients {
		email := strings.TrimSpace(client.Email)
		if email == "" {
			continue
		}
		enable := row.Enable
		hasEnable := false
		if client.Enable != nil {
			enable = enable && *client.Enable
			hasEnable = true
		}
		ref := ClientRef{
			SrcInboundID: row.ID,
			SrcTag:       row.Tag,
			Protocol:     protocol,
			Email:        email,
			UUID:         strings.TrimSpace(client.ID),
			Password:     strings.TrimSpace(client.Password),
			Flow:         strings.TrimSpace(client.Flow),
			Comment:      client.Comment,
			Enable:       enable,
			HasEnable:    hasEnable || !row.Enable,
			ExpiryTime:   client.ExpiryTime,
			TotalGB:      client.TotalGB,
			LimitIP:      client.LimitIP,
			SubID:        strings.TrimSpace(client.SubID),
			TgID:         stringifyTgID(client.TgID),
		}
		refs = append(refs, ref)
	}
	for _, account := range settings.Accounts {
		user := firstNonEmpty(account.Email, account.User, account.Username)
		if user == "" {
			continue
		}
		refs = append(refs, ClientRef{
			SrcInboundID: row.ID,
			SrcTag:       row.Tag,
			Protocol:     protocol,
			Email:        user,
			Password:     firstNonEmpty(account.Pass, account.Password),
			Enable:       row.Enable,
			HasEnable:    !row.Enable,
		})
	}
	return refs
}

func stringifyTgID(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(v)
	case float64:
		if v == 0 {
			return ""
		}
		return fmt.Sprintf("%.0f", v)
	case json.Number:
		return v.String()
	default:
		text := fmt.Sprint(v)
		if text == "0" {
			return ""
		}
		return strings.TrimSpace(text)
	}
}

// mapTransport maps an Xray stream's transport to the sing-box transport block.
// It is shared by inbound and outbound mapping; entity ("inbound"/"outbound")
// only labels the warning messages so the import report attributes them
// correctly.
func mapTransport(entity string, tag string, stream xuiStreamSettings) (map[string]any, []string) {
	network := strings.ToLower(strings.TrimSpace(stream.Network))
	switch network {
	case "", "tcp":
		return nil, nil
	case "ws":
		transport := map[string]any{"type": "ws"}
		if path, ok := stringFromMap(stream.WSSettings, "path"); ok && path != "" {
			transport["path"] = path
		}
		if headers := wsHeaders(stream.WSSettings); len(headers) > 0 {
			transport["headers"] = headers
		}
		return transport, nil
	case "grpc":
		transport := map[string]any{"type": "grpc"}
		if serviceName, ok := stringFromMap(stream.GRPCSettings, "serviceName"); ok && serviceName != "" {
			transport["service_name"] = serviceName
		}
		return transport, nil
	case "h2", "http":
		transport := map[string]any{"type": "http"}
		if hosts, ok := stringSliceFromMap(stream.HTTPSettings, "host"); ok {
			transport["host"] = hosts
		}
		if path, ok := stringFromMap(stream.HTTPSettings, "path"); ok && path != "" {
			transport["path"] = path
		}
		return transport, nil
	case "httpupgrade":
		transport := map[string]any{"type": "httpupgrade"}
		if host, ok := stringFromMap(stream.HTTPUPSettings, "host"); ok && host != "" {
			transport["host"] = host
		}
		if path, ok := stringFromMap(stream.HTTPUPSettings, "path"); ok && path != "" {
			transport["path"] = path
		}
		return transport, nil
	case "splithttp", "xhttp":
		transport := map[string]any{"type": "httpupgrade"}
		if path, ok := stringFromMap(stream.HTTPUPSettings, "path"); ok && path != "" {
			transport["path"] = path
		}
		if host, ok := stringFromMap(stream.HTTPUPSettings, "host"); ok && host != "" {
			transport["host"] = host
		}
		return transport, []string{fmt.Sprintf("%s %s: transport %q mapped to httpupgrade; manual review recommended", entity, tag, network)}
	default:
		return nil, []string{fmt.Sprintf("%s %s: transport %q requires manual review", entity, tag, network)}
	}
}

// wsHeaders collects all WebSocket request headers from Xray wsSettings. Xray
// stores custom headers under "headers"; some panels also keep the Host under a
// top-level "host". sing-box carries every header (including Host) in the
// transport headers map, so all are preserved rather than just Host.
func wsHeaders(ws map[string]any) map[string]any {
	out := map[string]any{}
	if headers, ok := mapFromMap(ws, "headers"); ok {
		for k, v := range headers {
			if s := strings.TrimSpace(fmt.Sprint(v)); s != "" {
				out[k] = s
			}
		}
	}
	if _, has := out["Host"]; !has {
		if host, ok := stringFromMap(ws, "host"); ok && host != "" {
			out["Host"] = host
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func mapOutboundTLSBlock(stream xuiStreamSettings, reality *realitySpec) (map[string]any, []string) {
	switch stream.Security {
	case "":
		return nil, nil
	case "none":
		return nil, nil
	case "reality":
		if reality == nil {
			return nil, []string{"reality TLS settings are incomplete; outbound preview has no TLS block"}
		}
		return map[string]any{
			"enabled":     true,
			"server_name": reality.ServerName,
			"utls": map[string]any{
				"enabled":     true,
				"fingerprint": reality.Fingerprint,
			},
			"reality": map[string]any{
				"enabled":    true,
				"public_key": reality.PublicKey,
				"short_id":   firstString(reality.ShortIDs),
			},
		}, nil
	case "tls":
		// Client-side TLS block for the subscription/preview link. The server-side
		// certificate is migrated separately (a model.Tls record) when inline, or
		// flagged for manual upload when only a file path is present.
		block := map[string]any{"enabled": true}
		if sni := strings.TrimSpace(stream.TLSSettings.ServerName); sni != "" {
			block["server_name"] = sni
		}
		if stream.TLSSettings.AllowInsecure {
			block["insecure"] = true
		}
		if len(stream.TLSSettings.ALPN) > 0 {
			block["alpn"] = stream.TLSSettings.ALPN
		}
		if fp := strings.TrimSpace(stream.TLSSettings.Fingerprint); fp != "" {
			block["utls"] = map[string]any{"enabled": true, "fingerprint": fp}
		}
		return block, nil
	default:
		return nil, []string{fmt.Sprintf("TLS security %q requires manual review", stream.Security)}
	}
}

func stringFromMap(values map[string]any, key string) (string, bool) {
	if values == nil {
		return "", false
	}
	value, ok := values[key]
	if !ok {
		return "", false
	}
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v), true
	default:
		return strings.TrimSpace(fmt.Sprint(v)), true
	}
}

func mapFromMap(values map[string]any, key string) (map[string]any, bool) {
	if values == nil {
		return nil, false
	}
	value, ok := values[key]
	if !ok {
		return nil, false
	}
	casted, ok := value.(map[string]any)
	return casted, ok
}

func stringSliceFromMap(values map[string]any, key string) ([]string, bool) {
	if values == nil {
		return nil, false
	}
	value, ok := values[key]
	if !ok {
		return nil, false
	}
	switch v := value.(type) {
	case []any:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if text := strings.TrimSpace(fmt.Sprint(item)); text != "" {
				result = append(result, text)
			}
		}
		return result, len(result) > 0
	case []string:
		return v, len(v) > 0
	case string:
		if strings.TrimSpace(v) == "" {
			return nil, false
		}
		return []string{strings.TrimSpace(v)}, true
	default:
		text := strings.TrimSpace(fmt.Sprint(v))
		if text == "" {
			return nil, false
		}
		return []string{text}, true
	}
}
