package util

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/deposist/s-ui-x/database/model"
	"github.com/deposist/s-ui-x/util/common"
)

var InboundTypeWithLink = []string{"socks", "http", "mixed", "shadowsocks", "naive", "hysteria", "hysteria2", "anytls", "tuic", "vless", "trojan", "vmess"}

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
	}

	return []string{}
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
	URL, _ := url.Parse(uri)
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
