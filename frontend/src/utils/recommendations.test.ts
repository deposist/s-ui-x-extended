import { reactive } from 'vue'
import { describe, expect, it } from 'vitest'
import { DnsTypes } from '@/types/dns'
import { EpTypes } from '@/types/endpoints'
import { InTypes } from '@/types/inbounds'
import { OutTypes } from '@/types/outbounds'
import { SrvTypes } from '@/types/services'
import {
  applyDnsRuleRecommendedValues,
  applyDnsServerRecommendedValues,
  applyEndpointRecommendedValues,
  applyInboundRecommendedValues,
  applyOutboundRecommendedValues,
  applyRouteRuleRecommendedValues,
  applyServiceRecommendedValues,
  applyTlsRecommendedValues,
  applyVlessInboundRecommendedValues,
  dnsRuleFieldHints,
  dnsServerFieldHintsForType,
  endpointFieldHintsForType,
  hasDnsRuleRecommendedPreset,
  hasDnsServerRecommendedPreset,
  hasEndpointRecommendedPreset,
  hasInboundRecommendedPreset,
  hasOutboundRecommendedPreset,
  hasRouteRuleRecommendedPreset,
  hasServiceRecommendedPreset,
  hasTlsRecommendedPreset,
  outboundFieldHintsForType,
  outboundRecommendationSpecs,
  routeRuleFieldHints,
  serviceFieldHintsForType,
  tlsFieldHintsForType,
} from './defaultRecommendations'
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

  it('applies explicit remaining-inbound presets only for supported protocols', () => {
    const vmess: any = {
      type: InTypes.VMess,
      listen_port: 23456,
      proxy_protocol: true,
      out_json: {},
    }

    applyInboundRecommendedValues(vmess)

    expect(vmess.listen).toBe('::')
    expect(vmess.sniff).toBe(true)
    expect(vmess.sniff_override_destination).toBe(true)
    expect(vmess.sniff_timeout).toBe('300ms')
    expect(vmess.proxy_protocol).toBeUndefined()
    expect(vmess.out_json.security).toBe('auto')
    expect(vmess.out_json.packet_encoding).toBe('xudp')
    expect(vmess.out_json.global_padding).toBe(true)
    expect(vmess.out_json.authenticated_length).toBe(true)

    const shadowTls: any = { type: InTypes.ShadowTLS, listen_port: 23457 }
    applyInboundRecommendedValues(shadowTls)
    expect(shadowTls.listen).toBeUndefined()
    expect(shadowTls.sniff).toBeUndefined()

    expect(hasInboundRecommendedPreset(InTypes.ShadowTLS)).toBe(false)
    expect(hasInboundRecommendedPreset(InTypes.MTProxy)).toBe(false)
    expect(hasInboundRecommendedPreset(InTypes.Tun)).toBe(false)
    expect(hasInboundRecommendedPreset(InTypes.Bond)).toBe(false)
    expect(hasInboundRecommendedPreset(InTypes.CoreFailover)).toBe(false)
    expect(hasInboundRecommendedPreset(InTypes.VMess)).toBe(true)
  })

  it('applies explicit outbound presets only for supported protocols', () => {
    const vless: any = {
      type: OutTypes.VLESS,
      tag: 'keep-me',
      server: 'edge.example.com',
      server_port: 0,
      packet_encoding: '',
      tls: {},
    }

    applyOutboundRecommendedValues(vless)

    expect(vless.tag).toBe('keep-me')
    expect(vless.server).toBe('edge.example.com')
    expect(vless.server_port).toBe(443)
    expect(vless.packet_encoding).toBe('xudp')
    expect(vless.tls.enabled).toBeUndefined()
    expect(vless.tls.min_version).toBeUndefined()

    const hysteria2: any = { type: OutTypes.Hysteria2, tls: {} }
    applyOutboundRecommendedValues(hysteria2)
    expect(hysteria2.server_port).toBe(443)
    expect(hysteria2.tls.enabled).toBe(true)
    expect(hysteria2.tls.min_version).toBe('1.3')
    expect(hysteria2.tls.utls).toEqual({ enabled: true, fingerprint: 'chrome' })
    expect(hysteria2.up_mbps).toBe(100)
    expect(hysteria2.down_mbps).toBe(100)

    const vmess: any = { type: OutTypes.VMess, tls: {}, security: 'legacy' }
    applyOutboundRecommendedValues(vmess)
    expect(vmess.server_port).toBe(443)
    expect(vmess.security).toBe('auto')
    expect(vmess.packet_encoding).toBe('xudp')
    expect(vmess.global_padding).toBe(true)
    expect(vmess.authenticated_length).toBe(true)

    const direct: any = { type: OutTypes.Direct, server_port: 0, tls: {} }
    applyOutboundRecommendedValues(direct)
    expect(direct).toEqual({ type: OutTypes.Direct, server_port: 0, tls: {} })

    expect(hasOutboundRecommendedPreset(OutTypes.Direct)).toBe(false)
    expect(hasOutboundRecommendedPreset(OutTypes.Shadowsocks)).toBe(false)
    expect(hasOutboundRecommendedPreset(OutTypes.SSH)).toBe(false)
    expect(hasOutboundRecommendedPreset(OutTypes.VLESS)).toBe(true)
    expect(hasOutboundRecommendedPreset(OutTypes.VMess)).toBe(true)
    expect(hasOutboundRecommendedPreset(OutTypes.OpenVPN)).toBe(true)
  })

  it('uses outbound-specific field-hint keys for outbound dial and shared sections', () => {
    const hints = outboundFieldHintsForType(OutTypes.VLESS)

    expect(hints.dial_options).toBe('types.outbound.hint.dial_options')
    expect(hints.transport_enable).toBe('types.outbound.hint.transport_enable')
    expect(hints.out_multiplex_enable).toBe('types.outbound.hint.out_multiplex_enable')
    expect(hints.tls_enable).toBe('types.outbound.hint.tls_enable')
  })

  it('applies explicit service, endpoint, TLS, DNS and rule presets safely', () => {
    const service: any = { type: SrvTypes.DERP, listen_port: 3478 }
    applyServiceRecommendedValues(service)
    expect(service.listen).toBe('::')

    const profiler: any = { type: SrvTypes.Profiler }
    applyServiceRecommendedValues(profiler)
    expect(profiler.listen).toBeUndefined()
    expect(hasServiceRecommendedPreset(SrvTypes.Profiler)).toBe(false)
    expect(serviceFieldHintsForType(SrvTypes.DERP).listen).toBe('types.service.hint.listen')

    const wg: any = { type: EpTypes.Wireguard, mtu: 0 }
    applyEndpointRecommendedValues(wg)
    expect(wg.mtu).toBe(1420)

    const vpnServer: any = { type: EpTypes.VpnServer, users: [] }
    applyEndpointRecommendedValues(vpnServer)
    expect(vpnServer.address).toBe('10.0.0.1')
    expect(vpnServer.users).toEqual([{ address: '10.0.0.2', key: '' }])
    expect(hasEndpointRecommendedPreset(EpTypes.VpnClient)).toBe(false)
    expect(endpointFieldHintsForType(EpTypes.Wireguard).mtu).toBe('types.endpoint.hint.mtu')

    const tls: any = { server: {}, client: {} }
    applyTlsRecommendedValues(tls)
    expect(tls.server.min_version).toBe('1.3')
    expect(tls.server.max_version).toBe('1.3')
    expect(tls.server.alpn).toEqual(['h3', 'h2', 'http/1.1'])
    expect(tls.client.utls).toEqual({ enabled: true, fingerprint: 'chrome' })

    const reality: any = { server: { reality: { handshake: {} } }, client: { reality: {} } }
    applyTlsRecommendedValues(reality)
    expect(reality.server.reality.handshake.server).toBeTruthy()
    expect(reality.server.reality.handshake.server_port).toBe(443)
    expect(hasTlsRecommendedPreset('reality')).toBe(true)
    expect(tlsFieldHintsForType('tls').min_version).toBe('types.tls.hint.min_version')

    const dns: any = { type: DnsTypes.HTTPS, tls: {} }
    applyDnsServerRecommendedValues(dns)
    expect(dns.server).toBe('cloudflare-dns.com')
    expect(dns.path).toBe('/dns-query')
    expect(dns.server_port).toBe(443)
    expect(dns.tls.enabled).toBe(true)
    expect(dns.tls.min_version).toBe('1.3')
    expect(hasDnsServerRecommendedPreset(DnsTypes.Local)).toBe(false)
    expect(dnsServerFieldHintsForType(DnsTypes.HTTPS).server).toBe('types.dns.hint.server')

    const dnsRule: any = { type: 'logical', action: 'route' }
    applyDnsRuleRecommendedValues(dnsRule)
    expect(dnsRule.mode).toBe('or')
    expect(dnsRule.strategy).toBe('prefer_ipv4')
    expect(hasDnsRuleRecommendedPreset()).toBe(true)
    expect(dnsRuleFieldHints().action).toBe('types.dnsRule.hint.action')

    const routeRule: any = { type: 'logical', action: 'route-options' }
    applyRouteRuleRecommendedValues(routeRule)
    expect(routeRule.mode).toBe('or')
    expect(routeRule.udp_timeout).toBe('5m')
    expect(hasRouteRuleRecommendedPreset()).toBe(true)
    expect(routeRuleFieldHints().action).toBe('types.rule.hint.action')
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
