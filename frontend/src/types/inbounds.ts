import RandomUtil from "@/plugins/randomUtil"
import { iMultiplex } from "./multiplex"
import { iTls } from "./tls"
import { Dial } from "./dial"
import { Transport } from "./transport"

export const InTypes = {
  Direct: 'direct',
  Mixed: 'mixed',
  SOCKS: 'socks',
  HTTP: 'http',
  Shadowsocks: 'shadowsocks',
  VMess: 'vmess',
  Trojan: 'trojan',
  Naive: 'naive',
  Hysteria: 'hysteria',
  ShadowTLS: 'shadowtls',
  TUIC: 'tuic',
  Hysteria2: 'hysteria2',
  VLESS: 'vless',
  AnyTls: 'anytls',
  Mieru: 'mieru',
  Sudoku: 'sudoku',
  TrustTunnel: 'trusttunnel',
  SSH: 'ssh',
  MTProxy: 'mtproxy',
  Call: 'call',
  Tun: 'tun',
  Redirect: 'redirect',
  TProxy: 'tproxy',
  Bond: 'bond',
  CoreFailover: 'core-failover',
}

type InType = typeof InTypes[keyof typeof InTypes]

export interface Addr {
  server: string
  server_port: number
  tls?: boolean
  insecure?: boolean
  server_name?: string
  remark?: string
}

export interface Listen {
  listen: string
  listen_port: number
  bind_interface?: string
  routing_mark?: number | string
  reuse_addr?: boolean
  netns?: string
  tcp_fast_open?: boolean
  tcp_multi_path?: boolean
  udp_fragment?: boolean
  udp_timeout?: string
  detour?: string
  disable_tcp_keep_alive?: boolean
  tcp_keep_alive?: string
  tcp_keep_alive_interval?: string
  proxy_protocol?: boolean
  proxy_protocol_accept_no_header?: boolean
  domain_strategy?: string
  domain_resolver?: string
  udp_disable_domain_unmapping?: boolean
}

interface InboundBasics extends Listen {
  id: number
  type: InType
  tag: string
  tls_id: number
  addrs?: Addr[]
  out_json?: any
}

interface ShadowTLSHandShake extends Dial {
  server: string
  server_port: number
}

export interface Direct extends InboundBasics {
  network?: "udp" | "tcp"
  override_address?: string
  override_port?: number
}
export interface Mixed extends InboundBasics {
  set_system_proxy?: boolean
  tls?: iTls
}
export interface SOCKS extends InboundBasics {}
export interface HTTP extends InboundBasics {
  set_system_proxy?: boolean
}
export interface Shadowsocks extends InboundBasics {
  method: string
  password: string
  network?: "udp" | "tcp"
  multiplex?: iMultiplex
  managed?: boolean
  destinations?: { server: string, server_port: number, method: string, password: string }[]
}
export interface VMess extends InboundBasics {
  tls: iTls
  multiplex?: iMultiplex
  transport?: Transport
}
export interface Trojan extends InboundBasics {
  tls: iTls
  fallback?: {
    server: string
    server_port: number
  }
  fallback_for_alpn?: {
    [alpn: string]: {
      server: string
      server_port: number
    }
  }
  multiplex?: iMultiplex
  transport?: Transport
}
export interface Naive extends InboundBasics {
  tls: iTls,
  quic_congestion_control?: "" | "bbr" | "bbr2" | "cubic" | "reno"
  network?: "udp" | "tcp"
}
export interface Hysteria extends InboundBasics {
  up_mbps: number
  down_mbps: number
  obfs?: string
  recv_window_conn?: number
  recv_window_client?: number
  max_conn_client?: number
  disable_mtu_discovery?: boolean
  up?: string
  down?: string
  tls?: iTls
}
export interface ShadowTLS extends InboundBasics {
  version: 1|2|3
  password?: string
  handshake: ShadowTLSHandShake
  handshake_for_server_name?: {
    [server_name: string]: ShadowTLSHandShake
  }
  strict_mode?: boolean
  wildcard_sni?: string
}
export interface VLESS extends InboundBasics {
  decryption?: string
  multiplex?: iMultiplex
  transport?: Transport
  tls: iTls
}

export interface AnyTls extends InboundBasics {
  padding_scheme: string[]
  tls: iTls
}
export interface TUIC extends InboundBasics {
  congestion_control: ""|"cubic"|"new_reno"|"bbr"
  auth_timeout?: string
  zero_rtt_handshake?: boolean
  heartbeat?: string
  tls?: iTls
}
export interface Hysteria2 extends InboundBasics {
  up_mbps?: number
  down_mbps?: number
  obfs?: {
    type?: "salamander"
    password: string
  }
  tls?: iTls
  ignore_client_bandwidth?: boolean
  masquerade?: string | {
    type: string
    directory?: string
    url?: string
    rewrite_host?: boolean
    status_code?: number
    headers?: Headers[]
    content?: string
  }
  brutal_debug?: boolean
}
export interface Tun extends InboundBasics {
  interface_name?: string
  address?: string[]
  mtu?: number
  endpoint_independent_nat?: boolean
  udp_timeout?: string
  stack?: string
  auto_route?: boolean
  strict_route?: boolean
  auto_redirect?: boolean
  exclude_mptcp?: boolean
  gso?: boolean
  inet4_address?: string[]
  inet4_route_address?: string[]
  inet4_route_exclude_address?: string[]
  inet6_address?: string[]
  inet6_route_address?: string[]
  inet6_route_exclude_address?: string[]
  auto_redirect_reset_mark?: string | number
  auto_redirect_nfqueue?: number
  auto_redirect_iproute2_fallback_rule_index?: number
  auto_redirect_input_mark?: string | number
  auto_redirect_output_mark?: string | number
  iproute2_table_index?: number
  iproute2_rule_index?: number
  loopback_address?: string[]
  route_address?: string[]
  route_exclude_address?: string[]
  route_address_set?: string[]
  route_exclude_address_set?: string[]
  include_interface?: string[]
  exclude_interface?: string[]
  include_uid?: number[]
  include_uid_range?: string[]
  exclude_uid?: number[]
  exclude_uid_range?: string[]
  include_android_user?: number[]
  include_package?: string[]
  exclude_package?: string[]
  platform?: {
    http_proxy?: {
      enabled?: boolean
      server?: string
      server_port?: number
      bypass_domain?: string[]
      match_domain?: string[]
    }
  }
}
export interface Redirect extends InboundBasics {}
export interface TProxy extends InboundBasics {
  network?: "udp" | "tcp"
}
export interface BondInbound extends InboundBasics {
  inbounds: string[]
}
export interface CoreFailoverInbound extends InboundBasics {
  inbounds: string[]
}
export interface Mieru extends InboundBasics {
  listen_ports?: string[]
  transport?: string
  traffic_pattern?: string
  user_hint_is_mandatory?: boolean
}
export interface Sudoku extends InboundBasics {
  key: string
  aead_method?: string
  table_type?: string
  padding_min?: number
  padding_max?: number
  handshake_timeout?: number
  enable_pure_downlink?: boolean
  custom_table?: string
  custom_tables?: string[]
  disable_http_mask?: boolean
  http_mask_mode?: string
  path_root?: string
  fallback?: string
}
export interface CallCookie {
  name: string
  value?: string
}

// Call inbound embeds DialerOptions in the kernel. The panel inbound form
// shows only the dialer fields that are absent from the shared Listen fields;
// domain_resolver stays the Listen form (string).
export interface Call extends InboundBasics {
  platform?: string
  mode?: string
  read_buffer?: number
  max_buffered_amount?: number
  memory_limit?: number
  cookies?: CallCookie[]
  join_link?: string
  email?: string
  password?: string
  inet4_bind_address?: string
  inet6_bind_address?: string
  bind_address_no_port?: boolean
  protect_path?: string
  connect_timeout?: string
  network_strategy?: 'default' | 'fallback' | 'hybrid'
  network_type?: string[]
  fallback_network_type?: string[]
  fallback_delay?: string
}

export interface TrustTunnel extends InboundBasics {
  tls: iTls
  network?: string[]
  congestion_controller?: string
  cwnd?: number
}
export interface SSH extends InboundBasics {
  host_key?: string[]
  host_key_path?: string[]
  server_version?: string
  max_auth_tries?: number
  fallback?: string
}
export interface MTProxy extends InboundBasics {
  concurrency?: number
  domain_fronting_port?: number
  domain_fronting_host?: string
  domain_fronting_proxy_protocol?: boolean
  prefer_ip?: string
  auto_update?: boolean
  allow_fallback_on_unknown_dc?: boolean
  tolerate_time_skewness?: string
  idle_timeout?: string
  handshake_timeout?: string
  doppelganger_urls?: string[]
  doppelganger_per_raid?: number
  doppelganger_each?: string
  doppelganger_drs?: boolean
  throttle_max_connections?: number
  throttle_check_interval?: string
}

// Create interfaces dynamically based on InTypes keys
type InterfaceMap = {
  direct: Direct
  mixed: Mixed
  socks: SOCKS
  http: HTTP
  shadowsocks: Shadowsocks
  vmess: VMess
  trojan: Trojan
  naive: Naive
  hysteria: Hysteria
  shadowtls: ShadowTLS
  tuic: TUIC
  hysteria2: Hysteria2
  vless: VLESS
  anytls: AnyTls
  mieru: Mieru
  sudoku: Sudoku
  trusttunnel: TrustTunnel
  ssh: SSH
  mtproxy: MTProxy
  call: Call
  tun: Tun
  redirect: Redirect
  tproxy: TProxy
}

// Create union type from InterfaceMap
export type Inbound = InterfaceMap[keyof InterfaceMap]

// Create defaultValues object dynamically
const defaultValues: Record<InType, Inbound> = {
  direct: <Direct>{ type: InTypes.Direct },
  mixed: <Mixed>{ type: InTypes.Mixed },
  socks: <SOCKS>{ type: InTypes.SOCKS },
  http: <HTTP>{ type: InTypes.HTTP, tls_id: 0 },
  shadowsocks: <Shadowsocks>{ type: InTypes.Shadowsocks, method: 'none' },
  vmess: <VMess>{ type: InTypes.VMess, tls_id: 0, transport: {} },
  trojan: <Trojan>{ type: InTypes.Trojan, tls_id: 0, transport: {} },
  naive: <Naive>{ type: InTypes.Naive, tls_id: 0 },
  hysteria: <Hysteria>{ type: InTypes.Hysteria, up_mbps: 100, down_mbps: 100, tls_id: 0 },
  shadowtls: <ShadowTLS>{ type: InTypes.ShadowTLS, version: 3, handshake: {}, handshake_for_server_name: {} },
  tuic: <TUIC>{ type: InTypes.TUIC, congestion_control: "cubic", tls_id: 0 },
  hysteria2: <Hysteria2>{ type: InTypes.Hysteria2, tls_id: 0 },
  vless: <VLESS>{ type: InTypes.VLESS, tls_id: 0, transport: {} },
  anytls: <AnyTls>{ type: InTypes.AnyTls, tls_id: 0, padding_scheme: [
    "stop=8",
    "0=30-30",
    "1=100-400",
    "2=400-500,c,500-1000,c,500-1000,c,500-1000,c,500-1000",
    "3=9-9,500-1000",
    "4=500-1000",
    "5=500-1000",
    "6=500-1000",
    "7=500-1000"
  ]},
  mieru: <Mieru>{ type: InTypes.Mieru, transport: 'TCP' },
  sudoku: <Sudoku>{ type: InTypes.Sudoku, key: '', aead_method: 'chacha20-poly1305', padding_min: 10, padding_max: 30, handshake_timeout: 5, enable_pure_downlink: true },
  trusttunnel: <TrustTunnel>{ type: InTypes.TrustTunnel, tls_id: 0, network: ['tcp', 'udp'], congestion_controller: 'bbr' },
  ssh: <SSH>{ type: InTypes.SSH },
  mtproxy: <MTProxy>{ type: InTypes.MTProxy, prefer_ip: 'prefer-ipv4' },
  call: <Call>{ type: InTypes.Call, platform: 'dion', read_buffer: 32768 },
  tun: <Tun>{ type: InTypes.Tun, mtu: 9000, stack: 'system', udp_timeout: '5m', auto_route: false },
  redirect: <Redirect>{ type: InTypes.Redirect },
  tproxy: <TProxy>{ type: InTypes.TProxy },
  bond: { type: InTypes.Bond, inbounds: [] } as unknown as BondInbound,
  'core-failover': { type: InTypes.CoreFailover, inbounds: [] } as unknown as CoreFailoverInbound,
}

function hasOwn(value: object | undefined, key: string): boolean {
  return value != null && Object.hasOwn(value, key)
}

function randomShadowsocksPassword(method: string): string {
  if (method.startsWith('2022')) {
    return method == '2022-blake3-aes-128-gcm'
      ? RandomUtil.randomShadowsocksPassword(16)
      : RandomUtil.randomShadowsocksPassword(32)
  }
  return RandomUtil.randomSeq(10)
}

function applyCreateSecrets<T extends Inbound>(type: InType, inbound: Inbound, json?: Partial<T>): void {
  const target = inbound as any
  const source = json as Record<string, unknown> | undefined
  switch (type) {
    case InTypes.Shadowsocks:
      if (!hasOwn(source, 'method')) target.method = '2022-blake3-aes-256-gcm'
      if (!hasOwn(source, 'password') && typeof target.method === 'string' && target.method.length > 0 && target.method !== 'none') {
        target.password = randomShadowsocksPassword(target.method)
      }
      break
    case InTypes.ShadowTLS:
      if (target.version === 2 && !hasOwn(source, 'password')) target.password = RandomUtil.randomSeq(16)
      break
    case InTypes.Sudoku:
      // The backend generates a canonical master scalar when the field is
      // omitted. Explicit values remain supported for legacy PSK/public keys.
      if (!hasOwn(source, 'key')) delete target.key
      break
  }
}

export function createInbound<T extends Inbound>(type: InType,json?: Partial<T>): Inbound {
  const defaultObject: Inbound = { ...defaultValues[type] ?? {}, ...(json ?? {}) }
  applyCreateSecrets(type, defaultObject, json)
  return defaultObject
}
