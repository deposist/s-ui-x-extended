import { reactive } from 'vue'
import { describe, expect, it } from 'vitest'
import { InTypes } from '@/types/inbounds'
import { OutTypes } from '@/types/outbounds'
import { applyVlessInboundRecommendedValues, outboundRecommendationSpecs } from './defaultRecommendations'
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

  it('applies the explicit VLESS inbound preset without choosing TLS or transport', () => {
    const inbound: any = {
      type: InTypes.VLESS,
      listen_port: 443,
      listen: '',
      tls_id: 0,
      decryption: '',
      sniff: false,
      sniff_override_destination: false,
      proxy_protocol: true,
      proxy_protocol_accept_no_header: true,
      domain_strategy: 'prefer_ipv4',
      udp_disable_domain_unmapping: true,
      transport: { type: 'ws' },
      multiplex: { enabled: true },
      out_json: { multiplex: { enabled: true } },
    }

    applyVlessInboundRecommendedValues(inbound)

    expect(inbound.listen).toBe('::')
    expect(inbound.listen_port).toBe(443)
    expect(inbound.tls_id).toBe(0)
    expect(inbound.decryption).toBe('none')
    expect(inbound.sniff).toBe(true)
    expect(inbound.sniff_override_destination).toBe(true)
    expect(inbound.sniff_timeout).toBe('300ms')
    expect(inbound.transport).toEqual({})
    expect(inbound.out_json.packet_encoding).toBe('xudp')
    expect(inbound.proxy_protocol).toBeUndefined()
    expect(inbound.proxy_protocol_accept_no_header).toBeUndefined()
    expect(inbound.domain_strategy).toBeUndefined()
    expect(inbound.udp_disable_domain_unmapping).toBeUndefined()
    expect(inbound.multiplex).toBeUndefined()
    expect(inbound.out_json.multiplex).toBeUndefined()
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
