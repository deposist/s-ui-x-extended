import { webcrypto } from 'node:crypto'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import Endpoint from './Endpoint.vue'
import HttpUtils from '@/plugins/httputil'

vi.mock('@/plugins/httputil', () => ({ default: { get: vi.fn() } }))
beforeEach(() => {
  vi.stubGlobal('window', { crypto: webcrypto })
  vi.mocked(HttpUtils.get).mockReset()
})
afterEach(() => vi.unstubAllGlobals())

describe('endpoint type initialization', () => {
  it('does not merge pending WireGuard keys into a newer VPN server form', async () => {
    let resolveKeys!: (keys: any) => void
    const vm: any = {
      ...Endpoint.data!.call({} as any, {} as any),
      $props: { id: 0 },
      endpoint: { type: 'wireguard', listen_port: 443 },
      genWgKey: () => new Promise(resolve => { resolveKeys = resolve }),
    }
    const wireguard = Endpoint.methods!.changeType.call(vm)
    vm.endpoint.type = 'vpn-server'
    await Endpoint.methods!.changeType.call(vm)
    resolveKeys({ private_key: 'test-private', public_key: 'test-public' })
    await wireguard
    expect(vm.endpoint.type).toBe('vpn-server')
    expect(vm.endpoint.address).toBe('10.0.0.1')
    expect(vm.endpoint).not.toHaveProperty('private_key')
  })

  it('does not overwrite a stored endpoint opened while creation keys are pending', async () => {
    let resolveKeys!: (keys: any) => void
    const stored = { id: 7, type: 'wireguard', tag: 'stored', private_key: 'stored-private', address: ['10.5.0.1/24'] }
    const vm: any = {
      ...Endpoint.data!.call({} as any, {} as any),
      $props: { id: 7, data: JSON.stringify(stored) },
      genWgKey: () => new Promise(resolve => { resolveKeys = resolve }),
    }
    const pending = Endpoint.methods!.changeType.call(vm)
    await Endpoint.methods!.updateData.call(vm, 7)
    resolveKeys({ private_key: 'new-private', public_key: 'new-public' })
    await pending
    expect(vm.endpoint).toEqual(stored)
  })

  it('resets to a valid fresh WireGuard model before creation keys return', async () => {
    let resolveKeys!: (keys: any) => void
    const previous = { type: 'vpn-client', tag: 'existing', address: '10.0.0.2', outbound: {} }
    const vm: any = {
      ...Endpoint.data!.call({} as any, {} as any),
      endpoint: previous,
      genWgKey: () => new Promise(resolve => { resolveKeys = resolve }),
    }
    let pending!: Promise<void>
    vm.changeType = () => { pending = Endpoint.methods!.changeType.call(vm) }
    await Endpoint.methods!.updateData.call(vm, 0)
    expect(vm.endpoint.type).toBe('wireguard')
    expect(Array.isArray(vm.endpoint.address)).toBe(true)
    expect(vm.endpoint.peers).toEqual([])
    expect(previous.type).toBe('vpn-client')
    resolveKeys({ private_key: 'new-private', public_key: 'new-public' })
    await pending
  })
})

describe('WireGuard key editing', () => {
  it.each(['newWgKey', 'addWgPeer', 'refreshWgPeerKey'] as const)('does not apply pending %s keys after switching endpoint forms', async method => {
    let resolveKeys!: (keys: any) => void
    const vm: any = {
      ...Endpoint.data!.call({} as any, {} as any),
      endpoint: { type: 'wireguard', private_key: 'old', peers: [{ public_key: 'old-public', allowed_ips: ['10.0.1.2/32'] }], ext: { public_key: 'old-public', keys: [] } },
      genWgKey: () => new Promise(resolve => { resolveKeys = resolve }),
      findFreeIP: () => '10.0.1.3/32',
    }
    const pending = Endpoint.methods![method].call(vm, 0)
    const current = { id: 7, type: 'vpn-server', address: '10.0.0.1' }
    vm.endpoint = current
    resolveKeys({ private_key: 'new-private', public_key: 'new-public' })
    await expect(pending).resolves.toBeUndefined()
    expect(vm.endpoint).toEqual({ id: 7, type: 'vpn-server', address: '10.0.0.1' })
  })

  it('does not apply a pending derived public key to another endpoint form', async () => {
    let resolveKey!: (message: any) => void
    vi.mocked(HttpUtils.get).mockImplementation(() => new Promise(resolve => { resolveKey = resolve }))
    const vm: any = {
      ...Endpoint.data!.call({} as any, {} as any),
      endpoint: { type: 'wireguard', private_key: 'old-private', ext: { keys: [] } },
    }
    const pending = Endpoint.methods!.getWgPubKey.call(vm, 'old-private')
    vm.endpoint = { type: 'vpn-server', address: '10.0.0.1' }
    resolveKey({ success: true, obj: ['old-public'] })
    await expect(pending).resolves.toBeUndefined()
    expect(vm.endpoint).not.toHaveProperty('ext')
  })

  it('does not apply a public key derived from an older private key', async () => {
    let resolveKey!: (message: any) => void
    vi.mocked(HttpUtils.get).mockImplementation(() => new Promise(resolve => { resolveKey = resolve }))
    const vm: any = {
      ...Endpoint.data!.call({} as any, {} as any),
      endpoint: { type: 'wireguard', private_key: 'old-private', ext: { keys: [] } },
    }
    const pending = Endpoint.methods!.getWgPubKey.call(vm, 'old-private')
    vm.endpoint.private_key = 'new-private'
    resolveKey({ success: true, obj: ['old-public'] })
    await pending
    expect(vm.endpoint.ext).not.toHaveProperty('public_key')
  })

  it('does not apply pending obfuscation headers to another WireGuard form', async () => {
    let resolveHeaders!: (message: any) => void
    vi.mocked(HttpUtils.get).mockImplementation(() => new Promise(resolve => { resolveHeaders = resolve }))
    const vm: any = {
      ...Endpoint.data!.call({} as any, {} as any),
      endpoint: { type: 'wireguard', amnezia: { h1: 1 } },
    }
    const pending = Endpoint.methods!.fillRandomAmneziaHeaders.call(vm)
    vm.endpoint = { type: 'wireguard', amnezia: { h1: 2 } }
    resolveHeaders({ success: true, obj: { h1: 3, h2: 4, h3: 5, h4: 6 } })
    await pending
    expect(vm.endpoint.amnezia).toEqual({ h1: 2 })
  })
})
