import { beforeEach, describe, expect, it, vi } from 'vitest'

const mocks = vi.hoisted(() => ({
  requestFulfilled: undefined as undefined | ((config: any) => any),
  responseFulfilled: undefined as undefined | ((response: any) => any),
  responseRejected: undefined as undefined | ((error: any) => any),
  axiosCreate: vi.fn(),
  axiosIsCancel: vi.fn((error: any) => Boolean(error?.__CANCEL__)),
  clearCSRFToken: vi.fn(),
  getCSRFToken: vi.fn(),
}))

vi.mock('axios', () => ({
  default: {
    create: mocks.axiosCreate,
    isCancel: mocks.axiosIsCancel,
  },
}))

vi.mock('@/store/csrf', () => ({
  clearCSRFToken: mocks.clearCSRFToken,
  getCSRFToken: mocks.getCSRFToken,
}))

vi.mock('@/plugins/base-url', () => ({
  getBaseUrl: () => '/app/',
}))

const loadApi = async () => {
  vi.resetModules()
  mocks.axiosCreate.mockReturnValue({
    interceptors: {
      request: {
        use: (fulfilled: (config: any) => any) => {
          mocks.requestFulfilled = fulfilled
        },
      },
      response: {
        use: (fulfilled: (response: any) => any, rejected: (error: any) => any) => {
          mocks.responseFulfilled = fulfilled
          mocks.responseRejected = rejected
        },
      },
    },
  })
  await import('../api')
}

describe('api axios interceptor regression anchors', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.requestFulfilled = undefined
    mocks.responseFulfilled = undefined
    mocks.responseRejected = undefined
    mocks.getCSRFToken.mockResolvedValue('csrf-token')
  })

  it('adds CSRF tokens to mutating api requests', async () => {
    await loadApi()

    const config = await mocks.requestFulfilled?.({
      method: 'post',
      url: 'api/save',
      headers: {},
    })

    expect(mocks.getCSRFToken).toHaveBeenCalledTimes(1)
    expect(config.headers['X-CSRF-Token']).toBe('csrf-token')
  })

  it('does not add CSRF tokens to login requests', async () => {
    await loadApi()

    const config = await mocks.requestFulfilled?.({
      method: 'post',
      url: 'api/login',
      headers: {},
    })

    expect(mocks.getCSRFToken).not.toHaveBeenCalled()
    expect(config.headers['X-CSRF-Token']).toBeUndefined()
  })

  it('aborts the previous duplicate idempotent request', async () => {
    await loadApi()
    const first = await mocks.requestFulfilled?.({
      method: 'get',
      url: 'api/load',
      params: { refresh: 1 },
      headers: {},
    })

    await mocks.requestFulfilled?.({
      method: 'get',
      url: 'api/load',
      params: { refresh: 1 },
      headers: {},
    })

    expect(first.signal.aborted).toBe(true)
    expect(first.signal.reason).toBe('Duplicate request cancelled')
  })

  it('does not deduplicate mutating requests with the same URL', async () => {
    await loadApi()
    const first = await mocks.requestFulfilled?.({
      method: 'post',
      url: 'api/save',
      data: { a: 1 },
      headers: {},
    })

    await mocks.requestFulfilled?.({
      method: 'post',
      url: 'api/save',
      data: { a: 2 },
      headers: {},
    })

    expect(first.signal).toBeUndefined()
  })

  it('clears the cached CSRF token on Invalid CSRF token responses', async () => {
    await loadApi()
    const error = {
      response: {
        status: 403,
        data: { msg: 'Invalid CSRF token' },
      },
    }

    await expect(mocks.responseRejected?.(error)).rejects.toBe(error)

    expect(mocks.clearCSRFToken).toHaveBeenCalledTimes(1)
  })
})
