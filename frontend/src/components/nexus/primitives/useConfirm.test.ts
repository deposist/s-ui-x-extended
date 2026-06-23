import { beforeEach, describe, expect, it } from 'vitest'

import { activeConfirm, confirm, resolveActiveConfirm, useConfirm } from './useConfirm'

describe('useConfirm', () => {
  beforeEach(() => {
    // Drain any leftover request so each test starts clean.
    resolveActiveConfirm(false)
  })

  it('exposes confirm() through the composable', () => {
    expect(typeof useConfirm().confirm).toBe('function')
  })

  it('publishes the active request for the host to render', () => {
    void confirm({ title: 'Delete inbound?', message: 'cannot undo', tone: 'error' })

    expect(activeConfirm.value).toMatchObject({
      title: 'Delete inbound?',
      message: 'cannot undo',
      tone: 'error',
    })
  })

  it('resolves true when confirmed', async () => {
    const pending = confirm({ title: 'go?' })

    resolveActiveConfirm(true)

    await expect(pending).resolves.toBe(true)
    expect(activeConfirm.value).toBeNull()
  })

  it('resolves false when cancelled', async () => {
    const pending = confirm({ title: 'go?' })

    resolveActiveConfirm(false)

    await expect(pending).resolves.toBe(false)
  })

  it('cancels a superseded request so its caller never hangs', async () => {
    const first = confirm({ title: 'first' })
    const second = confirm({ title: 'second' })

    await expect(first).resolves.toBe(false)
    expect(activeConfirm.value).toMatchObject({ title: 'second' })

    resolveActiveConfirm(true)
    await expect(second).resolves.toBe(true)
  })

  it('ignores resolve calls when nothing is pending', () => {
    expect(() => resolveActiveConfirm(true)).not.toThrow()
    expect(activeConfirm.value).toBeNull()
  })
})
