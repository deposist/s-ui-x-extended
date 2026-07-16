package util

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/deposist/s-ui-x-extended/core/capabilities"
	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/deposist/s-ui-x-extended/util/common"
)

// InboundTypeWithLink is the set of inbound types that produce an external link
// (URI or Telegram). Derived from the embedded capability manifest; consumers use
// it as a set (SQL IN / iteration), so order is not significant.
var InboundTypeWithLink = capabilities.InboundTypesWithLink()

type LinkParam struct {
	Key   string
	Value string
}

const defaultTUICUDPRelayMode = "quic"

// mapString returns m[key] as a string, or "" when the key is absent or not a
// string. Inbound/addr maps come from operator-supplied or imported config and
// may be malformed; using these accessors keeps a bad value from panicking the
// subscription request goroutine.
func mapString(m map[string]interface{}, key string) string {
	s, _ := m[key].(string)
	return s
}

// asBool returns v as a bool, or false when v is nil or not a bool.
func asBool(v interface{}) bool {
	b, _ := v.(bool)
	return b
}

// asInt coerces a JSON-decoded numeric (float64 from encoding/json, or the int
// forms that can appear when a map is built in Go) into an int. ok is false for
// nil or non-numeric values so callers can skip malformed ports.
func asInt(v interface{}) (int, bool) {
	switch n := v.(type) {
	case float64:
		return int(n), true
	case int:
		return n, true
	case int64:
		return int(n), true
	case json.Number:
		if i, err := n.Int64(); err == nil {
			return int(i), true
		}
	}
	return 0, false
}

func LinkGenerator(clientConfig json.RawMessage, i *model.Inbound, hostname string) []string {
	inbound, err := i.MarshalFull()
	if err != nil {
		return []string{}
	}

	var tls map[string]interface{}
	if i.TlsId > 0 {
		tls = prepareTls(i.Tls)
	}

	var userConfig map[string]map[string]interface{}
	if err := json.Unmarshal(clientConfig, &userConfig); err != nil {
		return []string{}
	}

	var Addrs []map[string]interface{}
	if err := json.Unmarshal(i.Addrs, &Addrs); err != nil {
		return []string{}
	}
	if len(Addrs) == 0 {
		Addrs = append(Addrs, map[string]interface{}{
			"server":      hostname,
			"server_port": (*inbound)["listen_port"],
			"remark":      i.Tag,
		})
		if i.TlsId > 0 {
			Addrs[0]["tls"] = tls
		}
	} else {
		for index, addr := range Addrs {
			addrRemark, _ := addr["remark"].(string)
			Addrs[index]["remark"] = i.Tag + addrRemark
			if i.TlsId > 0 {
				newTls := map[string]interface{}{}
				for k, v := range tls {
					newTls[k] = v
				}

				// Override tls
				if addrTls, ok := addr["tls"].(map[string]interface{}); ok {
					for k, v := range addrTls {
						newTls[k] = v
					}
				}
				Addrs[index]["tls"] = newTls
			}
		}
	}

	switch i.Type {
	case "socks":
		return socksLink(userConfig["socks"], Addrs)
	case "http":
		return httpLink(userConfig["http"], Addrs)
	case "mixed":
		return append(
			socksLink(userConfig["socks"], Addrs),
			httpLink(userConfig["http"], Addrs)...,
		)
	case "shadowsocks":
		return shadowsocksLink(userConfig, *inbound, Addrs)
	case "naive":
		return naiveLink(userConfig["naive"], *inbound, Addrs)
	case "hysteria":
		return hysteriaLink(userConfig["hysteria"], *inbound, Addrs)
	case "hysteria2":
		return hysteria2Link(userConfig["hysteria2"], *inbound, Addrs)
	case "tuic":
		return tuicLink(userConfig["tuic"], *inbound, Addrs)
	case "vless":
		return vlessLink(userConfig["vless"], *inbound, Addrs)
	case "anytls":
		return anytlsLink(userConfig["anytls"], Addrs)
	case "trojan":
		return trojanLink(userConfig["trojan"], *inbound, Addrs)
	case "vmess":
		return vmessLink(userConfig["vmess"], *inbound, Addrs)
	case "mtproxy":
		return mtproxyLink(userConfig["mtproxy"], Addrs)
	case "sudoku":
		return sudokuLink(userConfig["sudoku"], *inbound, Addrs)
	case "mieru":
		return mieruLink(userConfig["mieru"], *inbound, Addrs)
	}

	return []string{}
}

// mieruLink builds mierus:// simple-format share links for the enfein/mieru
// client. Unlike the standard mieru:// form (base64 of a protobuf ClientConfig),
// mierus:// is a plain URL, so we can emit it without pulling in the mieru
// protobuf schema. mieru is NOT keyless: the per-user name/password come from the
// client's own config block; host/ports/transport come from the inbound. One URL
// is produced per advertised address; a mieru server can bind a range of ports,
// so every listen_ports entry becomes a repeated port/protocol query pair.
func mieruLink(userConfig map[string]interface{}, inbound map[string]interface{}, addrs []map[string]interface{}) []string {
	username := mapString(userConfig, "name")
	password := mapString(userConfig, "password")
	if username == "" || password == "" {
		return []string{}
	}

	protocol := strings.ToUpper(strings.TrimSpace(mapString(inbound, "transport")))
	if protocol != "TCP" && protocol != "UDP" {
		protocol = "TCP" // mieru's own default transport
	}

	// listen_ports is a list of "begin:end" (or single) strings; mierus:// wants
	// "begin-end" (or single). Fall back to the single listen_port when absent.
	ports := mieruPortStrings(inbound)
	if len(ports) == 0 {
		return []string{}
	}

	profile := mapString(inbound, "tag")
	if profile == "" {
		profile = "mieru"
	}

	var links []string
	for _, addr := range addrs {
		server := mapString(addr, "server")
		if server == "" {
			continue
		}
		u := &url.URL{Scheme: "mierus", Host: server}
		u.User = url.UserPassword(username, password)
		q := url.Values{}
		q.Set("profile", profile)
		if mux := strings.TrimSpace(mapString(inbound, "multiplexing")); mux != "" {
			q.Set("multiplexing", mux)
		}
		for _, p := range ports {
			q.Add("port", p)
			q.Add("protocol", protocol)
		}
		u.RawQuery = q.Encode()
		links = append(links, u.String())
	}
	return links
}

// mieruPortStrings normalizes the inbound's port declaration into mierus:// port
// tokens: each listen_ports "begin:end" range becomes "begin-end", single values
// pass through, and a bare listen_port is used when no range list is present.
func mieruPortStrings(inbound map[string]interface{}) []string {
	var out []string
	if raw, ok := inbound["listen_ports"].([]interface{}); ok {
		for _, v := range raw {
			s, ok := v.(string)
			if !ok {
				continue
			}
			s = strings.TrimSpace(s)
			if s == "" {
				continue
			}
			out = append(out, strings.ReplaceAll(s, ":", "-"))
		}
	}
	if len(out) > 0 {
		return out
	}
	if port, ok := asInt(inbound["listen_port"]); ok && port > 0 {
		out = append(out, strconv.Itoa(port))
	}
	return out
}

// sudokuShortLinkPayload mirrors the upstream SUDOKU-ASCII/sudoku
// internal/config.shortLinkPayload (json keys are single letters). Only the
// fields our server inbound can supply are emitted; C-side-only http_mask
// host/mux/tls have no inbound source and stay omitted (omitempty).
type sudokuShortLinkPayload struct {
	Host            string   `json:"h"`
	Port            int      `json:"p"`
	Key             string   `json:"k"`
	ASCII           string   `json:"a,omitempty"`
	AEAD            string   `json:"e,omitempty"`
	PackedDownlink  bool     `json:"x,omitempty"`
	CustomTable     string   `json:"t,omitempty"`
	CustomTables    []string `json:"ts,omitempty"`
	DisableHTTPMask bool     `json:"hd,omitempty"`
	HTTPMaskMode    string   `json:"hm,omitempty"`
	HTTPMaskPath    string   `json:"hy,omitempty"`
}

// sudokuASCIIFromTableType maps the panel's table_type onto the short link `a`
// field, matching upstream encodeASCII: prefer_ascii->ascii, prefer_entropy (and
// empty) -> entropy, directional up_*_down_* values pass through unchanged.
func sudokuASCIIFromTableType(tableType string) string {
	switch strings.ToLower(strings.TrimSpace(tableType)) {
	case "prefer_ascii", "ascii":
		return "ascii"
	case "", "prefer_entropy", "entropy":
		return "entropy"
	default:
		return strings.ToLower(strings.TrimSpace(tableType))
	}
}

// sudokuLink builds sudoku:// short links for the SUDOKU-ASCII clients (Sudodroid
// and the Go CLI's -link). Sudoku is keyless: the shared inbound `key` and the
// transport-shaping fields are read from the inbound itself, never from a per-user
// config block. The payload is base64url(RawURLEncoding) of the JSON, exactly as
// the upstream core encodes and decodes it.
func sudokuLink(clientConfig map[string]interface{}, inbound map[string]interface{}, addrs []map[string]interface{}) []string {
	key := strings.TrimSpace(mapString(clientConfig, "key"))
	if key == "" {
		key = mapString(inbound, "key")
	}
	if key == "" {
		return []string{}
	}

	base := sudokuShortLinkPayload{
		Key:             key,
		AEAD:            mapString(inbound, "aead_method"),
		ASCII:           sudokuASCIIFromTableType(mapString(inbound, "table_type")),
		CustomTable:     mapString(inbound, "custom_table"),
		DisableHTTPMask: asBool(inbound["disable_http_mask"]),
		HTTPMaskPath:    mapString(inbound, "path_root"),
	}
	// packed downlink is the inverse of enable_pure_downlink; upstream omits `hm`
	// for the default legacy mode so the client falls back to its own default.
	base.PackedDownlink = !asBool(inbound["enable_pure_downlink"])
	if mode := strings.ToLower(strings.TrimSpace(mapString(inbound, "http_mask_mode"))); mode != "" && mode != "legacy" {
		base.HTTPMaskMode = mode
	}
	if tables, ok := inbound["custom_tables"].([]interface{}); ok {
		for _, t := range tables {
			if s, ok := t.(string); ok && s != "" {
				base.CustomTables = append(base.CustomTables, s)
			}
		}
	}

	var links []string
	for _, addr := range addrs {
		server := mapString(addr, "server")
		if server == "" {
			continue
		}
		port, ok := asInt(addr["server_port"])
		if !ok || port == 0 {
			continue
		}
		payload := base
		payload.Host = server
		payload.Port = port
		data, err := json.Marshal(payload)
		if err != nil {
			continue
		}
		links = append(links, "sudoku://"+base64.RawURLEncoding.EncodeToString(data))
	}
	return links
}

// mtproxyLink builds Telegram MTProto proxy deep links. There is no sing-box
// mtproxy OUTBOUND, so this is the only client delivery path. The per-user
// `secret` is already a self-contained faketls ('ee') secret (0xee || 16-byte key
// || faketls SNI host), so it is emitted verbatim (URL-escaped). No server-side
// material is read here.
func mtproxyLink(userConfig map[string]interface{}, addrs []map[string]interface{}) []string {
	secret, _ := userConfig["secret"].(string)
	if secret == "" {
		return []string{}
	}
	var links []string
	for _, addr := range addrs {
		port, _ := addr["server_port"].(float64)
		q := url.Values{}
		q.Set("server", mapString(addr, "server"))
		q.Set("port", fmt.Sprintf("%d", uint(port)))
		q.Set("secret", secret)
		links = append(links, "tg://proxy?"+q.Encode())
	}
	return links
}

func prepareTls(t *model.Tls) map[string]interface{} {
	var iTls, oTls map[string]interface{}
	if err := json.Unmarshal(t.Client, &oTls); err != nil {
		return nil
	}
	if err := json.Unmarshal(t.Server, &iTls); err != nil {
		return nil
	}

	for k, v := range iTls {
		switch k {
		case "enabled", "server_name", "alpn":
			oTls[k] = v
		case "reality":
			reality, okReality := v.(map[string]interface{})
			clientReality, okClient := oTls["reality"].(map[string]interface{})
			if !okReality || !okClient {
				continue
			}
			clientReality["enabled"] = reality["enabled"]
			if shortIDs, hasSIds := reality["short_id"].([]interface{}); hasSIds && len(shortIDs) > 0 {
				clientReality["short_id"] = shortIDs[common.RandomInt(len(shortIDs))]
			}
			oTls["reality"] = clientReality
		}
	}
	return oTls
}

func socksLink(userConfig map[string]interface{}, addrs []map[string]interface{}) []string {
	var links []string
	for _, addr := range addrs {
		port, _ := addr["server_port"].(float64)
		links = append(links, fmt.Sprintf("socks5://%s:%s@%s:%d", userConfig["username"], userConfig["password"], mapString(addr, "server"), uint(port)))
	}
	return links
}

func httpLink(userConfig map[string]interface{}, addrs []map[string]interface{}) []string {
	var links []string
	protocol := "http"
	for _, addr := range addrs {
		if addr["tls"] != nil {
			protocol = "https"
		}
		port, _ := addr["server_port"].(float64)
		links = append(links, fmt.Sprintf("%s://%s:%s@%s:%d", protocol, userConfig["username"], userConfig["password"], mapString(addr, "server"), uint(port)))
	}
	return links
}

func shadowsocksLink(
	userConfig map[string]map[string]interface{},
	inbound map[string]interface{},
	addrs []map[string]interface{}) []string {

	var userPass []string
	method, _ := inbound["method"].(string)
	if strings.HasPrefix(method, "2022") {
		inbPass, _ := inbound["password"].(string)
		userPass = append(userPass, inbPass)
	}
	var pass string
	if method == "2022-blake3-aes-128-gcm" {
		pass, _ = userConfig["shadowsocks16"]["password"].(string)
	} else {
		pass, _ = userConfig["shadowsocks"]["password"].(string)
	}
	userPass = append(userPass, pass)

	uriBase := fmt.Sprintf("ss://%s", toBase64([]byte(fmt.Sprintf("%s:%s", method, strings.Join(userPass, ":")))))

	var links []string
	for _, addr := range addrs {
		port, _ := addr["server_port"].(float64)
		links = append(links, fmt.Sprintf("%s@%s:%.0f#%s", uriBase, mapString(addr, "server"), port, mapString(addr, "remark")))
	}
	return links
}

func naiveLink(
	userConfig map[string]interface{},
	inbound map[string]interface{},
	addrs []map[string]interface{}) []string {

	password, _ := userConfig["password"].(string)
	username, _ := userConfig["username"].(string)

	baseUri := "http2://"
	var links []string

	for _, addr := range addrs {
		var params []LinkParam
		params = append(params, LinkParam{"padding", "1"})
		if tls, ok := addr["tls"].(map[string]interface{}); ok {
			if sni, ok := tls["server_name"].(string); ok {
				params = append(params, LinkParam{"peer", sni})
			}
			if alpn, ok := tls["alpn"].([]interface{}); ok {
				alpnList := make([]string, len(alpn))
				for i, v := range alpn {
					alpnList[i], _ = v.(string)
				}
				params = append(params, LinkParam{"alpn", strings.Join(alpnList, ",")})
			}
			if insecure, ok := tls["insecure"].(bool); ok && insecure {
				params = append(params, LinkParam{"insecure", "1"})
			}
		}
		if tfo, ok := inbound["tcp_fast_open"].(bool); ok && tfo {
			params = append(params, LinkParam{"tfo", "1"})
		} else {
			params = append(params, LinkParam{"tfo", "0"})
		}

		port, _ := addr["server_port"].(float64)
		uri := baseUri + toBase64([]byte(fmt.Sprintf("%s:%s@%s:%.0f", username, password, mapString(addr, "server"), port)))
		links = append(links, addParams(uri, params, mapString(addr, "remark")))
	}
	return links
}

func hysteriaLink(
	userConfig map[string]interface{},
	inbound map[string]interface{},
	addrs []map[string]interface{}) []string {

	baseUri := "hysteria://"
	var links []string

	for _, addr := range addrs {
		var params []LinkParam
		if upmbps, ok := inbound["up_mbps"].(float64); ok {
			params = append(params, LinkParam{"downmbps", fmt.Sprintf("%.0f", upmbps)})
		}
		if downmbps, ok := inbound["down_mbps"].(float64); ok {
			params = append(params, LinkParam{"upmbps", fmt.Sprintf("%.0f", downmbps)})
		}
		if auth, ok := userConfig["auth_str"].(string); ok {
			params = append(params, LinkParam{"auth", auth})
		}
		if tls, ok := addr["tls"].(map[string]interface{}); ok {
			getTlsParams(&params, tls, "insecure")
		}
		if obfs, ok := inbound["obfs"].(string); ok {
			params = append(params, LinkParam{"obfs", obfs})
		}
		if tfo, ok := inbound["tcp_fast_open"].(bool); ok && tfo {
			params = append(params, LinkParam{"fastopen", "1"})
		} else {
			params = append(params, LinkParam{"fastopen", "0"})
		}
		var outJson map[string]interface{}
		outRaw, _ := inbound["out_json"].(json.RawMessage)
		if err := json.Unmarshal(outRaw, &outJson); err != nil {
			return []string{} // Handle error
		}
		if mport, ok := outJson["server_ports"].([]interface{}); ok {
			mportList := make([]string, len(mport))
			for i, v := range mport {
				mportList[i], _ = v.(string)
			}
			params = append(params, LinkParam{"mport", strings.Join(mportList, ",")})
		}

		port, _ := addr["server_port"].(float64)
		uri := fmt.Sprintf("%s%s:%.0f", baseUri, mapString(addr, "server"), port)
		links = append(links, addParams(uri, params, mapString(addr, "remark")))
	}

	return links
}

func hysteria2Link(
	userConfig map[string]interface{},
	inbound map[string]interface{},
	addrs []map[string]interface{}) []string {

	password, _ := userConfig["password"].(string)
	baseUri := fmt.Sprintf("%s%s@", "hysteria2://", password)
	var links []string

	for _, addr := range addrs {
		var params []LinkParam
		if upmbps, ok := inbound["up_mbps"].(float64); ok {
			params = append(params, LinkParam{"downmbps", fmt.Sprintf("%.0f", upmbps)})
		}
		if downmbps, ok := inbound["down_mbps"].(float64); ok {
			params = append(params, LinkParam{"upmbps", fmt.Sprintf("%.0f", downmbps)})
		}
		if tls, ok := addr["tls"].(map[string]interface{}); ok {
			getTlsParams(&params, tls, "insecure")
		}
		if obfs, ok := inbound["obfs"].(map[string]interface{}); ok {
			if obfsType, ok := obfs["type"].(string); ok {
				params = append(params, LinkParam{"obfs", obfsType})
			}
			if obfsPassword, ok := obfs["password"].(string); ok {
				params = append(params, LinkParam{"obfs-password", obfsPassword})
			}
		}
		if tfo, ok := inbound["tcp_fast_open"].(bool); ok && tfo {
			params = append(params, LinkParam{"fastopen", "1"})
		} else {
			params = append(params, LinkParam{"fastopen", "0"})
		}
		var outJson map[string]interface{}
		outRaw, _ := inbound["out_json"].(json.RawMessage)
		if err := json.Unmarshal(outRaw, &outJson); err != nil {
			return []string{} // Handle error
		}
		if mport, ok := outJson["server_ports"].([]interface{}); ok {
			mportList := make([]string, len(mport))
			for i, v := range mport {
				mportList[i], _ = v.(string)
			}
			params = append(params, LinkParam{"mport", strings.Join(mportList, ",")})
		}

		port, _ := addr["server_port"].(float64)
		uri := fmt.Sprintf("%s%s:%.0f", baseUri, mapString(addr, "server"), port)
		links = append(links, addParams(uri, params, mapString(addr, "remark")))
	}

	return links
}

func anytlsLink(
	userConfig map[string]interface{},
	addrs []map[string]interface{}) []string {

	password, _ := userConfig["password"].(string)
	baseUri := fmt.Sprintf("%s%s@", "anytls://", password)
	var links []string

	for _, addr := range addrs {
		var params []LinkParam
		if tls, ok := addr["tls"].(map[string]interface{}); ok {
			getTlsParams(&params, tls, "insecure")
		}

		port, _ := addr["server_port"].(float64)
		uri := fmt.Sprintf("%s%s:%.0f", baseUri, mapString(addr, "server"), port)
		links = append(links, addParams(uri, params, mapString(addr, "remark")))
	}

	return links
}

func tuicLink(
	userConfig map[string]interface{},
	inbound map[string]interface{},
	addrs []map[string]interface{}) []string {

	password, _ := userConfig["password"].(string)
	uuid, _ := userConfig["uuid"].(string)
	baseUri := fmt.Sprintf("%s%s:%s@", "tuic://", uuid, password)
	udpRelayMode := tuicUDPRelayMode(inbound)
	var links []string

	for _, addr := range addrs {
		var params []LinkParam
		if tls, ok := addr["tls"].(map[string]interface{}); ok {
			getTlsParams(&params, tls, "insecure")
		}
		if congestionControl, ok := inbound["congestion_control"].(string); ok {
			params = append(params, LinkParam{"congestion_control", congestionControl})
		}
		if udpRelayMode != "" {
			params = append(params, LinkParam{"udp_relay_mode", udpRelayMode})
		}

		port, _ := addr["server_port"].(float64)
		uri := fmt.Sprintf("%s%s:%.0f", baseUri, mapString(addr, "server"), port)
		links = append(links, addParams(uri, params, mapString(addr, "remark")))
	}

	return links
}

func tuicUDPRelayMode(inbound map[string]interface{}) string {
	if outJson, ok := inbound["out_json"].(json.RawMessage); ok {
		var out map[string]interface{}
		if err := json.Unmarshal(outJson, &out); err == nil {
			if mode := normalizeTUICUDPRelayMode(out["udp_relay_mode"]); mode != "" {
				return mode
			}
		}
	}
	if outJson, ok := inbound["out_json"].(map[string]interface{}); ok {
		if mode := normalizeTUICUDPRelayMode(outJson["udp_relay_mode"]); mode != "" {
			return mode
		}
	}
	if mode := normalizeTUICUDPRelayMode(inbound["udp_relay_mode"]); mode != "" {
		return mode
	}
	return defaultTUICUDPRelayMode
}

func normalizeTUICUDPRelayMode(value interface{}) string {
	mode, _ := value.(string)
	switch strings.TrimSpace(mode) {
	case "native", "quic":
		return strings.TrimSpace(mode)
	default:
		return ""
	}
}

func vlessLink(
	userConfig map[string]interface{},
	inbound map[string]interface{},
	addrs []map[string]interface{}) []string {

	uuid, _ := userConfig["uuid"].(string)
	baseParams := getTransportParams(inbound["transport"])
	var links []string

	// `xtls-rprx-vision` is strictly TCP. Emitting it on a vless link
	// whose transport is grpc / ws / http / httpupgrade makes Xray-core
	// reject the connection on the client side (issue #1127). Decide
	// once per inbound so we never produce a self-broken link.
	transportType := "tcp"
	if tr, ok := inbound["transport"].(map[string]interface{}); ok {
		if tt, _ := tr["type"].(string); tt != "" {
			transportType = tt
		}
	}

	for _, addr := range addrs {
		params := make([]LinkParam, len(baseParams))
		copy(params, baseParams)
		if tls, ok := addr["tls"].(map[string]interface{}); ok && asBool(tls["enabled"]) {
			getTlsParams(&params, tls, "allowInsecure")
			if flow, ok := userConfig["flow"].(string); ok && flow != "" && transportType == "tcp" {
				params = append(params, LinkParam{"flow", flow})
			}
		}
		port, _ := addr["server_port"].(float64)
		uri := fmt.Sprintf("vless://%s@%s:%.0f", uuid, mapString(addr, "server"), port)
		uri = addParams(uri, params, mapString(addr, "remark"))
		links = append(links, uri)
	}

	return links
}

func trojanLink(
	userConfig map[string]interface{},
	inbound map[string]interface{},
	addrs []map[string]interface{}) []string {
	password, _ := userConfig["password"].(string)
	baseParams := getTransportParams(inbound["transport"])
	var links []string

	for _, addr := range addrs {
		params := make([]LinkParam, len(baseParams))
		copy(params, baseParams)
		if tls, ok := addr["tls"].(map[string]interface{}); ok && asBool(tls["enabled"]) {
			getTlsParams(&params, tls, "allowInsecure")
		}
		port, _ := addr["server_port"].(float64)
		uri := fmt.Sprintf("trojan://%s@%s:%.0f", password, mapString(addr, "server"), port)
		uri = addParams(uri, params, mapString(addr, "remark"))
		links = append(links, uri)
	}

	return links
}

func vmessLink(
	userConfig map[string]interface{},
	inbound map[string]interface{},
	addrs []map[string]interface{}) []string {

	uuid, _ := userConfig["uuid"].(string)
	transportParams := getTransportParams(inbound["transport"])
	var links []string

	baseParams := map[string]interface{}{
		"v":   "2",
		"id":  uuid,
		"aid": 0,
	}

	var net, typ, host, path string
	for _, p := range transportParams {
		switch p.Key {
		case "type":
			net = p.Value
		case "host":
			host = p.Value
		case "path":
			path = p.Value
		}
	}

	if net == "http" || net == "tcp" {
		baseParams["net"] = "tcp"
		if net == "http" {
			typ = "http"
		}
	} else {
		baseParams["net"] = net
	}

	for _, addr := range addrs {
		obj := make(map[string]interface{})
		for k, v := range baseParams {
			obj[k] = v
		}

		obj["add"], _ = addr["server"].(string)
		port, _ := addr["server_port"].(float64)
		obj["port"] = fmt.Sprintf("%.0f", port)
		obj["ps"], _ = addr["remark"].(string)
		if typ != "" {
			obj["type"] = typ
		}
		if host != "" {
			obj["host"] = host
		}
		if path != "" {
			obj["path"] = path
		}
		populateVmessTlsParams(obj, addr["tls"])

		jsonStr, _ := json.Marshal(obj)

		uri := fmt.Sprintf("vmess://%s", toBase64(jsonStr))
		links = append(links, uri)
	}
	return links
}

func populateVmessTlsParams(obj map[string]interface{}, tlsConfig interface{}) {
	if tlsMap, ok := tlsConfig.(map[string]interface{}); ok && asBool(tlsMap["enabled"]) {
		obj["tls"] = "tls"
		var tlsParams []LinkParam
		getTlsParams(&tlsParams, tlsMap, "allowInsecure")
		for _, p := range tlsParams {
			switch p.Key {
			case "security":
				// ignore, as "tls" is already set
			case "allowInsecure":
				obj["allowInsecure"] = 1
			case "sni":
				obj["sni"] = p.Value
			case "fp":
				obj["fp"] = p.Value
			case "alpn":
				obj["alpn"] = p.Value
			}
		}
	} else {
		obj["tls"] = "none"
	}
}

func toBase64(d []byte) string {
	return base64.StdEncoding.EncodeToString(d)
}

func addParams(uri string, params []LinkParam, remark string) string {
	URL, err := url.Parse(uri)
	if err != nil || URL == nil {
		// uri is assembled from operator-controlled inbound metadata (server addr);
		// a stray control byte / bad escape makes url.Parse return (nil, err). Bail
		// out instead of dereferencing nil and panicking the link-generation path.
		return uri
	}
	var q []string
	for _, p := range params {
		switch p.Key {
		case "mport", "alpn":
			q = append(q, fmt.Sprintf("%s=%s", p.Key, p.Value))
		default:
			q = append(q, fmt.Sprintf("%s=%s", p.Key, url.QueryEscape(p.Value)))
		}
	}
	URL.RawQuery = strings.Join(q, "&")
	URL.Fragment = remark
	return URL.String()
}

func getTransportParams(t interface{}) []LinkParam {
	var params []LinkParam
	trasport, _ := t.(map[string]interface{})
	var transportType string
	if tt, ok := trasport["type"].(string); ok {
		transportType = tt
	} else {
		transportType = "tcp"
	}
	params = append(params, LinkParam{"type", transportType})
	if transportType == "tcp" {
		return params
	}

	switch transportType {
	case "http":
		if host, ok := trasport["host"].([]interface{}); ok {
			var hosts []string
			for _, v := range host {
				if s, ok := v.(string); ok {
					hosts = append(hosts, s)
				}
			}
			params = append(params, LinkParam{"host", strings.Join(hosts, ",")})
		}
		if path, ok := trasport["path"].(string); ok {
			params = append(params, LinkParam{"path", path})
		}
	case "ws":
		if path, ok := trasport["path"].(string); ok {
			params = append(params, LinkParam{"path", path})
		}
		if headers, ok := trasport["headers"].(map[string]interface{}); ok {
			if host, ok := headers["Host"].(string); ok {
				params = append(params, LinkParam{"host", host})
			}
		}
	case "grpc":
		if serviceName, ok := trasport["service_name"].(string); ok {
			params = append(params, LinkParam{"serviceName", serviceName})
		}
	case "httpupgrade":
		if host, ok := trasport["host"].(string); ok {
			params = append(params, LinkParam{"host", host})
		}
		if path, ok := trasport["path"].(string); ok {
			params = append(params, LinkParam{"path", path})
		}
	}
	return params
}

func getTlsParams(params *[]LinkParam, tls map[string]interface{}, insecureKey string) {
	if reality, ok := tls["reality"].(map[string]interface{}); ok && asBool(reality["enabled"]) {
		*params = append(*params, LinkParam{"security", "reality"})
		if pbk, ok := reality["public_key"].(string); ok {
			*params = append(*params, LinkParam{"pbk", pbk})
		}
		if sid, ok := reality["short_id"].(string); ok {
			*params = append(*params, LinkParam{"sid", sid})
		}
	} else {
		*params = append(*params, LinkParam{"security", "tls"})
		if insecure, ok := tls["insecure"].(bool); ok && insecure {
			*params = append(*params, LinkParam{insecureKey, "1"})
		}
		if disableSni, ok := tls["disable_sni"].(bool); ok && disableSni {
			*params = append(*params, LinkParam{"disable_sni", "1"})
		}
	}
	if utls, ok := tls["utls"].(map[string]interface{}); ok {
		if fingerprint, ok := utls["fingerprint"].(string); ok {
			*params = append(*params, LinkParam{"fp", fingerprint})
		}
	}
	if sni, ok := tls["server_name"].(string); ok {
		*params = append(*params, LinkParam{"sni", sni})
	}
	if alpn, ok := tls["alpn"].([]interface{}); ok {
		alpnList := make([]string, len(alpn))
		for i, v := range alpn {
			alpnList[i], _ = v.(string)
		}
		*params = append(*params, LinkParam{"alpn", strings.Join(alpnList, ",")})
	}
}
