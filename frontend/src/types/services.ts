import { Dial } from './dial'
import { Listen } from "./inbounds"
import { iTls } from "./tls"

export const SrvTypes = {
  DERP: 'derp',
  Resolved: 'resolved',
  SSMAPI: 'ssm-api',
  OCM: 'ocm',
  CCM: 'ccm',
  OOMKiller: 'oom-killer',
  Profiler: 'profiler',
  API: 'api',
  HysteriaRealm: 'hysteria-realm',
  USBIPServer: 'usbip-server',
  USBIPClient: 'usbip-client',
}

type SrvType = typeof SrvTypes[keyof typeof SrvTypes]

interface SrvBasics extends Listen {
  id: number
  type: SrvType
  tag: string
  tls_id: number
}

// A service that only dials out has no listener, so it must not inherit the
// Listen fields (domain_resolver is a string on the listen side and an object on
// the dial side - extending both would make the type contradictory).
interface SrvClientBasics {
  id: number
  type: SrvType
  tag: string
  // Panel-side TLS reference. A dial-only service never uses it, and the backend
  // strips it from the stored options (model.Service.UnmarshalJSON), so it stays
  // part of the shared shape.
  tls_id: number
}

export interface DERP extends SrvBasics {
  tls: iTls
  config_path: string
  verify_client_endpoint?: string[]
  verify_client_url?: any[]
  home?: string
  mesh_with?: any[]
  mesh_psk?: string
  mesh_psk_file?: string
  stun?: any
}

export interface Resolved extends SrvBasics {}

export interface SSMAPI extends SrvBasics {
  servers: any
  cache_path?: string
  tls?: iTls
}

export interface OCM extends SrvBasics {
  credential_path?: string
  usages_path?: string
  users?: { name: string; token: string }[]
  headers?: { [key: string]: string | string[] }
  detour?: string
}

export interface CCM extends SrvBasics {
  credential_path?: string
  usages_path?: string
  users?: { name: string; token: string }[]
  headers?: { [key: string]: string | string[] }
  detour?: string
}

export interface OOMKiller extends SrvBasics {
  memory_limit?: string | number
  safety_margin?: string | number
  min_interval?: string
  max_interval?: string
  checks_before_limit?: number
}

export interface Profiler extends SrvBasics {
  listen: string
  read_timeout?: string
  write_timeout?: string
}

export interface APIDashboard {
  enabled?: boolean
  path?: string
  download_url?: string
  http_client?: string
  update_interval?: string
}

export interface API extends SrvBasics {
  secret?: string
  access_control_allow_origin?: string[]
  access_control_allow_private_network?: boolean
  dashboard?: APIDashboard
  tls?: iTls
}

export interface HysteriaRealmUser {
  name: string
  token: string
  max_realms?: number
}

export interface HysteriaRealm extends SrvBasics {
  users: HysteriaRealmUser[]
  idle_timeout?: string
  keep_alive_period?: string
  stream_receive_window?: string | number
  connection_receive_window?: string | number
  max_concurrent_streams?: number
  tls?: iTls
}

// USBIPServerServiceOptions: listen plus the device provider. `default` serves the
// devices listed here; `dynamic` lets the server pick devices at runtime.
export interface USBIPDeviceMatch {
  bus_id?: string
  vendor_id?: number
  product_id?: number
  serial?: string
}

export interface USBIPServer extends SrvBasics {
  provider?: '' | 'default' | 'dynamic'
  devices?: USBIPDeviceMatch[]
}

// USBIPClientServiceOptions: dial the remote USB/IP server and re-export the
// selected devices locally.
export interface USBIPClient extends SrvClientBasics, Dial {
  server: string
  server_port: number
  devices?: USBIPDeviceMatch[]
}

type InterfaceMap = {
  derp: DERP
  resolved: Resolved
  'ssm-api': SSMAPI
  ocm: OCM
  ccm: CCM
  'oom-killer': OOMKiller
  profiler: Profiler
  api: API
  'hysteria-realm': HysteriaRealm
  'usbip-server': USBIPServer
  'usbip-client': USBIPClient
}

export type Srv = InterfaceMap[keyof InterfaceMap]

const defaultValues: Record<SrvType, Srv> = {
  derp: <DERP>{ type: 'derp', config_path: '', tls_id:0 },
  resolved: <Resolved>{ type: 'resolved', listen: '::', listen_port: 53 },
  'ssm-api': <SSMAPI>{ type: 'ssm-api', tls_id: 0, servers: {} },
  ocm: { type: 'ocm', id: 0, tag: '', listen: '::', listen_port: 8080, tls_id: 0, users: [] } as OCM,
  ccm: { type: 'ccm', id: 0, tag: '', listen: '::', listen_port: 8080, tls_id: 0, users: [] } as CCM,
  'oom-killer': { type: 'oom-killer', id: 0, tag: '', checks_before_limit: 3, min_interval: '30s', max_interval: '5m' } as OOMKiller,
  profiler: { type: 'profiler', id: 0, tag: '', listen: '127.0.0.1:8964' } as Profiler,
  api: { type: 'api', id: 0, tag: '', listen: '127.0.0.1', listen_port: 9090, tls_id: 0 } as API,
  'hysteria-realm': { type: 'hysteria-realm', id: 0, tag: '', listen: '::', listen_port: 0, tls_id: 0, users: [] } as HysteriaRealm,
  // USB/IP is a local device server, so it defaults to the loopback address: an
  // operator who wants it reachable has to widen `listen` deliberately.
  'usbip-server': <USBIPServer>{ type: SrvTypes.USBIPServer, id: 0, tag: '', tls_id: 0, listen: '127.0.0.1', listen_port: 3240, provider: 'default', devices: [] },
  'usbip-client': <USBIPClient>{ type: SrvTypes.USBIPClient, id: 0, tag: '', tls_id: 0, server: '', server_port: 3240, devices: [] },
}

export function createSrv<T extends Srv>(type: string, json?: Partial<T>): Srv {
  // The partial is spread over the type's own defaults. Services do not share one
  // shape (listen vs dial only), so the spread cannot be proven assignable to the
  // Srv union here; each type's default above is the type-level guarantee.
  const defaultObject = { ...defaultValues[type], ...(json || {}) } as Srv
  return defaultObject
}
