import { beforeAll, describe, expect, it } from 'vitest'
import { webcrypto } from 'node:crypto'
import { createInbound, InTypes } from './inbounds'
import { createOutbound, OutTypes } from './outbounds'
import { createEndpoint, EpTypes } from './endpoints'
import { createSrv, SrvTypes } from './services'
import { createDnsServer, DnsTypes } from './dns'
import { createProvider, ProviderTypes } from './providers'
import { createClient } from './clients'

beforeAll(() => {
  Object.defineProperty(globalThis, 'window', {
    configurable: true,
    value: { crypto: webcrypto },
  })
})

describe('new editor forms do not reuse cancelled nested edits', () => {
  it('isolates inbound transport defaults', () => {
    const first = createInbound(InTypes.VLESS) as any
    first.transport.type = 'ws'
    expect((createInbound(InTypes.VLESS) as any).transport).toEqual({})
  })

  it('isolates outbound TLS defaults', () => {
    const first = createOutbound(OutTypes.Hysteria2)
    first.tls.insecure = true
    expect(createOutbound(OutTypes.Hysteria2).tls).toEqual({ enabled: true })
  })

  it('isolates endpoint address defaults', () => {
    const first = createEndpoint(EpTypes.Wireguard)
    first.address.push('192.0.2.1/32')
    expect(createEndpoint(EpTypes.Wireguard).address).toEqual(['10.0.0.2/32', 'fe80::2/128'])
  })

  it('isolates service device defaults', () => {
    const first = createSrv(SrvTypes.USBIPServer) as any
    first.devices.push({ bus_id: '1-1' })
    expect((createSrv(SrvTypes.USBIPServer) as any).devices).toEqual([])
  })

  it('isolates DNS hosts defaults', () => {
    const first = createDnsServer(DnsTypes.Hosts)
    first.path.push('/tmp/custom-hosts')
    expect(createDnsServer(DnsTypes.Hosts).path).toEqual(['/etc/hosts'])
  })

  it('isolates inline provider outbound defaults', () => {
    const first = createProvider(ProviderTypes.Inline)
    first.outbounds.push({ type: 'direct', tag: 'cancelled' })
    expect(createProvider(ProviderTypes.Inline).outbounds).toEqual([])
  })

  it('isolates new client assignments', () => {
    const first = createClient()
    first.inbounds.push(7)
    expect(createClient().inbounds).toEqual([])
  })
})
