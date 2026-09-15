package capabilities

// OptionCoverageMapping ties one core option struct to the panel TypeScript
// interface that models it. It is the single inventory used by both coverage
// gates:
//
//   - core/option_coverage_test.go asserts every field of the Go struct appears
//     in the TS interface (or is deliberately hidden with a reason), so a new
//     core field cannot be silently unavailable in the panel;
//   - scripts/gen-option-coverage.go renders the UI coverage matrix from the same
//     list, so "typed in TS" and "editable in the editor" are measured over the
//     same set of types.
//
// Keeping one list is the point: two hand-maintained copies would let a type be
// covered by one gate and invisible to the other.
type OptionCoverageMapping struct {
	// GoStruct is the option struct name in the core's option package.
	GoStruct string
	// TSInterface is the interface name in frontend/src/types/*.ts.
	TSInterface string
	// Context selects which TypeScript file the interface is read from: "in",
	// "out", "ep", "prov" or "svc".
	Context string
	// UIComponent is the editor file name when it cannot be derived from the TS
	// interface name (e.g. openvpn-client and openvpn-server share one editor whose
	// name mentions neither). Empty means "match by name".
	UIComponent string
}

// OptionCoverageMappings returns the inventory in a stable order (contexts in the
// order below, types in declaration order inside each context).
func OptionCoverageMappings() []OptionCoverageMapping {
	mappings := []OptionCoverageMapping{
		// Inbounds
		{"DirectInboundOptions", "Direct", "in", ""},
		{"SocksInboundOptions", "SOCKS", "in", ""},
		{"HTTPMixedInboundOptions", "Mixed", "in", ""},
		{"ShadowsocksInboundOptions", "Shadowsocks", "in", ""},
		{"VMessInboundOptions", "VMess", "in", ""},
		{"VLESSInboundOptions", "VLESS", "in", ""},
		{"TrojanInboundOptions", "Trojan", "in", ""},
		{"NaiveInboundOptions", "Naive", "in", ""},
		{"HysteriaInboundOptions", "Hysteria", "in", ""},
		{"Hysteria2InboundOptions", "Hysteria2", "in", ""},
		{"TUICInboundOptions", "TUIC", "in", ""},
		{"AnyTLSInboundOptions", "AnyTls", "in", ""},
		{"ShadowTLSInboundOptions", "ShadowTLS", "in", ""},
		{"MieruInboundOptions", "Mieru", "in", ""},
		{"SudokuInboundOptions", "Sudoku", "in", ""},
		{"TrustTunnelInboundOptions", "TrustTunnel", "in", ""},
		{"CallInboundOptions", "Call", "in", ""},
		{"CloudflaredInboundOptions", "Cloudflared", "in", ""},
		{"SSHInboundOptions", "SSH", "in", ""},
		{"MTProxyInboundOptions", "MTProxy", "in", ""},
		{"TunInboundOptions", "Tun", "in", ""},
		{"RedirectInboundOptions", "Redirect", "in", ""},
		{"TProxyInboundOptions", "TProxy", "in", ""},
		{"BondInboundOptions", "BondInbound", "in", ""},
		{"FailoverInboundOptions", "CoreFailoverInbound", "in", ""},
		// Outbounds
		{"_DirectOutboundOptions", "Direct", "out", ""},
		{"SOCKSOutboundOptions", "SOCKS", "out", ""},
		{"HTTPOutboundOptions", "HTTP", "out", ""},
		{"ShadowsocksOutboundOptions", "Shadowsocks", "out", ""},
		{"VMessOutboundOptions", "VMESS", "out", ""},
		{"VLESSOutboundOptions", "VLESS", "out", ""},
		{"TrojanOutboundOptions", "Trojan", "out", ""},
		{"NaiveOutboundOptions", "Naive", "out", ""},
		{"HysteriaOutboundOptions", "Hysteria", "out", ""},
		{"Hysteria2OutboundOptions", "Hysteria2", "out", ""},
		{"TUICOutboundOptions", "TUIC", "out", ""},
		{"AnyTLSOutboundOptions", "AnyTls", "out", ""},
		{"ShadowTLSOutboundOptions", "ShadowTLS", "out", ""},
		{"MieruOutboundOptions", "Mieru", "out", ""},
		{"SudokuOutboundOptions", "Sudoku", "out", ""},
		{"TrustTunnelOutboundOptions", "TrustTunnel", "out", ""},
		{"CallOutboundOptions", "Call", "out", ""},
		{"SSHOutboundOptions", "SSH", "out", ""},
		{"TorOutboundOptions", "Tor", "out", ""},
		{"MASQUEOutboundOptions", "MASQUE", "out", ""},
		{"ParserOutboundOptions", "Parser", "out", ""},
		{"SelectorOutboundOptions", "Selector", "out", ""},
		{"URLTestOutboundOptions", "URLTest", "out", ""},
		{"FallbackOutboundOptions", "Fallback", "out", ""},
		{"BondOutboundOptions", "Bond", "out", ""},
		{"BandwidthLimiterOutboundOptions", "BandwidthLimiter", "out", ""},
		{"ConnectionLimiterOutboundOptions", "ConnectionLimiter", "out", ""},
		{"TrafficLimiterOutboundOptions", "TrafficLimiter", "out", ""},
		{"RateLimiterOutboundOptions", "RateLimiter", "out", ""},
		{"FailoverOutboundOptions", "CoreFailover", "out", ""},
		{"StubOptions", "Block", "out", ""},
		{"SnellOutboundOptions", "Snell", "out", ""},
		{"BridgeOutboundOptions", "Bridge", "out", ""},
		// Endpoints
		{"WireGuardEndpointOptions", "WireGuard", "ep", "Wireguard.vue"},
		{"WARPEndpointOptions", "Warp", "ep", ""},
		{"TailscaleEndpointOptions", "Tailscale", "ep", ""},
		{"VPNServerEndpointOptions", "VpnServer", "ep", ""},
		{"VPNClientEndpointOptions", "VpnClient", "ep", ""},
		{"OpenVPNClientEndpointOptions", "OpenVPNClient", "ep", "OpenVPNEndpoint.vue"},
		{"OpenVPNServerEndpointOptions", "OpenVPNServer", "ep", "OpenVPNEndpoint.vue"},
		{"OpenConnectEndpointOptions", "OpenConnect", "ep", "OpenConnect.vue"},
		// Providers
		{"ProviderInlineOptions", "ProviderInline", "prov", ""},
		{"ProviderLocalOptions", "ProviderLocal", "prov", ""},
		{"ProviderRemoteOptions", "ProviderRemote", "prov", ""},
		// Services
		{"ResolvedServiceOptions", "Resolved", "svc", ""},
		{"_SSMAPIServiceOptions", "SSMAPI", "svc", ""},
		{"_DERPServiceOptions", "DERP", "svc", ""},
		{"_CCMServiceOptions", "CCM", "svc", ""},
		{"_OCMServiceOptions", "OCM", "svc", ""},
		{"_OOMKillerServiceOptions", "OOMKiller", "svc", ""},
		{"_ProfilerServiceOptions", "Profiler", "svc", ""},
		{"_APIServiceOptions", "API", "svc", ""},
		{"_HysteriaRealmServiceOptions", "HysteriaRealm", "svc", ""},
		{"_USBIPServerServiceOptions", "USBIPServer", "svc", ""},
		{"USBIPClientServiceOptions", "USBIPClient", "svc", ""},
	}
	return mappings
}
