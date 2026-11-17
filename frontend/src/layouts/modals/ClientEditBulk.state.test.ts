import { beforeEach, describe, expect, it, vi } from 'vitest'
import ClientEditBulk from './ClientEditBulk.vue'

const { save } = vi.hoisted(() => ({ save: vi.fn() }))
vi.mock('@/store/modules/data', () => ({ default: () => ({ save }) }))

beforeEach(() => save.mockReset().mockResolvedValue(false))

describe('bulk client editing', () => {
  it.each(['change_limits', 'add_inbounds'])('keeps stored clients unchanged when %s fails and is retried', async actionMode => {
    const clients = [{ id: 7, name: 'test', inbounds: [1], volume: 1024 ** 3, expiry: 0, up: 0, down: 0, enable: true }]
    const original = structuredClone(clients)
    const vm: any = {
      ...ClientEditBulk.data!.call({} as any, {} as any),
      $props: { visible: true },
      clients,
      actionMode,
      selectedClients: { model: 'all', values: [] },
      editData: { enable: true, addDays: 0, addVolume: 1, inboundTags: [2] },
      closeModal: vi.fn(),
    }
    vm.getTargetClients = ClientEditBulk.methods!.getTargetClients.bind(vm)
    await ClientEditBulk.methods!.saveChanges.call(vm)
    expect(clients).toEqual(original)
    await ClientEditBulk.methods!.saveChanges.call(vm)
    expect(clients).toEqual(original)
    const first = save.mock.calls[0][2][0]
    const second = save.mock.calls[1][2][0]
    expect(second).toEqual(first)
    if (actionMode === 'change_limits') expect(first.volume).toBe(2 * 1024 ** 3)
    else expect(first.inbounds).toEqual([1, 2])
    expect(vm.closeModal).not.toHaveBeenCalled()
  })
})
