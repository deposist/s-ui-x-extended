import { reactive } from 'vue'
import { describe, expect, it } from 'vitest'
import { OutTypes } from '@/types/outbounds'
import { outboundRecommendationSpecs } from './defaultRecommendations'
import {
  applyRecommendation,
  applyRecommendations,
  getByPath,
  isEmptyRecommendationValue,
  resolveRecommendations,
  setByPath,
  type RecommendationSpec,
} from './recommendations'

describe('recommendation helpers', () => {
  it('detects missing and empty values without treating false or zero as empty', () => {
    expect(isEmptyRecommendationValue(undefined)).toBe(true)
    expect(isEmptyRecommendationValue(null)).toBe(true)
    expect(isEmptyRecommendationValue('')).toBe(true)
    expect(isEmptyRecommendationValue([])).toBe(true)
    expect(isEmptyRecommendationValue({})).toBe(true)

    expect(isEmptyRecommendationValue(false)).toBe(false)
    expect(isEmptyRecommendationValue(0)).toBe(false)
    expect(isEmptyRecommendationValue('0')).toBe(false)
    expect(isEmptyRecommendationValue([false])).toBe(false)
    expect(isEmptyRecommendationValue({ enabled: false })).toBe(false)
  })

  it('sets deep paths and creates missing parents only when explicitly applied', () => {
    const model: Record<string, unknown> = {}
    setByPath(model, 'tls.server.alpn', ['h3'])

    expect(getByPath(model, 'tls.server.alpn')).toEqual(['h3'])
  })

  it('resolves recommendations without mutating the model', () => {
    const model = { tls: {} }
    const specs: RecommendationSpec<typeof model>[] = [
      { label: 'ALPN', path: 'tls.alpn', value: ['h2', 'http/1.1'] },
    ]

    const resolved = resolveRecommendations(specs, { mode: 'edit', model })

    expect(resolved).toHaveLength(1)
    expect(resolved[0].recommendedValue).toEqual(['h2', 'http/1.1'])
    expect(model).toEqual({ tls: {} })
  })

  it('deep applies recommendations while creating missing parents and preserving non-empty values', () => {
    const model: Record<string, unknown> = { existing: 'keep' }
    const specs: RecommendationSpec<typeof model>[] = [
      { label: 'ALPN', path: 'tls.server.alpn', value: ['h3', 'h2'] },
      { label: 'Existing', path: 'existing', value: 'replace' },
    ]

    applyRecommendations(model, specs, { mode: 'create', model })

    expect(getByPath(model, 'tls.server.alpn')).toEqual(['h3', 'h2'])
    expect(model.existing).toBe('keep')
  })

  it('deep clones values on apply so recommendation specs are not shared by reference', () => {
    const recommended = { alpn: ['h2'] }
    const model: Record<string, unknown> = {}
    const spec: RecommendationSpec<typeof model> = {
      label: 'TLS defaults',
      path: 'tls',
      value: recommended,
    }

    applyRecommendation(model, spec, { mode: 'edit', model })
    ;(getByPath(model, 'tls.alpn') as string[]).push('http/1.1')

    expect(recommended).toEqual({ alpn: ['h2'] })
  })

  it('clones Vue reactive recommendation values without throwing', () => {
    const recommended = reactive({ utls: { enabled: true, fingerprint: 'chrome' } })
    const model: Record<string, unknown> = {}
    const spec: RecommendationSpec<typeof model> = {
      label: 'uTLS defaults',
      path: 'tls',
      value: recommended,
    }

    const resolved = resolveRecommendations([spec], { mode: 'edit', model })
    expect(resolved[0].recommendedValue).toEqual({ utls: { enabled: true, fingerprint: 'chrome' } })

    applyRecommendation(model, spec, { mode: 'edit', model })
    expect(getByPath(model, 'tls.utls')).toEqual({ enabled: true, fingerprint: 'chrome' })
  })

  it('does not overwrite non-empty values unless forced or configured', () => {
    const model = { listen: '127.0.0.1' }
    const spec: RecommendationSpec<typeof model> = {
      label: 'Listen all',
      path: 'listen',
      value: '::',
    }

    applyRecommendation(model, spec, { mode: 'edit', model })
    expect(model.listen).toBe('127.0.0.1')

    applyRecommendation(model, spec, { mode: 'edit', model }, { force: true })
    expect(model.listen).toBe('::')
  })

  it('honors mode and capability predicates', () => {
    const model = { type: 'direct' }
    const specs: RecommendationSpec<typeof model>[] = [
      { label: 'Create only', path: 'a', value: 1, mode: 'create' },
      { label: 'Visible', path: 'b', value: 2, when: ({ type }) => type === 'direct' },
      { label: 'Hidden', path: 'c', value: 3, when: ({ type }) => type === 'vmess' },
    ]

    const resolved = resolveRecommendations(specs, { mode: 'edit', model, type: 'direct' })

    expect(resolved.map((item) => item.label)).toEqual(['Visible'])
  })

  it('filters default recommendation specs by unavailable protocol capabilities', () => {
    const model = { type: OutTypes.VLESS, tls: {} }

    const available = resolveRecommendations(outboundRecommendationSpecs, {
      mode: 'create',
      model,
      type: OutTypes.VLESS,
    })
    const unavailable = resolveRecommendations(outboundRecommendationSpecs, {
      mode: 'create',
      model,
      type: OutTypes.VLESS,
      unavailableTypes: [OutTypes.VLESS],
    })

    expect(available.map((item) => item.id)).toContain('outbound-server-port-443')
    expect(unavailable.map((item) => item.id)).not.toContain('outbound-server-port-443')
  })
})
