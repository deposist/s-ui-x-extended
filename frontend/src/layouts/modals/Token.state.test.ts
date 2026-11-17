import { beforeEach, describe, expect, it, vi } from 'vitest'
import Token from './Token.vue'

const { post, get } = vi.hoisted(() => ({ post: vi.fn(), get: vi.fn() }))
vi.mock('@/plugins/httputil', () => ({ default: { post, get } }))
beforeEach(() => { post.mockReset(); get.mockReset() })

describe('token management requests', () => {
  it('submits token creation once and keeps loading through the refreshed list', async () => {
    let resolvePost!: (value: any) => void
    let resolveGet!: (value: any) => void
    post.mockImplementation(() => new Promise(resolve => { resolvePost = resolve }))
    get.mockImplementation(() => new Promise(resolve => { resolveGet = resolve }))
    const vm: any = { ...Token.data!.call({} as any, {} as any) }
    vm.loadData = Token.methods!.loadData.bind(vm)
    const first = Token.methods!.addToken.call(vm)
    const duplicate = Token.methods!.addToken.call(vm)
    expect(post).toHaveBeenCalledTimes(1)
    resolvePost({ success: true, obj: 'new-token' })
    await Promise.resolve()
    expect(vm.loading).toBe(true)
    resolveGet({ success: true, obj: [] })
    await Promise.all([first, duplicate])
    expect(vm.loading).toBe(false)
  })

  it.each(['addToken', 'setTokenEnabled', 'deleteToken'] as const)('clears loading when %s is rejected', async method => {
    post.mockRejectedValue(new Error('request failed'))
    const vm: any = { ...Token.data!.call({} as any, {} as any) }
    await expect((Token.methods![method] as (...args: any[]) => any).call(vm, 7, true)).rejects.toThrow('request failed')
    expect(vm.loading).toBe(false)
  })
})
