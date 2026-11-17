import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createRenderer, nextTick, ref, ssrContextKey, type App, type Component } from 'vue'
import { createPinia, setActivePinia } from 'pinia'
import Basics from './Basics.vue'
import Dns from './Dns.vue'
import Rules from './Rules.vue'
import Settings from './Settings.vue'
import Data from '@/store/modules/data'
import { i18n, loadInitialLocaleMessages } from '@/locales'

vi.mock('@/plugins/httputil', () => ({ default: { get: vi.fn().mockResolvedValue({ success: false }), post: vi.fn() } }))
vi.mock('@/components/nexus/primitives/useConfirm', () => ({ useConfirm: () => ({ confirm: vi.fn() }) }))
vi.mock('vue-router', async importOriginal => ({ ...await importOriginal<typeof import('vue-router')>(), useRoute: () => ({ query: {} }) }))

const renderer = createRenderer({
  insert: () => {}, remove: () => {},
  createElement: () => ({}), createText: () => ({}), createComment: () => ({}),
  setText: () => {}, setElementText: () => {}, patchProp: () => {},
  parentNode: () => null, nextSibling: () => null,
})
const apps: App[] = []

beforeEach(async () => {
  await loadInitialLocaleMessages()
  setActivePinia(createPinia())
})
afterEach(() => {
  for (const app of apps.splice(0)) app.unmount()
  vi.restoreAllMocks()
  vi.clearAllTimers()
  vi.useRealTimers()
})

const mountEditor = async (component: Component, config: Record<string, any>, lastLoad = 1) => {
  const pinia = createPinia()
  const app = renderer.createApp({ ...component, render: () => null })
  app.use(pinia)
  app.use(i18n)
  app.provide(ssrContextKey, { modules: new Set() })
  app.provide('loading', ref(false))
  const store = Data(pinia)
  store.lastLoad = lastLoad
  store.config = config
  const vm = app.mount({})
  apps.push(app)
  await Promise.resolve()
  await nextTick()
  await Promise.resolve()
  return { app, state: (vm.$ as any).setupState as Record<string, any>, store }
}

describe('config editors', () => {
  for (const [name, component] of [['basics', Basics], ['routing', Rules]] as const) {
    it(`${name} preserves edits and allows retry after a rejected save`, async () => {
      const { state, store } = await mountEditor(component, { log: {}, experimental: {}, route: { rules: [], rule_set: [] } })
      state.appConfig.log.level = 'debug'
      vi.spyOn(store, 'save').mockResolvedValueOnce(false)
      await state.saveConfig()
      expect(state.appConfig.log.level).toBe('debug')
      expect(state.loading).toBe(false)
    })
  }

  for (const [name, component] of [['basics', Basics], ['DNS', Dns], ['routing', Rules]] as const) {
    it(`${name} stops waiting for the first store load after unmount`, async () => {
      vi.useFakeTimers()
      const { app } = await mountEditor(component, {}, 0)
      app.unmount()
      apps.splice(apps.indexOf(app), 1)
      expect(vi.getTimerCount()).toBe(0)
    })
    it(`${name} initializes from the first load and preserves later local edits`, async () => {
      const { state, store } = await mountEditor(component, {}, 0)
      store.config = { log: { level: 'warn' }, experimental: {}, dns: { servers: [], rules: [] }, route: { rules: [], rule_set: [] } }
      store.lastLoad = 1
      await nextTick()
      expect(state.loading).toBe(false)
      expect(state.appConfig.log.level).toBe('warn')
      state.appConfig.log.level = 'debug'
      store.config = { ...store.config, log: { level: 'error' } }
      store.lastLoad = 2
      await nextTick()
      expect(state.appConfig.log.level).toBe('debug')
    })
  }

  it('adds CORS origins to a newly enabled Clash API', async () => {
    const { state } = await mountEditor(Basics, { log: {}, experimental: {} })
    state.enableClashApi = true
    state.origin = 'https://panel.example,https://app.example'
    expect(state.appConfig.experimental.clash_api.access_control_allow_origin).toEqual(['https://panel.example', 'https://app.example'])
  })

  it('keeps routing edits when the stored config omits route', async () => {
    const { state } = await mountEditor(Rules, { log: {}, experimental: {} })
    state.route.final = 'direct'
    expect(state.appConfig.route?.final).toBe('direct')
  })

  for (const [name, component] of [['basics', Basics], ['settings', Settings]] as const) {
    it(`${name} keeps log and experimental editors usable without optional config sections`, async () => {
      const { state } = await mountEditor(component, {})
      expect(state.enableCacheFile).toBe(false)
      state.appConfig.log.disabled = true
      state.enableDebug = true
      expect(state.appConfig.log.disabled).toBe(true)
      expect(state.appConfig.experimental.debug).toEqual({})
    })
  }
})
