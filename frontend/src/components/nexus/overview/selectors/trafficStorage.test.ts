import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { safeRemoveItem } from '@/utils/safeStorage'
import { isValidTrafficTimeZone, loadTrafficTimeZone, persistTrafficTimeZone, trafficTimeZoneStorageKey } from './trafficSelectors'

beforeEach(() => {
  vi.stubGlobal('localStorage', undefined)
  Object.defineProperty(globalThis, 'localStorage', {
    configurable: true,
    get: () => { throw new Error('storage access denied') },
  })
})
afterEach(() => {
  safeRemoveItem(trafficTimeZoneStorageKey)
  vi.unstubAllGlobals()
})

describe('traffic timezone with blocked browser storage', () => {
  it('loads a valid timezone when accessing localStorage itself throws', () => {
    expect(isValidTrafficTimeZone(loadTrafficTimeZone())).toBe(true)
  })

  it('keeps the selected timezone for the session when localStorage is blocked', () => {
    expect(() => persistTrafficTimeZone('Europe/Moscow')).not.toThrow()
    expect(loadTrafficTimeZone()).toBe('Europe/Moscow')
  })
})
