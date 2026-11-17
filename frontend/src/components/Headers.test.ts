import { describe, expect, it } from 'vitest'
import Headers from './Headers.vue'

describe('HTTP header editing', () => {
  it.each(['constructor', '__proto__'])('preserves the valid HTTP header name %s', name => {
    const data: any = {}
    const vm: any = { $props: { data } }
    Headers.computed!.hdrs.set.call(vm, [{ name, value: 'test-value' }])
    expect(JSON.parse(JSON.stringify(data.headers))).toEqual({ [name]: 'test-value' })
  })

  it('preserves an empty first value of a repeated header', () => {
    const data: any = {}
    const vm: any = { $props: { data } }
    Headers.computed!.hdrs.set.call(vm, [{ name: 'X-Test', value: '' }, { name: 'X-Test', value: 'next' }])
    expect(data.headers['X-Test']).toEqual(['', 'next'])
  })
})
