import { beforeEach, describe, expect, it, vi } from 'vitest'

const mocks = vi.hoisted(() => ({
  apiGet: vi.fn(),
  apiPost: vi.fn(),
  clearCSRFToken: vi.fn(),
  pushError: vi.fn(),
  pushSuccess: vi.fn(),
  routerPush: vi.fn(),
}))

vi.mock('@/plugins/api', () => ({
  default: {
    get: mocks.apiGet,
    post: mocks.apiPost,
  },
}))

vi.mock('@/router', () => ({
  default: {
    push: mocks.routerPush,
  },
}))

vi.mock('@/locales', () => ({
  i18n: {
    global: {
      t: (key: string) => key,
    },
  },
}))

vi.mock('@/store/csrf', () => ({
  clearCSRFToken: mocks.clearCSRFToken,
}))

vi.mock('notivue', () => ({
  push: {
    error: mocks.pushError,
    success: mocks.pushSuccess,
  },
}))

const loadHttpUtils = async () => {
  vi.resetModules()
  return import('../httputil')
}

describe('HttpUtils regression anchors', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('maps a successful GET response and shows the success toast', async () => {
    const { default: HttpUtils } = await loadHttpUtils()
    mocks.apiGet.mockResolvedValueOnce({ data: { success: true, msg: 'saved', obj: { ok: true } } })

    const msg = await HttpUtils.get('api/load', { refresh: 1 })

    expect(mocks.apiGet).toHaveBeenCalledWith('api/load', { params: { refresh: 1 } })
    expect(msg).toEqual({ success: true, msg: 'saved', obj: { ok: true } })
    expect(mocks.pushSuccess).toHaveBeenCalledWith({
      message: 'success: actions.saved',
    })
  })

  it('maps backend error bodies from POST failures without throwing', async () => {
    const { default: HttpUtils } = await loadHttpUtils()
    mocks.apiPost.mockRejectedValueOnce({
      response: {
        data: { success: false, msg: 'Invalid CSRF token', obj: null },
      },
    })

    const msg = await HttpUtils.post('api/save', { key: 'value' })

    expect(msg).toEqual({ success: false, msg: 'Invalid CSRF token', obj: null })
    expect(mocks.pushError).toHaveBeenCalledWith({
      title: 'failed',
      message: 'Invalid CSRF token',
    })
  })

  it('clears CSRF and navigates to login locally on Invalid login responses', async () => {
    const { default: HttpUtils } = await loadHttpUtils()
    mocks.apiGet.mockResolvedValueOnce({ data: { success: false, msg: 'Invalid login', obj: null } })

    await HttpUtils.get('api/load')
    await Promise.resolve()

    expect(mocks.pushError).toHaveBeenCalledWith({ title: 'invalidLogin' })
    expect(mocks.clearCSRFToken).toHaveBeenCalledTimes(1)
    expect(mocks.routerPush).toHaveBeenCalledWith('/login')
    expect(mocks.apiPost).not.toHaveBeenCalled()
  })

  it('handles Invalid login errors from CSRF loading as a local logout', async () => {
    const { default: HttpUtils } = await loadHttpUtils()
    mocks.apiPost.mockRejectedValueOnce(new Error('Invalid login'))

    const msg = await HttpUtils.post('api/save', { key: 'value' })

    expect(msg).toEqual({ success: false, msg: 'Invalid login', obj: null })
    expect(mocks.pushError).toHaveBeenCalledWith({ title: 'invalidLogin' })
    expect(mocks.clearCSRFToken).toHaveBeenCalledTimes(1)
    expect(mocks.routerPush).toHaveBeenCalledWith('/login')
  })

  it('handles repeated Invalid login responses only once until reset', async () => {
    const mod = await loadHttpUtils()
    const HttpUtils = mod.default
    mocks.apiGet.mockResolvedValue({ data: { success: false, msg: 'Invalid login', obj: null } })

    await HttpUtils.get('api/load')
    await HttpUtils.get('api/status')

    expect(mocks.pushError).toHaveBeenCalledTimes(1)
    expect(mocks.clearCSRFToken).toHaveBeenCalledTimes(1)
    expect(mocks.routerPush).toHaveBeenCalledTimes(1)

    mod.resetInvalidLoginHandling()
    await HttpUtils.get('api/load')

    expect(mocks.pushError).toHaveBeenCalledTimes(2)
    expect(mocks.clearCSRFToken).toHaveBeenCalledTimes(2)
    expect(mocks.routerPush).toHaveBeenCalledTimes(2)
  })

  it('does not retry HTTP 401 responses in the current axios-based contract', async () => {
    const { default: HttpUtils } = await loadHttpUtils()
    mocks.apiGet.mockRejectedValueOnce({
      response: {
        status: 401,
        data: { success: false, msg: 'unauthorized', obj: null },
      },
    })

    const msg = await HttpUtils.get('api/load')

    expect(msg).toEqual({ success: false, msg: 'unauthorized', obj: null })
    expect(mocks.apiGet).toHaveBeenCalledTimes(1)
  })

  it('suppresses toast noise for AbortController cancellation', async () => {
    const { default: HttpUtils } = await loadHttpUtils()
    mocks.apiGet.mockRejectedValueOnce(Object.assign(new Error('Duplicate request cancelled'), {
      code: 'ERR_CANCELED',
      name: 'CanceledError',
    }))

    const msg = await HttpUtils.get('api/load')

    expect(msg).toEqual({ success: false, msg: '', obj: null })
    expect(mocks.pushError).not.toHaveBeenCalled()
  })

  it('marks non-Msg JSON payloads as unknown data instead of throwing', async () => {
    const { default: HttpUtils } = await loadHttpUtils()
    mocks.apiGet.mockResolvedValueOnce({ data: { unexpected: true } })

    const msg = await HttpUtils.get('api/load')

    expect(msg).toEqual({ success: false, msg: 'unknown data: [object Object]', obj: null })
    expect(mocks.pushError).toHaveBeenCalledWith({
      title: 'failed',
      message: 'unknown data: [object Object]',
    })
  })
})
