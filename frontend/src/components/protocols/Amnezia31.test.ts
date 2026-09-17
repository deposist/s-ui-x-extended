import { describe, expect, it, vi } from 'vitest'
import { reactive } from 'vue'

vi.mock('@/router', () => ({ default: { push: vi.fn() } }))
vi.mock('@/locales', () => ({ i18n: { global: { t: (key: string) => key } } }))
vi.mock('@/store/csrf', () => ({ clearCSRFToken: vi.fn() }))
vi.mock('notivue', () => ({ push: { error: vi.fn(), success: vi.fn() } }))
vi.mock('@/plugins/api', () => ({ default: { get: vi.fn(), post: vi.fn() } }))

// Relative import (not '@/'): the alias drags in the router setup, which needs
// a browser window; this file mirrors QUICOptions.test.ts.
import Amnezia from './Amnezia.vue'
const component = Amnezia as any
const stateWith = (amnezia: Record<string, unknown> | undefined) => {
  const data = reactive({ amnezia, mtu: 1420 })
  const state: any = { $props: { data, full: true }, ...component.data?.() }
  // computed amnezia is the same contract the template binds to.
  state.amnezia = data.amnezia
  return state
}


describe('AmneziaWG 3.1 switches', () => {
  it('default to off without writing keys', () => {
    const state = stateWith({})
    expect(component.computed.randomTrailers.get.call(state)).toBe(false)
    expect(component.computed.disableCookies.get.call(state)).toBe(false)
  })

  it('reflect a persisted true flag', () => {
    const state = stateWith({ random_trailers: true, disable_cookies: true })
    expect(component.computed.randomTrailers.get.call(state)).toBe(true)
    expect(component.computed.disableCookies.get.call(state)).toBe(true)
  })

  it('turning on writes true and turning off deletes the key', () => {
    const state = stateWith({})
    component.computed.randomTrailers.set.call(state, true)
    component.computed.disableCookies.set.call(state, true)
    expect(state.$props.data.amnezia.random_trailers).toBe(true)
    expect(state.$props.data.amnezia.disable_cookies).toBe(true)
    component.computed.randomTrailers.set.call(state, false)
    component.computed.disableCookies.set.call(state, false)
    expect('random_trailers' in state.$props.data.amnezia).toBe(false)
    expect('disable_cookies' in state.$props.data.amnezia).toBe(false)
  })
})
