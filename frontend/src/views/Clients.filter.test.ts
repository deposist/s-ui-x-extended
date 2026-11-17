import { describe, expect, it, vi } from 'vitest'
import { createRenderer, ref, ssrContextKey } from 'vue'
import Clients from './Clients.vue'

const store = vi.hoisted(() => ({
  clients: [{ id: 1, name: 'user[1]', desc: 'a.b', group: '', inbounds: [] }, { id: 2, name: 'other', desc: 'axb', group: '', inbounds: [] }],
  inbounds: [], onlines: { user: [] }, enableTraffic: false,
}))
vi.mock('@/store/modules/data', () => ({ default: () => store }))
vi.mock('vuetify', () => ({ useDisplay: () => ({ smAndDown: ref(false) }) }))
vi.mock('@/uiMode/useUiMode', () => ({ useUiMode: () => ({ mode: ref('classic') }) }))
vi.mock('@/locales', () => ({ i18n: { global: { t: (key: string) => key } }, locale: ref('en') }))
vi.mock('@/layouts/modals/Client.vue', () => ({ default: {} }))
vi.mock('@/layouts/modals/ClientAddBulk.vue', () => ({ default: {} }))
vi.mock('@/layouts/modals/ClientEditBulk.vue', () => ({ default: {} }))
vi.mock('@/layouts/modals/QrCode.vue', () => ({ default: {} }))
vi.mock('@/layouts/modals/ClientDoctor.vue', () => ({ default: {} }))
vi.mock('@/layouts/modals/Stats.vue', () => ({ default: {} }))
vi.mock('@/components/IpHistoryModal.vue', () => ({ default: {} }))

const renderer = createRenderer({
  insert: () => {}, remove: () => {},
  createElement: () => ({}), createText: () => ({}), createComment: () => ({}),
  setText: () => {}, setElementText: () => {}, patchProp: () => {},
  parentNode: () => null, nextSibling: () => null,
})

describe('client text filtering', () => {
  for (const text of ['[', '.']) {
    it(`matches ${text} as literal text`, () => {
      const app = renderer.createApp({ ...Clients, render: () => null })
      app.provide(ssrContextKey, { modules: new Set() })
      app.mount({})
      try {
        const state = (app._instance as any).setupState
        state.filterSettings.text = text
        expect(() => state.doFilter()).not.toThrow()
        expect(state.filterSettings.filteredClients.map((client: any) => client.id)).toEqual([1])
      } finally {
        app.unmount()
      }
    })
  }
})
