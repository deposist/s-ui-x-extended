import { Dial } from "./dial"

export const EpTypes = {
  Wireguard: 'wireguard',
  Warp: 'warp',
  Tailscale: 'tailscale',
  VpnServer: 'vpn-server',
  VpnClient: 'vpn-client',
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
  amnezia?: WireGuardAmnezia
  disable_pauses?: boolean
  preallocated_buffers_per_pool?: number
  ext: any
}

export interface Warp extends WireGuard {
  persistent_keepalive_interval?: number
  profile?: string
  reserved?: number[]
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

export interface VpnClient extends EndpointBasics {
  address: string
  key: string
  outbound: any
  default_gateway?: string
  reconnect_delay?: string
  reject_delay?: string
  pool_size?: number
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
}

export function createEndpoint<T extends Endpoint>(type: string,json?: Partial<T>): Endpoint {
  const defaultObject: Endpoint = { ...defaultValues[type], ...(json || {}) }
  return defaultObject
}
