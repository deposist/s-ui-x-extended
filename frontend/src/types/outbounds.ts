import { oTls } from "./tls"
import { oMultiplex } from "./multiplex"
import { Transport } from "./transport"
import { Dial } from "./dial"

export const OutTypes = {
  Direct: 'direct',
  SOCKS: 'socks',
  HTTP: 'http',
  Shadowsocks: 'shadowsocks',
  VMess: 'vmess',
  Trojan: 'trojan',
  Naive: 'naive',
  Hysteria: 'hysteria',
  VLESS: 'vless',
  ShadowTLS: 'shadowtls',
  TUIC: 'tuic',
  Hysteria2: 'hysteria2',
  AnyTls: 'anytls',
  Tor: 'tor',
  SSH: 'ssh',
  Mieru: 'mieru',
  Sudoku: 'sudoku',
  TrustTunnel: 'trusttunnel',
  MASQUE: 'masque',
  OpenVPN: 'openvpn',
  Parser: 'parser',
  Selector: 'selector',
  URLTest: 'urltest',
  Bond: 'bond',
  Failover: 'failover',
  Fallback: 'fallback',
  BandwidthLimiter: 'bandwidth-limiter',
  ConnectionLimiter: 'connection-limiter',
  TrafficLimiter: 'traffic-limiter',
  RateLimiter: 'rate-limiter',
}

type OutType = typeof OutTypes[keyof typeof OutTypes]

interface OutboundBasics {
  id: number
  type: OutType
  tag: string
}

export interface WgPeer {
  server: string
  server_port: number
  public_key: string
  pre_shared_key?: string
  allowed_ips?: string[]
  reserved?: number[]
}

export interface Direct extends OutboundBasics, Dial {}

export interface SOCKS extends OutboundBasics, Dial {
  server: string
  server_port: number
  version?: "4" | "4a" | "5"
  username?: string
  password?: string
  network?: "udp" | "tcp"
  udp_over_tcp?: false | {
    enabled: true
    version?: number
  }
}

export interface HTTP extends OutboundBasics, Dial {
  server: string
  server_port: number
  username?: string
  password?: string
  path?: string
  headers?: {
    [key: string]: string
  }
  tls?: oTls
}

export interface Shadowsocks extends OutboundBasics, Dial {
  server: string
  server_port: number
  method: string
  password: string
  plugin?: string
  plugin_opts?: string
  network?: "udp" | "tcp"
  udp_over_tcp?: false | {
    enabled: true
    version?: number
  }
  multiplex?: oMultiplex
}

export interface VMESS extends OutboundBasics, Dial {
  server: string
  server_port: number
  uuid: string
  security?: string
  alter_id: 0
  global_padding?: boolean
  authenticated_length?: boolean
  network?: "udp" | "tcp"
  packet_encoding?: string
  tls?: oTls
  multiplex?: oMultiplex
  transport?: Transport
}

export interface Trojan extends OutboundBasics, Dial {
  server: string
  server_port: number
  password: string
  network?: "udp" | "tcp"
  tls?: oTls
  multiplex?: oMultiplex
  transport?: Transport
}

export interface Naive extends OutboundBasics, Dial {
  server: string
  server_port: number
  username?: string
  password?: string
  insecure_concurrency?: number
  extra_headers?: { [key: string]: string }
  udp_over_tcp?: false | { enabled?: boolean; version?: number }
  stream_receive_window?: string | number
  quic?: boolean
  quic_congestion_control?: "" | "bbr" | "bbr2" | "cubic" | "reno"
  quic_session_receive_window?: string | number
  tls: oTls
}

export interface Hysteria extends OutboundBasics, Dial {
  server: string
  server_port: number
  server_ports?: string[]
  hop_interval?: string
  up?: string
  down?: string
  up_mbps: number
  down_mbps: number
  obfs?: string
  auth_str?: string
  recv_window_conn?: number
  recv_window?: number
  disable_mtu_discovery?: boolean
  network?: "udp" | "tcp"
  tls: oTls
}

export interface ShadowTLS extends OutboundBasics, Dial {
  server: string
  server_port: number
  version: 1|2|3
  password?: string
  tls: oTls
}

export interface VLESS extends OutboundBasics, Dial {
  server: string
  server_port: number
  uuid: string
  flow?: string
  encryption?: string
  network?: "udp" | "tcp"
  packet_encoding?: string
  tls?: oTls
  multiplex?: oMultiplex
  transport?: Transport
}

export interface TUIC extends OutboundBasics, Dial {
  server: string
  server_port: number
  uuid: string
  password?: string
  congestion_control?: "cubic"|"new_reno"|"bbr"
  udp_relay_mode?: "native" | "quic"
  udp_over_stream?: boolean
  zero_rtt_handshake?: boolean
  heartbeat?: string
  network?: "udp" | "tcp"
  tls: oTls
}

export interface Hysteria2 extends OutboundBasics, Dial {
  server: string
  server_port: number
  server_ports?: string[]
  hop_interval: string
  up_mbps?: number
  down_mbps?: number
  obfs?: {
    type?: "salamander"
    password: string
  }
  password?: string
  network?: "udp" | "tcp"
  tls: oTls
  brutal_debug?: boolean
}

export interface AnyTls extends OutboundBasics, Dial {
  server: string
  server_port: number
  password: string
  idle_session_check_interval: string
  idle_session_timeout: string
  min_idle_session: number
  tls: oTls
}

export interface Tor extends OutboundBasics, Dial {
  executable_path?: string
  extra_args?: string[]
  data_directory: string
  torrc?: {
    [options: string]: string
  }
}

export interface SSH extends OutboundBasics, Dial  {
  server: string
  server_port?: number
  user?: string
  password?: string
  private_key?: string
  private_key_path?: string
  private_key_passphrase?: string
  host_key?: string[]
  host_key_algorithms?: string[]
  client_version?: string
}

export interface Mieru extends OutboundBasics, Dial {
  server: string
  server_port?: number
  server_ports?: string[]
  transport?: string
  username?: string
  password?: string
  multiplexing?: string
  traffic_pattern?: string
}

export interface Sudoku extends OutboundBasics, Dial {
  server: string
  server_port: number
  key: string
  aead_method?: string
  table_type?: string
  padding_min?: number
  padding_max?: number
  enable_pure_downlink?: boolean
  custom_table?: string
  custom_tables?: string[]
  http_mask?: {
    enabled?: boolean
    mode?: string
    host?: string
    path_root?: string
    multiplex?: string
  }
}

export interface TrustTunnel extends OutboundBasics, Dial {
  server: string
  server_port: number
  username?: string
  password?: string
  network?: string[]
  health_check?: boolean
  quic?: boolean
  congestion_controller?: string
  bbr_profile?: string
  cwnd?: number
  multiplex?: {
    enabled?: boolean
    max_connections?: number
    min_streams?: number
    max_streams?: number
  }
  tls?: oTls
}

export interface MasqueTls {
  insecure?: boolean
  cipher_suites?: string[]
  curve_preferences?: string[]
  fragment?: boolean
  fragment_fallback_delay?: string
  record_fragment?: boolean
  kernel_tx?: boolean
  kernel_rx?: boolean
}

export interface CloudflareProfile {
  id?: string
  auth_token?: string
  private_key?: string
  recreate?: boolean
  detour?: string
}

export interface MASQUE extends OutboundBasics, Dial {
  system?: boolean
  name?: string
  allowed_ips?: string[]
  use_http2?: boolean
  use_ipv6?: boolean
  profile?: CloudflareProfile
  udp_timeout?: string
  udp_keepalive_period?: string
  udp_initial_packet_size?: number
  reconnect_delay?: string
  tls?: MasqueTls
}

export interface OpenVPNTls {
  certificate?: string
  certificate_path?: string
  key?: string
  key_path?: string
  ca?: string
  ca_path?: string
  cipher_suites?: string[]
  verify_x509_name?: string
  verify_x509_name_mode?: string
  kernel_tx?: boolean
  kernel_rx?: boolean
}

export interface OpenVPN extends OutboundBasics, Dial {
  system?: boolean
  name?: string
  allowed_ips?: string[]
  servers: { server: string; server_port: number }[]
  proto?: "udp" | "tcp"
  cipher?: string
  auth?: string
  username?: string
  password?: string
  tls_crypt?: string
  tls_crypt_path?: string
  tls_crypt_v2?: boolean
  tls_auth?: string
  tls_auth_path?: string
  key_direction?: number
  reconnect_delay?: string
  ping_interval?: string
  tls?: OpenVPNTls
}

export interface Parser extends OutboundBasics, Dial {
  link: string
}

export interface Selector extends OutboundBasics {
  outbounds: string[]
  url?: string
  interval?: string
  tolerance?: number
  idle_timeout?: string
  interrupt_exist_connections?: boolean
}

export interface URLTest extends OutboundBasics {
  outbounds: string[]
  default?: string
  interrupt_exist_connections?: boolean
}

export interface BondOutbound {
  outbound: { [key: string]: any }
  download_ratio: number
  upload_ratio: number
  count?: number
}

export interface Bond extends OutboundBasics {
  outbounds: BondOutbound[]
}

export interface Failover extends OutboundBasics {
  strategy?: "sequential" | "cycle"
  delay?: string
  outbounds: { [key: string]: any }[]
}

export interface Fallback extends OutboundBasics {
  outbounds: string[]
}

export interface LimiterRoute {
  rules?: { [key: string]: any }[]
  final?: string
  [key: string]: any
}

export interface BandwidthLimiterUser {
  name: string
  strategy: string
  connection_type?: string
  mode: string
  speed: string
}

export interface BandwidthLimiter extends OutboundBasics {
  strategy: string
  connection_type?: string
  mode: string
  flow_keys?: string[]
  speed: string
  users?: BandwidthLimiterUser[]
  route: LimiterRoute
}

export interface ConnectionLimiterUser {
  name: string
  strategy: string
  connection_type?: string
  count: number
}

export interface ConnectionLimiter extends OutboundBasics {
  strategy: string
  connection_type?: string
  count: number
  users?: ConnectionLimiterUser[]
  route: LimiterRoute
}

export interface TrafficLimiterUser {
  name: string
  strategy: string
  mode: string
  total: string
}

export interface TrafficLimiter extends OutboundBasics {
  strategy: string
  mode: string
  total: string
  users?: TrafficLimiterUser[]
  route: LimiterRoute
}

export interface RateLimiterUser {
  name: string
  strategy: string
  connection_type?: string
  count: number
  interval: string
}

export interface RateLimiter extends OutboundBasics {
  strategy: string
  connection_type?: string
  count: number
  interval: string
  users?: RateLimiterUser[]
  route: LimiterRoute
}

// Create interfaces dynamically based on OutTypes keys
type InterfaceMap = {
  [Key in keyof typeof OutTypes]: {
    type: string
    [otherProperties: string]: any // You can add other properties as needed
  }
}

// Create union type from InterfaceMap
export type Outbound = InterfaceMap[keyof InterfaceMap]

// Create defaultValues object dynamically
const defaultValues: Record<OutType, Outbound> = {
  direct: { type: OutTypes.Direct },
  socks: { type: OutTypes.SOCKS, version: "5" },
  http: { type: OutTypes.HTTP, tls: {} },
  shadowsocks: { type: OutTypes.Shadowsocks, method: 'none', multiplex: {} },
  vmess: { type: OutTypes.VMess, tls: {}, multiplex: {}, transport: {}, security: 'auto', global_padding: false, alter_id: 0, packet_encoding: 'xudp' },
  trojan: { type: OutTypes.Trojan, tls: {}, multiplex: {}, transport: {} },
  naive: { type: OutTypes.Naive, tls: { enabled: true }, quic_congestion_control: 'bbr' },
  hysteria: { type: OutTypes.Hysteria, up_mbps: 100, down_mbps: 100, tls: { enabled: true } },
  shadowtls: { type: OutTypes.ShadowTLS, version: 3, tls: { enabled: true } },
  vless: { type: OutTypes.VLESS, tls: {}, multiplex: {}, transport: {}, packet_encoding: 'xudp' },
  tuic: { type: OutTypes.TUIC, congestion_control: 'bbr', udp_relay_mode: 'native', tls: { enabled: true } },
  hysteria2: { type: OutTypes.Hysteria2, tls: { enabled: true } },
  anytls: { type: OutTypes.AnyTls, tls: { enabled: true } },
  tor: { type: OutTypes.Tor, executable_path: './tor', data_directory: '$HOME/.cache/tor', torrc: { ClientOnly: '1' } },
  ssh: { type: OutTypes.SSH },
  mieru: { type: OutTypes.Mieru, transport: 'TCP', multiplexing: 'MULTIPLEXING_LOW' },
  sudoku: { type: OutTypes.Sudoku, key: '', aead_method: 'chacha20-poly1305', padding_min: 10, padding_max: 30, enable_pure_downlink: true },
  trusttunnel: { type: OutTypes.TrustTunnel, network: ['tcp', 'udp'], congestion_controller: 'bbr', tls: { enabled: true } },
  masque: { type: OutTypes.MASQUE, use_http2: false, use_ipv6: false, profile: { detour: 'direct' }, udp_timeout: '5m0s', udp_keepalive_period: '30s', reconnect_delay: '5s', tls: {} },
  openvpn: { type: OutTypes.OpenVPN, servers: [{ server: '', server_port: 1194 }], proto: 'udp', cipher: 'AES-256-GCM', auth: 'SHA256', tls: {} },
  parser: { type: OutTypes.Parser, link: '' },
  selector: { type: OutTypes.Selector },
  urltest: { type: OutTypes.URLTest },
  bond: { type: OutTypes.Bond, outbounds: [] },
  failover: { type: OutTypes.Failover, strategy: 'sequential', outbounds: [] },
  fallback: { type: OutTypes.Fallback, outbounds: [] },
  'bandwidth-limiter': { type: OutTypes.BandwidthLimiter, strategy: 'global', mode: 'bidirectional', speed: '2MB', route: { final: 'direct' } },
  'connection-limiter': { type: OutTypes.ConnectionLimiter, strategy: 'connection', connection_type: 'hwid', count: 5, route: { final: 'direct' } },
  'traffic-limiter': { type: OutTypes.TrafficLimiter, strategy: 'global', mode: 'bidirectional', total: '10GB', route: { final: 'direct' } },
  'rate-limiter': { type: OutTypes.RateLimiter, strategy: 'leaky-bucket', count: 10, interval: '1s', route: { final: 'direct' } },
}

export function createOutbound<T extends Outbound>(type: string,json?: Partial<T>): Outbound {
  const defaultObject: Outbound = { ...defaultValues[type], ...(json || {}) }
  return defaultObject
}
