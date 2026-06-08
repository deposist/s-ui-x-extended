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

describe('HttpUtils cancellation handling', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('does not show a failed toast for canceled duplicate requests', async () => {
    const { default: HttpUtils } = await import('@/plugins/httputil')
    mocks.apiGet.mockRejectedValueOnce(Object.assign(new Error('canceled'), {
      code: 'ERR_CANCELED',
      name: 'CanceledError',
    }))

    const msg = await HttpUtils.get('api/load')

    expect(msg).toEqual({ success: false, msg: '', obj: null })
    expect(mocks.pushError).not.toHaveBeenCalled()
  })

  it('still shows a failed toast for real request errors', async () => {
    const { default: HttpUtils } = await import('@/plugins/httputil')
    mocks.apiGet.mockRejectedValueOnce(new Error('network down'))

    const msg = await HttpUtils.get('api/load')

    expect(msg).toEqual({ success: false, msg: 'Error: network down', obj: null })
    expect(mocks.pushError).toHaveBeenCalledTimes(1)
    expect(mocks.pushError).toHaveBeenCalledWith(expect.objectContaining({
      message: 'Error: network down',
    }))
  })
})
