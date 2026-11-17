import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createRenderer, nextTick, ssrContextKey, type App } from 'vue'
import { createPinia } from 'pinia'
import HttpUtils from '@/plugins/httputil'
import Data from '@/store/modules/data'
import Overview from './Overview.vue'

vi.mock('@/plugins/httputil', () => ({ default: { get: vi.fn() } }))

const renderer = createRenderer({
  insert: () => {}, remove: () => {},
  createElement: () => ({}), createText: () => ({}), createComment: () => ({}),
  setText: () => {}, setElementText: () => {}, patchProp: () => {},
  parentNode: () => null, nextSibling: () => null,
})
const apps: App[] = []
let requests: Array<{ url: string; resolve: (response: any) => void }>
const flushRequests = async () => {
  await Promise.resolve()
  await nextTick()
  await Promise.resolve()
}
const mountOverview = () => {
  const app = renderer.createApp({ ...Overview, render: () => null })
  const pinia = createPinia()
  app.use(pinia)
  app.provide(ssrContextKey, { modules: new Set() })
  Data(pinia).lastLoad = 1
  const vm = app.mount({})
  apps.push(app)
  return (vm.$ as any).setupState
}
const finishRequest = (request: typeof requests[number], marker: number) => {
  const obj = request.url === 'api/status'
    ? { cpu: marker, sbd: { running: true } }
    : request.url === 'api/security/audit'
      ? { events: [{ id: marker, event: 'login_success' }] }
      : { buckets: [{ startTime: marker, download: marker, upload: marker }] }
  request.resolve({ success: true, msg: '', obj })
}

beforeEach(() => {
  vi.useFakeTimers()
  vi.stubGlobal('window', new EventTarget())
  vi.stubGlobal('document', Object.assign(new EventTarget(), { hidden: false }))
  vi.stubGlobal('navigator', { onLine: true })
  requests = []
  vi.mocked(HttpUtils.get).mockReset()
  vi.mocked(HttpUtils.get).mockImplementation(url => new Promise(resolve => requests.push({ url, resolve })))
})
afterEach(() => {
  for (const app of apps.splice(0)) app.unmount()
  vi.clearAllTimers()
  vi.useRealTimers()
  vi.unstubAllGlobals()
})

describe('overview connectivity lifecycle', () => {
  it('does not publish pending successful responses after going offline', async () => {
    const state = mountOverview()
    expect(requests).toHaveLength(3)
    window.dispatchEvent(new Event('offline'))
    for (const request of requests) finishRequest(request, 1)
    await flushRequests()
    expect(state.browserOnline).toBe(false)
    expect(state.trafficSummary).toBeUndefined()
    expect(state.statusPayload).toBeUndefined()
    expect(state.auditEvents).toEqual([])
    expect(state.statusUnavailable).toBe(true)
    expect(state.auditUnavailable).toBe(true)
    expect(state.trafficLoading).toBe(false)
    expect(state.statusLoading).toBe(false)
    expect(state.auditLoading).toBe(false)
  })

  it('refreshes all data immediately on reconnect and retains the newer responses', async () => {
    const state = mountOverview()
    const original = [...requests]
    window.dispatchEvent(new Event('offline'))
    window.dispatchEvent(new Event('online'))
    expect(requests).toHaveLength(6)
    for (const request of requests.slice(3)) finishRequest(request, 2)
    await flushRequests()
    for (const request of original) finishRequest(request, 1)
    await flushRequests()
    expect(state.statusPayload.cpu).toBe(2)
    expect(state.auditEvents.map((event: any) => event.id)).toEqual([2])
    expect(state.trafficSummary.buckets[0].download).toBe(2)
    expect(state.statusUnavailable).toBe(false)
    expect(state.auditUnavailable).toBe(false)
  })
})
