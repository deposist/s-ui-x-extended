import { beforeEach, describe, expect, it, vi } from 'vitest'

const mocks = vi.hoisted(() => ({
  axiosGet: vi.fn(),
}))

vi.mock('axios', () => ({
  default: {
    get: mocks.axiosGet,
  },
}))

vi.mock('@/plugins/base-url', () => ({
  getBaseUrl: () => '/app/',
}))

const loadCSRFStore = async () => {
  vi.resetModules()
  return import('../csrf')
}

describe('csrf store regression anchors', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('fetches and reuses a token until clearCSRFToken is called', async () => {
    const { clearCSRFToken, getCSRFToken } = await loadCSRFStore()
    mocks.axiosGet
      .mockResolvedValueOnce({ data: { obj: { token: 'token-1' } } })
      .mockResolvedValueOnce({ data: { obj: { token: 'token-2' } } })

    await expect(getCSRFToken()).resolves.toBe('token-1')
    await expect(getCSRFToken()).resolves.toBe('token-1')
    expect(mocks.axiosGet).toHaveBeenCalledTimes(1)

    clearCSRFToken()
    await expect(getCSRFToken()).resolves.toBe('token-2')
    expect(mocks.axiosGet).toHaveBeenCalledTimes(2)
  })

  it('coalesces concurrent token requests into one backend call', async () => {
    const { getCSRFToken } = await loadCSRFStore()
    let resolveToken: (value: unknown) => void = () => {}
    mocks.axiosGet.mockReturnValueOnce(new Promise((resolve) => {
      resolveToken = resolve
    }))

    const first = getCSRFToken()
    const second = getCSRFToken()
    expect(mocks.axiosGet).toHaveBeenCalledTimes(1)

    resolveToken({ data: { obj: { token: 'shared-token' } } })
    await expect(first).resolves.toBe('shared-token')
    await expect(second).resolves.toBe('shared-token')
  })

  it('does not let a stale in-flight request repopulate the cache after clear', async () => {
    const { clearCSRFToken, getCSRFToken } = await loadCSRFStore()
    let resolveToken: (value: unknown) => void = () => {}
    mocks.axiosGet
      .mockReturnValueOnce(new Promise((resolve) => {
        resolveToken = resolve
      }))
      .mockResolvedValueOnce({ data: { obj: { token: 'fresh-token' } } })

    const stale = getCSRFToken()
    clearCSRFToken()
    resolveToken({ data: { obj: { token: 'stale-token' } } })

    await expect(stale).resolves.toBe('stale-token')
    await expect(getCSRFToken()).resolves.toBe('fresh-token')
    expect(mocks.axiosGet).toHaveBeenCalledTimes(2)
  })

  it('documents that client-side expiry is clear-driven, not time-driven', async () => {
    const { getCSRFToken } = await loadCSRFStore()
    mocks.axiosGet.mockResolvedValueOnce({ data: { obj: { token: 'cached-token' } } })

    await expect(getCSRFToken()).resolves.toBe('cached-token')
    await new Promise((resolve) => setTimeout(resolve, 0))
    await expect(getCSRFToken()).resolves.toBe('cached-token')
    expect(mocks.axiosGet).toHaveBeenCalledTimes(1)
  })

  it('rejects empty token responses', async () => {
    const { getCSRFToken } = await loadCSRFStore()
    mocks.axiosGet.mockResolvedValueOnce({ data: { obj: { token: '' } } })

    await expect(getCSRFToken()).rejects.toThrow('CSRF token was not returned')
  })
})
