import { Listen } from "./inbounds"
import { iTls } from "./tls"

export const SrvTypes = {
  DERP: 'derp',
  Resolved: 'resolved',
  SSMAPI: 'ssm-api',
  OCM: 'ocm', 
  CCM: 'ccm', 
  OOMKiller: 'oom-killer',
}

type SrvType = typeof SrvTypes[keyof typeof SrvTypes]

interface SrvBasics extends Listen {
  id: number
  type: SrvType
  tag: string
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

type InterfaceMap = {
  derp: DERP
  resolved: Resolved
  'ssm-api': SSMAPI
  ocm: OCM
  ccm: CCM
  'oom-killer': OOMKiller
}

export type Srv = InterfaceMap[keyof InterfaceMap]

const defaultValues: Record<SrvType, Srv> = {
  derp: <DERP>{ type: 'derp', config_path: '', tls_id:0 },
  resolved: <Resolved>{ type: 'resolved', listen: '::', listen_port: 53 },
  'ssm-api': <SSMAPI>{ type: 'ssm-api', tls_id: 0, servers: {} },
  ocm: { type: 'ocm', id: 0, tag: '', listen: '::', listen_port: 8080, tls_id: 0, users: [] } as OCM,
  ccm: { type: 'ccm', id: 0, tag: '', listen: '::', listen_port: 8080, tls_id: 0, users: [] } as CCM,
  'oom-killer': { type: 'oom-killer', id: 0, tag: '', checks_before_limit: 3 } as OOMKiller,
}

export function createSrv<T extends Srv>(type: string, json?: Partial<T>): Srv {
  const defaultObject: Srv = { ...defaultValues[type], ...(json || {}) }
  return defaultObject
}
