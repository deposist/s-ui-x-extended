import { describe, expect, it, vi } from 'vitest'
import { createRenderer, nextTick, ssrContextKey } from 'vue'
import HttpUtils from '@/plugins/httputil'
import Nodes from './Nodes.vue'

vi.mock('@/plugins/httputil', () => ({ default: { get: vi.fn() } }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
vi.mock('@/components/nexus/primitives/useConfirm', () => ({ useConfirm: () => ({ confirm: vi.fn() }) }))
vi.mock('@/components/nexus/data/NexusDataTable.vue', () => ({ default: {} }))
vi.mock('@/components/nexus/data/RowActions.vue', () => ({ default: {} }))
vi.mock('@/components/nexus/data/TablePagination.vue', () => ({ default: {} }))
vi.mock('@/components/nexus/primitives/Badge.vue', () => ({ default: {} }))
vi.mock('@/components/nexus/primitives/EmptyState.vue', () => ({ default: {} }))
vi.mock('@/components/nexus/primitives/PageHeader.vue', () => ({ default: {} }))
vi.mock('@/components/nexus/primitives/PageToolbar.vue', () => ({ default: {} }))

const renderer = createRenderer({
  insert: () => {}, remove: () => {},
  createElement: () => ({}), createText: () => ({}), createComment: () => ({}),
  setText: () => {}, setElementText: () => {}, patchProp: () => {},
  parentNode: () => null, nextSibling: () => null,
})
function mountNodes() {
  const app = renderer.createApp({ ...Nodes, render: () => null })
  app.provide(ssrContextKey, { modules: new Set() })
  app.mount({})
  return { app, state: (app._instance as any).setupState }
}

describe('node client selection', () => {
  it('refreshes the Master node list and stops on unmount', async () => {
    vi.useFakeTimers()
    const get = vi.mocked(HttpUtils.get).mockClear()
      .mockResolvedValueOnce({ success: true, msg: '', obj: [] })
      .mockResolvedValueOnce({ success: true, msg: '', obj: [{ uuid: 'new-slave' }] })
    const { app, state } = mountNodes()
    try {
      state.panelRole = 'master'
      await vi.advanceTimersByTimeAsync(5000)
      expect(state.nodes).toEqual([{ uuid: 'new-slave' }])
    } finally {
      app.unmount()
      await vi.advanceTimersByTimeAsync(5000)
      vi.useRealTimers()
    }
    expect(get).toHaveBeenCalledTimes(2)
  })

  it('keeps both add and remove selections from different pages', async () => {
    vi.mocked(HttpUtils.get).mockResolvedValueOnce({ success: true, msg: '', obj: [] })
      .mockResolvedValueOnce({ success: true, msg: '', obj: { items: [{ id: 1, assigned: false }], total: 50 } })
      .mockResolvedValueOnce({ success: true, msg: '', obj: { items: [{ id: 2, assigned: true }], total: 50 } })
    const { app, state } = mountNodes()
    try {
      await state.openClients({ uuid: 'a', name: 'A' })
      state.toggleClient(1)
      await state.loadClients(2)
      state.toggleClient(2)
      expect(state.selectionAddable).toEqual([1])
      expect(state.selectionRemovable).toEqual([2])
    } finally { app.unmount() }
  })

  it('ignores a response for a previously opened node', async () => {
    let resolveA!: (value: any) => void
    let resolveB!: (value: any) => void
    vi.mocked(HttpUtils.get).mockResolvedValueOnce({ success: true, msg: '', obj: [] })
      .mockImplementationOnce(() => new Promise(resolve => { resolveA = resolve }))
      .mockImplementationOnce(() => new Promise(resolve => { resolveB = resolve }))
    const { app, state } = mountNodes()
    try {
      const first = state.openClients({ uuid: 'a', name: 'A' })
      const second = state.openClients({ uuid: 'b', name: 'B' })
      resolveB({ success: true, obj: { items: [{ id: 2 }], total: 1 } })
      await second
      resolveA({ success: true, obj: { items: [{ id: 1 }], total: 1 } })
      await first
      await nextTick()
      expect(state.clientsPanel.items.map((client: any) => client.id)).toEqual([2])
    } finally { app.unmount() }
  })
})
