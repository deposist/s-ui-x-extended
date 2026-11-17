import { describe, expect, it, vi } from 'vitest'
import { createRenderer, ssrContextKey } from 'vue'
import { createPinia } from 'pinia'
import Inbounds from './Inbounds.vue'
import Data from '@/store/modules/data'

vi.mock('@/plugins/httputil', () => ({ default: {} }))
vi.mock('@/layouts/modals/Inbound.vue', () => ({ default: {} }))
vi.mock('@/layouts/modals/Stats.vue', () => ({ default: {} }))

const renderer = createRenderer({
  insert: () => {}, remove: () => {},
  createElement: () => ({}), createText: () => ({}), createComment: () => ({}),
  setText: () => {}, setElementText: () => {}, patchProp: () => {},
  parentNode: () => null, nextSibling: () => null,
})

describe('inbound cloning', () => {
  it('stops cloning without saving when the source inbound is missing', async () => {
    const app = renderer.createApp({ ...Inbounds, render: () => null })
    const pinia = createPinia()
    app.use(pinia)
    app.provide(ssrContextKey, { modules: new Set() })
    vi.spyOn(Data(pinia), 'loadInbounds').mockResolvedValue([])
    const save = vi.spyOn(Data(pinia), 'save').mockResolvedValue(true)
    app.mount({})
    try {
      const state = (app._instance as any).setupState
      await expect(state.clone(1)).resolves.toBeUndefined()
      expect(state.cloneLoading).toBe(false)
      expect(save).not.toHaveBeenCalled()
    } finally {
      app.unmount()
    }
  })
})
