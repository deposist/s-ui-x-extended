import { afterEach, describe, expect, it, vi } from 'vitest'

import type { safeGetItem as safeGetItemType, safeRemoveItem as safeRemoveItemType, safeSetItem as safeSetItemType } from './safeStorage'

// The wrappers keep per-session module state (the fallback map and the
// "storage misbehaved" flag), so each case imports a fresh module copy.
const freshSafeStorage = async () => {
  vi.resetModules()
  return await import('./safeStorage')
}

// A storage whose property access itself throws — Chrome with "Block all
// cookies" behaves this way: reading the global `localStorage` rejects with a
// SecurityError before any method can run.
const blockedStorage = (): never => {
  throw new DOMException('Access to storage is denied', 'SecurityError')
}

const workingStorage = () => {
  const map = new Map<string, string>()
  return {
    getItem: (key: string) => map.get(key) ?? null,
    setItem: (key: string, value: string) => map.set(key, value),
    removeItem: (key: string) => map.delete(key),
  }
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('safeStorage', () => {
  it('never throws when localStorage access is denied', async () => {
    vi.stubGlobal('localStorage', blockedStorage)
    const { safeGetItem, safeSetItem, safeRemoveItem } = await freshSafeStorage()

    expect(() => safeSetItem('blocked-key', 'value')).not.toThrow()
    expect(safeGetItem('blocked-key')).toBe('value')
    expect(safeGetItem('never-set')).toBeNull()
    expect(() => safeRemoveItem('blocked-key')).not.toThrow()
    expect(safeGetItem('blocked-key')).toBeNull()
  })

  it('falls back to a per-session in-memory copy on denied access', async () => {
    vi.stubGlobal('localStorage', blockedStorage)
    const { safeGetItem, safeSetItem, safeRemoveItem } = await freshSafeStorage()

    safeSetItem('memory-key', 'first')
    safeSetItem('memory-key', 'second')
    expect(safeGetItem('memory-key')).toBe('second')

    safeRemoveItem('memory-key')
    expect(safeGetItem('memory-key')).toBeNull()
  })

  it('uses real storage when it works', async () => {
    const storage = workingStorage()
    vi.stubGlobal('localStorage', storage)
    const { safeGetItem, safeSetItem, safeRemoveItem } = await freshSafeStorage()

    safeSetItem('real-key', 'persisted')
    expect(storage.getItem('real-key')).toBe('persisted')
    expect(safeGetItem('real-key')).toBe('persisted')

    safeRemoveItem('real-key')
    expect(safeGetItem('real-key')).toBeNull()
  })

  it('keeps the in-memory copy when a write fails mid-session', async () => {
    const storage = workingStorage()
    let failWrites = false
    vi.stubGlobal('localStorage', {
      getItem: storage.getItem,
      setItem: (key: string, value: string) => {
        if (failWrites) throw new DOMException('quota exceeded', 'QuotaExceededError')
        storage.setItem(key, value)
      },
      removeItem: storage.removeItem,
    })
    const { safeGetItem, safeSetItem } = await freshSafeStorage()

    safeSetItem('quota-key', 'before-failure')
    failWrites = true
    safeSetItem('quota-key', 'after-failure')

    expect(safeGetItem('quota-key')).toBe('after-failure')
  })
})

// Compile-time check that the dynamic import above matches the public API.
type _apiShape = [
  typeof safeGetItemType,
  typeof safeSetItemType,
  typeof safeRemoveItemType,
]
