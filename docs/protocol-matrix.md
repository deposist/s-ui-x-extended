<!-- GENERATED from core/capabilities/protocols.json by RenderMatrix (capabilities/matrix.go).
     Do not edit by hand; regenerate with: go test ./core/capabilities -run TestProtocolMatrixDoc -update -->

# Protocol capability matrix

Single source of truth: `core/capabilities/protocols.json`. Derived backend maps, frontend lists, and this matrix all come from it.

| Type | in | out | group | endpoint | service | tls-tmpl | users | clientDelivery | clash | buildTag | platforms | assembledAs | notes/gap |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| socks | ✓ | ✓ | – | – | – | – | ✓ | uri | proxy | – | all | – | — |
| http | ✓ | ✓ | – | – | – | ✓ | ✓ | uri | proxy | – | all | – | — |
| mixed | ✓ | – | – | – | – | – | ✓ | uri | – | – | all | – | — |
| shadowsocks | ✓ | ✓ | – | – | – | – | ✓ | uri | proxy | – | all | – | — |
| vmess | ✓ | ✓ | – | – | – | ✓ | ✓ | uri | proxy | – | all | – | — |
| vless | ✓ | ✓ | – | – | – | ✓ | ✓ | uri | proxy | – | all | – | — |
| trojan | ✓ | ✓ | – | – | – | ✓ | ✓ | uri | proxy | – | all | – | — |
| naive | ✓ | ✓ | – | – | – | ✓ | ✓ | uri | unsupported | with_naive_outbound | all | – | — |
| hysteria | ✓ | ✓ | – | – | – | ✓ | ✓ | uri | proxy | with_quic | all | – | — |
| hysteria2 | ✓ | ✓ | – | – | – | ✓ | ✓ | uri | proxy | with_quic | all | – | — |
| tuic | ✓ | ✓ | – | – | – | ✓ | ✓ | uri | proxy | with_quic | all | – | — |
| anytls | ✓ | ✓ | – | – | – | ✓ | ✓ | uri | proxy | – | all | – | — |
| shadowtls | ✓ | ✓ | – | – | – | – | ✓ | broken | unsupported | – | all | – | BROKEN delivery: server backing-shadowsocks detour is never created and the client out_json is a lone shadowtls outbound with no paired shadowsocks. Also NOT fail-closed: a shadowtls inbound without listen.detour does not reject; the decrypted inner stream falls through to normal routing (see docs/project-context.md). Auto-pair design deferred. |
| mieru | ✓ | ✓ | – | – | – | – | ✓ | json | unsupported | – | all | – | out_json copies transport/traffic_pattern and maps listen_ports->server_ports; username/password merged per-user |
| sudoku | ✓ | ✓ | – | – | – | – | – | json | unsupported | with_sudoku | all | – | keyless (no server-side per-user objects); an optional client split key overrides only outbound.key during delivery, otherwise the inbound key is used. http_mask is built nested from flat inbound fields, merged with C-side host/multiplex |
| trusttunnel | ✓ | ✓ | – | – | – | ✓ | ✓ | json | unsupported | with_trusttunnel | all | – | out_json copies network/quic/congestion_controller/cwnd; NOT health_check/multiplex/username/password; username/password merged per-user |
| ssh | ✓ | ✓ | – | – | – | – | ✓ | json | unsupported | – | all | – | out_json copies NOTHING server-side (host_key* are PRIVATE server keys); user/password merged per-user |
| mtproxy | ✓ | – | – | – | – | – | ✓ | telegram | – | with_mtproxy | all | – | no sing-box mtproxy outbound exists: delivered ONLY as a tg://proxy link (linkScheme tg) from the per-user secret. Excluded from JSON/Clash by design (outJsonBuilder empty -> out_json wiped -> subscription len<5 guard skips it). |
| direct | ✓ | ✓ | – | – | – | – | – | none | none | – | all | – | — |
| call | ✓ | ✓ | – | – | – | – | – | none | unsupported | with_call | all | – | server-side WebRTC bridge (dion/telemost/vk/wbstream); inbound creates a call room, clients join via join_link. No per-user credentials or subscription outbound.; client-side WebRTC bridge; requires join_link (empty = create a new call on the platform) |
| cloudflared | ✓ | – | – | – | – | – | – | none | – | with_cloudflared | all | – | Cloudflare tunnel client inbound (token based); it carries edge traffic into the core, so it has no client outbound and no subscription delivery. |
| tun | ✓ | – | – | – | – | – | – | none | – | – | all | – | — |
| redirect | ✓ | – | – | – | – | – | – | none | – | – | linux/darwin | – | REDIRECT is implemented on linux/darwin only; on other platforms the core returns os.ErrInvalid at start |
| tproxy | ✓ | – | – | – | – | – | – | none | – | – | linux | – | TPROXY is linux only; other platforms get os.ErrInvalid from the core at start |
| bond | ✓ | ✓ | – | – | – | – | – | none | none | – | all | – | native core bond inbound; panel bond outbound aggregating member outbounds |
| core-failover | ✓ | ✓ | – | – | – | – | – | none | none | – | all | – | native core failover inbound; native core failover outbound (dial-time) |
| block | – | ✓ | – | – | – | – | – | – | none | – | all | – | — |
| tor | – | ✓ | – | – | – | – | – | – | unsupported | – | all | – | — |
| masque | – | ✓ | – | – | – | – | – | – | unsupported | with_masque | all | – | — |
| snell | – | ✓ | – | – | – | – | – | – | unsupported | – | all | – | snell proxy (v4 obfs / v6 mode); the core also registers a snell inbound, the panel exposes the outbound only |
| bridge | – | ✓ | – | – | – | – | – | – | none | – | all | – | — |
| parser | – | ✓ | – | – | – | – | – | – | none | – | all | – | panel parser outbound wrapping a detour with protocol parsing |
| bandwidth-limiter | – | ✓ | – | – | – | – | – | – | none | – | all | – | panel bandwidth limiter outbound |
| connection-limiter | – | ✓ | – | – | – | – | – | – | none | – | all | – | panel connection limiter outbound |
| traffic-limiter | – | ✓ | – | – | – | – | – | – | none | – | all | – | panel traffic limiter outbound |
| rate-limiter | – | ✓ | – | – | – | – | – | – | none | – | all | – | panel rate limiter outbound |
| selector | – | – | ✓ | – | – | – | – | – | – | – | all | selector | Manual operator-selected group backed directly by the core selector outbound. |
| urltest | – | – | ✓ | – | – | – | – | – | – | – | all | urltest | Latency-based group backed directly by the core urltest outbound. |
| fallback | – | – | ✓ | – | – | – | – | – | – | – | all | fallback | Dial-time fallback group backed directly by the core fallback outbound. |
| failover | – | – | ✓ | – | – | – | – | – | – | – | all | selector | Panel-managed priority failover assembled as a core selector. Switches new connections only; existing sessions may break. |
| wireguard | – | – | – | ✓ | – | – | – | – | – | with_wireguard | all | – | warp maps to a wireguard endpoint |
| tailscale | – | – | – | ✓ | – | – | – | – | – | with_tailscale | all | – | — |
| warp | – | – | – | ✓ | – | – | – | – | – | with_wireguard | all | – | warp maps to a wireguard endpoint |
| vpn-server | – | – | – | ✓ | – | – | – | – | – | – | all | – | panel VPN server endpoint (core type vpn-server); assembled from server users |
| vpn-client | – | – | – | ✓ | – | – | – | – | – | – | all | – | panel VPN client endpoint (core type vpn-client); assembled from the selected outbound |
| openvpn-client | – | – | – | ✓ | – | – | – | – | – | with_openvpn | all | – | — |
| openvpn-server | – | – | – | ✓ | – | – | – | – | – | with_openvpn | all | – | — |
| openconnect | – | – | – | ✓ | – | – | – | – | – | with_openconnect | all | – | — |
| resolved | – | – | – | – | ✓ | – | – | – | – | – | all | – | — |
| ssm-api | – | – | – | – | ✓ | – | – | – | – | – | all | – | — |
| derp | – | – | – | – | ✓ | – | – | – | – | – | all | – | — |
| ccm | – | – | – | – | ✓ | – | – | – | – | with_ccm | all | – | — |
| ocm | – | – | – | – | ✓ | – | – | – | – | with_ocm | all | – | — |
| oom-killer | – | – | – | – | ✓ | – | – | – | – | with_oomkiller | all | – | — |
| profiler | – | – | – | – | ✓ | – | – | – | – | with_profiler | all | – | dev-only |
| usbip-server | – | – | – | – | ✓ | – | – | – | – | with_usbip | all | – | Local USB/IP device server; the core registers it for linux/windows (and darwin with cgo), so it is not offered elsewhere. Defaults to a loopback listen. |
| usbip-client | – | – | – | – | ✓ | – | – | – | – | with_usbip | all | – | Dials a remote USB/IP server and re-exports the selected devices; same platform limits as usbip-server. |
| hysteria-realm | – | – | – | – | ✓ | – | – | – | – | with_quic | all | – | — |
| api | – | – | – | – | ✓ | – | – | – | – | – | all | – | — |
