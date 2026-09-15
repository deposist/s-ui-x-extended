package migrateutil

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/deposist/s-ui-x-extended/database/model"

	E "github.com/sagernet/sing/common/exceptions"
	"gorm.io/gorm"
)

// openvpnOutboundToEndpoint migrates legacy `openvpn` OUTBOUND rows (core ≤
// 1.13) to `openvpn-client` ENDPOINT rows (core 1.14). The fork moved OpenVPN
// from the outbound registry to the endpoint registry; the outbound type is
// gone, so every stored row must be converted or the core fails to start.
//
// Tag and dialer options are preserved verbatim, so references from route
// rules, selector/urltest/failover members, detours and providers keep working
// (the core resolves outbound-namespace tags through the endpoint manager).
//
// The migration is transactional (the caller owns the tx), idempotent (a row
// already migrated is type openvpn-client and is skipped), and aborts with a
// precise path+reason when it meets a shape it cannot map safely:
//
//   - a tag collision with an existing endpoint (would silently shadow it);
//   - an inline openvpn outbound nested inside a bond member (bond builds
//     members through the outbound registry, which no longer has openvpn, and
//     a bond member cannot reference an endpoint tag);
//   - a key_direction value outside the OpenVPN 0/1 pair;
//   - a proto value that is not udp/tcp.
func MigrateOpenVPNOutboundToEndpoint(tx *gorm.DB) error {
	if tx == nil || !tx.Migrator().HasTable(&model.Outbound{}) {
		return nil
	}
	var outbounds []model.Outbound
	if err := tx.Model(&model.Outbound{}).Find(&outbounds).Error; err != nil {
		return err
	}

	// Abort on inline openvpn nested in a bond before touching anything: there
	// is no endpoint equivalent for a bond member, so the whole migration must
	// roll back rather than strand the bond.
	for _, outbound := range outbounds {
		if outbound.Type != "bond" {
			continue
		}
		if path := findNestedOpenVPN(outbound.Options); path != "" {
			return E.New("cannot migrate openvpn outbound ", fmt.Sprintf("%q", outbound.Tag),
				": bond member at ", path, " nests an inline openvpn outbound, which has no endpoint equivalent; ",
				"move it to a top-level openvpn outbound first")
		}
	}

	existingEndpointTags, err := endpointTagSet(tx)
	if err != nil {
		return err
	}

	for _, outbound := range outbounds {
		if outbound.Type != "openvpn" {
			continue
		}
		if _, collision := existingEndpointTags[outbound.Tag]; collision {
			return E.New("cannot migrate openvpn outbound ", fmt.Sprintf("%q", outbound.Tag),
				": an endpoint already uses this tag")
		}
		endpointOptions, err := convertOpenVPNOutboundOptions(outbound.Options, outbound.Tag)
		if err != nil {
			return err
		}
		endpoint := model.Endpoint{
			Type:    "openvpn-client",
			Tag:     outbound.Tag,
			Options: endpointOptions,
		}
		if err := tx.Create(&endpoint).Error; err != nil {
			return E.Cause(err, "insert openvpn-client endpoint ", fmt.Sprintf("%q", outbound.Tag))
		}
		if err := tx.Delete(&model.Outbound{}, outbound.Id).Error; err != nil {
			return E.Cause(err, "delete openvpn outbound ", fmt.Sprintf("%q", outbound.Tag))
		}
		existingEndpointTags[outbound.Tag] = struct{}{}
	}
	return nil
}

// endpointTagSet returns the set of tags already taken by endpoint rows.
func endpointTagSet(tx *gorm.DB) (map[string]struct{}, error) {
	if !tx.Migrator().HasTable(&model.Endpoint{}) {
		return map[string]struct{}{}, nil
	}
	var endpoints []model.Endpoint
	if err := tx.Model(&model.Endpoint{}).Select("tag").Find(&endpoints).Error; err != nil {
		return nil, err
	}
	tags := make(map[string]struct{}, len(endpoints))
	for _, endpoint := range endpoints {
		tags[endpoint.Tag] = struct{}{}
	}
	return tags, nil
}

// findNestedOpenVPN returns the path of an inline openvpn outbound nested in a
// bond member, or "" when the row carries none.
func findNestedOpenVPN(options json.RawMessage) string {
	var parsed struct {
		Outbounds []struct {
			Outbound struct {
				Type string `json:"type"`
				Tag  string `json:"tag"`
			} `json:"outbound"`
		} `json:"outbounds"`
	}
	if err := json.Unmarshal(options, &parsed); err != nil {
		return ""
	}
	for i, member := range parsed.Outbounds {
		if member.Outbound.Type == "openvpn" {
			name := member.Outbound.Tag
			if name == "" {
				name = fmt.Sprintf("#%d", i)
			}
			return fmt.Sprintf("outbounds[%d] (%s)", i, name)
		}
	}
	return ""
}

// convertOpenVPNOutboundOptions maps a legacy openvpn outbound options blob to
// the openvpn-client endpoint options shape. Unknown-to-1.14 legacy keys are
// rejected with the field name so nothing is silently dropped.
func convertOpenVPNOutboundOptions(raw json.RawMessage, tag string) (json.RawMessage, error) {
	var old map[string]json.RawMessage
	if len(raw) > 0 && string(raw) != "null" {
		if err := json.Unmarshal(raw, &old); err != nil {
			return nil, E.Cause(err, "decode openvpn outbound ", fmt.Sprintf("%q", tag))
		}
	}
	if old == nil {
		old = map[string]json.RawMessage{}
	}

	out := map[string]json.RawMessage{}
	consume := func(key string) (json.RawMessage, bool) {
		value, ok := old[key]
		if ok {
			delete(old, key)
		}
		return value, ok
	}
	set := func(key string, value any) error {
		encoded, err := json.Marshal(value)
		if err != nil {
			return err
		}
		out[key] = encoded
		return nil
	}
	setRaw := func(key string, value json.RawMessage) { out[key] = value }

	// Direct scalar/bool copies (same JSON name and type on both sides).
	for _, key := range []string{"system", "name", "auth", "username", "password",
		"reconnect_delay", "ping_interval", "ping_restart"} {
		if value, ok := consume(key); ok {
			setRaw(key, value)
		}
	}

	// mode: the legacy constructor always negotiated TLS (tls.NewOpenVPNClient
	// was unconditional), so every migrated row is a TLS client.
	if err := set("mode", "tls"); err != nil {
		return nil, err
	}

	// proto → network, applying the legacy normalization (default udp).
	if value, ok := consume("proto"); ok {
		network, err := legacyOpenVPNProto(value)
		if err != nil {
			return nil, E.Cause(err, "openvpn outbound ", fmt.Sprintf("%q", tag), " proto")
		}
		setRaw("network", network)
	}

	// servers: same {server, server_port} element shape on both sides.
	if value, ok := consume("servers"); ok {
		setRaw("servers", value)
	}

	// allowed_ips → routes: the legacy system device turned these into the
	// interface route addresses (Inet4RouteAddress/Inet6RouteAddress), which is
	// exactly what the new `routes` field drives.
	if value, ok := consume("allowed_ips"); ok {
		setRaw("routes", value)
	}

	// cipher → data_ciphers: legacy TLS-mode cipher. The new `cipher` field is
	// static_key-only, so TLS rows must use the data-channel list instead. The
	// list is kept to exactly the configured cipher: the negotiated cipher then
	// matches the old single-cipher behavior with no silent widening.
	if value, ok := consume("cipher"); ok {
		cipher, err := legacyOpenVPNCipher(value)
		if err != nil {
			return nil, E.Cause(err, "openvpn outbound ", fmt.Sprintf("%q", tag), " cipher")
		}
		if cipher != "" {
			if err := set("data_ciphers", []string{cipher}); err != nil {
				return nil, err
			}
		}
	}

	// TLS sub-object.
	tlsOut, err := convertOpenVPNLegacyTLS(old, tag)
	if err != nil {
		return nil, err
	}
	if tlsOut != nil {
		out["tls"] = tlsOut
	}
	// consume the handled control-wrap scalars at the top level
	consume("tls_auth")
	consume("tls_auth_path")
	consume("tls_crypt")
	consume("tls_crypt_path")
	consume("tls_crypt_v2")
	consume("key_direction")

	// Dialer options carry over unchanged (shared embedded struct).
	dialerKeys := []string{
		"detour", "bind_interface", "inet4_bind_address", "inet6_bind_address",
		"bind_address_no_port", "protect_path", "routing_mark", "reuse_addr",
		"netns", "connect_timeout", "tcp_fast_open", "tcp_multi_path",
		"disable_tcp_keep_alive", "tcp_keep_alive", "tcp_keep_alive_interval",
		"udp_fragment", "domain_resolver", "network_strategy", "network_type",
		"fallback_network_type", "fallback_delay", "domain_strategy",
	}
	for _, key := range dialerKeys {
		if value, ok := consume(key); ok {
			setRaw(key, value)
		}
	}

	// Whatever is left was never part of the legacy schema or has no 1.14
	// equivalent. kernel_tx/kernel_rx are intentionally absent from the new
	// schema and never had a runtime effect (kTLS never engaged on the
	// packet-framed control channel), so they are dropped on purpose.
	delete(old, "kernel_tx") // in tls sub-object, already consumed; defensive
	delete(old, "kernel_rx")
	if len(old) > 0 {
		keys := make([]string, 0, len(old))
		for key := range old {
			keys = append(keys, key)
		}
		return nil, E.New("cannot migrate openvpn outbound ", fmt.Sprintf("%q", tag),
			": unsupported legacy field(s) with no 1.14 equivalent: ", strings.Join(keys, ", "))
	}

	encoded, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return nil, err
	}
	return json.RawMessage(encoded), nil
}

// legacyOpenVPNProto reproduces the legacy normalizeProto default: empty and
// udp variants collapse to udp; tcp variants collapse to tcp; anything else is
// a hard error, exactly like the legacy Prepare.
func legacyOpenVPNProto(raw json.RawMessage) (json.RawMessage, error) {
	var proto string
	if err := json.Unmarshal(raw, &proto); err != nil {
		return nil, err
	}
	// Matches the legacy normalizeProto exactly: udp/udp4 → udp,
	// tcp/tcp-client/tcp4/tcp4-client → tcp; anything else was a Prepare error.
	switch strings.ToLower(strings.TrimSpace(proto)) {
	case "", "udp", "udp4":
		return json.Marshal("udp")
	case "tcp", "tcp-client", "tcp4", "tcp4-client":
		return json.Marshal("tcp")
	default:
		return nil, E.New("unsupported legacy proto ", fmt.Sprintf("%q", proto))
	}
}

// legacyOpenVPNCipher uppercases/trims the cipher the way the legacy Prepare
// did, so data_ciphers matches the negotiated value.
func legacyOpenVPNCipher(raw json.RawMessage) (string, error) {
	var cipher string
	if err := json.Unmarshal(raw, &cipher); err != nil {
		return "", err
	}
	return strings.ToUpper(strings.TrimSpace(cipher)), nil
}

// convertOpenVPNLegacyTLS builds the new tls object (certificate material +
// control_wrap) from the legacy flat auth/crypt keys and the legacy tls
// sub-object. It consumes those keys from old.
//
// Precedence preserved from the legacy constructor: tls_auth (inline) >
// tls_auth_path > tls_crypt (inline) > tls_crypt_path. The legacy code chose
// tls_auth whenever it or its path was set, otherwise tls_crypt, and applied
// key_direction only on the tls_auth branch.
func convertOpenVPNLegacyTLS(old map[string]json.RawMessage, tag string) (json.RawMessage, error) {
	get := func(key string) (json.RawMessage, bool) {
		value, ok := old[key]
		return value, ok
	}

	var legacyTLS map[string]json.RawMessage
	if value, ok := get("tls"); ok && string(value) != "null" {
		if err := json.Unmarshal(value, &legacyTLS); err != nil {
			return nil, E.Cause(err, "openvpn outbound ", fmt.Sprintf("%q", tag), " tls")
		}
		delete(old, "tls")
	}

	tlsOut := map[string]json.RawMessage{}

	// Certificate material: CA stays CA, client cert/key move to client_*.
	if legacyTLS != nil {
		tlsMove := func(from, to string) {
			if value, ok := legacyTLS[from]; ok {
				tlsOut[to] = value
				delete(legacyTLS, from)
			}
		}
		tlsMove("ca", "certificate")
		tlsMove("ca_path", "certificate_path")
		tlsMove("certificate", "client_certificate")
		tlsMove("certificate_path", "client_certificate_path")
		tlsMove("key", "client_key")
		tlsMove("key_path", "client_key_path")
		tlsMove("verify_x509_name", "server_name")
		tlsMove("verify_x509_name_mode", "server_name_type")
		// cipher_suites (Go names list) → cipher (":"-joined), matching the
		// legacy lookupTLSCipherSuite translation table.
		if value, ok := legacyTLS["cipher_suites"]; ok {
			joined, err := legacyOpenVPNCipherSuites(value)
			if err != nil {
				return nil, E.Cause(err, "openvpn outbound ", fmt.Sprintf("%q", tag), " tls.cipher_suites")
			}
			if joined != "" {
				encoded, err := json.Marshal(joined)
				if err != nil {
					return nil, err
				}
				tlsOut["cipher"] = encoded
			}
			delete(legacyTLS, "cipher_suites")
		}
		// kernel_tx/kernel_rx never engaged (packet-framed control channel); the
		// new schema has no such fields, so they are dropped rather than carried.
		delete(legacyTLS, "kernel_tx")
		delete(legacyTLS, "kernel_rx")
		if len(legacyTLS) > 0 {
			keys := make([]string, 0, len(legacyTLS))
			for key := range legacyTLS {
				keys = append(keys, key)
			}
			return nil, E.New("cannot migrate openvpn outbound ", fmt.Sprintf("%q", tag),
				": unsupported legacy tls field(s): ", strings.Join(keys, ", "))
		}
	}

	// Control wrap (tls_auth / tls_crypt). Inline wins over path; auth wins over
	// crypt — identical to the legacy constructor.
	controlWrap, err := buildLegacyControlWrap(old, tag)
	if err != nil {
		return nil, err
	}
	if controlWrap != nil {
		tlsOut["control_wrap"] = controlWrap
	}

	if len(tlsOut) == 0 {
		return nil, nil
	}
	encoded, err := json.Marshal(tlsOut)
	if err != nil {
		return nil, err
	}
	return encoded, nil
}

// buildLegacyControlWrap applies the legacy tls_auth > tls_crypt precedence and
// produces the new control_wrap object. It reads (but does not delete) the
// relevant keys from old; deletion happens at the caller.
func buildLegacyControlWrap(old map[string]json.RawMessage, tag string) (json.RawMessage, error) {
	getStr := func(key string) string {
		value, ok := old[key]
		if !ok {
			return ""
		}
		var s string
		if err := json.Unmarshal(value, &s); err != nil {
			return ""
		}
		return s
	}
	tlsAuth := getStr("tls_auth")
	tlsAuthPath := getStr("tls_auth_path")
	tlsCrypt := getStr("tls_crypt")
	tlsCryptPath := getStr("tls_crypt_path")

	var cryptV2 bool
	if value, ok := old["tls_crypt_v2"]; ok {
		_ = json.Unmarshal(value, &cryptV2)
	}

	wrap := map[string]json.RawMessage{}
	setStr := func(key, value string) error {
		encoded, err := json.Marshal(value)
		if err != nil {
			return err
		}
		wrap[key] = encoded
		return nil
	}

	switch {
	case tlsAuth != "" || tlsAuthPath != "":
		// tls_auth branch
		if err := setStr("type", "tls_auth"); err != nil {
			return nil, err
		}
		if tlsAuth != "" {
			encoded, err := json.Marshal([]string{tlsAuth})
			if err != nil {
				return nil, err
			}
			wrap["key"] = encoded
		} else {
			if err := setStr("key_path", tlsAuthPath); err != nil {
				return nil, err
			}
		}
		// key_direction applies to tls_auth only. Legacy effective value: when
		// tls_auth was present, keyDirection became options.KeyDirection whose Go
		// zero value is 0 (server) — an absent key_direction meant "server", NOT
		// no-direction. The new endpoint treats an empty direction as -1
		// (no-direction), so to preserve behavior we always set direction here.
		direction := "server"
		if value, ok := old["key_direction"]; ok {
			parsed, err := legacyKeyDirection(value)
			if err != nil {
				return nil, E.Cause(err, "openvpn outbound ", fmt.Sprintf("%q", tag), " key_direction")
			}
			if parsed != "" {
				direction = parsed
			}
		}
		if err := setStr("direction", direction); err != nil {
			return nil, err
		}
	case tlsCrypt != "" || tlsCryptPath != "":
		// tls_crypt branch
		cryptType := "tls_crypt"
		if cryptV2 {
			cryptType = "tls_crypt_v2"
		}
		if err := setStr("type", cryptType); err != nil {
			return nil, err
		}
		if tlsCrypt != "" {
			encoded, err := json.Marshal([]string{tlsCrypt})
			if err != nil {
				return nil, err
			}
			wrap["key"] = encoded
		} else {
			if err := setStr("key_path", tlsCryptPath); err != nil {
				return nil, err
			}
		}
		// key_direction without tls_auth had no effect in the legacy
		// constructor (keyDirection stayed -1), so it is intentionally dropped.
	default:
		return nil, nil
	}

	encoded, err := json.Marshal(wrap)
	if err != nil {
		return nil, err
	}
	return encoded, nil
}

// legacyKeyDirection maps the legacy integer key_direction to the new string
// enum, preserving the effective zero. Only 0 and 1 are valid OpenVPN values.
func legacyKeyDirection(raw json.RawMessage) (string, error) {
	// The legacy field was an int with omitempty, so a stored 0 is meaningful.
	var asInt int
	if err := json.Unmarshal(raw, &asInt); err == nil {
		switch asInt {
		case 0:
			return "server", nil
		case 1:
			return "client", nil
		default:
			return "", E.New("invalid value ", asInt, " (expected 0 or 1)")
		}
	}
	// Tolerate a hand-edited string for robustness.
	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		switch asString {
		case "server", "client":
			return asString, nil
		case "":
			return "", nil
		}
		return "", E.New("invalid value ", fmt.Sprintf("%q", asString), ` (expected "server" or "client")`)
	}
	return "", E.New("unreadable value")
}

// legacyOpenVPNCipherSuites joins the Go cipher-name list with ":" for the new
// single cipher field.
func legacyOpenVPNCipherSuites(raw json.RawMessage) (string, error) {
	var suites []string
	if err := json.Unmarshal(raw, &suites); err != nil {
		return "", err
	}
	return strings.Join(suites, ":"), nil
}
