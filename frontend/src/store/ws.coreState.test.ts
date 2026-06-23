import { beforeEach, describe, expect, it, vi } from 'vitest'

const { warn } = vi.hoisted(() => ({ warn: vi.fn() }))

vi.mock('notivue', () => ({
  push: { warning: warn, error: vi.fn(), success: vi.fn() },
}))
vi.mock('@/locales', () => ({
  i18n: { global: { t: (key: string) => key } },
}))
vi.mock('@/plugins/httputil', () => ({
  default: { get: vi.fn() },
}))
vi.mock('@/store/modules/data', () => ({
  default: () => ({ loadData: vi.fn(), onlines: {} }),
}))

import { handleCoreStateWarning } from './ws'

describe('core_state realtime warning', () => {
  beforeEach(() => warn.mockClear())

  it('toasts the failover all-down warning with the group tag', () => {
    handleCoreStateWarning({ warning: 'failover_all_down', group: 'auto' })
    expect(warn).toHaveBeenCalledTimes(1)
    expect(String(warn.mock.calls[0][0].message)).toContain('auto')
  })

  it('stays silent for unrelated or empty core_state payloads', () => {
    handleCoreStateWarning({ warning: 'stats_commit_failed' })
    handleCoreStateWarning({})
    handleCoreStateWarning(undefined)
    expect(warn).not.toHaveBeenCalled()
  })
})
