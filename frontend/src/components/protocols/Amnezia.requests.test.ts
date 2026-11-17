import { describe, expect, it, vi } from 'vitest'
import Amnezia from './Amnezia.vue'
import HttpUtils from '@/plugins/httputil'

vi.mock('@/plugins/httputil', () => ({ default: { get: vi.fn() } }))

describe('Amnezia pending parameter generation', () => {
  it.each(['applyHardened', 'fetchRandom'] as const)('keeps a newly selected profile intact during %s', async method => {
    let complete!: (value: any) => void
    vi.mocked(HttpUtils.get).mockImplementationOnce(() => new Promise(resolve => { complete = resolve }))
    const vm: any = { data: { amnezia: { jc: 4 } }, full: true, hardening: false, randomizing: false, selectedPreset: 'balanced' }
    Object.defineProperty(vm, 'amnezia', { get: () => vm.data.amnezia })
    const pending = Amnezia.methods![method].call(vm, true)
    const current = { jc: 7, h1: 999 }
    vm.data.amnezia = current
    complete({ success: true, obj: { jc: 5, jmin: 10, jmax: 50, h1: 11, h2: 12, h3: 13, h4: 14 } })
    await expect(pending).resolves.toBeUndefined()
    expect(vm.data.amnezia).toEqual({ jc: 7, h1: 999 })
    expect(vm.selectedPreset).toBe('balanced')
  })

  it('handles disabling Amnezia while hardened generation is pending', async () => {
    let complete!: (value: any) => void
    vi.mocked(HttpUtils.get).mockImplementationOnce(() => new Promise(resolve => { complete = resolve }))
    const vm: any = { data: { amnezia: {} }, full: true, hardening: false }
    Object.defineProperty(vm, 'amnezia', { get: () => vm.data.amnezia })
    const pending = Amnezia.methods!.applyHardened.call(vm)
    delete vm.data.amnezia
    complete({ success: true, obj: { jc: 5 } })
    await expect(pending).resolves.toBeUndefined()
    expect(vm.data).not.toHaveProperty('amnezia')
    expect(vm.hardening).toBe(false)
  })
})
