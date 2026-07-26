import { Outbound } from "./outbounds"

export const ProviderTypes = {
  Inline: 'inline',
  Local: 'local',
  Remote: 'remote',
}

type ProviderType = typeof ProviderTypes[keyof typeof ProviderTypes]

interface ProviderBasics {
  id: number
  type: ProviderType
  tag: string
}

export interface ProviderHealthCheck {
  enabled?: boolean
  url?: string
  interval?: string
  timeout?: string
}

export interface ProviderInline extends ProviderBasics {
  outbounds: Outbound[]
  remove_emojis?: boolean
  health_check?: ProviderHealthCheck
}

export interface ProviderLocal extends ProviderBasics {
  path: string
  remove_emojis?: boolean
  health_check?: ProviderHealthCheck
}

export interface ProviderRemote extends ProviderBasics {
  url: string
  user_agent?: string
  headers?: { [key: string]: string | string[] }
  download_detour?: string
  update_interval?: string
  exclude?: string
  include?: string
  remove_emojis?: boolean
  health_check?: ProviderHealthCheck
}

// Create interfaces dynamically based on ProviderTypes keys
type InterfaceMap = {
  [Key in keyof typeof ProviderTypes]: {
    type: string
    [otherProperties: string]: any
  }
}

// Create union type from InterfaceMap
export type Provider = InterfaceMap[keyof InterfaceMap]

// Create defaultValues object dynamically
const defaultValues: Record<ProviderType, Provider> = {
  inline: { type: ProviderTypes.Inline, outbounds: [], health_check: {} },
  local: { type: ProviderTypes.Local, path: '', health_check: {} },
  remote: { type: ProviderTypes.Remote, url: '', user_agent: 'sing-box', health_check: {} },
}

export function createProvider<T extends Provider>(type: string, json?: Partial<T>): Provider {
  const defaultObject: Provider = { ...defaultValues[type], ...(json || {}) }
  return defaultObject
}
