import { describe, expect, it } from 'vitest'
import { reactive } from 'vue'
import QUICOptions from './QUICOptions.vue'

describe('QUIC optional settings', () => {
  it('opens an empty settings group without writing defaults and clears only QUIC fields', async () => {
    const data = reactive<Record<string, unknown>>({ tag: 'keep' })
    const component = QUICOptions as any
    const state: any = { $props: { data }, ...component.data?.() }
    const toggle = component.computed.optionQuic
    expect(toggle.get.call(state)).toBe(false)
    toggle.set.call(state, true)
    expect(toggle.get.call(state)).toBe(true)
    expect(data).toEqual({ tag: 'keep' })
    data.idle_timeout = '12s'
    toggle.set.call(state, false)
    expect(toggle.get.call(state)).toBe(false)
    expect(data).toEqual({ tag: 'keep' })
  })
})
