import { DnsTypes } from '@/types/dns'
import { EpTypes } from '@/types/endpoints'
import { InTypes } from '@/types/inbounds'
import { OutTypes } from '@/types/outbounds'
import { SrvTypes } from '@/types/services'
import { RECOMMENDED, dnsResolvers, dohPaths, sniFrontHosts, tlsAlpn } from '@/types/recommended'
import type { Inbound } from '@/types/inbounds'
import type { RecommendationContext, RecommendationSpec } from '@/utils/recommendations'

const anyTlsRecommendedPaddingScheme = [
  'stop=8',
  '0=30-30',
  '1=100-400',
  '2=400-500,c,500-1000,c,500-1000,c,500-1000,c,500-1000',
  '3=9-9,500-1000',
  '4=500-1000',
  '5=500-1000',
  '6=500-1000',
  '7=500-1000',
]

function typeAvailable(context: RecommendationContext<any>): boolean {
  return !context.type || !context.unavailableTypes?.includes(context.type)
}

export const inboundRecommendationSpecs: RecommendationSpec<Record<string, unknown>>[] = [
  { id: 'inbound-listen-any', label: 'Listen on all interfaces', description: 'Use :: for dual-stack listen when the protocol supports Listen.', path: 'listen', value: '::', when: (ctx) => typeAvailable(ctx) && ctx.type !== InTypes.Tun },
  { id: 'inbound-domain-resolver', label: 'Use local domain resolver', description: 'For local proxy inbounds, resolve domains through the local DNS server.', path: 'domain_resolver.server', value: 'local', when: (ctx) => [InTypes.SOCKS, InTypes.HTTP, InTypes.Mixed].includes(ctx.type as any) },
]

export const vlessInboundFieldHintKeys: Record<string, string> = {
  type: 'types.vless.hint.type',
  tag: 'types.vless.hint.tag',
  listen: 'types.vless.hint.listen',
  listen_port: 'types.vless.hint.listen_port',
  listen_detour: 'types.vless.hint.listen_detour',
  bind_interface: 'types.vless.hint.bind_interface',
  routing_mark: 'types.vless.hint.routing_mark',
  reuse_addr: 'types.vless.hint.reuse_addr',
  netns: 'types.vless.hint.netns',
  tcp_fast_open: 'types.vless.hint.tcp_fast_open',
  tcp_multi_path: 'types.vless.hint.tcp_multi_path',
  udp_fragment: 'types.vless.hint.udp_fragment',
  udp_timeout: 'types.vless.hint.udp_timeout',
  disable_tcp_keep_alive: 'types.vless.hint.disable_tcp_keep_alive',
  tcp_keep_alive: 'types.vless.hint.tcp_keep_alive',
  tcp_keep_alive_interval: 'types.vless.hint.tcp_keep_alive_interval',
  decryption: 'types.vless.hint.decryption',
  sniff: 'types.vless.hint.sniff',
  sniff_override_destination: 'types.vless.hint.sniff_override_destination',
  sniff_timeout: 'types.vless.hint.sniff_timeout',
  proxy_protocol: 'types.vless.hint.proxy_protocol',
  proxy_protocol_accept_no_header: 'types.vless.hint.proxy_protocol_accept_no_header',
  domain_strategy: 'types.vless.hint.domain_strategy',
  udp_disable_domain_unmapping: 'types.vless.hint.udp_disable_domain_unmapping',
  transport_enable: 'types.vless.hint.transport_enable',
  transport_type: 'types.vless.hint.transport_type',
  users: 'types.vless.hint.users',
  users_group: 'types.vless.hint.users_group',
  users_client: 'types.vless.hint.users_client',
  tls_id: 'types.vless.hint.tls_id',
  inbound_multiplex_enable: 'types.vless.hint.inbound_multiplex_enable',
  inbound_multiplex_padding: 'types.vless.hint.inbound_multiplex_padding',
  inbound_multiplex_brutal: 'types.vless.hint.inbound_multiplex_brutal',
  inbound_multiplex_brutal_up_mbps: 'types.vless.hint.inbound_multiplex_brutal_up_mbps',
  inbound_multiplex_brutal_down_mbps: 'types.vless.hint.inbound_multiplex_brutal_down_mbps',
  out_json_network: 'types.vless.hint.out_json_network',
  out_json_packet_encoding: 'types.vless.hint.out_json_packet_encoding',
  dial_options: 'types.vless.hint.dial_options',
  dial_tcp_fast_open: 'types.vless.hint.dial_tcp_fast_open',
  dial_tcp_multi_path: 'types.vless.hint.dial_tcp_multi_path',
  dial_udp_fragment: 'types.vless.hint.dial_udp_fragment',
  dial_connect_timeout: 'types.vless.hint.dial_connect_timeout',
  dial_disable_tcp_keep_alive: 'types.vless.hint.dial_disable_tcp_keep_alive',
  dial_tcp_keep_alive: 'types.vless.hint.dial_tcp_keep_alive',
  dial_tcp_keep_alive_interval: 'types.vless.hint.dial_tcp_keep_alive_interval',
  multi_domain: 'types.vless.hint.multi_domain',
  addr_server: 'types.vless.hint.addr_server',
  addr_server_port: 'types.vless.hint.addr_server_port',
  addr_remark: 'types.vless.hint.addr_remark',
  addr_tls: 'types.vless.hint.addr_tls',
  out_multiplex_enable: 'types.vless.hint.out_multiplex_enable',
  out_multiplex_protocol: 'types.vless.hint.out_multiplex_protocol',
  out_multiplex_max_connections: 'types.vless.hint.out_multiplex_max_connections',
  out_multiplex_min_streams: 'types.vless.hint.out_multiplex_min_streams',
  out_multiplex_max_streams: 'types.vless.hint.out_multiplex_max_streams',
  out_multiplex_padding: 'types.vless.hint.out_multiplex_padding',
  out_multiplex_brutal: 'types.vless.hint.out_multiplex_brutal',
  out_multiplex_brutal_up_mbps: 'types.vless.hint.out_multiplex_brutal_up_mbps',
  out_multiplex_brutal_down_mbps: 'types.vless.hint.out_multiplex_brutal_down_mbps',
}

export const vlessInboundFieldHints = vlessInboundFieldHintKeys

export const commonInboundFieldHintKeys: Record<string, string> = {
  type: 'types.inbound.hint.type',
  tag: 'types.inbound.hint.tag',
  listen: 'types.inbound.hint.listen',
  listen_port: 'types.inbound.hint.listen_port',
  listen_detour: 'types.inbound.hint.listen_detour',
  bind_interface: 'types.inbound.hint.bind_interface',
  routing_mark: 'types.inbound.hint.routing_mark',
  reuse_addr: 'types.inbound.hint.reuse_addr',
  netns: 'types.inbound.hint.netns',
  tcp_fast_open: 'types.inbound.hint.tcp_fast_open',
  tcp_multi_path: 'types.inbound.hint.tcp_multi_path',
  udp_fragment: 'types.inbound.hint.udp_fragment',
  udp_timeout: 'types.inbound.hint.udp_timeout',
  disable_tcp_keep_alive: 'types.inbound.hint.disable_tcp_keep_alive',
  tcp_keep_alive: 'types.inbound.hint.tcp_keep_alive',
  tcp_keep_alive_interval: 'types.inbound.hint.tcp_keep_alive_interval',
  network: 'types.inbound.hint.network',
  sniff: 'types.inbound.hint.sniff',
  sniff_override_destination: 'types.inbound.hint.sniff_override_destination',
  sniff_timeout: 'types.inbound.hint.sniff_timeout',
  proxy_protocol: 'types.inbound.hint.proxy_protocol',
  proxy_protocol_accept_no_header: 'types.inbound.hint.proxy_protocol_accept_no_header',
  domain_strategy: 'types.inbound.hint.domain_strategy',
  udp_disable_domain_unmapping: 'types.inbound.hint.udp_disable_domain_unmapping',
  transport_enable: 'types.inbound.hint.transport_enable',
  transport_type: 'types.inbound.hint.transport_type',
  users: 'types.inbound.hint.users',
  users_group: 'types.inbound.hint.users_group',
  users_client: 'types.inbound.hint.users_client',
  tls_id: 'types.inbound.hint.tls_id',
  inbound_multiplex_enable: 'types.inbound.hint.inbound_multiplex_enable',
  inbound_multiplex_padding: 'types.inbound.hint.inbound_multiplex_padding',
  inbound_multiplex_brutal: 'types.inbound.hint.inbound_multiplex_brutal',
  inbound_multiplex_brutal_up_mbps: 'types.inbound.hint.inbound_multiplex_brutal_up_mbps',
  inbound_multiplex_brutal_down_mbps: 'types.inbound.hint.inbound_multiplex_brutal_down_mbps',
  out_json_network: 'types.inbound.hint.out_json_network',
  out_json_packet_encoding: 'types.inbound.hint.out_json_packet_encoding',
  dial_options: 'types.inbound.hint.dial_options',
  dial_tcp_fast_open: 'types.inbound.hint.dial_tcp_fast_open',
  dial_tcp_multi_path: 'types.inbound.hint.dial_tcp_multi_path',
  dial_udp_fragment: 'types.inbound.hint.dial_udp_fragment',
  dial_connect_timeout: 'types.inbound.hint.dial_connect_timeout',
  dial_disable_tcp_keep_alive: 'types.inbound.hint.dial_disable_tcp_keep_alive',
  dial_tcp_keep_alive: 'types.inbound.hint.dial_tcp_keep_alive',
  dial_tcp_keep_alive_interval: 'types.inbound.hint.dial_tcp_keep_alive_interval',
  multi_domain: 'types.inbound.hint.multi_domain',
  addr_server: 'types.inbound.hint.addr_server',
  addr_server_port: 'types.inbound.hint.addr_server_port',
  addr_remark: 'types.inbound.hint.addr_remark',
  addr_tls: 'types.inbound.hint.addr_tls',
  out_multiplex_enable: 'types.inbound.hint.out_multiplex_enable',
  out_multiplex_protocol: 'types.inbound.hint.out_multiplex_protocol',
  out_multiplex_max_connections: 'types.inbound.hint.out_multiplex_max_connections',
  out_multiplex_min_streams: 'types.inbound.hint.out_multiplex_min_streams',
  out_multiplex_max_streams: 'types.inbound.hint.out_multiplex_max_streams',
  out_multiplex_padding: 'types.inbound.hint.out_multiplex_padding',
  out_multiplex_brutal: 'types.inbound.hint.out_multiplex_brutal',
  out_multiplex_brutal_up_mbps: 'types.inbound.hint.out_multiplex_brutal_up_mbps',
  out_multiplex_brutal_down_mbps: 'types.inbound.hint.out_multiplex_brutal_down_mbps',
  override_address: 'types.inbound.hint.override_address',
  override_port: 'types.inbound.hint.override_port',
  set_system_proxy: 'types.inbound.hint.set_system_proxy',
  method: 'types.inbound.hint.method',
  password: 'types.inbound.hint.password',
  managed: 'types.inbound.hint.managed',
  security: 'types.inbound.hint.security',
  global_padding: 'types.inbound.hint.global_padding',
  authenticated_length: 'types.inbound.hint.authenticated_length',
  quic_congestion_control: 'types.inbound.hint.quic_congestion_control',
  congestion_control: 'types.inbound.hint.congestion_control',
  up_mbps: 'types.inbound.hint.up_mbps',
  down_mbps: 'types.inbound.hint.down_mbps',
  padding_scheme: 'types.inbound.hint.padding_scheme',
  server_version: 'types.inbound.hint.server_version',
  max_auth_tries: 'types.inbound.hint.max_auth_tries',
  protocol_note: 'types.inbound.hint.protocol_note',
}

const inboundTypesWithoutPreset = new Set<string>([
  InTypes.ShadowTLS,
  InTypes.MTProxy,
  InTypes.Tun,
  InTypes.Bond,
  InTypes.CoreFailover,
  InTypes.Redirect,
  InTypes.Mixed,
  InTypes.SOCKS,
  InTypes.HTTP,
])

export function inboundFieldHintsForType(type: string): Record<string, string> {
  if (type === InTypes.VLESS) return vlessInboundFieldHints
  return commonInboundFieldHintKeys
}

export function hasInboundRecommendedPreset(type: string): boolean {
  return !inboundTypesWithoutPreset.has(type) && type !== InTypes.VLESS
}

function applyRecommendedListen(target: Record<string, any>): void {
  target.listen = target.listen || '::'
}

function applyRecommendedAdvanced(target: Record<string, any>): void {
  target.sniff = true
  target.sniff_override_destination = true
  target.sniff_timeout = '300ms'
  delete target.proxy_protocol
  delete target.proxy_protocol_accept_no_header
  delete target.domain_strategy
  delete target.udp_disable_domain_unmapping
}

export function applyInboundRecommendedValues(inbound: Inbound): void {
  const target = inbound as Record<string, any>
  if (target.type === InTypes.VLESS) {
    applyVlessInboundRecommendedValues(inbound)
    return
  }

  if (!hasInboundRecommendedPreset(target.type)) return

  applyRecommendedListen(target)
  applyRecommendedAdvanced(target)

  switch (target.type) {
    case InTypes.VMess:
      target.out_json = target.out_json && typeof target.out_json === 'object' ? target.out_json : {}
      target.out_json.security = 'auto'
      target.out_json.packet_encoding = RECOMMENDED.vlessPacketEncoding
      target.out_json.global_padding = true
      target.out_json.authenticated_length = true
      break
    case InTypes.Naive:
      target.quic_congestion_control = 'bbr'
      break
    case InTypes.TUIC:
      target.congestion_control = 'bbr'
      break
    case InTypes.Hysteria:
    case InTypes.Hysteria2:
      target.up_mbps = 100
      target.down_mbps = 100
      break
    case InTypes.AnyTls:
      target.padding_scheme = [...anyTlsRecommendedPaddingScheme]
      break
    case InTypes.Mieru:
      target.transport = 'TCP'
      break
    case InTypes.Sudoku:
      target.aead_method = RECOMMENDED.sudokuAead
      target.padding_min = RECOMMENDED.sudokuPaddingMin
      target.padding_max = RECOMMENDED.sudokuPaddingMax
      target.handshake_timeout = RECOMMENDED.sudokuHandshakeTimeout
      target.enable_pure_downlink = true
      break
    case InTypes.TrustTunnel:
      target.network = ['tcp', 'udp']
      target.congestion_controller = RECOMMENDED.trustTunnelCongestion
      break
    case InTypes.SSH:
      target.server_version = 'SSH-2.0-OpenSSH_9.7'
      target.max_auth_tries = 3
      break
  }
}

export function applyVlessInboundRecommendedValues(inbound: Inbound): void {
  const target = inbound as Record<string, any>

  target.listen = target.listen || '::'
  target.decryption = 'none'
  target.sniff = true
  target.sniff_override_destination = true
  target.sniff_timeout = '300ms'
  target.transport = {}

  delete target.proxy_protocol
  delete target.proxy_protocol_accept_no_header
  delete target.domain_strategy
  delete target.udp_disable_domain_unmapping
  delete target.multiplex

  target.out_json = target.out_json && typeof target.out_json === 'object' ? target.out_json : {}
  target.out_json.packet_encoding = RECOMMENDED.vlessPacketEncoding
  delete target.out_json.multiplex
}

export const outboundRecommendationSpecs: RecommendationSpec<Record<string, unknown>>[] = [
  { id: 'outbound-server-port-443', label: 'Use HTTPS port', description: 'Common TLS/QUIC protocols usually listen on port 443.', path: 'server_port', value: 443, when: (ctx) => typeAvailable(ctx) && [OutTypes.VLESS, OutTypes.VMess, OutTypes.Trojan, OutTypes.Hysteria2, OutTypes.TUIC, OutTypes.AnyTls, OutTypes.Naive].includes(ctx.type as any) },
  { id: 'outbound-tls-min-13', label: 'Require TLS 1.3', description: 'Prefer TLS 1.3 where outbound TLS settings are available.', path: 'tls.min_version', value: RECOMMENDED.tlsMinVersion, when: (ctx) => typeAvailable(ctx) && ctx.model != null && Object.hasOwn(ctx.model as object, 'tls') },
  { id: 'outbound-utls-chrome', label: 'Use Chrome uTLS fingerprint', description: 'Recommended client fingerprint for outbound TLS.', path: 'tls.utls', value: { enabled: true, fingerprint: 'chrome' }, when: (ctx) => typeAvailable(ctx) && ctx.model != null && Object.hasOwn(ctx.model as object, 'tls') },
]

export const serviceRecommendationSpecs: RecommendationSpec<Record<string, unknown>>[] = [
  { id: 'service-listen-any', label: 'Listen on all interfaces', description: 'Use :: for dual-stack listen when this service has Listen settings.', path: 'listen', value: '::', when: (ctx) => ![SrvTypes.OOMKiller, SrvTypes.Profiler].includes(ctx.type as any) && (ctx.model as any)?.listen_port != null },
]

export const endpointRecommendationSpecs: RecommendationSpec<Record<string, unknown>>[] = [
  { id: 'endpoint-wireguard-mtu', label: 'WireGuard MTU 1420', description: 'A safe MTU for common WireGuard deployments.', path: 'mtu', value: 1420, when: (ctx) => ctx.type === EpTypes.Wireguard },
  { id: 'endpoint-vpn-server-route', label: 'VPN server first client address', description: 'Use the default first client address for a new VPN server user.', path: 'users.0.address', value: '10.0.0.2', when: (ctx) => ctx.type === EpTypes.VpnServer },
]

export const tlsRecommendationSpecs: RecommendationSpec<Record<string, unknown>>[] = [
  { id: 'tls-min-13', label: 'Minimum TLS 1.3', description: 'Use TLS 1.3 as the minimum version when enabled.', path: 'server.min_version', value: RECOMMENDED.tlsMinVersion, when: (ctx) => !(ctx.model as any)?.server?.reality },
  { id: 'tls-max-13', label: 'Maximum TLS 1.3', description: 'Pin maximum TLS version to TLS 1.3 when enabled.', path: 'server.max_version', value: RECOMMENDED.tlsMaxVersion, when: (ctx) => !(ctx.model as any)?.server?.reality },
  { id: 'tls-alpn-modern', label: 'Modern ALPN list', description: 'Advertise HTTP/3, HTTP/2 and HTTP/1.1.', path: 'server.alpn', value: tlsAlpn.map((item) => item.value), when: (ctx) => !(ctx.model as any)?.server?.reality },
  { id: 'tls-utls-chrome', label: 'Chrome client fingerprint', description: 'Use Chrome as the outbound uTLS fingerprint.', path: 'client.utls', value: { enabled: true, fingerprint: 'chrome' }, when: (ctx) => !(ctx.model as any)?.server?.reality },
  { id: 'reality-handshake-server', label: 'Reality handshake server', description: 'Use a well-known TLS 1.3 host as the Reality handshake destination.', path: 'server.reality.handshake.server', value: sniFrontHosts[0], when: (ctx) => Boolean((ctx.model as any)?.server?.reality) },
]

export const dnsRecommendationSpecs: RecommendationSpec<Record<string, unknown>>[] = [
  { id: 'dns-server-cloudflare', label: 'Cloudflare resolver', description: 'Use 1.1.1.1 as an editable upstream DNS server.', path: 'server', value: dnsResolvers[0], when: (ctx) => [DnsTypes.TCP, DnsTypes.UDP, DnsTypes.TLS, DnsTypes.QUIC].includes(ctx.type as any) },
  { id: 'dns-doh-server', label: 'Cloudflare DoH host', description: 'Use cloudflare-dns.com with the standard DoH path.', path: 'server', value: 'cloudflare-dns.com', when: (ctx) => [DnsTypes.HTTPS, DnsTypes.HTTP3].includes(ctx.type as any) },
  { id: 'dns-doh-path', label: 'Standard DoH path', description: 'Use /dns-query for DoH and HTTP/3 DNS.', path: 'path', value: dohPaths[0], when: (ctx) => [DnsTypes.HTTPS, DnsTypes.HTTP3].includes(ctx.type as any) },
]

export const dnsRuleRecommendationSpecs: RecommendationSpec<Record<string, unknown>>[] = [
  { id: 'dns-rule-mode-or', label: 'Logical OR mode', description: 'OR mode is usually expected when adding multiple match rules.', path: 'mode', value: 'or', when: (ctx) => (ctx.model as any).type === 'logical' },
  { id: 'dns-rule-strategy-ipv4', label: 'Prefer IPv4 strategy', description: 'Prefer IPv4 for DNS route actions when a strategy is needed.', path: 'strategy', value: 'prefer_ipv4', when: (ctx) => (ctx.model as any).action === 'route' },
]

export const ruleRecommendationSpecs: RecommendationSpec<Record<string, unknown>>[] = [
  { id: 'rule-mode-or', label: 'Logical OR mode', description: 'OR mode is usually expected when adding multiple match rules.', path: 'mode', value: 'or', when: (ctx) => (ctx.model as any).type === 'logical' },
  { id: 'rule-udp-timeout', label: 'UDP timeout 5m', description: 'Use an explicit UDP timeout for route options.', path: 'udp_timeout', value: '5m', when: (ctx) => ['route', 'route-options', 'bypass'].includes((ctx.model as any).action) },
]
