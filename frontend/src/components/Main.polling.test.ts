import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createRenderer, nextTick, ssrContextKey } from 'vue'
import { createPinia, setActivePinia } from 'pinia'
import HttpUtils from '@/plugins/httputil'
import Data from '@/store/modules/data'
import Main from './Main.vue'
import { loadInitialLocaleMessages } from '@/locales'

vi.mock('@/plugins/httputil', () => ({ default: { get: vi.fn(), post: vi.fn() } }))

const renderer = createRenderer({
  insert: () => {}, remove: () => {},
  createElement: () => ({}), createText: () => ({}), createComment: () => ({}),
  setText: () => {}, setElementText: () => {}, patchProp: () => {},
  parentNode: () => null, nextSibling: () => null,
})

const flushRequests = async () => {
  await Promise.resolve()
  await nextTick()
  await Promise.resolve()
}

beforeEach(async () => {
  await loadInitialLocaleMessages()
  vi.useFakeTimers()
  vi.stubGlobal('document', Object.assign(new EventTarget(), { hidden: false }))
  setActivePinia(createPinia())
  vi.mocked(HttpUtils.get).mockReset()
})

afterEach(() => {
  vi.useRealTimers()
  vi.unstubAllGlobals()
})

describe('classic dashboard polling', () => {
  it('does not restart polling after unmount while the first status is pending', async () => {
    let resolve!: (value: any) => void
    vi.mocked(HttpUtils.get).mockImplementation(() => new Promise(done => { resolve = done }))
    // Store preferences are read when the component's Pinia instance is created.
    const app = renderer.createApp({ ...Main, render: () => null })
    const pinia = createPinia()
    app.use(pinia)
    app.provide(ssrContextKey, { modules: new Set() })
    Data(pinia).reloadItems = ['g-cpu']
    app.mount({})
    app.unmount()

    resolve({ success: true, msg: '', obj: { cpu: 5 } })
    await flushRequests()
    expect(vi.getTimerCount()).toBe(0)
  })

  it('does not overlap slow status requests', async () => {
    vi.mocked(HttpUtils.get).mockResolvedValueOnce({ success: true, msg: '', obj: { cpu: 5 } })
    vi.mocked(HttpUtils.get).mockImplementation(() => new Promise(() => {}))
    const app = renderer.createApp({ ...Main, render: () => null })
    const pinia = createPinia()
    app.use(pinia)
    app.provide(ssrContextKey, { modules: new Set() })
    Data(pinia).reloadItems = ['g-cpu']
    app.mount({})
    await flushRequests()
    vi.mocked(HttpUtils.get).mockClear()

    await vi.advanceTimersByTimeAsync(6000)
    expect(HttpUtils.get).toHaveBeenCalledTimes(1)
    app.unmount()
  })

  it('keeps cached system details between metric-only responses', async () => {
    vi.mocked(HttpUtils.get).mockResolvedValueOnce({ success: true, msg: '', obj: { sys: { appVersion: '1.0' }, cpu: 5 } })
    vi.mocked(HttpUtils.get).mockResolvedValue({ success: true, msg: '', obj: { cpu: 6 } })
    const app = renderer.createApp({ ...Main, render: () => null })
    const pinia = createPinia()
    app.use(pinia)
    app.provide(ssrContextKey, { modules: new Set() })
    Data(pinia).reloadItems = ['i-sys', 'g-cpu']
    app.mount({})
    await flushRequests()

    await vi.advanceTimersByTimeAsync(4000)
    expect(vi.mocked(HttpUtils.get).mock.calls.map(call => call[1])).toEqual([
      { r: 'sys,cpu' }, { r: 'cpu' }, { r: 'cpu' },
    ])
    app.unmount()
  })
})
