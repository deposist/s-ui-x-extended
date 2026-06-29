import { describe, expect, it } from 'vitest'

import { InTypes } from '@/types/inbounds'
import {
  inboundAllowedTlsTemplateKinds,
  isInboundTlsTemplateCompatible,
  tlsTemplateKind,
} from './tlsCompatibility'

const tlsTemplate = { id: 1, name: 'TLS', server: {} }
const realityTemplate = { id: 2, name: 'Reality', server: { reality: { enabled: true } } }

describe('inbound TLS template compatibility', () => {
  it('detects ordinary TLS and Reality templates', () => {
    expect(tlsTemplateKind(tlsTemplate)).toBe('tls')
    expect(tlsTemplateKind(realityTemplate)).toBe('reality')
  })

  it('allows Reality templates only for inbound protocols that support them', () => {
    expect(inboundAllowedTlsTemplateKinds(InTypes.VLESS)).toEqual(['tls', 'reality'])
    expect(inboundAllowedTlsTemplateKinds(InTypes.Trojan)).toEqual(['tls', 'reality'])

    expect(inboundAllowedTlsTemplateKinds(InTypes.VMess)).toEqual(['tls'])
    expect(inboundAllowedTlsTemplateKinds(InTypes.HTTP)).toEqual(['tls'])
    expect(inboundAllowedTlsTemplateKinds(InTypes.Hysteria2)).toEqual(['tls'])
    expect(inboundAllowedTlsTemplateKinds(InTypes.AnyTls)).toEqual(['tls'])
  })

  it('rejects unsupported Reality template choices for inbound protocols', () => {
    expect(isInboundTlsTemplateCompatible(InTypes.VLESS, realityTemplate)).toBe(true)
    expect(isInboundTlsTemplateCompatible(InTypes.VMess, realityTemplate)).toBe(false)
    expect(isInboundTlsTemplateCompatible(InTypes.Hysteria2, realityTemplate)).toBe(false)
    expect(isInboundTlsTemplateCompatible(InTypes.VMess, tlsTemplate)).toBe(true)
  })
})
