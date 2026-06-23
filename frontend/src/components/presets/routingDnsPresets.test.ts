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

const state = (region: 'RU' | 'ZH', enabled: boolean, direction: 'direct' | 'proxy', exceptions: string[] = []): RegionalPresetState => ({
  region,
  enabled,
  direction,
  exceptions,
})

const options = {
  proxyOutbound: 'my-proxy',
  directOutbound: 'my-direct',
}

const hasUnknownPresetMetadata = (value: unknown): boolean => JSON.stringify(value).includes('"x-preset-')

describe('routing DNS preset catalog', () => {
  it('uses safe HTTPS SRS source URLs', () => {
    expect(validatePresetCatalogShape()).toBe(true)
    expect(routingDnsPresetCatalog.map(preset => preset.id)).toEqual([
      'ru-direct',
      'ru-proxy',
      'zh-direct',
      'zh-proxy',
    ])

    for (const preset of routingDnsPresetCatalog) {
      for (const source of preset.sources) {
        expect(source.url).toMatch(/^https:\/\//)
        expect(source.url).toContain('raw.githubusercontent.com')
        expect(source.url).not.toContain('@')
        expect(source.url).toMatch(/\.srs$/)
      }
    }
  })

  it('applies RU proxy with preset tags and keeps private RU direct', () => {
    const result = applyRoutingDnsPreset(baseConfig(), 'ru-proxy', options)

    expect(result.config.experimental.cache_file?.enabled).toBe(true)
    expect(result.config.route.rule_set.map((item: any) => item.tag)).toEqual([
      'preset-ru-proxy-blocked',
      'preset-ru-proxy-private',
    ])

    const outboundByRuleSet = Object.fromEntries(
      (result.config.route.rules as any[]).map(rule => [rule.rule_set?.join(','), rule.outbound]),
    )
    expect(outboundByRuleSet['preset-ru-proxy-blocked']).toBe('my-proxy')
    expect(outboundByRuleSet['preset-ru-proxy-private']).toBe('my-direct')
    expect(hasUnknownPresetMetadata(result.config)).toBe(false)
  })

  it('applies RU direct with blocked and private RU direct', () => {
    const { config } = applyRoutingDnsPreset(baseConfig(), 'ru-direct', options)

    const outboundByRuleSet = Object.fromEntries(
      (config.route.rules as any[]).map(rule => [rule.rule_set?.join(','), rule.outbound]),
    )
    expect(outboundByRuleSet['preset-ru-direct-blocked']).toBe('my-direct')
    expect(outboundByRuleSet['preset-ru-direct-private']).toBe('my-direct')
  })

  it('applies ZH direct and ZH proxy in opposite directions', () => {
    const zhDirect = applyRoutingDnsPreset(baseConfig(), 'zh-direct', options).config
    const zhProxy = applyRoutingDnsPreset(baseConfig(), 'zh-proxy', options).config

    const directOutboundByRuleSet = (zhDirect.route.rules as any[]).map(rule => ({ rule_set: rule.rule_set, outbound: rule.outbound }))
    expect(directOutboundByRuleSet).toContainEqual({
      rule_set: ['preset-zh-direct-geosite-cn', 'preset-zh-direct-geoip-cn'],
      outbound: 'my-direct',
    })
    expect(directOutboundByRuleSet).toContainEqual({
      rule_set: ['preset-zh-direct-geosite-non-cn'],
      outbound: 'my-proxy',
    })

    const proxyOutboundByRuleSet = (zhProxy.route.rules as any[]).map(rule => ({ rule_set: rule.rule_set, outbound: rule.outbound }))
    expect(proxyOutboundByRuleSet).toContainEqual({
      rule_set: ['preset-zh-proxy-geosite-cn', 'preset-zh-proxy-geoip-cn'],
      outbound: 'my-proxy',
    })
    expect(proxyOutboundByRuleSet).toContainEqual({
      rule_set: ['preset-zh-proxy-geosite-non-cn'],
      outbound: 'my-direct',
    })
    expect(hasUnknownPresetMetadata(zhDirect)).toBe(false)
    expect(hasUnknownPresetMetadata(zhProxy)).toBe(false)
  })

  it('throws for an unknown preset id instead of silently doing nothing', () => {
    expect(() => applyRoutingDnsPreset(baseConfig(), 'does-not-exist', options)).toThrow(/unknown preset/)
  })

  it('detects preset state from deterministic tags', () => {
    const first = applyPresets(baseConfig(), state('RU', true, 'proxy', ['Example.RU']), state('ZH', true, 'direct'), options).config
    const detected = detectPresetState(first)

    expect(detected.ru).toEqual({
      region: 'RU',
      enabled: true,
      direction: 'proxy',
      exceptions: ['example.ru'],
    })
    expect(detected.zh).toMatchObject({
      region: 'ZH',
      enabled: true,
      direction: 'direct',
    })
  })

  it('preserves custom items when applying presets', () => {
    const cfg = baseConfig()
    cfg.route.rule_set.push({ type: 'remote', tag: 'custom-rs', format: 'binary', url: 'https://example.test/custom.srs' } as any)
    cfg.route.rules.push({ rule_set: ['custom-rs'], outbound: 'custom-out' } as any)
    cfg.dns.servers.push({ type: 'udp', tag: 'custom-dns', server: '9.9.9.9' } as any)
    cfg.dns.rules.push({ action: 'route', domain_suffix: ['custom.test'], server: 'custom-dns' } as any)

    const result = applyPresets(cfg, state('RU', true, 'direct'), state('ZH', false, 'direct'), options).config

    expect(result.route.rule_set).toContainEqual(expect.objectContaining({ tag: 'custom-rs' }))
    expect(result.route.rules).toContainEqual(expect.objectContaining({ rule_set: ['custom-rs'], outbound: 'custom-out' }))
    expect(result.dns.servers).toContainEqual(expect.objectContaining({ tag: 'custom-dns' }))
    expect(result.dns.rules).toContainEqual(expect.objectContaining({ domain_suffix: ['custom.test'], server: 'custom-dns' }))
  })

  it('disables a preset by removing only managed items', () => {
    const withPreset = applyPresets(baseConfig(), state('RU', true, 'proxy'), state('ZH', false, 'direct'), options).config
    withPreset.route.rules.push({ domain_suffix: ['keep.test'], outbound: 'custom-out' } as any)

    const disabled = applyPresets(withPreset, state('RU', false, 'direct'), state('ZH', false, 'direct'), options).config

    expect(disabled.route.rule_set.some((item: any) => String(item.tag).startsWith('preset-ru-'))).toBe(false)
    expect(disabled.route.rules.some((item: any) => item.rule_set?.some((tag: string) => tag.startsWith('preset-ru-')))).toBe(false)
    expect(disabled.route.rules).toContainEqual(expect.objectContaining({ domain_suffix: ['keep.test'], outbound: 'custom-out' }))
  })

  it('computes preview groups and proxy security warnings', () => {
    const cfg = baseConfig()
    cfg.route.rules.push({ domain_suffix: ['custom.test'], outbound: 'custom-out' } as any)
    cfg.dns.rules.push({ domain_suffix: ['custom.test'], server: 'custom-dns' } as any)

    const preview = computePreview(cfg, state('RU', true, 'proxy'), state('ZH', false, 'direct'), options)

    expect(preview.ru.willAdd).toContain('rule-set preset-ru-proxy-blocked')
    expect(preview.ru.willKeep).toContain('1 custom route rule(s)')
    expect(preview.ru.willKeep).toContain('1 custom DNS rule(s)')
    expect(preview.ru.securityWarnings).toContain('regionalPresets.security.dnsLeakRisk')
    expect(preview.zh.securityWarnings).toHaveLength(0)
  })

  it('adds exception rule sets and routes exceptions to the opposite direction', () => {
    const { config, preview } = applyPresets(
      baseConfig(),
      state('RU', true, 'proxy', ['Example.RU']),
      state('ZH', false, 'direct'),
      options,
    )

    expect(config.route.rule_set).toContainEqual(expect.objectContaining({
      type: 'inline',
      tag: 'preset-ru-proxy-exceptions',
      rules: [{ domain_suffix: ['example.ru'] }],
    }))
    expect(config.route.rules).toContainEqual(expect.objectContaining({
      rule_set: ['preset-ru-proxy-exceptions'],
      outbound: 'my-direct',
    }))
    expect(config.dns.rules).toContainEqual(expect.objectContaining({
      rule_set: ['preset-ru-proxy-exceptions'],
      server: 'preset-dns-direct',
    }))
    expect(preview.ru.willAdd).toContain('rule-set preset-ru-proxy-exceptions')
  })

  it('identifies preset-managed items without metadata fields', () => {
    expect(isPresetManagedItem({ tag: 'preset-zh-direct-geosite-cn' })).toBe(true)
    expect(isPresetManagedItem({ tag: 'preset-dns-direct' })).toBe(true)
    expect(isPresetManagedItem({ rule_set: ['preset-ru-proxy-blocked'] })).toBe(true)
    expect(isPresetManagedItem({ tag: 'custom-rs' })).toBe(false)
  })
})
