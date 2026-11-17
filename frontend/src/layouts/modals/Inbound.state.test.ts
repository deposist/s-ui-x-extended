import { webcrypto } from 'node:crypto'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import Inbound from './Inbound.vue'
import InboundDrawer from '@/components/nexus/drawers/InboundDrawer.vue'
import Data from '@/store/modules/data'

vi.mock('@/plugins/httputil', () => ({ default: { get: vi.fn() } }))

beforeEach(() => {
  setActivePinia(createPinia())
  vi.stubGlobal('window', { crypto: webcrypto })
})
afterEach(() => vi.unstubAllGlobals())

describe.each([['classic', Inbound], ['nexus', InboundDrawer]])('%s inbound loading', (_mode, component) => {
  const form = (): any => {
    const makeData = component.data! as (this: any, vm: any) => Record<string, any>
    const vm: any = { ...makeData.call({}, {}), $props: { id: 1, visible: true }, $t: (key: string) => key }
    for (const [name, method] of Object.entries(component.methods!)) vm[name] = (method as (...args: any[]) => any).bind(vm)
    return vm
  }

  it('stops loading and blocks save when the requested inbound is missing', async () => {
    const vm = form()
    vm.inbound = { id: 1, tag: 'old', type: 'direct', listen_port: 443 }
    vi.spyOn(Data(), 'loadInbounds').mockResolvedValue([])
    const save = vi.spyOn(Data(), 'save').mockResolvedValue(true)
    await expect(vm.loadData(2)).resolves.toBeUndefined()
    expect(vm.loading).toBe(false)
    await vm.saveChanges()
    expect(save).not.toHaveBeenCalled()
  })

  it('does not replace a new inbound with a pending previous edit', async () => {
    const vm = form()
    let resolveFirst!: (value: any) => void
    vi.spyOn(Data(), 'loadInbounds').mockImplementation(() => new Promise(resolve => { resolveFirst = resolve }))
    const oldLoad = vm.loadData(1)
    vm.$props.id = 0
    vm.updateData(0)
    const newTag = vm.inbound.tag
    resolveFirst([{ id: 1, type: 'direct', tag: 'previous', listen_port: 443 }])
    await oldLoad
    expect(vm.inbound.id).toBe(0)
    expect(vm.inbound.tag).toBe(newTag)
  })
})
