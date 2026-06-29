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

export const commonOutboundFieldHintKeys: Record<string, string> = {
  type: 'types.outbound.hint.type',
  tag: 'types.outbound.hint.tag',
  server: 'types.outbound.hint.server',
  server_port: 'types.outbound.hint.server_port',
  network: 'types.outbound.hint.network',
  method: 'types.outbound.hint.method',
  password: 'types.outbound.hint.password',
  security: 'types.outbound.hint.security',
  global_padding: 'types.outbound.hint.global_padding',
  authenticated_length: 'types.outbound.hint.authenticated_length',
  packet_encoding: 'types.outbound.hint.packet_encoding',
  flow: 'types.outbound.hint.flow',
  quic: 'types.outbound.hint.quic',
  quic_congestion_control: 'types.outbound.hint.quic_congestion_control',
  congestion_control: 'types.outbound.hint.congestion_control',
  congestion_controller: 'types.outbound.hint.congestion_controller',
  udp_relay_mode: 'types.outbound.hint.udp_relay_mode',
  up_mbps: 'types.outbound.hint.up_mbps',
  down_mbps: 'types.outbound.hint.down_mbps',
  idle_session_check_interval: 'types.outbound.hint.idle_session_check_interval',
  idle_session_timeout: 'types.outbound.hint.idle_session_timeout',
  min_idle_session: 'types.outbound.hint.min_idle_session',
  transport: 'types.outbound.hint.transport',
  multiplexing: 'types.outbound.hint.multiplexing',
  aead_method: 'types.outbound.hint.aead_method',
  padding_min: 'types.outbound.hint.padding_min',
  padding_max: 'types.outbound.hint.padding_max',
  enable_pure_downlink: 'types.outbound.hint.enable_pure_downlink',
  proto: 'types.outbound.hint.proto',
  cipher: 'types.outbound.hint.cipher',
  auth: 'types.outbound.hint.auth',
  transport_enable: 'types.outbound.hint.transport_enable',
  transport_type: 'types.outbound.hint.transport_type',
  out_multiplex_enable: 'types.outbound.hint.out_multiplex_enable',
  out_multiplex_protocol: 'types.outbound.hint.out_multiplex_protocol',
  out_multiplex_max_connections: 'types.outbound.hint.out_multiplex_max_connections',
  out_multiplex_min_streams: 'types.outbound.hint.out_multiplex_min_streams',
  out_multiplex_max_streams: 'types.outbound.hint.out_multiplex_max_streams',
  out_multiplex_padding: 'types.outbound.hint.out_multiplex_padding',
  out_multiplex_brutal: 'types.outbound.hint.out_multiplex_brutal',
  out_multiplex_brutal_up_mbps: 'types.outbound.hint.out_multiplex_brutal_up_mbps',
  out_multiplex_brutal_down_mbps: 'types.outbound.hint.out_multiplex_brutal_down_mbps',
  tls_enable: 'types.outbound.hint.tls_enable',
  tls_min_version: 'types.outbound.hint.tls_min_version',
  tls_utls: 'types.outbound.hint.tls_utls',
  tls_reality: 'types.outbound.hint.tls_reality',
  dial_options: 'types.outbound.hint.dial_options',
  dial_tcp_fast_open: 'types.outbound.hint.dial_tcp_fast_open',
  dial_tcp_multi_path: 'types.outbound.hint.dial_tcp_multi_path',
  dial_udp_fragment: 'types.outbound.hint.dial_udp_fragment',
  dial_connect_timeout: 'types.outbound.hint.dial_connect_timeout',
  dial_disable_tcp_keep_alive: 'types.outbound.hint.dial_disable_tcp_keep_alive',
  dial_tcp_keep_alive: 'types.outbound.hint.dial_tcp_keep_alive',
  dial_tcp_keep_alive_interval: 'types.outbound.hint.dial_tcp_keep_alive_interval',
}

const outboundTypesWithoutPreset = new Set<string>([
  OutTypes.Direct,
  OutTypes.SOCKS,
  OutTypes.HTTP,
  OutTypes.Shadowsocks,
  OutTypes.ShadowTLS,
  OutTypes.Tor,
  OutTypes.SSH,
  OutTypes.MASQUE,
  OutTypes.Parser,
  OutTypes.Selector,
  OutTypes.URLTest,
  OutTypes.Bond,
  OutTypes.Failover,
  OutTypes.Fallback,
  OutTypes.BandwidthLimiter,
  OutTypes.ConnectionLimiter,
  OutTypes.TrafficLimiter,
  OutTypes.RateLimiter,
  OutTypes.Block,
  OutTypes.CoreFailover,
])

const outboundPort443Types = new Set<string>([
  OutTypes.VLESS,
  OutTypes.VMess,
  OutTypes.Trojan,
  OutTypes.Naive,
  OutTypes.Hysteria,
  OutTypes.Hysteria2,
  OutTypes.TUIC,
  OutTypes.AnyTls,
  OutTypes.TrustTunnel,
])

const outboundTlsRequiredTypes = new Set<string>([
  OutTypes.Naive,
  OutTypes.Hysteria,
  OutTypes.Hysteria2,
  OutTypes.ShadowTLS,
  OutTypes.TUIC,
  OutTypes.AnyTls,
  OutTypes.TrustTunnel,
])

export function outboundFieldHintsForType(_type: string): Record<string, string> {
  return commonOutboundFieldHintKeys
}

export function hasOutboundRecommendedPreset(type: string): boolean {
  return !outboundTypesWithoutPreset.has(type)
}

function ensureObject(target: Record<string, any>, key: string): Record<string, any> {
  if (!target[key] || typeof target[key] !== 'object' || Array.isArray(target[key])) target[key] = {}
  return target[key]
}

function hasOwn(target: object, key: string): boolean {
  return Object.prototype.hasOwnProperty.call(target, key)
}

function applyRecommendedOutboundTls(target: Record<string, any>): void {
  if (!hasOwn(target, 'tls')) return
  const tls = ensureObject(target, 'tls')
  if (outboundTlsRequiredTypes.has(target.type)) tls.enabled = true
  if (tls.enabled === true) {
    tls.min_version = RECOMMENDED.tlsMinVersion
    tls.utls = { enabled: true, fingerprint: 'chrome' }
  }
}

export function applyOutboundRecommendedValues(outbound: Record<string, any>): void {
  const target = outbound as Record<string, any>
  if (!hasOutboundRecommendedPreset(target.type)) return

  if (outboundPort443Types.has(target.type)) target.server_port = target.server_port || 443
  applyRecommendedOutboundTls(target)

  switch (target.type) {
    case OutTypes.VMess:
      target.security = RECOMMENDED.vmessSecurity
      target.packet_encoding = RECOMMENDED.vmessPacketEncoding
      target.global_padding = true
      target.authenticated_length = true
      break
    case OutTypes.VLESS:
      target.packet_encoding = RECOMMENDED.vlessPacketEncoding
      break
    case OutTypes.Naive:
      if (target.quic) target.quic_congestion_control = RECOMMENDED.naiveCongestion
      break
    case OutTypes.Hysteria:
    case OutTypes.Hysteria2:
      target.up_mbps = target.up_mbps || 100
      target.down_mbps = target.down_mbps || 100
      break
    case OutTypes.TUIC:
      target.congestion_control = RECOMMENDED.tuicCongestion
      target.udp_relay_mode = RECOMMENDED.tuicUdpRelayMode
      break
    case OutTypes.AnyTls:
      target.idle_session_check_interval = target.idle_session_check_interval || '30s'
      target.idle_session_timeout = target.idle_session_timeout || '30s'
      target.min_idle_session = target.min_idle_session ?? 0
      break
    case OutTypes.Mieru:
      target.transport = 'TCP'
      target.multiplexing = target.multiplexing || 'MULTIPLEXING_LOW'
      break
    case OutTypes.Sudoku:
      target.aead_method = RECOMMENDED.sudokuAead
      target.padding_min = RECOMMENDED.sudokuPaddingMin
      target.padding_max = RECOMMENDED.sudokuPaddingMax
      target.enable_pure_downlink = true
      break
    case OutTypes.TrustTunnel:
      target.network = ['tcp', 'udp']
      target.congestion_controller = RECOMMENDED.trustTunnelCongestion
      break
    case OutTypes.OpenVPN:
      target.proto = target.proto || 'udp'
      target.cipher = RECOMMENDED.openvpnCipher
      target.auth = RECOMMENDED.openvpnAuth
      break
  }
}

export const outboundRecommendationSpecs: RecommendationSpec<Record<string, unknown>>[] = [
  { id: 'outbound-server-port-443', label: 'Use HTTPS port', description: 'Common TLS/QUIC protocols usually listen on port 443.', path: 'server_port', value: 443, when: (ctx) => typeAvailable(ctx) && [OutTypes.VLESS, OutTypes.VMess, OutTypes.Trojan, OutTypes.Hysteria2, OutTypes.TUIC, OutTypes.AnyTls, OutTypes.Naive].includes(ctx.type as any) },
  { id: 'outbound-tls-min-13', label: 'Require TLS 1.3', description: 'Prefer TLS 1.3 where outbound TLS settings are available.', path: 'tls.min_version', value: RECOMMENDED.tlsMinVersion, when: (ctx) => typeAvailable(ctx) && ctx.model != null && hasOwn(ctx.model as object, 'tls') },
  { id: 'outbound-utls-chrome', label: 'Use Chrome uTLS fingerprint', description: 'Recommended client fingerprint for outbound TLS.', path: 'tls.utls', value: { enabled: true, fingerprint: 'chrome' }, when: (ctx) => typeAvailable(ctx) && ctx.model != null && hasOwn(ctx.model as object, 'tls') },
]

export const commonServiceFieldHintKeys: Record<string, string> = {
  type: 'types.service.hint.type',
  tag: 'types.service.hint.tag',
  listen: 'types.service.hint.listen',
  listen_port: 'types.service.hint.listen_port',
  tls_id: 'types.service.hint.tls_id',
  config_path: 'types.service.hint.config_path',
  verify_client_endpoint: 'types.service.hint.verify_client_endpoint',
  verify_client_url: 'types.service.hint.verify_client_url',
  home: 'types.service.hint.home',
  mesh_with: 'types.service.hint.mesh_with',
  mesh_psk: 'types.service.hint.mesh_psk',
  stun: 'types.service.hint.stun',
  ssm_servers: 'types.service.hint.ssm_servers',
  cache_path: 'types.service.hint.cache_path',
  credential_path: 'types.service.hint.credential_path',
  usages_path: 'types.service.hint.usages_path',
  detour: 'types.service.hint.detour',
  users: 'types.service.hint.users',
  headers: 'types.service.hint.headers',
  memory_limit: 'types.service.hint.memory_limit',
  safety_margin: 'types.service.hint.safety_margin',
  min_interval: 'types.service.hint.min_interval',
  max_interval: 'types.service.hint.max_interval',
  checks_before_limit: 'types.service.hint.checks_before_limit',
  profiler_listen: 'types.service.hint.profiler_listen',
  read_timeout: 'types.service.hint.read_timeout',
  write_timeout: 'types.service.hint.write_timeout',
}

const serviceTypesWithoutPreset = new Set<string>([
  SrvTypes.OOMKiller,
  SrvTypes.Profiler,
])

export function serviceFieldHintsForType(_type: string): Record<string, string> {
  return commonServiceFieldHintKeys
}

export function hasServiceRecommendedPreset(type: string): boolean {
  return !serviceTypesWithoutPreset.has(type)
}

export function applyServiceRecommendedValues(service: Record<string, any>): void {
  const target = service as Record<string, any>
  if (!hasServiceRecommendedPreset(target.type)) return
  target.listen = target.listen || '::'
}

export const serviceRecommendationSpecs: RecommendationSpec<Record<string, unknown>>[] = [
  { id: 'service-listen-any', label: 'Listen on all interfaces', description: 'Use :: for dual-stack listen when this service has Listen settings.', path: 'listen', value: '::', when: (ctx) => ![SrvTypes.OOMKiller, SrvTypes.Profiler].includes(ctx.type as any) && (ctx.model as any)?.listen_port != null },
]

export const commonEndpointFieldHintKeys: Record<string, string> = {
  type: 'types.endpoint.hint.type',
  tag: 'types.endpoint.hint.tag',
  address: 'types.endpoint.hint.address',
  listen_port: 'types.endpoint.hint.listen_port',
  mtu: 'types.endpoint.hint.mtu',
  private_key: 'types.endpoint.hint.private_key',
  public_key: 'types.endpoint.hint.public_key',
  peers: 'types.endpoint.hint.peers',
  system: 'types.endpoint.hint.system',
  domain_strategy: 'types.endpoint.hint.domain_strategy',
  connect_timeout: 'types.endpoint.hint.connect_timeout',
  vpn_user: 'types.endpoint.hint.vpn_user',
  vpn_inbounds: 'types.endpoint.hint.vpn_inbounds',
  vpn_outbound: 'types.endpoint.hint.vpn_outbound',
  dial_options: 'types.endpoint.hint.dial_options',
  dial_tcp_fast_open: 'types.endpoint.hint.dial_tcp_fast_open',
  dial_tcp_multi_path: 'types.endpoint.hint.dial_tcp_multi_path',
  dial_udp_fragment: 'types.endpoint.hint.dial_udp_fragment',
  dial_connect_timeout: 'types.endpoint.hint.dial_connect_timeout',
  dial_disable_tcp_keep_alive: 'types.endpoint.hint.dial_disable_tcp_keep_alive',
  dial_tcp_keep_alive: 'types.endpoint.hint.dial_tcp_keep_alive',
  dial_tcp_keep_alive_interval: 'types.endpoint.hint.dial_tcp_keep_alive_interval',
}

const endpointTypesWithoutPreset = new Set<string>([
  EpTypes.Warp,
  EpTypes.Tailscale,
  EpTypes.VpnClient,
])

export function endpointFieldHintsForType(_type: string): Record<string, string> {
  return commonEndpointFieldHintKeys
}

export function hasEndpointRecommendedPreset(type: string): boolean {
  return !endpointTypesWithoutPreset.has(type)
}

export function applyEndpointRecommendedValues(endpoint: Record<string, any>): void {
  const target = endpoint as Record<string, any>
  if (!hasEndpointRecommendedPreset(target.type)) return

  switch (target.type) {
    case EpTypes.Wireguard:
      target.mtu = target.mtu || 1420
      break
    case EpTypes.VpnServer:
      target.address = target.address || '10.0.0.1'
      if (!Array.isArray(target.users) || target.users.length === 0) {
        target.users = [{ address: '10.0.0.2', key: '' }]
      } else if (!target.users[0].address) {
        target.users[0].address = '10.0.0.2'
      }
      break
  }
}

export const endpointRecommendationSpecs: RecommendationSpec<Record<string, unknown>>[] = [
  { id: 'endpoint-wireguard-mtu', label: 'WireGuard MTU 1420', description: 'A safe MTU for common WireGuard deployments.', path: 'mtu', value: 1420, when: (ctx) => ctx.type === EpTypes.Wireguard },
  { id: 'endpoint-vpn-server-route', label: 'VPN server first client address', description: 'Use the default first client address for a new VPN server user.', path: 'users.0.address', value: '10.0.0.2', when: (ctx) => ctx.type === EpTypes.VpnServer },
]

export const tlsFieldHintKeys: Record<string, string> = {
  name: 'types.tls.hint.name',
  tls_type: 'types.tls.hint.tls_type',
  server_name: 'types.tls.hint.server_name',
  min_version: 'types.tls.hint.min_version',
  max_version: 'types.tls.hint.max_version',
  alpn: 'types.tls.hint.alpn',
  cipher_suites: 'types.tls.hint.cipher_suites',
  certificate_path: 'types.tls.hint.certificate_path',
  key_path: 'types.tls.hint.key_path',
  certificate: 'types.tls.hint.certificate',
  key: 'types.tls.hint.key',
  disable_sni: 'types.tls.hint.disable_sni',
  insecure: 'types.tls.hint.insecure',
  client_authentication: 'types.tls.hint.client_authentication',
  store: 'types.tls.hint.store',
  ktls: 'types.tls.hint.ktls',
  utls: 'types.tls.hint.utls',
  ech: 'types.tls.hint.ech',
  reality_handshake_server: 'types.tls.hint.reality_handshake_server',
  reality_handshake_port: 'types.tls.hint.reality_handshake_port',
  reality_private_key: 'types.tls.hint.reality_private_key',
  reality_public_key: 'types.tls.hint.reality_public_key',
  reality_short_id: 'types.tls.hint.reality_short_id',
  reality_max_time_difference: 'types.tls.hint.reality_max_time_difference',
}

export function tlsFieldHintsForType(_type: string): Record<string, string> {
  return tlsFieldHintKeys
}

export function hasTlsRecommendedPreset(_type: string): boolean {
  return true
}

export function applyTlsRecommendedValues(tls: Record<string, any>): void {
  const target = tls as Record<string, any>
  const isReality = target.server?.reality != null

  if (isReality) {
    const server = ensureObject(target, 'server')
    const reality = ensureObject(server, 'reality')
    const handshake = ensureObject(reality, 'handshake')
    handshake.server = handshake.server || sniFrontHosts[0]
    handshake.server_port = handshake.server_port || 443
    return
  }

  const server = ensureObject(target, 'server')
  server.min_version = RECOMMENDED.tlsMinVersion
  server.max_version = RECOMMENDED.tlsMaxVersion
  server.alpn = tlsAlpn.map((item) => item.value)
  const client = ensureObject(target, 'client')
  client.utls = { enabled: true, fingerprint: 'chrome' }
}

export const tlsRecommendationSpecs: RecommendationSpec<Record<string, unknown>>[] = [
  { id: 'tls-min-13', label: 'Minimum TLS 1.3', description: 'Use TLS 1.3 as the minimum version when enabled.', path: 'server.min_version', value: RECOMMENDED.tlsMinVersion, when: (ctx) => !(ctx.model as any)?.server?.reality },
  { id: 'tls-max-13', label: 'Maximum TLS 1.3', description: 'Pin maximum TLS version to TLS 1.3 when enabled.', path: 'server.max_version', value: RECOMMENDED.tlsMaxVersion, when: (ctx) => !(ctx.model as any)?.server?.reality },
  { id: 'tls-alpn-modern', label: 'Modern ALPN list', description: 'Advertise HTTP/3, HTTP/2 and HTTP/1.1.', path: 'server.alpn', value: tlsAlpn.map((item) => item.value), when: (ctx) => !(ctx.model as any)?.server?.reality },
  { id: 'tls-utls-chrome', label: 'Chrome client fingerprint', description: 'Use Chrome as the outbound uTLS fingerprint.', path: 'client.utls', value: { enabled: true, fingerprint: 'chrome' }, when: (ctx) => !(ctx.model as any)?.server?.reality },
  { id: 'reality-handshake-server', label: 'Reality handshake server', description: 'Use a well-known TLS 1.3 host as the Reality handshake destination.', path: 'server.reality.handshake.server', value: sniFrontHosts[0], when: (ctx) => Boolean((ctx.model as any)?.server?.reality) },
]

export const dnsServerFieldHintKeys: Record<string, string> = {
  type: 'types.dns.hint.type',
  tag: 'types.dns.hint.tag',
  server: 'types.dns.hint.server',
  server_port: 'types.dns.hint.server_port',
  path: 'types.dns.hint.path',
  headers: 'types.dns.hint.headers',
  tls_enable: 'types.dns.hint.tls_enable',
  tls_min_version: 'types.dns.hint.tls_min_version',
  tls_utls: 'types.dns.hint.tls_utls',
  dial_options: 'types.dns.hint.dial_options',
  dial_tcp_fast_open: 'types.dns.hint.dial_tcp_fast_open',
  dial_tcp_multi_path: 'types.dns.hint.dial_tcp_multi_path',
  dial_udp_fragment: 'types.dns.hint.dial_udp_fragment',
  dial_connect_timeout: 'types.dns.hint.dial_connect_timeout',
  dial_disable_tcp_keep_alive: 'types.dns.hint.dial_disable_tcp_keep_alive',
  dial_tcp_keep_alive: 'types.dns.hint.dial_tcp_keep_alive',
  dial_tcp_keep_alive_interval: 'types.dns.hint.dial_tcp_keep_alive_interval',
  hosts_path: 'types.dns.hint.hosts_path',
  predefined: 'types.dns.hint.predefined',
  prefer_go: 'types.dns.hint.prefer_go',
  dhcp_interface: 'types.dns.hint.dhcp_interface',
  fakeip_range: 'types.dns.hint.fakeip_range',
  sdns_stamp: 'types.dns.hint.sdns_stamp',
  fallback_servers: 'types.dns.hint.fallback_servers',
  fallback_strategy: 'types.dns.hint.fallback_strategy',
  endpoint: 'types.dns.hint.endpoint',
  service: 'types.dns.hint.service',
  accept_default_resolvers: 'types.dns.hint.accept_default_resolvers',
}

const dnsServerTypesWithPreset = new Set<string>([
  DnsTypes.TCP,
  DnsTypes.UDP,
  DnsTypes.TLS,
  DnsTypes.QUIC,
  DnsTypes.HTTPS,
  DnsTypes.HTTP3,
])

const dnsTlsTypes = new Set<string>([
  DnsTypes.TLS,
  DnsTypes.QUIC,
  DnsTypes.HTTPS,
  DnsTypes.HTTP3,
])

export function dnsServerFieldHintsForType(_type: string): Record<string, string> {
  return dnsServerFieldHintKeys
}

export function hasDnsServerRecommendedPreset(type: string): boolean {
  return dnsServerTypesWithPreset.has(type)
}

export function applyDnsServerRecommendedValues(server: Record<string, any>): void {
  const target = server as Record<string, any>
  if (!hasDnsServerRecommendedPreset(target.type)) return

  if ([DnsTypes.TCP, DnsTypes.UDP, DnsTypes.TLS, DnsTypes.QUIC].includes(target.type)) {
    target.server = target.server || dnsResolvers[0]
  }
  if ([DnsTypes.HTTPS, DnsTypes.HTTP3].includes(target.type)) {
    target.server = target.server || 'cloudflare-dns.com'
    target.path = target.path || dohPaths[0]
  }
  if ([DnsTypes.TCP, DnsTypes.UDP].includes(target.type)) target.server_port = target.server_port || 53
  if ([DnsTypes.TLS, DnsTypes.QUIC].includes(target.type)) target.server_port = target.server_port || 853
  if ([DnsTypes.HTTPS, DnsTypes.HTTP3].includes(target.type)) target.server_port = target.server_port || 443

  if (dnsTlsTypes.has(target.type) && hasOwn(target, 'tls')) {
    const tls = ensureObject(target, 'tls')
    tls.enabled = true
    tls.min_version = RECOMMENDED.tlsMinVersion
    tls.utls = { enabled: true, fingerprint: 'chrome' }
  }
}

export const dnsRuleFieldHintKeys: Record<string, string> = {
  logical: 'types.dnsRule.hint.logical',
  action: 'types.dnsRule.hint.action',
  mode: 'types.dnsRule.hint.mode',
  invert: 'types.dnsRule.hint.invert',
  server: 'types.dnsRule.hint.server',
  strategy: 'types.dnsRule.hint.strategy',
  disable_cache: 'types.dnsRule.hint.disable_cache',
  rewrite_ttl: 'types.dnsRule.hint.rewrite_ttl',
  client_subnet: 'types.dnsRule.hint.client_subnet',
  method: 'types.dnsRule.hint.method',
  no_drop: 'types.dnsRule.hint.no_drop',
  rcode: 'types.dnsRule.hint.rcode',
  answer: 'types.dnsRule.hint.answer',
  match_fields: 'types.dnsRule.hint.match_fields',
}

export function dnsRuleFieldHints(): Record<string, string> {
  return dnsRuleFieldHintKeys
}

export function hasDnsRuleRecommendedPreset(): boolean {
  return true
}

export function applyDnsRuleRecommendedValues(rule: Record<string, any>): void {
  const target = rule as Record<string, any>
  if (target.type === 'logical') target.mode = target.mode || 'or'
  if (target.action === 'route') target.strategy = target.strategy || 'prefer_ipv4'
}

export const routeRuleFieldHintKeys: Record<string, string> = {
  logical: 'types.rule.hint.logical',
  action: 'types.rule.hint.action',
  mode: 'types.rule.hint.mode',
  invert: 'types.rule.hint.invert',
  outbound: 'types.rule.hint.outbound',
  override_address: 'types.rule.hint.override_address',
  override_port: 'types.rule.hint.override_port',
  network_strategy: 'types.rule.hint.network_strategy',
  fallback_delay: 'types.rule.hint.fallback_delay',
  udp_disable_domain_unmapping: 'types.rule.hint.udp_disable_domain_unmapping',
  udp_connect: 'types.rule.hint.udp_connect',
  udp_timeout: 'types.rule.hint.udp_timeout',
  tls_fragment: 'types.rule.hint.tls_fragment',
  tls_record_fragment: 'types.rule.hint.tls_record_fragment',
  method: 'types.rule.hint.method',
  no_drop: 'types.rule.hint.no_drop',
  sniffer: 'types.rule.hint.sniffer',
  timeout: 'types.rule.hint.timeout',
  strategy: 'types.rule.hint.strategy',
  server: 'types.rule.hint.server',
  match_fields: 'types.rule.hint.match_fields',
}

export function routeRuleFieldHints(): Record<string, string> {
  return routeRuleFieldHintKeys
}

export function hasRouteRuleRecommendedPreset(): boolean {
  return true
}

export function applyRouteRuleRecommendedValues(rule: Record<string, any>): void {
  const target = rule as Record<string, any>
  if (target.type === 'logical') target.mode = target.mode || 'or'
  if (['route', 'route-options', 'bypass'].includes(target.action)) target.udp_timeout = target.udp_timeout || '5m'
}

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
