import { beforeEach, describe, expect, it, vi } from 'vitest'
import Changes from './Changes.vue'
import Logs from './Logs.vue'
import QrCode from './QrCode.vue'

const { get, loadClients } = vi.hoisted(() => ({ get: vi.fn(), loadClients: vi.fn() }))
vi.mock('@/plugins/httputil', () => ({ default: { get } }))
vi.mock('@/store/modules/data', () => ({ default: () => ({ loadClients }) }))

const cases = [
  { name: 'Logs', component: Logs, method: 'loadData', value: 'lines', request: get },
  { name: 'Changes', component: Changes, method: 'loadData', value: 'changes', request: get },
  { name: 'QrCode', component: QrCode, method: 'load', value: 'client', request: loadClients },
]

beforeEach(() => { get.mockReset(); loadClients.mockReset() })

describe('modal data loading', () => {
  it.each(cases)('$name ignores an older response and keeps the newest request loading', async ({ component, method, value, request }) => {
    const data = component.data! as (this: any, vm: any) => Record<string, any>
    const vm: any = { ...data.call({}, {}), $props: { id: 1 } }
    let resolveOld!: (value: any) => void
    let resolveNew!: (value: any) => void
    request.mockImplementationOnce(() => new Promise(resolve => { resolveOld = resolve }))
    request.mockImplementationOnce(() => new Promise(resolve => { resolveNew = resolve }))
    const load = (component.methods! as any)[method].bind(vm)
    const old = load()
    const current = load()
    resolveOld(component === QrCode ? { name: 'old', subSecret: 'old-secret' } : { success: true, obj: ['old'] })
    await old
    expect(vm.loading).toBe(true)
    resolveNew(component === QrCode ? { name: 'new', subSecret: 'new-secret' } : { success: true, obj: ['new'] })
    await current
    expect(vm[value]).toEqual(component === QrCode ? { name: 'new', subSecret: 'new-secret' } : ['new'])
    expect(vm.loading).toBe(false)
  })

  it.each(cases)('$name clears loading after a rejected request', async ({ component, method, request }) => {
    const data = component.data! as (this: any, vm: any) => Record<string, any>
    const vm: any = { ...data.call({}, {}), $props: { id: 1 } }
    request.mockRejectedValue(new Error('request failed'))
    await expect((component.methods! as any)[method].call(vm)).rejects.toThrow('request failed')
    expect(vm.loading).toBe(false)
  })
})
