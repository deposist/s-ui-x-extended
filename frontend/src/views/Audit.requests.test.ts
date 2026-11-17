import { describe, expect, it, vi } from 'vitest'
import { createRenderer, nextTick, ref, ssrContextKey } from 'vue'
import HttpUtils from '@/plugins/httputil'
import Audit from './Audit.vue'

vi.mock('@/plugins/httputil', () => ({ default: { get: vi.fn() } }))
vi.mock('@/uiMode/useUiMode', () => ({ useUiMode: () => ({ mode: ref('classic') }) }))
vi.mock('@/locales', () => ({ i18n: { global: { t: (key: string) => key } }, locale: 'en' }))
vi.mock('vue3-persian-datetime-picker', () => ({ default: {} }))
vi.mock('@/components/nexus/data/NexusDataTable.vue', () => ({ default: {} }))
vi.mock('@/components/nexus/primitives/EmptyState.vue', () => ({ default: {} }))
vi.mock('@/components/nexus/primitives/PageHeader.vue', () => ({ default: {} }))
vi.mock('@/components/nexus/primitives/StatusBadge.vue', () => ({ default: {} }))

const renderer = createRenderer({
  insert: () => {}, remove: () => {},
  createElement: () => ({}), createText: () => ({}), createComment: () => ({}),
  setText: () => {}, setElementText: () => {}, patchProp: () => {},
  parentNode: () => null, nextSibling: () => null,
})

describe('audit requests', () => {
  it('keeps the newest filter response when requests finish out of order', async () => {
    const responses: Array<(value: any) => void> = []
    vi.mocked(HttpUtils.get).mockImplementation(() => new Promise(resolve => responses.push(resolve)))
    const app = renderer.createApp({ ...Audit, render: () => null })
    app.provide(ssrContextKey, { modules: new Set() })
    app.mount({})
    try {
      const state = (app._instance as any).setupState
      state.eventFilter = 'login_failed'
      state.resetAndLoad()
      responses[1]({ success: true, obj: { events: [{ id: 2 }], nextCursor: 22 } })
      await nextTick()
      responses[0]({ success: true, obj: { events: [{ id: 1 }], nextCursor: 11 } })
      await nextTick()
      expect(state.events.map((event: any) => event.id)).toEqual([2])
      expect(state.nextCursor).toBe(22)
    } finally {
      app.unmount()
    }
  })
})
