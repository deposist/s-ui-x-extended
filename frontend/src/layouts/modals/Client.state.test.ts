import { webcrypto } from 'node:crypto'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import Client from './Client.vue'
import Data from '@/store/modules/data'
import HttpUtils from '@/plugins/httputil'
import { createClient } from '@/types/clients'

vi.mock('@/plugins/httputil', () => ({ default: { get: vi.fn(), patch: vi.fn(), post: vi.fn() } }))
vi.mock('@/plugins/api', () => ({ default: {} }))
vi.mock('vue3-persian-datetime-picker', () => ({ default: {} }))

const form = (id = 0): any => {
  const vm: any = { ...Client.data!.call({} as any, {} as any), $props: { id, visible: true }, $emit: vi.fn() }
  for (const [name, method] of Object.entries(Client.methods!)) vm[name] = (method as (...args: any[]) => any).bind(vm)
  return vm
}

beforeEach(() => {
  setActivePinia(createPinia())
  vi.stubGlobal('window', { crypto: webcrypto })
  vi.mocked(HttpUtils.get).mockReset().mockResolvedValue({ success: true, msg: '', obj: [] })
  vi.mocked(HttpUtils.patch).mockReset().mockResolvedValue({ success: true, msg: '', obj: null })
})

afterEach(() => vi.unstubAllGlobals())

describe('client form state', () => {
  it('resets limits when a new form follows an edited client', async () => {
    const vm = form()
    vm.limits = { speedShared: false, speedMbps: 10, downloadMbps: 20, uploadMbps: 30, connections: 4 }
    await vm.updateData(0)
    expect(vm.limits).toEqual({ speedShared: true, speedMbps: 0, downloadMbps: 0, uploadMbps: 0, connections: 0 })
  })

  it('keeps the newer client when an earlier client load finishes last', async () => {
    const vm = form(1)
    let resolveFirst!: (value: any) => void
    vi.spyOn(Data(), 'loadClients').mockImplementation(id => id === 1
      ? new Promise(resolve => { resolveFirst = resolve })
      : Promise.resolve(createClient({ id: 2, name: 'second' })))
    const first = vm.updateData(1)
    vm.$props.id = 2
    await vm.updateData(2)
    resolveFirst(createClient({ id: 1, name: 'first' }))
    await first
    expect(vm.client.id).toBe(2)
    expect(vm.client.name).toBe('second')
  })

  it('waits for saved assignments and limits before allowing an edit', async () => {
    const vm = form(1)
    vi.spyOn(Data(), 'loadClients').mockResolvedValue(createClient({ id: 1, name: 'client' }))
    let resolveLimits!: (value: any) => void
    vi.mocked(HttpUtils.get).mockImplementation(url => url.endsWith('/limits')
      ? new Promise(resolve => { resolveLimits = resolve })
      : Promise.resolve({ success: true, msg: '', obj: [] }))
    const loading = vm.updateData(1)
    await vi.waitFor(() => expect(resolveLimits).toBeTypeOf('function'))
    expect(vm.loading).toBe(true)
    resolveLimits({ success: true, msg: '', obj: { speed: 1250000, connections: 2 } })
    await loading
    expect(vm.loading).toBe(false)
    expect(vm.limits.speedMbps).toBe(10)
  })

  it('ignores assignments that finish after opening a new client', async () => {
    const vm = form(1)
    let resolveNodes!: (value: any) => void
    vi.mocked(HttpUtils.get).mockImplementation(url => url === 'api/clients/1/nodes'
      ? new Promise(resolve => { resolveNodes = resolve })
      : Promise.resolve({ success: true, msg: '', obj: [] }))
    const oldNodes = vm.loadClientNodes(1)
    vm.$props.id = 0
    await vm.updateData(0)
    resolveNodes({ success: true, msg: '', obj: ['old-node'] })
    await oldNodes
    expect(vm.clientNodes).toEqual([])
  })

  it('keeps the form open if node assignments cannot be saved', async () => {
    const vm = form(1)
    vm.client = createClient({ id: 1, name: 'client' })
    vi.spyOn(Data(), 'save').mockResolvedValue(true)
    vi.mocked(HttpUtils.patch).mockResolvedValue({ success: false, msg: 'unavailable', obj: null })
    await vm.saveChanges()
    expect(vm.$emit).not.toHaveBeenCalledWith('close')
    expect(HttpUtils.patch).toHaveBeenCalledTimes(1)
  })

  it('retries a newly created client as an edit after a limits save fails', async () => {
    const vm = form()
    vm.client.name = 'new-client'
    const save = vi.spyOn(Data(), 'save').mockImplementation(async () => {
      Data().clients = [createClient({ id: 7, name: 'new-client' })]
      return true
    })
    vi.mocked(HttpUtils.patch)
      .mockResolvedValueOnce({ success: true, msg: '', obj: null })
      .mockResolvedValueOnce({ success: false, msg: 'unavailable', obj: null })
    await vm.saveChanges()
    expect(vm.$emit).not.toHaveBeenCalledWith('close')
    await vm.saveChanges()
    expect(save.mock.calls.map(call => call[1])).toEqual(['new', 'edit'])
    expect(vm.$emit).toHaveBeenCalledWith('close')
  })

  it('saves the submitted assignments and limits if the form closes during the client save', async () => {
    const vm = form(7)
    vm.client = createClient({ id: 7, name: 'submitted-client' })
    vm.clientNodes = ['submitted-node']
    vm.limits.speedMbps = 8
    let resolveSave!: (success: boolean) => void
    vi.spyOn(Data(), 'save').mockImplementation(() => new Promise(resolve => { resolveSave = resolve }))
    const pending = vm.saveChanges()
    vm.closeModal()
    resolveSave(true)
    await pending
    expect(HttpUtils.patch).toHaveBeenCalledWith('api/clients/7/nodes', { nodes: '["submitted-node"]' })
    expect(HttpUtils.patch).toHaveBeenCalledWith('api/clients/7/limits', expect.objectContaining({ speed: '1000000' }))
    expect(vm.client.id).toBeUndefined()
  })

  it('does not close or unlock a newer form when an earlier client save finishes', async () => {
    const vm = form(7)
    vm.client = createClient({ id: 7, name: 'submitted-client' })
    let resolveSave!: (success: boolean) => void
    let resolveClient!: (client: any) => void
    vi.spyOn(Data(), 'save').mockImplementation(() => new Promise(resolve => { resolveSave = resolve }))
    vi.spyOn(Data(), 'loadClients').mockImplementation(() => new Promise(resolve => { resolveClient = resolve }))
    const pending = vm.saveChanges()
    vm.closeModal()
    vm.$props.id = 8
    vm.$emit.mockClear()
    const newerLoad = vm.updateData(8)
    resolveSave(true)
    await pending
    const loadingWhileNewClientPending = vm.loading
    resolveClient(createClient({ id: 8, name: 'newer-client' }))
    await newerLoad
    expect(loadingWhileNewClientPending).toBe(true)
    expect(vm.$emit).not.toHaveBeenCalledWith('close')
    expect(vm.client.id).toBe(8)
  })
})
