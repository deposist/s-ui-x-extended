import { describe, expect, it, vi } from 'vitest'
import { createRenderer, nextTick, ssrContextKey } from 'vue'
import HttpUtils from '@/plugins/httputil'
import Nodes from './Nodes.vue'

vi.mock('@/plugins/httputil', () => ({ default: { get: vi.fn(), post: vi.fn() } }))
vi.mock('@/store/modules/data', () => ({ default: () => ({ clients: [] }) }))
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

  it('selects a group and then all search results across pages', async () => {
    const page = (items: unknown[], total: number) => ({ success: true, msg: '', obj: { items, total } })
    const first = page([{ id: 1, assigned: false, group: 'Team' }, { id: 2, assigned: false, group: '' }], 3)
    const second = page([{ id: 3, assigned: true, group: 'Team' }], 3)
    const get = vi.mocked(HttpUtils.get).mockClear()
      .mockResolvedValueOnce({ success: true, msg: '', obj: [] })
      .mockResolvedValueOnce(first)
      .mockResolvedValueOnce(first)
      .mockResolvedValueOnce(second)
      .mockResolvedValueOnce(first)
      .mockResolvedValueOnce(second)
    const { app, state } = mountNodes()
    try {
      await state.openClients({ uuid: 'a', name: 'A', paired: true, peerInbounds: [{ id: 12 }] })
      state.clientsPanel.query = 'filtered'
      await state.selectMatching('Team')
      expect(state.selectionAddable).toEqual([1])
      expect(state.selectionRemovable).toEqual([3])
      expect(get).toHaveBeenLastCalledWith('api/nodes/a/clients', { limit: 200, offset: 2, q: 'filtered', inboundId: 12 })
      await state.selectMatching()
      expect(state.selectionAddable).toEqual([1, 2])
      expect(state.selectionRemovable).toEqual([3])
    } finally { app.unmount() }
  })

  it('does not carry a bulk selection to a different inbound', async () => {
    let finish!: (value: any) => void
    const page = { success: true, msg: '', obj: { items: [], total: 0 } }
    vi.mocked(HttpUtils.get)
      .mockResolvedValueOnce({ success: true, msg: '', obj: [] })
      .mockResolvedValueOnce(page)
      .mockImplementationOnce(() => new Promise(resolve => { finish = resolve }))
      .mockResolvedValueOnce(page)
    const { app, state } = mountNodes()
    try {
      await state.openClients({ uuid: 'a', name: 'A', paired: true, peerInbounds: [{ id: 12 }] })
      const selecting = state.selectMatching()
      await state.selectPeerInbound(13)
      finish({ success: true, msg: '', obj: { items: [{ id: 1, assigned: false }], total: 1 } })
      await selecting
      expect(state.selectionAddable).toEqual([])
    } finally { app.unmount() }
  })

  it('submits selections in API-sized batches and retains unsuccessful ids', async () => {
    const page = { success: true, msg: '', obj: { items: [], total: 0 } }
    vi.mocked(HttpUtils.get)
      .mockResolvedValueOnce({ success: true, msg: '', obj: [] })
      .mockResolvedValueOnce(page)
      .mockResolvedValueOnce(page)
      .mockResolvedValueOnce({ success: true, msg: '', obj: [] })
    const post = vi.mocked(HttpUtils.post).mockClear()
      .mockResolvedValueOnce({ success: true, msg: '', obj: null })
      .mockResolvedValueOnce({ success: false, msg: '', obj: null })
    const { app, state } = mountNodes()
    try {
      await state.openClients({ uuid: 'a', name: 'A' })
      for (let id = 1; id <= 501; id++) state.selection.ids.set(id, false)
      state.selection.ids.set(502, true)
      await state.assignSelected(true)
      expect(post.mock.calls.map(([, body]) => JSON.parse((body as URLSearchParams).get('ids')!).length)).toEqual([500, 1])
      expect(state.selectionAddable).toEqual([501])
      expect(state.selectionRemovable).toEqual([502])
    } finally { app.unmount() }
  })

  it('applies only the reviewed fingerprint and keeps the selection after a conflict', async () => {
    const page = { success: true, msg: '', obj: { items: [{ id: 3, assigned: false }], total: 1 } }
    vi.mocked(HttpUtils.get)
      .mockResolvedValueOnce({ success: true, msg: '', obj: [] })
      .mockResolvedValueOnce(page)
    const post = vi.mocked(HttpUtils.post).mockClear()
      .mockResolvedValueOnce({ success: true, msg: '', obj: { changes: 1, skipped: 0, items: [], fingerprint: 'calculated' } })
      .mockResolvedValueOnce({ success: false, msg: '', obj: { reason: 'stale_preview' } })
    const { app, state } = mountNodes()
    try {
      await state.openClients({ uuid: 'peer', name: 'Slave', paired: true, peerInbounds: [{ id: 12 }] })
      state.toggleClient(3)
      await state.assignSelected(true)
      expect(post).toHaveBeenCalledTimes(1)
      expect(post.mock.calls[0]?.[0]).toBe('api/nodes/peer/clients/preview')
      expect(state.assignmentPreview.fingerprint).toBe('calculated')
      await state.confirmAssignmentPreview()
      expect((post.mock.calls[1]?.[1] as URLSearchParams).get('previewFingerprint')).toBe('calculated')
      expect(state.assignmentResult).toBe('nodes.preview.stale')
      expect(state.selectionAddable).toEqual([3])
    } finally { app.unmount() }
  })
})
