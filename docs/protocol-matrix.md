<!-- GENERATED from core/capabilities/protocols.json by RenderMatrix (capabilities/matrix.go).
     Do not edit by hand; regenerate with: go test ./core/capabilities -run TestProtocolMatrixDoc -update -->

# Protocol capability matrix

Single source of truth: `core/capabilities/protocols.json`. Derived backend maps, frontend lists, and this matrix all come from it.

| Type | in | out | group | endpoint | service | tls-tmpl | users | clientDelivery | buildTag | assembledAs | notes/gap |
|---|---|---|---|---|---|---|---|---|---|---|---|
| socks | ✓ | ✓ | – | – | – | – | ✓ | uri | – | – | — |
| http | ✓ | ✓ | – | – | – | ✓ | ✓ | uri | – | – | — |
| mixed | ✓ | – | – | – | – | – | ✓ | uri | – | – | — |
| shadowsocks | ✓ | ✓ | – | – | – | – | ✓ | uri | – | – | — |
| vmess | ✓ | ✓ | – | – | – | ✓ | ✓ | uri | – | – | — |
| vless | ✓ | ✓ | – | – | – | ✓ | ✓ | uri | – | – | — |
| trojan | ✓ | ✓ | – | – | – | ✓ | ✓ | uri | – | – | — |
| naive | ✓ | ✓ | – | – | – | ✓ | ✓ | uri | with_naive_outbound | – | — |
| hysteria | ✓ | ✓ | – | – | – | ✓ | ✓ | uri | with_quic | – | — |
| hysteria2 | ✓ | ✓ | – | – | – | ✓ | ✓ | uri | with_quic | – | — |
| tuic | ✓ | ✓ | – | – | – | ✓ | ✓ | uri | with_quic | – | — |
| anytls | ✓ | ✓ | – | – | – | ✓ | ✓ | uri | – | – | — |
| shadowtls | ✓ | ✓ | – | – | – | – | ✓ | broken | – | – | BROKEN delivery: server backing-shadowsocks detour is never created and the client out_json is a lone shadowtls outbound with no paired shadowsocks. Also NOT fail-closed: a shadowtls inbound without listen.detour does not reject; the decrypted inner stream falls through to normal routing (see docs/project-context.md). Auto-pair design deferred. |
| mieru | ✓ | ✓ | – | – | – | – | ✓ | json | – | – | out_json copies transport/traffic_pattern and maps listen_ports->server_ports; username/password merged per-user |
| sudoku | ✓ | ✓ | – | – | – | – | – | json | with_sudoku | – | keyless (no server-side per-user objects); an optional client split key overrides only outbound.key during delivery, otherwise the inbound key is used. http_mask is built nested from flat inbound fields, merged with C-side host/multiplex |
| trusttunnel | ✓ | ✓ | – | – | – | ✓ | ✓ | json | with_trusttunnel | – | out_json copies network/quic/congestion_controller/cwnd; NOT health_check/multiplex/username/password; username/password merged per-user |
| ssh | ✓ | ✓ | – | – | – | – | ✓ | json | – | – | out_json copies NOTHING server-side (host_key* are PRIVATE server keys); user/password merged per-user |
| mtproxy | ✓ | – | – | – | – | – | ✓ | telegram | with_mtproxy | – | no sing-box mtproxy outbound exists: delivered ONLY as a tg://proxy link (linkScheme tg) from the per-user secret. Excluded from JSON/Clash by design (outJsonBuilder empty -> out_json wiped -> subscription len<5 guard skips it). |
| direct | ✓ | ✓ | – | – | – | – | – | none | – | – | — |
| call | ✓ | ✓ | – | – | – | – | – | none | with_call | – | server-side WebRTC bridge (dion/telemost/vk/wbstream); inbound creates a call room, clients join via join_link. No per-user credentials or subscription outbound.; client-side WebRTC bridge; requires join_link (empty = create a new call on the platform) |
| tun | ✓ | – | – | – | – | – | – | none | – | – | — |
| redirect | ✓ | – | – | – | – | – | – | none | – | – | — |
| tproxy | ✓ | – | – | – | – | – | – | none | – | – | — |
| bond | ✓ | – | – | – | – | – | – | none | – | – | native core bond inbound |
| core-failover | ✓ | ✓ | – | – | – | – | – | none | – | – | native core failover inbound; native core failover outbound (dial-time) |
| block | – | ✓ | – | – | – | – | – | – | – | – | — |
| tor | – | ✓ | – | – | – | – | – | – | – | – | — |
| masque | – | ✓ | – | – | – | – | – | – | with_masque | – | — |
| openvpn | – | ✓ | – | – | – | – | – | – | with_openvpn | – | — |
| selector | – | – | ✓ | – | – | – | – | – | – | selector | Manual operator-selected group backed directly by the core selector outbound. |
| urltest | – | – | ✓ | – | – | – | – | – | – | urltest | Latency-based group backed directly by the core urltest outbound. |
| fallback | – | – | ✓ | – | – | – | – | – | – | fallback | Dial-time fallback group backed directly by the core fallback outbound. |
| failover | – | – | ✓ | – | – | – | – | – | – | selector | Panel-managed priority failover assembled as a core selector. Switches new connections only; existing sessions may break. |
| wireguard | – | – | – | ✓ | – | – | – | – | with_wireguard | – | warp maps to a wireguard endpoint |
| tailscale | – | – | – | ✓ | – | – | – | – | with_tailscale | – | — |
| vpn | – | – | – | ✓ | – | – | – | – | – | – | warp/vpn client+server endpoint |
| resolved | – | – | – | – | ✓ | – | – | – | – | – | — |
| ssm-api | – | – | – | – | ✓ | – | – | – | – | – | — |
| derp | – | – | – | – | ✓ | – | – | – | – | – | — |
| ccm | – | – | – | – | ✓ | – | – | – | with_ccm | – | — |
| ocm | – | – | – | – | ✓ | – | – | – | with_ocm | – | — |
| oom-killer | – | – | – | – | ✓ | – | – | – | with_oomkiller | – | — |
| profiler | – | – | – | – | ✓ | – | – | – | with_profiler | – | dev-only |
