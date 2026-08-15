import { describe, expect, it } from 'vitest'

import {
  amneziaPresetCatalog,
  amneziaPresetById,
  applyAmneziaPreset,
  applyAmneziaTimingDefaults,
  amneziaTimingDefaults,
  detectAmneziaPreset,
} from './amneziaPresets'
import { validateAmnezia } from '@/utils/amneziaValidation'

describe('amnezia presets', () => {
  it('catalog contains balanced, mobile and custom', () => {
    expect(amneziaPresetCatalog.map(p => p.id)).toEqual(['balanced', 'mobile', 'custom'])
  })

  it('balanced matches the ADVANCED.md default distribution midpoint', () => {
    expect(amneziaPresetById('balanced').junk).toEqual({ jc: 4, jmin: 40, jmax: 90 })
  })

  it('mobile matches the ADVANCED.md mobile preset (fixed Jc=3, narrow window)', () => {
    expect(amneziaPresetById('mobile').junk).toEqual({ jc: 3, jmin: 40, jmax: 70 })
  })

  it('custom has no junk values (manual mode)', () => {
    expect(amneziaPresetById('custom').junk).toBeUndefined()
  })

  it('applying a preset writes only junk parameters and never touches headers', () => {
    const amnezia: Record<string, unknown> = {
      jc: 99, jmin: 1, jmax: 2,
      s1: 15, s2: 20,
      h1: '1000-1099', h2: 2000, h3: 3000, h4: 4000,
      i1: '<b 0x01>',
    }
    applyAmneziaPreset(amnezia, 'mobile')
    expect(amnezia).toMatchObject({
      jc: 3, jmin: 40, jmax: 70,
      s1: 15, s2: 20,
      h1: '1000-1099', h2: 2000, h3: 3000, h4: 4000,
      i1: '<b 0x01>',
    })
  })

  it('applying custom is a no-op', () => {
    const amnezia: Record<string, unknown> = { jc: 7, jmin: 11, jmax: 22 }
    applyAmneziaPreset(amnezia, 'custom')
    expect(amnezia).toEqual({ jc: 7, jmin: 11, jmax: 22 })
  })

  it('every preset with junk values passes the obfuscation validator', () => {
    for (const preset of amneziaPresetCatalog) {
      if (!preset.junk) continue
      const amnezia = {
        ...preset.junk,
        s1: 15, s2: 20, s3: 12, s4: 8,
        h1: '1000-1099', h2: 2000, h3: '3000-3099', h4: '4000-4099',
      }
      expect(validateAmnezia(amnezia), preset.id).toEqual({})
    }
  })

  it('detects the active preset from junk values', () => {
    expect(detectAmneziaPreset({ jc: 4, jmin: 40, jmax: 90 })).toBe('balanced')
    expect(detectAmneziaPreset({ jc: 3, jmin: 40, jmax: 70 })).toBe('mobile')
    expect(detectAmneziaPreset({ jc: 5, jmin: 40, jmax: 90 })).toBe('custom')
    expect(detectAmneziaPreset({})).toBe('custom')
    expect(detectAmneziaPreset(undefined)).toBe('custom')
    expect(detectAmneziaPreset(null)).toBe('custom')
  })

  it('throws on an unknown preset id', () => {
    expect(() => amneziaPresetById('nope' as never)).toThrow('unknown amnezia preset')
  })

  it('timing defaults cover the six AWG 3.0 range fields', () => {
    expect(Object.keys(amneziaTimingDefaults).sort()).toEqual([
      'content_padding_addition',
      'keepalive_timeout',
      'max_handshake_attempts',
      'reject_after_time',
      'rekey_after_time',
      'rekey_timeout',
    ])
  })

  it('applying timing defaults fills only missing fields', () => {
    const amnezia: Record<string, unknown> = { rekey_after_time: '300-400' }
    applyAmneziaTimingDefaults(amnezia)
    expect(amnezia.rekey_after_time).toBe('300-400')
    expect(amnezia.rekey_timeout).toBe('1-5')
    expect(amnezia.keepalive_timeout).toBe('5-10')
  })

  it('a profile seeded with junk + timing defaults passes the validator', () => {
    const amnezia: Record<string, unknown> = {}
    applyAmneziaPreset(amnezia, 'balanced')
    applyAmneziaTimingDefaults(amnezia)
    const wireguard = { ...amnezia, s1: 15, s2: 20, s3: 12, s4: 8, h1: '1000-1099', h2: 2000, h3: '3000-3099', h4: '4000-4099' }
    expect(validateAmnezia(wireguard)).toEqual({})
    expect(validateAmnezia(amnezia, { warp: true })).toEqual({})
  })
})
