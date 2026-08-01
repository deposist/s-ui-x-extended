import { describe, expect, it, vi } from 'vitest'

import { SnapshotSession } from './snapshotSession'

describe('authenticated snapshot session', () => {
  it('loads one shared snapshot and does not install healthy-socket polling', () => {
    const loadSnapshot = vi.fn()
    const connectRealtime = vi.fn()
    const disconnectRealtime = vi.fn()
    const setIntervalSpy = vi.spyOn(globalThis, 'setInterval')
    const session = new SnapshotSession({ loadSnapshot, connectRealtime, disconnectRealtime })

    session.enter()
    session.enter()
    session.enter()

    expect(loadSnapshot).toHaveBeenCalledTimes(1)
    expect(connectRealtime).toHaveBeenCalledTimes(1)
    expect(setIntervalSpy).not.toHaveBeenCalled()
    expect(disconnectRealtime).not.toHaveBeenCalled()
  })

  it('disconnects on login and starts one fresh snapshot on resume', () => {
    const loadSnapshot = vi.fn()
    const connectRealtime = vi.fn()
    const disconnectRealtime = vi.fn()
    const session = new SnapshotSession({ loadSnapshot, connectRealtime, disconnectRealtime })

    session.enter()
    session.leave()
    session.leave()
    session.enter()

    expect(loadSnapshot).toHaveBeenCalledTimes(2)
    expect(connectRealtime).toHaveBeenCalledTimes(2)
    expect(disconnectRealtime).toHaveBeenCalledTimes(1)
  })
})
