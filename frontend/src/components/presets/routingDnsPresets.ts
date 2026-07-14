import type { Config } from '@/types/config'

export type PresetRegion = 'RU' | 'ZH'
export type PresetDirection = 'direct' | 'proxy'
export type PresetRegionKey = 'ru' | 'zh'

export interface PresetSource {
  name: string
  url?: string
  path?: string
}

export interface RegionalPresetState {
  region: PresetRegion
  enabled: boolean
  direction: PresetDirection
  exceptions: string[]
}

export interface PresetManagedIdentity {
  region: PresetRegion
  direction?: PresetDirection
  tags: string[]
}

export interface ApplyPresetOptions {
  proxyOutbound?: string
  directOutbound: string
  direction?: PresetDirection
  exceptions?: string[]
}

export interface ApplyPresetsOptions {
  proxyOutbound?: string
  directOutbound: string
}

export interface PresetPreviewGroup {
  willAdd: string[]
  willChange: string[]
  willKeep: string[]
  willRemove: string[]
  securityWarnings: string[]
}

export type PresetPreview = Record<PresetRegionKey, PresetPreviewGroup>

export interface ApplyPresetResult {
  config: Config
  changes: string[]
  preview: PresetPreviewGroup
}

export interface ApplyPresetsResult {
  config: Config
  changes: string[]
  preview: PresetPreview
}

export type PresetApplier = (config: Config, options: ApplyPresetOptions, changes: string[]) => void

export interface RoutingDnsPreset {
  id: string
  region: PresetRegion
  direction: PresetDirection
  titleKey: string
  descriptionKey: string
  sources: PresetSource[]
  apply: PresetApplier
}

export interface DetectedPresetState {
  ru: RegionalPresetState
  zh: RegionalPresetState
}

const SOURCE_URLS = {
  ruSmartGeositeDat: 'https://github.com/wastrel-g/geosite-ru-smart/releases/latest/download/geosite.dat',
  ruGeoip: 'https://raw.githubusercontent.com/runetfreedom/russia-v2ray-rules-dat/release/sing-box/rule-set-geoip/geoip-ru.srs',
  cnGeosite: 'https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-geolocation-cn.srs',
  cnGeoip: 'https://raw.githubusercontent.com/SagerNet/sing-geoip/rule-set/geoip-cn.srs',
} as const

const MANAGED_RU_SMART_RULESET_PATH = 'rulesets/geosite-ru-smart/direct-ru.srs'

const DNS_DIRECT_TAG = 'preset-dns-direct'
const DNS_PROXY_TAG = 'preset-dns-proxy'
const BUILTIN_DIRECT_OUTBOUND_TAG = 'direct'

const COUNTRY_PRESET_NAMES = ['geosite', 'geoip'] as const
const LEGACY_RU_PRESET_NAMES = ['blocked', 'private', 'exceptions'] as const
const LEGACY_ZH_PRESET_NAMES = ['geosite-cn', 'geoip-cn', 'geosite-non-cn', 'exceptions'] as const

const cloneConfig = (config: Config): Config => JSON.parse(JSON.stringify(config ?? {}))

const ensureConfigShape = (config: Config) => {
  if (!config.route) config.route = { rules: [], rule_set: [] }
  if (!Array.isArray(config.route.rules)) config.route.rules = []
  if (!Array.isArray(config.route.rule_set)) config.route.rule_set = []
  if (!config.dns) config.dns = { servers: [], rules: [] }
  if (!Array.isArray(config.dns.servers)) config.dns.servers = []
  if (!Array.isArray(config.dns.rules)) config.dns.rules = []
  if (!Array.isArray(config.outbounds)) config.outbounds = []
  if (!config.experimental) config.experimental = {}
  if (!config.experimental.cache_file) config.experimental.cache_file = {}
}

const asArray = (value: unknown): string[] => {
  if (Array.isArray(value)) return value.map(String)
  if (typeof value === 'string') return [value]
  return []
}

const regionKey = (region: PresetRegion): PresetRegionKey => region === 'RU' ? 'ru' : 'zh'

const tagPrefix = (region: PresetRegion, direction: PresetDirection) =>
  `preset-${regionKey(region)}-${direction}`

const presetRuleSetTag = (region: PresetRegion, direction: PresetDirection, name: string) =>
  `${tagPrefix(region, direction)}-${name}`

const currentRegionRuleSetTags = (region: PresetRegion): string[] =>
  COUNTRY_PRESET_NAMES.map(name => presetRuleSetTag(region, 'direct', name))

const legacyRegionRuleSetTags = (region: PresetRegion): string[] => {
  const names = region === 'RU' ? LEGACY_RU_PRESET_NAMES : LEGACY_ZH_PRESET_NAMES
  return (['direct', 'proxy'] as const).flatMap(direction =>
    names.map(name => presetRuleSetTag(region, direction, name)))
}

const regionRuleSetTags = (region: PresetRegion): string[] => [
  ...currentRegionRuleSetTags(region),
  ...legacyRegionRuleSetTags(region),
]

const allPresetRuleSetTags = () => [
  ...regionRuleSetTags('RU'),
  ...regionRuleSetTags('ZH'),
]

const hasAnyRuleSet = (item: any, tags: Set<string>) =>
  asArray(item?.rule_set).some(tag => tags.has(tag))

export const isPresetManagedItem = (item: any): boolean => {
  const tag = String(item?.tag ?? '')
  if (tag === DNS_DIRECT_TAG || tag === DNS_PROXY_TAG) return true
  if (tag.startsWith('preset-ru-') || tag.startsWith('preset-zh-')) return true
  return hasAnyRuleSet(item, new Set(allPresetRuleSetTags()))
}

const remoteRuleSet = (tag: string, url: string, downloadDetour: string) => ({
  type: 'remote',
  tag,
  format: 'binary',
  url,
  download_detour: downloadDetour,
  update_interval: '24h',
})

const localRuleSet = (tag: string, path: string) => ({
  type: 'local',
  tag,
  format: 'binary',
  path,
})

const dnsServer = (tag: string, server: string, detour?: string) => ({
  type: 'udp',
  tag,
  server,
  server_port: 53,
  ...(detour ? { detour } : {}),
})

const isMeaningfulOutboundValue = (value: unknown): boolean => {
  if (value === undefined || value === null || value === '') return false
  if (typeof value === 'boolean') return value
  if (typeof value === 'number') return value !== 0
  if (Array.isArray(value)) return value.some(item => isMeaningfulOutboundValue(item))
  if (typeof value === 'object') return Object.values(value).some(item => isMeaningfulOutboundValue(item))
  return true
}

const isEmptyBuiltinDirectOutbound = (config: Config, tag: string): boolean => {
  if (tag !== BUILTIN_DIRECT_OUTBOUND_TAG) return false
  const outbound = (config.outbounds as any[]).find(item => String(item?.tag ?? '') === tag)
  if (!outbound) return true
  if (String(outbound?.type ?? '') !== 'direct') return false
  return !Object.entries(outbound)
    .some(([key, value]) => !['id', 'type', 'tag'].includes(key) && isMeaningfulOutboundValue(value))
}

const dnsDetourForOutbound = (config: Config, outboundTag: string): string | undefined =>
  isEmptyBuiltinDirectOutbound(config, outboundTag) ? undefined : outboundTag

const upsertByTag = (items: any[], item: any) => {
  const index = items.findIndex(existing => existing?.tag === item.tag)
  if (index === -1) {
    items.push(item)
    return 'add'
  }
  items[index] = item
  return 'update'
}

const addDirectDnsServer = (config: Config, directOutbound: string, changes: string[]) => {
  const server = dnsServer(DNS_DIRECT_TAG, '223.5.5.5', dnsDetourForOutbound(config, directOutbound))
  const action = upsertByTag(config.dns.servers as any[], server)
  changes.push(`${action} dns server ${server.tag}`)
}

const addRuleSet = (config: Config, ruleSet: any, changes: string[]) => {
  ;(config.route.rule_set as any[]).push(ruleSet)
  changes.push(`add rule-set ${ruleSet.tag}`)
}

const addRouteRule = (config: Config, rule: any, label: string, changes: string[]) => {
  ;(config.route.rules as any[]).push(rule)
  changes.push(`add route rule ${label}`)
}

const addDnsRule = (config: Config, rule: any, label: string, changes: string[]) => {
  ;(config.dns.rules as any[]).push(rule)
  changes.push(`add dns rule ${label}`)
}

const countrySources = (region: PresetRegion) => region === 'RU'
  ? {
      geosite: MANAGED_RU_SMART_RULESET_PATH,
      geositeType: 'local' as const,
      geoip: SOURCE_URLS.ruGeoip,
      names: [
        { name: 'wastrel-g/geosite-ru-smart: direct-ru', url: SOURCE_URLS.ruSmartGeositeDat, path: MANAGED_RU_SMART_RULESET_PATH },
        { name: 'runetfreedom/russia-v2ray-rules-dat: geoip-ru', url: SOURCE_URLS.ruGeoip },
      ],
    }
  : {
      geosite: SOURCE_URLS.cnGeosite,
      geositeType: 'remote' as const,
      geoip: SOURCE_URLS.cnGeoip,
      names: [
        { name: 'SagerNet/sing-geosite: geolocation-cn', url: SOURCE_URLS.cnGeosite },
        { name: 'SagerNet/sing-geoip: cn', url: SOURCE_URLS.cnGeoip },
      ],
    }

const applyRegionDirectPreset = (region: PresetRegion): PresetApplier => (config, options, changes) => {
  const directOutbound = options.directOutbound
  const sources = countrySources(region)
  const geositeTag = presetRuleSetTag(region, 'direct', 'geosite')
  const geoipTag = presetRuleSetTag(region, 'direct', 'geoip')

  addRuleSet(
    config,
    sources.geositeType === 'local'
      ? localRuleSet(geositeTag, sources.geosite)
      : remoteRuleSet(geositeTag, sources.geosite, directOutbound),
    changes,
  )
  addRuleSet(config, remoteRuleSet(geoipTag, sources.geoip, directOutbound), changes)
  addRouteRule(config, { rule_set: [geositeTag, geoipTag], outbound: directOutbound }, `${geositeTag}, ${geoipTag}`, changes)
  addDnsRule(config, { action: 'route', rule_set: [geositeTag], server: DNS_DIRECT_TAG }, geositeTag, changes)
}

const applyRuDirect = applyRegionDirectPreset('RU')
const applyZhDirect = applyRegionDirectPreset('ZH')

const makePreset = (region: PresetRegion, id: string): RoutingDnsPreset => ({
  id,
  region,
  direction: 'direct',
  titleKey: `regionalPresets.region.${regionKey(region)}.title`,
  descriptionKey: `regionalPresets.region.${regionKey(region)}.description`,
  sources: countrySources(region).names,
  apply: region === 'RU' ? applyRuDirect : applyZhDirect,
})

export const routingDnsPresetCatalog: RoutingDnsPreset[] = [
  makePreset('RU', 'ru-direct'),
  makePreset('ZH', 'zh-direct'),
]

const presetByRegion = (region: PresetRegion) => {
  const preset = routingDnsPresetCatalog.find(item => item.region === region)
  if (!preset) throw new Error(`unknown preset for ${region}`)
  return preset
}

const defaultState = (region: PresetRegion): RegionalPresetState => ({
  region,
  enabled: false,
  direction: 'direct',
  exceptions: [],
})

const detectRegionState = (config: Config, region: PresetRegion): RegionalPresetState => {
  ensureConfigShape(config)
  const ruleSets = config.route.rule_set as any[]
  const tags = new Set(regionRuleSetTags(region))
  const matchingRuleSets = ruleSets.filter(item => tags.has(String(item?.tag ?? '')))

  if (matchingRuleSets.length === 0) return defaultState(region)
  return {
    region,
    enabled: true,
    direction: 'direct',
    exceptions: [],
  }
}

export const detectPresetState = (input: Config): DetectedPresetState => {
  const config = cloneConfig(input)
  ensureConfigShape(config)
  return {
    ru: detectRegionState(config, 'RU'),
    zh: detectRegionState(config, 'ZH'),
  }
}

const removeItemsReferencingTags = (items: any[], tags: Set<string>) =>
  items.filter(item => !hasAnyRuleSet(item, tags))

const presetDnsServersInUse = (config: Config) => {
  const rules = (config.dns.rules as any[]) ?? []
  return new Set(rules.map(rule => String(rule?.server ?? '')).filter(server => server === DNS_DIRECT_TAG || server === DNS_PROXY_TAG))
}

const pruneUnusedPresetDnsServers = (config: Config) => {
  const usedServers = presetDnsServersInUse(config)
  config.dns.servers = (config.dns.servers as any[]).filter(server => {
    const tag = String(server?.tag ?? '')
    if (tag !== DNS_DIRECT_TAG && tag !== DNS_PROXY_TAG) return true
    return usedServers.has(tag)
  }) as any
}

export const removePresetManagedItems = (config: Config, region: PresetRegion) => {
  ensureConfigShape(config)
  const tags = new Set(regionRuleSetTags(region))
  config.route.rule_set = (config.route.rule_set as any[]).filter(item => !tags.has(String(item?.tag ?? ''))) as any
  config.route.rules = removeItemsReferencingTags(config.route.rules as any[], tags) as any
  config.dns.rules = removeItemsReferencingTags(config.dns.rules as any[], tags) as any
  pruneUnusedPresetDnsServers(config)
}

const existingManagedItemsForRegion = (config: Config, region: PresetRegion) => {
  const tags = new Set(regionRuleSetTags(region))
  const routeRuleSets = (config.route.rule_set as any[]).filter(item => tags.has(String(item?.tag ?? '')))
  const routeRules = (config.route.rules as any[]).filter(item => hasAnyRuleSet(item, tags))
  const dnsRules = (config.dns.rules as any[]).filter(item => hasAnyRuleSet(item, tags))
  return { routeRuleSets, routeRules, dnsRules }
}

const countCustomItems = (config: Config, region: PresetRegion) => {
  const tags = new Set(regionRuleSetTags(region))
  const routeRules = (config.route.rules as any[]).filter(item => !hasAnyRuleSet(item, tags)).length
  const dnsRules = (config.dns.rules as any[]).filter(item => !hasAnyRuleSet(item, tags)).length
  return { routeRules, dnsRules }
}

const emptyPreviewGroup = (): PresetPreviewGroup => ({
  willAdd: [],
  willChange: [],
  willKeep: [],
  willRemove: [],
  securityWarnings: [],
})

const makeRegionPreview = (
  input: Config,
  state: RegionalPresetState,
  options: ApplyPresetsOptions,
): PresetPreviewGroup => {
  const before = cloneConfig(input)
  ensureConfigShape(before)
  const preview = emptyPreviewGroup()
  const existing = existingManagedItemsForRegion(before, state.region)
  const custom = countCustomItems(before, state.region)

  if (custom.routeRules > 0) preview.willKeep.push(`${custom.routeRules} custom route rule(s)`)
  if (custom.dnsRules > 0) preview.willKeep.push(`${custom.dnsRules} custom DNS rule(s)`)

  if (!state.enabled) {
    preview.willRemove.push(...existing.routeRuleSets.map(item => `rule-set ${item.tag}`))
    preview.willRemove.push(...existing.routeRules.map(item => `route rule ${asArray(item.rule_set).join(', ')}`))
    preview.willRemove.push(...existing.dnsRules.map(item => `dns rule ${asArray(item.rule_set).join(', ')}`))
    return preview
  }

  const after = cloneConfig(before)
  const changes: string[] = []
  removePresetManagedItems(after, state.region)
  addDirectDnsServer(after, options.directOutbound, changes)
  presetByRegion(state.region).apply(after, { directOutbound: options.directOutbound }, changes)
  pruneUnusedPresetDnsServers(after)

  const desired = existingManagedItemsForRegion(after, state.region)
  const existingRuleSetTags = new Set(existing.routeRuleSets.map(item => String(item?.tag ?? '')))
  const existingRouteRuleKeys = new Set(existing.routeRules.map(item => asArray(item?.rule_set).join('|')))
  const existingDnsRuleKeys = new Set(existing.dnsRules.map(item => asArray(item?.rule_set).join('|')))

  for (const item of desired.routeRuleSets) {
    const label = `rule-set ${item.tag}`
    ;(existingRuleSetTags.has(item.tag) ? preview.willChange : preview.willAdd).push(label)
  }
  for (const item of desired.routeRules) {
    const key = asArray(item?.rule_set).join('|')
    const label = `route rule ${asArray(item?.rule_set).join(', ')}`
    ;(existingRouteRuleKeys.has(key) ? preview.willChange : preview.willAdd).push(label)
  }
  for (const item of desired.dnsRules) {
    const key = asArray(item?.rule_set).join('|')
    const label = `dns rule ${asArray(item?.rule_set).join(', ')}`
    ;(existingDnsRuleKeys.has(key) ? preview.willChange : preview.willAdd).push(label)
  }

  const desiredRuleSetTags = new Set(desired.routeRuleSets.map(item => String(item?.tag ?? '')))
  const removed = existing.routeRuleSets.filter(item => !desiredRuleSetTags.has(String(item?.tag ?? '')))
  preview.willRemove.push(...removed.map(item => `rule-set ${item.tag}`))

  return preview
}

export const computePreview = (
  input: Config,
  ruState: RegionalPresetState,
  zhState: RegionalPresetState,
  options: ApplyPresetsOptions,
): PresetPreview => ({
  ru: makeRegionPreview(input, ruState, options),
  zh: makeRegionPreview(input, zhState, options),
})

export const applyPresets = (
  input: Config,
  ruState: RegionalPresetState,
  zhState: RegionalPresetState,
  options: ApplyPresetsOptions,
): ApplyPresetsResult => {
  if (!options.directOutbound) {
    throw new Error('directOutbound is required')
  }
  if (!validatePresetCatalogShape()) {
    throw new Error('preset catalog contains invalid source URLs')
  }

  const config = cloneConfig(input)
  const changes: string[] = []
  ensureConfigShape(config)

  if (config.experimental.cache_file?.enabled !== true) {
    config.experimental.cache_file!.enabled = true
    changes.push('enable experimental.cache_file')
  }

  const states = [ruState, zhState]
  for (const state of states) {
    removePresetManagedItems(config, state.region)
  }

  if (states.some(state => state.enabled)) {
    addDirectDnsServer(config, options.directOutbound, changes)
  }

  for (const state of states) {
    if (!state.enabled) continue
    presetByRegion(state.region).apply(config, { directOutbound: options.directOutbound }, changes)
  }

  pruneUnusedPresetDnsServers(config)

  return {
    config,
    changes,
    preview: computePreview(input, ruState, zhState, options),
  }
}

export const applyRoutingDnsPreset = (
  input: Config,
  presetId: string,
  options: ApplyPresetOptions,
): ApplyPresetResult => {
  if (!options.directOutbound) {
    throw new Error('directOutbound is required')
  }

  const preset = routingDnsPresetCatalog.find(item => item.id === presetId)
  if (!preset) throw new Error(`unknown preset: ${presetId}`)

  const detected = detectPresetState(input)
  const state: RegionalPresetState = {
    region: preset.region,
    enabled: true,
    direction: 'direct',
    exceptions: [],
  }
  const result = applyPresets(
    input,
    preset.region === 'RU' ? state : detected.ru,
    preset.region === 'ZH' ? state : detected.zh,
    { directOutbound: options.directOutbound },
  )

  return {
    config: result.config,
    changes: result.changes,
    preview: result.preview[regionKey(preset.region)],
  }
}

export const validatePresetCatalogShape = () => routingDnsPresetCatalog.every(preset =>
  typeof preset.apply === 'function' &&
  preset.sources.length > 0 &&
  preset.sources.every(source => {
    const hasSafeURL = !source.url || (() => {
      const parsed = new URL(source.url)
      return parsed.protocol === 'https:' &&
        parsed.username === '' &&
        parsed.password === '' &&
        (parsed.pathname.endsWith('.srs') || parsed.pathname.endsWith('/geosite.dat'))
    })()
    const hasSafeManagedPath = !source.path || source.path === MANAGED_RU_SMART_RULESET_PATH
    return hasSafeURL && hasSafeManagedPath && (Boolean(source.url) || Boolean(source.path))
  }))
