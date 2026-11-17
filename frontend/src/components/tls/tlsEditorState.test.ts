import { afterEach, describe, expect, it, vi } from 'vitest'
import OutTLS from './OutTLS.vue'
import Acme from './Acme.vue'
import Tls from '@/layouts/modals/Tls.vue'
import { defaultInTls, defaultOutTls } from '@/types/tls'

vi.mock('@/plugins/httputil', () => ({ default: {} }))
vi.mock('@/plugins/randomUtil', () => ({ default: { randomShortId: () => ['0123456789abcdef'] } }))

const originalInTls = structuredClone(defaultInTls)
const originalOutTls = structuredClone(defaultOutTls)

afterEach(() => {
  Object.assign(defaultInTls, structuredClone(originalInTls))
  Object.assign(defaultOutTls, structuredClone(originalOutTls))
})

describe('TLS editor state', () => {
  it('uses the first complete certificate domain as the default server name', () => {
    const acme = { domain: ['vpn.example.com', 'alt.example.com'] } as any
    const vm = { acme, domains: 'vpn.example.com,alt.example.com' } as any
    Acme.computed!.optionDefault.set.call(vm, true)
    expect(acme.default_server_name).toBe('vpn.example.com')
  })

  it.each([
    ['optionALPN', 'alpn', (value: any) => value.push('custom')],
    ['optionCS', 'cipher_suites', (value: any) => value.push('custom')],
    ['optionFP', 'utls', (value: any) => { value.fingerprint = 'firefox' }],
    ['optionReality', 'reality', (value: any) => { value.public_key = 'custom' }],
    ['optionEch', 'ech', (value: any) => { value.config_path = 'custom' }],
  ] as const)('keeps outbound %s edits separate from later forms', (option, field, edit) => {
    const first = { $props: { outbound: { tls: {} } } } as any
    OutTLS.computed![option].set.call(first, true)
    edit(first.$props.outbound.tls[field])
    const next = { $props: { outbound: { tls: {} } } } as any
    OutTLS.computed![option].set.call(next, true)
    expect(next.$props.outbound.tls[field]).toEqual(originalOutTls[field])
  })

  it.each([
    ['optionALPN', 'inTls', 'alpn', (value: any) => value.push('custom')],
    ['optionCS', 'inTls', 'cipher_suites', (value: any) => value.push('custom')],
    ['optionFP', 'outTls', 'utls', (value: any) => { value.fingerprint = 'firefox' }],
    ['optionEch', 'outTls', 'ech', (value: any) => { value.config_path = 'custom' }],
  ] as const)('keeps TLS modal %s edits separate from later forms', (option, side, field, edit) => {
    const first = { inTls: {}, outTls: {} } as any
    Tls.computed![option].set.call(first, true)
    edit(first[side][field])
    const next = { inTls: {}, outTls: {} } as any
    Tls.computed![option].set.call(next, true)
    const defaults = side === 'inTls' ? originalInTls : originalOutTls
    expect(next[side][field]).toEqual((defaults as any)[field])
  })

  it('keeps Reality fingerprint edits separate from later TLS modals', () => {
    const first = { tlsType: 1, tls: {} } as any
    Tls.methods!.changeTlsType.call(first)
    first.tls.client.utls.fingerprint = 'firefox'
    const next = { tlsType: 1, tls: {} } as any
    Tls.methods!.changeTlsType.call(next)
    expect(next.tls.client.utls).toEqual(originalOutTls.utls)
  })
})
