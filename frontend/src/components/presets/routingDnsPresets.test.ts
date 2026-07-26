import { describe, expect, it } from 'vitest'

import type { Config } from '@/types/config'
import {
  applyPresets,
  applyRoutingDnsPreset,
  computePreview,
  detectPresetState,
  isPresetManagedItem,
  routingDnsPresetCatalog,
  type RegionalPresetState,
  requiredRuleSetSources,
  validatePresetCatalogShape,
} from './routingDnsPresets'

const baseConfig = (): Config => ({
  log: {},
  dns: { servers: [], rules: [] },
  inbounds: [],
  outbounds: [],
  route: { rules: [], rule_set: [] },
  experimental: {},
})

const state = (region: 'RU' | 'ZH', enabled: boolean): RegionalPresetState => ({
  region,
  enabled,
  direction: 'direct',
  exceptions: [],
})

const options = {
  proxyOutbound: 'my-proxy',
  directOutbound: 'direct',
}

const hasUnknownPresetMetadata = (value: unknown): boolean => JSON.stringify(value).includes('"x-preset-')

const byTag = (items: any[]) => Object.fromEntries(items.map(item => [item.tag, item]))

const ruleOutboundByRuleSet = (config: Config) => Object.fromEntries(
  (config.route.rules as any[]).map(rule => [rule.rule_set?.join(','), rule.outbound]),
)

describe('routing DNS preset catalog', () => {
  it('uses safe HTTPS SRS source URLs', () => {
    expect(validatePresetCatalogShape()).toBe(true)
    expect(routingDnsPresetCatalog.map(preset => preset.id)).toEqual([
      'ru-direct',
      'zh-direct',
    ])

    for (const preset of routingDnsPresetCatalog) {
      for (const source of preset.sources) {
        expect(source.url || source.path).toBeTruthy()
        if (source.url) {
          expect(source.url).toMatch(/^https:\/\//)
          expect(source.url).not.toContain('@')
          expect(source.url).toMatch(/(\.srs|\/geosite\.dat)$/)
        }
        if (source.path) {
          expect(source.path).toBe('rulesets/geosite-ru-smart/direct-ru.srs')
        }
      }
    }
  })

  it('emits a local rule-set for every materialized source', () => {
    const sources = requiredRuleSetSources(state('RU', true), state('ZH', true))
    const ruleSetPaths = Object.fromEntries(
      sources.map(source => [source.url, `rulesets/${source.tag}.srs`]),
    )

    const { config } = applyPresets(baseConfig(), state('RU', true), state('ZH', true), {
      ...options,
      ruleSetPaths,
      requireLocalAssets: true,
    })

    expect((config.route.rule_set as any[]).length).toBeGreaterThan(0)
    for (const ruleSet of config.route.rule_set as any[]) {
      expect(ruleSet.type).toBe('local')
      expect(ruleSet.path).toMatch(/^rulesets\/.*\.srs$/)
      // A local rule-set must not keep remote-only fields: sing-box would
      // reject the unknown keys and refuse to start.
      expect(ruleSet).not.toHaveProperty('url')
      expect(ruleSet).not.toHaveProperty('download_detour')
      expect(ruleSet).not.toHaveProperty('update_interval')
    }
  })

  // The whole point of the fail-closed flow: saving a config that points at a
  // file which was never downloaded would stop sing-box from starting.
  it('refuses to build a config referencing an undownloaded source', () => {
    expect(() => applyPresets(baseConfig(), state('RU', true), state('ZH', false), {
      ...options,
      ruleSetPaths: {},
      requireLocalAssets: true,
    })).toThrow(/has not been downloaded/)
  })

  it('keeps the regional resolver scoped to its own rule', () => {
    const { config } = applyPresets(baseConfig(), state('RU', true), state('ZH', false), options)

    const doh = (config.dns.servers as any[]).find(server => server.tag === 'preset-dns-proxy')
    expect(doh.type).toBe('https')
    // An IP literal: resolving a resolver's own hostname would need the very
    // plaintext DNS this change avoids.
    expect(doh.server).toMatch(/^\d+\.\d+\.\d+\.\d+$/)

    const rulesUsingRegional = (config.dns.rules as any[])
      .filter(rule => rule.server === 'preset-dns-direct')
    expect(rulesUsingRegional.length).toBe(1)
  })

  // An unknown detour tag passes config validation but makes sing-box fail at
  // startup, so the preset must never emit one it has not verified.
  it('omits the DoH detour when the proxy outbound does not exist', () => {
    const { config } = applyPresets(baseConfig(), state('RU', true), state('ZH', false), {
      ...options,
      proxyOutbound: 'not-in-config',
    })
    const doh = (config.dns.servers as any[]).find(server => server.tag === 'preset-dns-proxy')
    expect(doh).not.toHaveProperty('detour')
  })

  it('sets the DoH detour when the proxy outbound exists', () => {
    const cfg = baseConfig()
    cfg.outbounds.push({ id: 1, type: 'socks', tag: 'upstream' } as any)

    const { config } = applyPresets(cfg, state('RU', true), state('ZH', false), {
      ...options,
      proxyOutbound: 'upstream',
    })
    const doh = (config.dns.servers as any[]).find(server => server.tag === 'preset-dns-proxy')
    expect(doh.detour).toBe('upstream')
  })

  // A final pointing at a pruned server is accepted by validation but breaks DNS
  // at runtime, so disabling the presets has to clean up both together.
  it('removes the DoH server and final when every preset is disabled', () => {
    const enabled = applyPresets(baseConfig(), state('RU', true), state('ZH', true), options).config
    expect(enabled.dns.final).toBe('preset-dns-proxy')

    const disabled = applyPresets(enabled, state('RU', false), state('ZH', false), options).config
    expect(disabled.dns.servers as any[]).toEqual([])
    expect(disabled.dns).not.toHaveProperty('final')
  })

  it('leaves a user-defined dns final alone', () => {
    const cfg = baseConfig()
    cfg.dns.final = 'my-own-resolver'

    const { config } = applyPresets(cfg, state('RU', false), state('ZH', false), options)
    expect(config.dns.final).toBe('my-own-resolver')
  })

  it('lists only the sources the enabled regions need', () => {
    expect(requiredRuleSetSources(state('RU', false), state('ZH', false))).toEqual([])

    const ruOnly = requiredRuleSetSources(state('RU', true), state('ZH', false))
    expect(ruOnly.length).toBe(2)
    for (const source of ruOnly) {
      expect(source.url).toMatch(/^https:\/\/.*\.srs$/)
      expect(source.tag).toBeTruthy()
    }
    // Tags name the files on disk, so collisions would overwrite each other.
    const both = requiredRuleSetSources(state('RU', true), state('ZH', true))
    expect(new Set(both.map(source => source.tag)).size).toBe(both.length)
  })

  it('applies RU country rule sets directly and leaves other traffic untouched', () => {
    const result = applyRoutingDnsPreset(baseConfig(), 'ru-direct', options)
    const ruleSets = byTag(result.config.route.rule_set as any[])

    expect(result.config.experimental.cache_file?.enabled).toBe(true)
    expect(Object.keys(ruleSets)).toEqual([
      'preset-ru-direct-geosite',
      'preset-ru-direct-geoip',
    ])
    expect(ruleSets['preset-ru-direct-geosite']).toMatchObject({
      type: 'remote',
      format: 'binary',
      url: 'https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-geolocation-ru.srs',
      download_detour: 'direct',
    })
    expect(ruleSets['preset-ru-direct-geoip']).toMatchObject({
      type: 'remote',
      format: 'binary',
      url: 'https://raw.githubusercontent.com/runetfreedom/russia-v2ray-rules-dat/release/sing-box/rule-set-geoip/geoip-ru.srs',
      download_detour: 'direct',
    })

    expect(ruleOutboundByRuleSet(result.config)['preset-ru-direct-geosite,preset-ru-direct-geoip']).toBe('direct')
    // Route final stays untouched: the preset only claims regional traffic.
    expect(result.config.route).not.toHaveProperty('final')
    // DNS final is now the preset's DoH resolver, so non-regional lookups no
    // longer travel over the regional server's plaintext UDP 53.
    expect(result.config.dns.final).toBe('preset-dns-proxy')
    expect(hasUnknownPresetMetadata(result.config)).toBe(false)
  })

  it('applies ZH country rule sets directly and leaves non-CN traffic untouched', () => {
    const { config } = applyRoutingDnsPreset(baseConfig(), 'zh-direct', {
      directOutbound: 'direct',
    })
    const ruleSets = byTag(config.route.rule_set as any[])

    expect(Object.keys(ruleSets)).toEqual([
      'preset-zh-direct-geosite',
      'preset-zh-direct-geoip',
    ])
    expect(ruleSets['preset-zh-direct-geosite']).toMatchObject({
      url: 'https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-geolocation-cn.srs',
      download_detour: 'direct',
    })
    expect(ruleSets['preset-zh-direct-geoip']).toMatchObject({
      url: 'https://raw.githubusercontent.com/SagerNet/sing-geoip/rule-set/geoip-cn.srs',
      download_detour: 'direct',
    })
    expect(ruleOutboundByRuleSet(config)['preset-zh-direct-geosite,preset-zh-direct-geoip']).toBe('direct')
    expect((config.route.rules as any[]).some(rule => rule.rule_set?.includes('preset-zh-direct-geosite-non-cn'))).toBe(false)
  })

  it('routes country DNS domains to a regional resolver and the rest over DoH', () => {
    const { config } = applyPresets(
      baseConfig(),
      state('RU', true),
      state('ZH', true),
      { directOutbound: 'direct' },
    )

    expect(config.dns.servers).toContainEqual(expect.objectContaining({
      type: 'udp',
      tag: 'preset-dns-direct',
      server: '223.5.5.5',
      server_port: 53,
    }))
    expect((config.dns.servers as any[]).find(server => server.tag === 'preset-dns-direct')).not.toHaveProperty('detour')
    expect(config.dns.final).toBe('preset-dns-proxy')
    expect(config.dns.rules).toContainEqual(expect.objectContaining({
      action: 'route',
      rule_set: ['preset-ru-direct-geosite'],
      server: 'preset-dns-direct',
    }))
    expect(config.dns.rules).toContainEqual(expect.objectContaining({
      action: 'route',
      rule_set: ['preset-zh-direct-geosite'],
      server: 'preset-dns-direct',
    }))
  })

  it('removes stale DNS detour when reapplying presets with the empty built-in direct outbound', () => {
    const cfg = baseConfig()
    cfg.outbounds.push({ id: 1, type: 'direct', tag: 'direct' } as any)
    cfg.dns.servers.push({
      type: 'udp',
      tag: 'preset-dns-direct',
      server: '223.5.5.5',
      server_port: 53,
      detour: 'direct',
    } as any)

    const { config } = applyRoutingDnsPreset(cfg, 'ru-direct', {
      directOutbound: 'direct',
    })

    const directDns = (config.dns.servers as any[]).find(server => server.tag === 'preset-dns-direct')
    expect(directDns).not.toHaveProperty('detour')
  })

  it('keeps DNS detour for a configured direct outbound', () => {
    const cfg = baseConfig()
    cfg.outbounds.push({ id: 1, type: 'direct', tag: 'direct', default_interface: 'eth0' } as any)

    const { config } = applyRoutingDnsPreset(cfg, 'ru-direct', {
      directOutbound: 'direct',
    })

    const directDns = (config.dns.servers as any[]).find(server => server.tag === 'preset-dns-direct')
    expect(directDns).toEqual(expect.objectContaining({ detour: 'direct' }))
  })

  it('keeps DNS detour for a custom direct outbound tag', () => {
    const cfg = baseConfig()
    cfg.outbounds.push({ id: 1, type: 'direct', tag: 'wan-direct' } as any)

    const { config } = applyRoutingDnsPreset(cfg, 'ru-direct', {
      directOutbound: 'wan-direct',
    })

    const directDns = (config.dns.servers as any[]).find(server => server.tag === 'preset-dns-direct')
    expect(directDns).toEqual(expect.objectContaining({ detour: 'wan-direct' }))
  })

  it('throws for an unknown preset id instead of silently doing nothing', () => {
    expect(() => applyRoutingDnsPreset(baseConfig(), 'does-not-exist', { directOutbound: 'direct' })).toThrow(/unknown preset/)
  })

  it('detects current and legacy preset state from deterministic tags', () => {
    const first = applyPresets(baseConfig(), state('RU', true), state('ZH', true), options).config
    const detected = detectPresetState(first)

    expect(detected.ru).toEqual({
      region: 'RU',
      enabled: true,
      direction: 'direct',
      exceptions: [],
    })
    expect(detected.zh).toEqual({
      region: 'ZH',
      enabled: true,
      direction: 'direct',
      exceptions: [],
    })

    const legacy = baseConfig()
    legacy.route.rule_set.push({ type: 'remote', tag: 'preset-ru-proxy-blocked' } as any)
    expect(detectPresetState(legacy).ru.enabled).toBe(true)
  })

  it('preserves custom items when applying presets', () => {
    const cfg = baseConfig()
    cfg.route.rule_set.push({ type: 'remote', tag: 'custom-rs', format: 'binary', url: 'https://example.test/custom.srs' } as any)
    cfg.route.rules.push({ rule_set: ['custom-rs'], outbound: 'custom-out' } as any)
    cfg.dns.servers.push({ type: 'udp', tag: 'custom-dns', server: '9.9.9.9' } as any)
    cfg.dns.rules.push({ action: 'route', domain_suffix: ['custom.test'], server: 'custom-dns' } as any)

    const result = applyPresets(cfg, state('RU', true), state('ZH', false), options).config

    expect(result.route.rule_set).toContainEqual(expect.objectContaining({ tag: 'custom-rs' }))
    expect(result.route.rules).toContainEqual(expect.objectContaining({ rule_set: ['custom-rs'], outbound: 'custom-out' }))
    expect(result.dns.servers).toContainEqual(expect.objectContaining({ tag: 'custom-dns' }))
    expect(result.dns.rules).toContainEqual(expect.objectContaining({ domain_suffix: ['custom.test'], server: 'custom-dns' }))
  })

  it('disables a preset by removing current and legacy managed items only', () => {
    const withPreset = applyPresets(baseConfig(), state('RU', true), state('ZH', false), options).config
    withPreset.route.rule_set.push({ type: 'remote', tag: 'preset-ru-proxy-blocked' } as any)
    withPreset.route.rules.push({ rule_set: ['preset-ru-proxy-blocked'], outbound: 'my-proxy' } as any)
    withPreset.route.rules.push({ domain_suffix: ['keep.test'], outbound: 'custom-out' } as any)

    const disabled = applyPresets(withPreset, state('RU', false), state('ZH', false), options).config

    expect(disabled.route.rule_set.some((item: any) => String(item.tag).startsWith('preset-ru-'))).toBe(false)
    expect(disabled.route.rules.some((item: any) => item.rule_set?.some((tag: string) => tag.startsWith('preset-ru-')))).toBe(false)
    expect(disabled.route.rules).toContainEqual(expect.objectContaining({ domain_suffix: ['keep.test'], outbound: 'custom-out' }))
  })

  it('computes preview groups without proxy-route warnings', () => {
    const cfg = baseConfig()
    cfg.route.rules.push({ domain_suffix: ['custom.test'], outbound: 'custom-out' } as any)
    cfg.dns.rules.push({ domain_suffix: ['custom.test'], server: 'custom-dns' } as any)

    const preview = computePreview(cfg, state('RU', true), state('ZH', false), options)

    expect(preview.ru.willAdd).toContain('rule-set preset-ru-direct-geosite')
    expect(preview.ru.willAdd).toContain('rule-set preset-ru-direct-geoip')
    expect(preview.ru.willKeep).toContain('1 custom route rule(s)')
    expect(preview.ru.willKeep).toContain('1 custom DNS rule(s)')
    expect(preview.ru.securityWarnings).toHaveLength(0)
    expect(preview.zh.securityWarnings).toHaveLength(0)
  })

  it('identifies preset-managed items without metadata fields', () => {
    expect(isPresetManagedItem({ tag: 'preset-zh-direct-geosite' })).toBe(true)
    expect(isPresetManagedItem({ tag: 'preset-dns-direct' })).toBe(true)
    expect(isPresetManagedItem({ rule_set: ['preset-ru-direct-geoip'] })).toBe(true)
    expect(isPresetManagedItem({ tag: 'custom-rs' })).toBe(false)
  })
})
