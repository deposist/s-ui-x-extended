import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

const { get, post } = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
}))

vi.mock('@/plugins/httputil', () => ({
  default: { get, post },
}))

vi.mock('notivue', () => ({
  push: { warning: vi.fn(), error: vi.fn(), success: vi.fn() },
}))

vi.mock('@/locales', () => ({
  i18n: { global: { t: (key: string) => key } },
}))

import Data from './data'

interface TestResponse {
  success: true
  msg: string
  obj: Record<string, unknown>
}

const response = (obj: Record<string, unknown>): TestResponse => ({ success: true, msg: '', obj })

const deferred = <T>() => {
  let resolve!: (value: T) => void
  const promise = new Promise<T>(done => { resolve = done })
  return { promise, resolve }
}

describe('Data server cursor and subscription availability', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    get.mockReset()
    post.mockReset()
    vi.stubGlobal('localStorage', {
      getItem: vi.fn(() => null),
      setItem: vi.fn(),
      removeItem: vi.fn(),
    })
  })

  it('uses the exact authoritative server revision for the next load', async () => {
    vi.spyOn(Date, 'now').mockReturnValue(9_999_999_999_000)
    get
      .mockResolvedValueOnce(response({ revision: 41, onlines: {}, config: { log: {} } }))
      .mockResolvedValueOnce(response({ revision: 41, onlines: { user: ['alice'] } }))
    const data = Data()

    await data.loadData()
    await data.loadData()

    expect(data.lastLoad).toBe(41)
    expect(get).toHaveBeenNthCalledWith(1, 'api/load', {})
    expect(get).toHaveBeenNthCalledWith(2, 'api/load', { lu: 41 })
  })

  it('deduplicates concurrent loads', async () => {
    const pending = deferred<TestResponse>()
    get.mockReturnValueOnce(pending.promise)
    const data = Data()

    const first = data.loadData()
    const second = data.loadData()
    expect(get).toHaveBeenCalledTimes(1)

    pending.resolve(response({ revision: 12, onlines: {}, config: { marker: 'fresh' } }))
    await first

    expect(data.lastLoad).toBe(12)
    expect(data.config).toEqual({ marker: 'fresh' })
  })

  it('removes and restores subscription URIs as availability changes', () => {
    const data = Data()

    data.setNewData({
      revision: 1,
      config: {},
      subURI: 'https://base.example/sub/',
      subJsonURI: 'https://json.example/sub/',
      subClashURI: 'https://clash.example/sub/',
    })
    expect([data.subURI, data.subJsonURI, data.subClashURI]).toEqual([
      'https://base.example/sub/',
      'https://json.example/sub/',
      'https://clash.example/sub/',
    ])

    data.setNewData({ revision: 2, config: {}, subURI: '' })
    expect([data.subURI, data.subJsonURI, data.subClashURI]).toEqual(['', '', ''])

    data.setNewData({ revision: 3, config: {}, subURI: 'https://back.example/sub/' })
    expect(data.subURI).toBe('https://back.example/sub/')
  })
})
