import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createRenderer, nextTick, ssrContextKey, type App } from 'vue'
import HttpUtils from '@/plugins/httputil'
import PanelUpdateCard from './PanelUpdateCard.vue'

vi.mock('@/plugins/httputil', () => ({ default: { get: vi.fn(), post: vi.fn() } }))

const renderer = createRenderer({
  insert: () => {}, remove: () => {},
  createElement: () => ({}), createText: () => ({}), createComment: () => ({}),
  setText: () => {}, setElementText: () => {}, patchProp: () => {},
  parentNode: () => null, nextSibling: () => null,
})
const apps: App[] = []
const response = (channel = 'main', stage = 'idle') => ({
  success: true, msg: '', obj: { current: '1.0.0', channel, job: { stage } },
})
const flushRequests = async () => {
  await Promise.resolve()
  await nextTick()
  await Promise.resolve()
}
const mountCard = () => {
  const app = renderer.createApp({ ...PanelUpdateCard, render: () => null })
  app.provide(ssrContextKey, { modules: new Set() })
  const vm = app.mount({})
  apps.push(app)
  return { app, state: (vm.$ as any).setupState }
}

beforeEach(() => {
  vi.useFakeTimers()
  vi.mocked(HttpUtils.get).mockReset()
  vi.mocked(HttpUtils.post).mockReset()
})
afterEach(() => {
  for (const app of apps.splice(0)) app.unmount()
  vi.clearAllTimers()
  vi.useRealTimers()
})

describe('panel update status lifecycle', () => {
  it('resumes polling when the card is opened during an update', async () => {
    vi.mocked(HttpUtils.get).mockResolvedValueOnce(response('main', 'downloading'))
    vi.mocked(HttpUtils.get).mockResolvedValue(response('main', 'idle'))
    mountCard()
    await flushRequests()
    await vi.advanceTimersByTimeAsync(2000)
    expect(HttpUtils.get).toHaveBeenCalledTimes(2)
    expect(vi.getTimerCount()).toBe(0)
  })

  it('ignores initial status loaded before a newer channel check', async () => {
    let resolveStatus!: (value: any) => void
    vi.mocked(HttpUtils.get).mockImplementation(() => new Promise(resolve => { resolveStatus = resolve }))
    vi.mocked(HttpUtils.post).mockResolvedValue(response('beta'))
    const { state } = mountCard()
    state.channel = 'beta'
    await flushRequests()
    resolveStatus(response('main'))
    await flushRequests()
    expect(state.channel).toBe('beta')
    expect(state.status.channel).toBe('beta')
  })

  it('keeps the latest channel check pending when an earlier check completes', async () => {
    vi.mocked(HttpUtils.get).mockResolvedValue(response())
    const resolvers: Array<(value: any) => void> = []
    vi.mocked(HttpUtils.post).mockImplementation(() => new Promise(resolve => { resolvers.push(resolve) }))
    const { state } = mountCard()
    await flushRequests()
    state.channel = 'beta'
    await flushRequests()
    state.channel = 'main'
    await flushRequests()
    resolvers[0](response('beta'))
    await flushRequests()
    expect(state.channel).toBe('main')
    expect(state.checking).toBe(true)
    resolvers[1](response('main'))
    await flushRequests()
    expect(state.channel).toBe('main')
    expect(state.checking).toBe(false)
  })

  it('does not start polling from a response received after unmount', async () => {
    let resolveStatus!: (value: any) => void
    vi.mocked(HttpUtils.get).mockImplementation(() => new Promise(resolve => { resolveStatus = resolve }))
    const { app } = mountCard()
    app.unmount()
    apps.splice(apps.indexOf(app), 1)
    resolveStatus(response('main', 'downloading'))
    await flushRequests()
    expect(vi.getTimerCount()).toBe(0)
  })
})
