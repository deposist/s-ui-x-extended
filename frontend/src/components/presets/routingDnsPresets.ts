import type { Config } from '@/types/config'

export type PresetRegion = 'RU' | 'ZH'
export type PresetDirection = 'direct' | 'proxy'
export type PresetRegionKey = 'ru' | 'zh'

export interface PresetSource {
  name: string
  url: string
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
  proxyOutbound: string
  directOutbound: string
  direction?: PresetDirection
  exceptions?: string[]
}

export interface ApplyPresetsOptions {
  proxyOutbound: string
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

export type PresetApplier = (config: Config, options: Required<ApplyPresetOptions>, changes: string[]) => void

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
  cnGeosite: 'https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-geolocation-cn.srs',
  nonCnGeosite: 'https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-geolocation-!cn.srs',
  cnGeoip: 'https://raw.githubusercontent.com/SagerNet/sing-geoip/rule-set/geoip-cn.srs',
  ruBlocked: 'https://raw.githubusercontent.com/runetfreedom/russia-blocked-geoip/release/srs/re-filter.srs',
  ruPrivate: 'https://raw.githubusercontent.com/runetfreedom/russia-blocked-geoip/release/srs/private.srs',
} as const

const DNS_DIRECT_TAG = 'preset-dns-direct'
const DNS_PROXY_TAG = 'preset-dns-proxy'

const RU_PRESET_NAMES = ['blocked', 'private', 'exceptions'] as const
const ZH_PRESET_NAMES = ['geosite-cn', 'geoip-cn', 'geosite-non-cn', 'exceptions'] as const

const cloneConfig = (config: Config): Config => JSON.parse(JSON.stringify(config ?? {}))

const ensureConfigShape = (config: Config) => {
  if (!config.route) config.route = { rules: [], rule_set: [] }
  if (!Array.isArray(config.route.rules)) config.route.rules = []
  if (!Array.isArray(config.route.rule_set)) config.route.rule_set = []
  if (!config.dns) config.dns = { servers: [], rules: [] }
  if (!Array.isArray(config.dns.servers)) config.dns.servers = []
  if (!Array.isArray(config.dns.rules)) config.dns.rules = []
  if (!config.experimental) config.experimental = {}
  if (!config.experimental.cache_file) config.experimental.cache_file = {}
}

const asArray = (value: unknown): string[] => {
  if (Array.isArray(value)) return value.map(String)
  if (typeof value === 'string') return [value]
  return []
}

const unique = (items: string[]) => [...new Set(items.filter(Boolean))]

const regionKey = (region: PresetRegion): PresetRegionKey => region === 'RU' ? 'ru' : 'zh'

const tagPrefix = (region: PresetRegion, direction: PresetDirection) =>
  `preset-${regionKey(region)}-${direction}`

const presetRuleSetTag = (region: PresetRegion, direction: PresetDirection, name: string) =>
  `${tagPrefix(region, direction)}-${name}`

const directionFromTag = (tag: string, region: PresetRegion): PresetDirection | undefined => {
  if (tag.startsWith(`${tagPrefix(region, 'proxy')}-`)) return 'proxy'
  if (tag.startsWith(`${tagPrefix(region, 'direct')}-`)) return 'direct'
  return undefined
}

const regionRuleSetTags = (region: PresetRegion): string[] => {
  const names = region === 'RU' ? RU_PRESET_NAMES : ZH_PRESET_NAMES
  return (['direct', 'proxy'] as const).flatMap(direction =>
    names.map(name => presetRuleSetTag(region, direction, name)))
}

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

const inlineExceptionRuleSet = (tag: string, exceptions: string[]) => ({
  type: 'inline',
  tag,
  rules: [
    { domain_suffix: unique(exceptions) },
  ],
})

const dnsServer = (tag: string, server: string, detour: string) => ({
  type: 'udp',
  tag,
  server,
  server_port: 53,
  detour,
})

const addCommonDnsServers = (config: Config, options: ApplyPresetsOptions, changes: string[]) => {
  const servers = config.dns.servers as any[]
  const upsert = (server: any) => {
    const index = servers.findIndex(item => item?.tag === server.tag)
    if (index === -1) {
      servers.push(server)
      changes.push(`add dns server ${server.tag}`)
      return
    }
    servers[index] = { ...servers[index], ...server }
    changes.push(`update dns server ${server.tag}`)
  }

  upsert(dnsServer(DNS_DIRECT_TAG, '223.5.5.5', options.directOutbound))
  upsert(dnsServer(DNS_PROXY_TAG, '1.1.1.1', options.proxyOutbound))
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

const oppositeDirection = (direction: PresetDirection): PresetDirection =>
  direction === 'direct' ? 'proxy' : 'direct'

const outboundForDirection = (direction: PresetDirection, options: ApplyPresetsOptions) =>
  direction === 'direct' ? options.directOutbound : options.proxyOutbound

const dnsForDirection = (direction: PresetDirection) =>
  direction === 'direct' ? DNS_DIRECT_TAG : DNS_PROXY_TAG

const normalizedExceptions = (exceptions: string[] = []) =>
  unique(exceptions.map(item => item.trim().replace(/^\.+|\.+$/g, '').toLowerCase()).filter(Boolean))

const addExceptionRules = (
  config: Config,
  region: PresetRegion,
  direction: PresetDirection,
  exceptions: string[],
  options: ApplyPresetsOptions,
  changes: string[],
) => {
  const normalized = normalizedExceptions(exceptions)
  if (normalized.length === 0) return

  const exceptionDirection = oppositeDirection(direction)
  const exceptionTag = presetRuleSetTag(region, direction, 'exceptions')
  addRuleSet(config, inlineExceptionRuleSet(exceptionTag, normalized), changes)
  addRouteRule(config, { rule_set: [exceptionTag], outbound: outboundForDirection(exceptionDirection, options) }, `${exceptionTag}`, changes)
  addDnsRule(config, { action: 'route', rule_set: [exceptionTag], server: dnsForDirection(exceptionDirection) }, `${exceptionTag}`, changes)
}

const applyRuPreset: PresetApplier = (config, options, changes) => {
  const direction = options.direction
  const blockedTag = presetRuleSetTag('RU', direction, 'blocked')
  const privateTag = presetRuleSetTag('RU', direction, 'private')
  const blockedOutbound = direction === 'proxy' ? options.proxyOutbound : options.directOutbound
  const blockedDns = direction === 'proxy' ? DNS_PROXY_TAG : DNS_DIRECT_TAG

  addExceptionRules(config, 'RU', direction, options.exceptions, options, changes)
  addRuleSet(config, remoteRuleSet(blockedTag, SOURCE_URLS.ruBlocked, blockedOutbound), changes)
  addRuleSet(config, remoteRuleSet(privateTag, SOURCE_URLS.ruPrivate, options.directOutbound), changes)
  addRouteRule(config, { rule_set: [privateTag], outbound: options.directOutbound }, privateTag, changes)
  addRouteRule(config, { rule_set: [blockedTag], outbound: blockedOutbound }, blockedTag, changes)
  addDnsRule(config, { action: 'route', rule_set: [privateTag], server: DNS_DIRECT_TAG }, privateTag, changes)
  addDnsRule(config, { action: 'route', rule_set: [blockedTag], server: blockedDns }, blockedTag, changes)
}

const applyZhPreset: PresetApplier = (config, options, changes) => {
  const direction = options.direction
  const cnTag = presetRuleSetTag('ZH', direction, 'geosite-cn')
  const geoipTag = presetRuleSetTag('ZH', direction, 'geoip-cn')
  const nonCnTag = presetRuleSetTag('ZH', direction, 'geosite-non-cn')
  const regionalOutbound = outboundForDirection(direction, options)
  const regionalDns = dnsForDirection(direction)
  const nonRegionalDirection = oppositeDirection(direction)
  const nonRegionalOutbound = outboundForDirection(nonRegionalDirection, options)
  const nonRegionalDns = dnsForDirection(nonRegionalDirection)

  addExceptionRules(config, 'ZH', direction, options.exceptions, options, changes)
  addRuleSet(config, remoteRuleSet(cnTag, SOURCE_URLS.cnGeosite, regionalOutbound), changes)
  addRuleSet(config, remoteRuleSet(geoipTag, SOURCE_URLS.cnGeoip, regionalOutbound), changes)
  addRuleSet(config, remoteRuleSet(nonCnTag, SOURCE_URLS.nonCnGeosite, nonRegionalOutbound), changes)
  addRouteRule(config, { rule_set: [cnTag, geoipTag], outbound: regionalOutbound }, `${cnTag}, ${geoipTag}`, changes)
  addRouteRule(config, { rule_set: [nonCnTag], outbound: nonRegionalOutbound }, nonCnTag, changes)
  addDnsRule(config, { action: 'route', rule_set: [cnTag], server: regionalDns }, cnTag, changes)
  addDnsRule(config, { action: 'route', rule_set: [nonCnTag], server: nonRegionalDns }, nonCnTag, changes)
}

const applyRuDirect: PresetApplier = (config, options, changes) => applyRuPreset(config, { ...options, direction: 'direct' }, changes)
const applyRuProxy: PresetApplier = (config, options, changes) => applyRuPreset(config, { ...options, direction: 'proxy' }, changes)
const applyZhDirect: PresetApplier = (config, options, changes) => applyZhPreset(config, { ...options, direction: 'direct' }, changes)
const applyZhProxy: PresetApplier = (config, options, changes) => applyZhPreset(config, { ...options, direction: 'proxy' }, changes)

export const routingDnsPresetCatalog: RoutingDnsPreset[] = [
  {
    id: 'ru-direct',
    region: 'RU',
    direction: 'direct',
    titleKey: 'regionalPresets.region.ru.title',
    descriptionKey: 'regionalPresets.region.ru.description',
    sources: [
      { name: 'runetfreedom/russia-blocked-geoip: re-filter', url: SOURCE_URLS.ruBlocked },
      { name: 'runetfreedom/russia-blocked-geoip: private', url: SOURCE_URLS.ruPrivate },
    ],
    apply: applyRuDirect,
  },
  {
    id: 'ru-proxy',
    region: 'RU',
    direction: 'proxy',
    titleKey: 'regionalPresets.region.ru.title',
    descriptionKey: 'regionalPresets.region.ru.description',
    sources: [
      { name: 'runetfreedom/russia-blocked-geoip: re-filter', url: SOURCE_URLS.ruBlocked },
      { name: 'runetfreedom/russia-blocked-geoip: private', url: SOURCE_URLS.ruPrivate },
    ],
    apply: applyRuProxy,
  },
  {
    id: 'zh-direct',
    region: 'ZH',
    direction: 'direct',
    titleKey: 'regionalPresets.region.zh.title',
    descriptionKey: 'regionalPresets.region.zh.description',
    sources: [
      { name: 'SagerNet/sing-geosite: geolocation-cn', url: SOURCE_URLS.cnGeosite },
      { name: 'SagerNet/sing-geoip: cn', url: SOURCE_URLS.cnGeoip },
      { name: 'SagerNet/sing-geosite: geolocation-!cn', url: SOURCE_URLS.nonCnGeosite },
    ],
    apply: applyZhDirect,
  },
  {
    id: 'zh-proxy',
    region: 'ZH',
    direction: 'proxy',
    titleKey: 'regionalPresets.region.zh.title',
    descriptionKey: 'regionalPresets.region.zh.description',
    sources: [
      { name: 'SagerNet/sing-geosite: geolocation-cn', url: SOURCE_URLS.cnGeosite },
      { name: 'SagerNet/sing-geoip: cn', url: SOURCE_URLS.cnGeoip },
      { name: 'SagerNet/sing-geosite: geolocation-!cn', url: SOURCE_URLS.nonCnGeosite },
    ],
    apply: applyZhProxy,
  },
]

const presetByRegionDirection = (region: PresetRegion, direction: PresetDirection) => {
  const preset = routingDnsPresetCatalog.find(item => item.region === region && item.direction === direction)
  if (!preset) throw new Error(`unknown preset for ${region}/${direction}`)
  return preset
}

const defaultState = (region: PresetRegion): RegionalPresetState => ({
  region,
  enabled: false,
  direction: 'direct',
  exceptions: [],
})

const extractExceptions = (ruleSet: any): string[] => {
  const rules = Array.isArray(ruleSet?.rules) ? ruleSet.rules : []
  return unique(rules.flatMap((rule: any) => asArray(rule?.domain_suffix)))
}

const detectRegionState = (config: Config, region: PresetRegion): RegionalPresetState => {
  ensureConfigShape(config)
  const ruleSets = config.route.rule_set as any[]
  const regionTags = new Set(regionRuleSetTags(region))
  const matchingRuleSets = ruleSets.filter(item => regionTags.has(String(item?.tag ?? '')))

  if (matchingRuleSets.length === 0) return defaultState(region)

  const direction = matchingRuleSets.some(item => directionFromTag(String(item?.tag ?? ''), region) === 'proxy')
    ? 'proxy'
    : 'direct'
  const exceptionRuleSet = matchingRuleSets.find(item => String(item?.tag ?? '') === presetRuleSetTag(region, direction, 'exceptions')) ??
    matchingRuleSets.find(item => String(item?.tag ?? '').endsWith('-exceptions'))

  return {
    region,
    enabled: true,
    direction,
    exceptions: exceptionRuleSet ? extractExceptions(exceptionRuleSet) : [],
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
  addCommonDnsServers(after, options, changes)
  presetByRegionDirection(state.region, state.direction).apply(after, {
    ...options,
    direction: state.direction,
    exceptions: normalizedExceptions(state.exceptions),
  }, changes)
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

  const removed = existingManagedItemsForRegion(before, state.region).routeRuleSets
    .filter(item => !desired.routeRuleSets.some(next => next.tag === item.tag))
  preview.willRemove.push(...removed.map(item => `rule-set ${item.tag}`))

  if (state.direction === 'proxy') {
    preview.securityWarnings.push('regionalPresets.security.dnsLeakRisk')
    preview.securityWarnings.push('regionalPresets.security.routeExposureRisk')
  }

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
  if (!options.proxyOutbound || !options.directOutbound) {
    throw new Error('proxyOutbound and directOutbound are required')
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
    addCommonDnsServers(config, options, changes)
  }

  for (const state of states) {
    if (!state.enabled) continue
    presetByRegionDirection(state.region, state.direction).apply(config, {
      ...options,
      direction: state.direction,
      exceptions: normalizedExceptions(state.exceptions),
    }, changes)
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
  if (!options.proxyOutbound || !options.directOutbound) {
    throw new Error('proxyOutbound and directOutbound are required')
  }

  const preset = routingDnsPresetCatalog.find(item => item.id === presetId)
  if (!preset) throw new Error(`unknown preset: ${presetId}`)

  const detected = detectPresetState(input)
  const state: RegionalPresetState = {
    region: preset.region,
    enabled: true,
    direction: preset.direction,
    exceptions: normalizedExceptions(options.exceptions ?? []),
  }
  const result = applyPresets(
    input,
    preset.region === 'RU' ? state : detected.ru,
    preset.region === 'ZH' ? state : detected.zh,
    { proxyOutbound: options.proxyOutbound, directOutbound: options.directOutbound },
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
    const parsed = new URL(source.url)
    return parsed.protocol === 'https:' &&
      parsed.username === '' &&
      parsed.password === '' &&
      parsed.pathname.endsWith('.srs')
  }))
