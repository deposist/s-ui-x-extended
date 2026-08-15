package util

import (
	"encoding/json"

	"github.com/deposist/s-ui-x-extended/core/capabilities"
	"github.com/deposist/s-ui-x-extended/util/common"

	"github.com/deposist/s-ui-x-extended/database/model"
)

// outJSONBuilder mutates the client out_json from the marshalled inbound. Each
// builder is an ALLOW-LIST: it copies only the fields it explicitly names, so no
// server-side secret can leak by being copied wholesale.
type outJSONBuilder func(out *map[string]interface{}, inbound map[string]interface{})

// outJSONBuilders is the dispatch table keyed by the manifest's outJsonBuilder
// value. "base" keeps only the common fields (type/tag/server/server_port[/tls]);
// a protocol key runs its builder; any inbound type whose manifest builder key is
// empty (or is otherwise absent here) falls through to a wipe — the legacy
// default branch and safety net for not-yet-delivered / unknown types.
var outJSONBuilders = map[string]outJSONBuilder{
	"base":        func(*map[string]interface{}, map[string]interface{}) {},
	"naive":       naiveOut,
	"shadowsocks": shadowsocksOut,
	"shadowtls":   shadowTlsOut,
	"hysteria":    hysteriaOut,
	"hysteria2":   hysteria2Out,
	"tuic":        tuicOut,
	"vless":       vlessOut,
	"trojan":      trojanOut,
	"vmess":       vmessOut,
	// Phase 1: extended protocols now delivered as JSON outbounds.
	"ssh":         sshOut,
	"mieru":       mieruOut,
	"trusttunnel": trustTunnelOut,
	"sudoku":      sudokuOut,
}

// Derived once from the embedded capability manifest.
var (
	outJSONSkip          = capabilities.SkipOutJSONTypes()
	outJSONBuilderByType = capabilities.OutJSONBuilders()
)

// FillOutJson fills an Inbound's out_json (the client-facing outbound template).
func FillOutJson(i *model.Inbound, hostname string) error {
	if _, skip := outJSONSkip[i.Type]; skip {
		// Transparent/local inbounds (direct/tun/redirect/tproxy) have no client
		// outbound; leave out_json untouched.
		return nil
	}
	var outJson map[string]interface{}
	err := json.Unmarshal(i.OutJson, &outJson)
	if err != nil {
		return err
	}

	if outJson == nil {
		outJson = make(map[string]interface{})
	}

	if i.TlsId > 0 {
		addTls(&outJson, i.Tls)
	} else {
		delete(outJson, "tls")
	}

	inbound, err := i.MarshalFull()
	if err != nil {
		return err
	}

	outJson["type"] = i.Type
	outJson["tag"] = i.Tag
	outJson["server"] = hostname
	outJson["server_port"] = (*inbound)["listen_port"]

	if builder, ok := outJSONBuilders[outJSONBuilderByType[i.Type]]; ok {
		builder(&outJson, *inbound)
	} else {
		// Empty/unknown builder key: wipe out_json (legacy default branch). Types
		// not yet wired for client delivery (mieru/sudoku/trusttunnel/ssh/mtproxy)
		// land here until their builder is registered.
		for key := range outJson {
			delete(outJson, key)
		}
	}

	i.OutJson, err = json.MarshalIndent(outJson, "", "  ")
	if err != nil {
		return err
	}

	return nil
}

// addTls function
func addTls(out *map[string]interface{}, tls *model.Tls) {
	var tlsServer, tlsConfig map[string]interface{}
	err := json.Unmarshal(tls.Server, &tlsServer)
	if err != nil {
		return
	}
	err = json.Unmarshal(tls.Client, &tlsConfig)
	if err != nil {
		return
	}

	// tls.Client may be JSON null/absent (nil map) or an object that omits the
	// reality/ech sub-maps the server enables. addTls writes into tlsConfig and
	// merges those sub-maps, so guard every access: a non-lockstep server/client
	// pair (import, apiv2, future UI regression) must not panic the save path.
	if tlsConfig == nil {
		tlsConfig = map[string]interface{}{}
	}

	if enabled, ok := tlsServer["enabled"]; ok {
		tlsConfig["enabled"] = enabled
	}
	if serverName, ok := tlsServer["server_name"]; ok {
		tlsConfig["server_name"] = serverName
	}
	if alpn, ok := tlsServer["alpn"]; ok {
		tlsConfig["alpn"] = alpn
	}
	if minVersion, ok := tlsServer["min_version"]; ok {
		tlsConfig["min_version"] = minVersion
	}
	if maxVersion, ok := tlsServer["max_version"]; ok {
		tlsConfig["max_version"] = maxVersion
	}
	if certificate, ok := tlsServer["certificate"]; ok {
		tlsConfig["certificate"] = certificate
	}
	if cipherSuites, ok := tlsServer["cipher_suites"]; ok {
		tlsConfig["cipher_suites"] = cipherSuites
	}
	if reality, ok := tlsServer["reality"].(map[string]interface{}); ok {
		if enabled, _ := reality["enabled"].(bool); enabled {
			realityConfig, ok := tlsConfig["reality"].(map[string]interface{})
			if !ok {
				realityConfig = map[string]interface{}{}
			}
			realityConfig["enabled"] = true
			if shortIDs, ok := reality["short_id"].([]interface{}); ok && len(shortIDs) > 0 {
				realityConfig["short_id"] = shortIDs[common.RandomInt(len(shortIDs))]
			}
			tlsConfig["reality"] = realityConfig
		}
	}
	if ech, ok := tlsServer["ech"].(map[string]interface{}); ok {
		if enabled, _ := ech["enabled"].(bool); enabled {
			echConfig, ok := tlsConfig["ech"].(map[string]interface{})
			if !ok {
				echConfig = map[string]interface{}{}
			}
			echConfig["enabled"] = true
			echConfig["pq_signature_schemes_enabled"] = ech["pq_signature_schemes_enabled"]
			echConfig["dynamic_record_sizing_disabled"] = ech["dynamic_record_sizing_disabled"]
			tlsConfig["ech"] = echConfig
		}
	}

	(*out)["tls"] = tlsConfig
}

func naiveOut(out *map[string]interface{}, inbound map[string]interface{}) {
	if quic_congestion_control, ok := inbound["quic_congestion_control"].(string); ok {
		(*out)["quic"] = true
		switch quic_congestion_control {
		case "bbr_standard":
			(*out)["quic_congestion_control"] = "bbr"
		case "bbr2_variant":
			(*out)["quic_congestion_control"] = "bbr2"
		default:
			(*out)["quic_congestion_control"] = quic_congestion_control
		}
	}

}

func shadowsocksOut(out *map[string]interface{}, inbound map[string]interface{}) {
	if method, ok := inbound["method"].(string); ok {
		(*out)["method"] = method
	}
}

func shadowTlsOut(out *map[string]interface{}, inbound map[string]interface{}) {
	if version, ok := inbound["version"].(float64); ok && int(version) == 3 {
		(*out)["version"] = 3
	} else {
		for key := range *out {
			delete(*out, key)
		}
	}
	(*out)["tls"] = map[string]interface{}{"enabled": true}
}

func hysteriaOut(out *map[string]interface{}, inbound map[string]interface{}) {
	delete(*out, "down_mbps")
	delete(*out, "up_mbps")
	delete(*out, "obfs")
	delete(*out, "recv_window_conn")
	delete(*out, "disable_mtu_discovery")

	if upMbps, ok := inbound["down_mbps"]; ok {
		(*out)["up_mbps"] = upMbps
	}
	if downMbps, ok := inbound["up_mbps"]; ok {
		(*out)["down_mbps"] = downMbps
	}
	if obfs, ok := inbound["obfs"]; ok {
		(*out)["obfs"] = obfs
	}
	if recvWindow, ok := inbound["recv_window_conn"]; ok {
		(*out)["recv_window_conn"] = recvWindow
	}
	if disableMTU, ok := inbound["disable_mtu_discovery"]; ok {
		(*out)["disable_mtu_discovery"] = disableMTU
	}
}

func hysteria2Out(out *map[string]interface{}, inbound map[string]interface{}) {
	delete(*out, "down_mbps")
	delete(*out, "up_mbps")
	delete(*out, "obfs")

	if upMbps, ok := inbound["down_mbps"]; ok {
		(*out)["up_mbps"] = upMbps
	}
	if downMbps, ok := inbound["up_mbps"]; ok {
		(*out)["down_mbps"] = downMbps
	}
	if obfs, ok := inbound["obfs"]; ok {
		(*out)["obfs"] = obfs
	}
}

func tuicOut(out *map[string]interface{}, inbound map[string]interface{}) {
	delete(*out, "zero_rtt_handshake")
	delete(*out, "heartbeat")
	if congestionControl, ok := inbound["congestion_control"].(string); ok {
		(*out)["congestion_control"] = congestionControl
	} else {
		(*out)["congestion_control"] = "cubic"
	}
	if zeroRTT, ok := inbound["zero_rtt_handshake"].(bool); ok {
		(*out)["zero_rtt_handshake"] = zeroRTT
	}
	if heartbeat, ok := inbound["heartbeat"]; ok {
		(*out)["heartbeat"] = heartbeat
	}
}

func vlessOut(out *map[string]interface{}, inbound map[string]interface{}) {
	delete(*out, "transport")
	if transport, ok := inbound["transport"]; ok {
		(*out)["transport"] = transport
	}
}

func trojanOut(out *map[string]interface{}, inbound map[string]interface{}) {
	delete(*out, "transport")
	if transport, ok := inbound["transport"]; ok {
		(*out)["transport"] = transport
	}
}

func vmessOut(out *map[string]interface{}, inbound map[string]interface{}) {
	(*out)["alter_id"] = 0
	delete(*out, "transport")
	if transport, ok := inbound["transport"]; ok {
		(*out)["transport"] = transport
	}
}

// sshOut copies NOTHING from the inbound. host_key / host_key_path are the
// server's PRIVATE host keys and must never reach a client; server_version /
// max_auth_tries are server-only. The base fields (type/tag/server/server_port)
// are enough; user/password are merged per-user in the subscription. This is an
// allow-list of size zero, kept explicit so the intent (and the forbidden-keys
// test) is unambiguous.
func sshOut(out *map[string]interface{}, inbound map[string]interface{}) {
	_ = out
	_ = inbound
}

// mieruOut (EXTENDED) copies only the transport-shaping client fields and maps the
// server's listen_ports to the client's server_ports. username/password are merged
// per-user in the subscription.
func mieruOut(out *map[string]interface{}, inbound map[string]interface{}) {
	for _, k := range []string{"transport", "traffic_pattern", "server_ports"} {
		delete(*out, k)
	}
	if v, ok := inbound["transport"]; ok {
		(*out)["transport"] = v
	}
	if v, ok := inbound["traffic_pattern"]; ok {
		(*out)["traffic_pattern"] = v
	}
	if v, ok := inbound["listen_ports"]; ok {
		(*out)["server_ports"] = v
	}
}

// trustTunnelOut (EXTENDED) copies only the transport client fields. It must NOT
// copy health_check / multiplex / username / password (out-direction-only fields):
// username/password are merged per-user; health_check/multiplex are client-local
// preferences set by the operator on the outbound, not derivable from the inbound.
// bbr_profile was removed from the trusttunnel options in sing-box 2.6.x.
func trustTunnelOut(out *map[string]interface{}, inbound map[string]interface{}) {
	keys := []string{"network", "quic", "congestion_controller", "cwnd"}
	for _, k := range keys {
		delete(*out, k)
	}
	for _, k := range keys {
		if v, ok := inbound[k]; ok {
			(*out)[k] = v
		}
	}
}

// sudokuOut (EXTENDED, keyless) copies client-safe transport parameters. The
// mandatory public `key` is client-visible; `master_key` is intentionally never
// copied. Per-client split keys override `key` during delivery.
// / `handshake_timeout`. http_mask is a nested object on the outbound but flat on
// the inbound: build it from the inbound's flat fields and MERGE onto any existing
// out_json.http_mask so an operator's C-side host/multiplex survive.
func sudokuOut(out *map[string]interface{}, inbound map[string]interface{}) {
	for _, k := range []string{"key", "aead_method", "table_type", "padding_min", "padding_max", "enable_pure_downlink", "custom_table", "custom_tables"} {
		delete(*out, k)
		if v, ok := inbound[k]; ok {
			(*out)[k] = v
		}
	}

	httpMask, _ := (*out)["http_mask"].(map[string]interface{})
	if httpMask == nil {
		httpMask = map[string]interface{}{}
	}
	// enabled mirrors the server's disable switch; mode / path_root mirror the
	// server. host / multiplex are C-side-only and are preserved (never set here).
	disable, _ := inbound["disable_http_mask"].(bool)
	httpMask["enabled"] = !disable
	if v, ok := inbound["http_mask_mode"]; ok {
		httpMask["mode"] = v
	}
	if v, ok := inbound["path_root"]; ok {
		httpMask["path_root"] = v
	}
	(*out)["http_mask"] = httpMask
}
