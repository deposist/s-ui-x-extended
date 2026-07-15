import { describe, expect, it } from 'vitest'

import {
  AWG_RU_RULESET_TAG,
  applyAWGRuDirect,
  applyAWGRuDirectState,
  computeAWGRuDirectPreview,
  detectAWGRuDirect,
  removeAWGRuDirect,
} from './awgRuDirectPreset'
import type { Config } from '@/types/config'

const emptyConfig = (): Config => ({} as Config)

describe('awg ru-direct preset', () => {
  it('adds the rule-set and an inbound-scoped rule', () => {
    const config = emptyConfig()
    const changes = applyAWGRuDirect(config, 'awg-endpoint', 'direct')
    expect(changes).toHaveLength(2)
    expect((config.route!.rule_set as any[]).map(r => r.tag)).toContain(AWG_RU_RULESET_TAG)
    const rule = (config.route!.rules as any[])[0]
    expect(rule.inbound).toEqual(['awg-endpoint'])
    expect(rule.rule_set).toEqual([AWG_RU_RULESET_TAG])
    expect(rule.outbound).toBe('direct')
  })

  it('rule-set uses a safe https .srs source', () => {
    const config = emptyConfig()
    applyAWGRuDirect(config, 'awg-endpoint', 'direct')
    const ruleSet = (config.route!.rule_set as any[])[0]
    const url = new URL(ruleSet.url)
    expect(url.protocol).toBe('https:')
    expect(url.pathname.endsWith('.srs')).toBe(true)
    expect(ruleSet.format).toBe('binary')
  })

  it('is idempotent', () => {
    const config = emptyConfig()
    applyAWGRuDirect(config, 'awg-endpoint', 'direct')
    const changes = applyAWGRuDirect(config, 'awg-endpoint', 'direct')
    expect(changes).toEqual([])
    expect((config.route!.rules as any[]).length).toBe(1)
    expect((config.route!.rule_set as any[]).length).toBe(1)
  })

  it('prepends the rule so it wins over broader regional rules', () => {
    const config = {
      route: {
        rules: [{ rule_set: ['other'], outbound: 'proxy' }],
        rule_set: [],
      },
    } as unknown as Config
    applyAWGRuDirect(config, 'awg-endpoint', 'direct')
    expect((config.route!.rules as any[])[0].inbound).toEqual(['awg-endpoint'])
  })

  it('supports two endpoints sharing one rule-set', () => {
    const config = emptyConfig()
    applyAWGRuDirect(config, 'awg-1', 'direct')
    applyAWGRuDirect(config, 'awg-2', 'direct')
    expect((config.route!.rule_set as any[]).length).toBe(1)
    expect(detectAWGRuDirect(config)).toEqual(['awg-1', 'awg-2'])
  })

  it('removes the rule and drops the rule-set when unused', () => {
    const config = emptyConfig()
    applyAWGRuDirect(config, 'awg-1', 'direct')
    applyAWGRuDirect(config, 'awg-2', 'direct')
    removeAWGRuDirect(config, 'awg-1')
    expect(detectAWGRuDirect(config)).toEqual(['awg-2'])
    expect((config.route!.rule_set as any[]).length).toBe(1)
    removeAWGRuDirect(config, 'awg-2')
    expect(detectAWGRuDirect(config)).toEqual([])
    expect((config.route!.rule_set as any[]).length).toBe(0)
  })

  it('keeps unrelated rules and rule-sets intact', () => {
    const config = {
      route: {
        rules: [{ rule_set: ['custom-rs'], outbound: 'proxy' }],
        rule_set: [{ type: 'local', tag: 'custom-rs', format: 'binary', path: 'x.srs' }],
      },
    } as unknown as Config
    applyAWGRuDirect(config, 'awg-endpoint', 'direct')
    removeAWGRuDirect(config, 'awg-endpoint')
    expect((config.route!.rules as any[]).length).toBe(1)
    expect((config.route!.rules as any[])[0].rule_set).toEqual(['custom-rs'])
    expect((config.route!.rule_set as any[]).map((r: any) => r.tag)).toEqual(['custom-rs'])
  })

  it('applyAWGRuDirectState moves the rule between endpoints', () => {
    const config = emptyConfig()
    applyAWGRuDirectState(config, { enabled: true, endpointTag: 'awg-1' }, 'direct')
    expect(detectAWGRuDirect(config)).toEqual(['awg-1'])
    applyAWGRuDirectState(config, { enabled: true, endpointTag: 'awg-2' }, 'direct')
    expect(detectAWGRuDirect(config)).toEqual(['awg-2'])
    applyAWGRuDirectState(config, { enabled: false, endpointTag: '' }, 'direct')
    expect(detectAWGRuDirect(config)).toEqual([])
    expect((config.route!.rule_set as any[]).length).toBe(0)
  })

  it('preview reflects add, keep and remove', () => {
    const config = emptyConfig()
    let preview = computeAWGRuDirectPreview(config, { enabled: true, endpointTag: 'awg-1' }, 'direct')
    expect(preview.willAdd.length).toBe(2)
    expect(preview.willRemove).toEqual([])

    applyAWGRuDirect(config, 'awg-1', 'direct')
    preview = computeAWGRuDirectPreview(config, { enabled: true, endpointTag: 'awg-1' }, 'direct')
    expect(preview.willKeep.length).toBe(1)
    expect(preview.willAdd).toEqual([])

    preview = computeAWGRuDirectPreview(config, { enabled: false, endpointTag: '' }, 'direct')
    expect(preview.willRemove).toEqual(['route rule awg-1'])
  })

  it('rejects empty arguments', () => {
    expect(() => applyAWGRuDirect(emptyConfig(), '', 'direct')).toThrow()
    expect(() => applyAWGRuDirect(emptyConfig(), 'awg', '')).toThrow()
  })
})
