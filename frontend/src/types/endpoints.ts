import { Dial } from "./dial"

export const EpTypes = {
  Wireguard: 'wireguard',
  Warp: 'warp',
  Tailscale: 'tailscale',
  VpnServer: 'vpn-server',
  VpnClient: 'vpn-client',
  OpenVPNClient: 'openvpn-client',
  OpenVPNServer: 'openvpn-server',
  OpenConnect: 'openconnect',
}

type EpType = typeof EpTypes[keyof typeof EpTypes]

interface EndpointBasics {
  id: number
  type: EpType
  tag: string
  awgManaged?: boolean
}

export interface WgPeer {
  address: string
  port: number
  public_key: string
  pre_shared_key?: string
  allowed_ips?: string[]
  persistent_keepalive_interval?: number
  reserved?: number[]
}

export interface WireGuardAmnezia {
  jc?: number
  jmin?: number
  jmax?: number
  s1?: number
  s2?: number
  s3?: number
  s4?: number
  h1?: number | string
  h2?: number | string
  h3?: number | string
  h4?: number | string
  i1?: string
  i2?: string
  i3?: string
  i4?: string
  i5?: string
  header_protection_key?: string
  content_padding_addition?: number | string
  rekey_after_time?: number | string
  rekey_timeout?: number | string
  reject_after_time?: number | string
  keepalive_timeout?: number | string
  max_handshake_attempts?: number | string
}

export interface WireGuard extends EndpointBasics, Dial {
  system?: boolean
  name?: string
  mtu?: number
  address: string[]
  private_key: string
  listen_port: number
  peers: WgPeer[]
  udp_timeout?: string
  workers?: number
  udp_mapping?: "" | "endpoint_independent" | "address_dependent" | "address_and_port_dependent"
  udp_filtering?: "" | "endpoint_independent" | "address_dependent" | "address_and_port_dependent"
  udp_nat_max?: number
  amnezia?: WireGuardAmnezia
  disable_pauses?: boolean
  preallocated_buffers_per_pool?: number
  ext: any
}

export interface Warp extends WireGuard {
  persistent_keepalive_interval?: number
  profile?: string
  reserved?: number[]
  port?: number
}

export interface Tailscale extends EndpointBasics, Dial {
  state_directory?: string
  auth_key?: string
  control_url?: string
  ephemeral?: boolean
  hostname?: string
  accept_routes?: boolean
  exit_node?: string
  exit_node_allow_lan_access?: boolean
  advertise_routes?: string[]
  advertise_tags?: string[]
  advertise_exit_node?: boolean
  relay_server_port?: number
  relay_server_static_endpoints?: string[]
  system_interface?: boolean
  system_interface_name?: string
  system_interface_mtu?: number
  udp_timeout?: string
  listen_port?: number
  taildrop_directory?: string
  ssh_server?: boolean | {
    enabled?: boolean
    disable_pty?: boolean
    disable_sftp?: boolean
    disable_forwarding?: boolean
  }
}

export interface VpnUser {
  address: string
  key: string
}

export interface VpnServer extends EndpointBasics {
  address: string
  users: VpnUser[]
  inbounds: any[]
  connect_timeout?: string
  default_gateway?: string
  pool_size?: number
}

export interface OpenVPNRemote {
  server: string
  server_port: number
  network?: "" | "udp" | "udp4" | "udp6" | "tcp" | "tcp4" | "tcp6"
}

// OpenVPNControlWrap mirrors option.OpenVPNControlWrapOptions (tls_auth /
// tls_crypt / tls_crypt_v2 static key wrapping for the control channel).
export interface OpenVPNControlWrap {
  type?: "" | "tls_auth" | "tls_crypt" | "tls_crypt_v2"
  key?: string[]
  key_path?: string
  direction?: "" | "server" | "client"
}

// OpenVPNEndpointTLS mirrors option.OpenVPNOutboundTLSOptions /
// OpenVPNInboundTLSOptions: certificate material plus the control channel wrap.
export interface OpenVPNEndpointTLS {
  server_name?: string
  server_name_type?: "" | "subject" | "name" | "name-prefix" | "name-suffix"
  certificate?: string[]
  certificate_path?: string
  client_certificate?: string[]
  client_certificate_path?: string
  client_key?: string[]
  client_key_path?: string
  version_min?: "" | "1.0" | "1.1" | "1.2" | "1.3"
  version_max?: "" | "1.0" | "1.1" | "1.2" | "1.3"
  cipher?: string
  control_wrap?: OpenVPNControlWrap
}

export interface OpenVPNPullFilter {
  action: "accept" | "ignore" | "reject"
  match: string
  comment?: string
}

export interface OpenVPNClient extends EndpointBasics, Dial {
  system?: boolean
  name?: string
  mtu?: number
  udp_mapping?: "" | "endpoint_independent" | "address_dependent" | "address_and_port_dependent"
  udp_filtering?: "" | "endpoint_independent" | "address_dependent" | "address_and_port_dependent"
  udp_nat_max?: number
  mode?: "" | "tls" | "static_key"
  network?: "" | "udp" | "udp4" | "udp6" | "tcp" | "tcp4" | "tcp6"
  server?: string
  server_port?: number
  servers: OpenVPNRemote[]
  remote_random?: boolean
  address?: string[]
  peer_address?: string
  peer_address_ipv6?: string
  topology?: "" | "net30" | "p2p" | "subnet"
  username?: string
  password?: string
  auth_retry?: "" | "none" | "nointeract" | "interact"
  static_challenge?: string
  static_challenge_echo?: boolean
  static_key?: string[]
  static_key_path?: string
  key_direction?: "" | "server" | "client"
  auth?: string
  cipher?: string
  data_ciphers?: string[]
  data_ciphers_fallback?: string
  mss_fix?: number
  mss_fix_disabled?: boolean
  mss_fix_mode?: "" | "mtu" | "fixed"
  fragment?: number
  replay_window?: number
  replay_window_time?: string
  compression?: "" | "none" | "no" | "lz4" | "lz4-v2" | "stub" | "stub-v2" | "disabled" | "off"
  compression_lzo?: "" | "none" | "no" | "yes" | "adaptive" | "asym" | "disabled" | "off"
  allow_compression?: "" | "no" | "asym" | "yes"
  route_no_pull?: boolean
  pull_filters?: OpenVPNPullFilter[]
  routes?: string[]
  route_gateway?: string
  route_metric?: number
  redirect_gateway?: boolean
  redirect_gateway_flags?: string[]
  redirect_private?: boolean
  block_ipv6?: boolean
  ping_interval?: string
  ping_restart?: string
  ping_restart_disabled?: boolean
  renegotiate_interval?: string
  renegotiate_disabled?: boolean
  renegotiate_bytes?: number
  renegotiate_packets?: number
  tls_timeout?: string
  handshake_window?: string
  reconnect_delay?: string
  explicit_exit_notify?: number
  udp_timeout?: string
  tls?: OpenVPNEndpointTLS
}

// OpenVPNServerControlWrap mirrors OpenVPNInboundControlWrapOptions: the client
// wrap plus force_cookie.
export interface OpenVPNServerControlWrap extends OpenVPNControlWrap {
  force_cookie?: boolean
}

// OpenVPNServerTLS mirrors option.OpenVPNInboundTLSOptions: the server holds
// key/key_path and optionally verifies client certificates.
export interface OpenVPNServerTLS {
  certificate?: string[]
  certificate_path?: string
  key?: string[]
  key_path?: string
  client_certificate?: string[]
  client_certificate_path?: string
  verify_client_certificate?: "" | "require" | "optional" | "none"
  client_name?: string
  client_name_type?: "" | "subject" | "name" | "name-prefix"
  peer_fingerprint?: string[]
  crl_path?: string
  remote_certificate_ku?: string[]
  remote_certificate_eku?: string
  remote_certificate_tls?: "" | "server" | "client" | "none"
  certificate_profile?: "" | "legacy" | "preferred" | "insecure" | "suiteb"
  ns_certificate_type?: "" | "server" | "client"
  version_min?: "" | "1.0" | "1.1" | "1.2" | "1.3"
  version_max?: "" | "1.0" | "1.1" | "1.2" | "1.3"
  cipher?: string
  groups?: string
  control_wrap?: OpenVPNServerControlWrap
}

export interface OpenVPNUser {
  username: string
  password: string
}

export interface OpenVPNPushDNSServer {
  priority: number
  addresses: string[]
  resolve_domains?: string[]
  dnssec?: "" | "yes" | "optional" | "no"
  transport?: "" | "plain" | "dot" | "doh"
  sni?: string
}

// OpenVPNPush mirrors option.OpenVPNPushOptions: routes/DNS pushed to clients.
export interface OpenVPNPush {
  routes?: string[]
  dns?: string[]
  dns_servers?: OpenVPNPushDNSServer[]
  search_domains?: string[]
  dhcp_options?: string[]
  redirect_gateway?: boolean
  redirect_gateway_flags?: string[]
  block_outside_dns?: boolean
  ping_interval?: string
  ping_restart?: string
}

export interface OpenVPNServer extends EndpointBasics, Dial {
  system?: boolean
  name?: string
  mtu?: number
  udp_mapping?: "" | "endpoint_independent" | "address_dependent" | "address_and_port_dependent"
  udp_filtering?: "" | "endpoint_independent" | "address_dependent" | "address_and_port_dependent"
  udp_nat_max?: number
  mode?: "" | "tls" | "static_key"
  network?: "" | "udp" | "udp4" | "udp6" | "tcp" | "tcp4" | "tcp6"
  listen?: string
  listen_port?: number
  proxy_protocol?: boolean
  proxy_protocol_accept_no_header?: boolean
  udp_timeout?: string
  remote?: string
  remote_port?: number
  max_clients?: number
  address?: string[]
  peer_address?: string
  peer_address_ipv6?: string
  topology?: "" | "net30" | "p2p" | "subnet"
  duplicate_cn?: boolean
  users?: OpenVPNUser[]
  static_key?: string[]
  static_key_path?: string
  key_direction?: "" | "server" | "client"
  auth?: string
  cipher?: string
  data_ciphers?: string[]
  data_ciphers_fallback?: string
  mss_fix?: number
  mss_fix_disabled?: boolean
  mss_fix_mode?: "" | "mtu" | "fixed"
  replay_window?: number
  replay_window_time?: string
  push?: OpenVPNPush
  ping_interval?: string
  ping_restart?: string
  renegotiate_interval?: string
  renegotiate_disabled?: boolean
  renegotiate_bytes?: number
  renegotiate_packets?: number
  handshake_window?: string
  tls?: OpenVPNServerTLS
}

export interface VpnClient extends EndpointBasics {
  address: string
  key: string
  outbound: any
  default_gateway?: string
  reconnect_delay?: string
  reject_delay?: string
  pool_size?: number
}

// OpenConnect VPN client endpoint (AnyConnect/GlobalProtect/Fortinet/F5/Pulse/
// NC). The nested blocks (token, tls, csd, hip, tncc, mobile, fortinet_host_check)
// are kept as open records so the JSON sub-editors round-trip every field.
export interface OpenConnect extends EndpointBasics, Dial {
  system?: boolean
  name?: string
  udp_timeout?: string
  udp_mapping?: string
  udp_filtering?: string
  udp_nat_max?: number
  server: string
  flavor?: 'anyconnect' | 'gp' | 'fortinet' | 'f5' | 'pulse' | 'nc'
  username?: string
  password?: string
  auth_group?: string
  cookie?: string
  token?: {
    mode?: 'totp' | 'hotp' | 'stoken' | 'oidc'
    secret?: string
    secret_path?: string
    pin?: string
    password?: string
    device_id?: string
    counter?: number
  }
  reported_os?: string
  user_agent?: string
  version?: string
  local_hostname?: string
  mobile?: Record<string, any>
  csd?: Record<string, any>
  hip?: Record<string, any>
  tncc?: Record<string, any>
  fortinet_host_check?: Record<string, any>
  no_udp?: boolean
  dtls_local_port?: number
  compression_disabled?: boolean
  compression_mode?: 'stateless' | 'all'
  ipv6_disabled?: boolean
  http_keepalive_disabled?: boolean
  xml_post_disabled?: boolean
  external_auth_disabled?: boolean
  password_authentication_disabled?: boolean
  tcp_keep_alive_enabled?: boolean
  pfs?: boolean
  mtu?: number
  base_mtu?: number
  dpd_interval?: string
  reconnect_timeout?: string
  trojan_interval?: string
  queue_length?: number
  allow_insecure_crypto?: boolean
  tls?: Record<string, any>
  form_entries?: Record<string, any>[]
}

// Create interfaces dynamically based on EpTypes keys
type InterfaceMap = {
  [Key in keyof typeof EpTypes]: {
    type: string
    [otherProperties: string]: any // You can add other properties as needed
  }
}

// Create union type from InterfaceMap
export type Endpoint = InterfaceMap[keyof InterfaceMap]

// Create defaultValues object dynamically
const defaultValues: Record<EpType, Endpoint> = {
  wireguard: { type: EpTypes.Wireguard, address: ['10.0.0.2/32','fe80::2/128'], private_key: '', listen_port: 0 },
  warp: { type: EpTypes.Warp, address: [], private_key: '', listen_port: 0, mtu: 1420, peers: [{ address: '', port: 0, public_key: ''}] },
  tailscale: { type: EpTypes.Tailscale, domain_resolver: 'local' },
  'vpn-server': { type: EpTypes.VpnServer, address: '10.0.0.1', users: [], inbounds: [] },
  'vpn-client': { type: EpTypes.VpnClient, address: '10.0.0.2', key: '', outbound: {} },
  'openvpn-client': { type: EpTypes.OpenVPNClient, mode: 'tls', network: 'udp', servers: [{ server: '', server_port: 1194 }], data_ciphers: ['AES-256-GCM'], auth: 'SHA256', tls: {} },
  'openvpn-server': { type: EpTypes.OpenVPNServer, network: 'udp', address: ['10.8.0.1/24'], tls: {} },
  'openconnect': { type: EpTypes.OpenConnect, server: '', flavor: 'anyconnect', tls: {} },
}

export function createEndpoint<T extends Endpoint>(type: string,json?: Partial<T>): Endpoint {
  const defaultObject: Endpoint = { ...defaultValues[type], ...(json || {}) }
  return defaultObject
}
