// Browsers with storage fully disabled (Chrome "Block all cookies") throw a
// DOMException/SecurityError on any localStorage ACCESS — even a plain
// `localStorage.getItem` in module scope aborts app boot with a blank page.
// Every persisted UI preference must go through these wrappers: once real
// storage misbehaves (access denied, quota exceeded) the session switches to
// an in-memory map, so get-after-set keeps working for its remainder.

const memoryFallback = new Map<string, string>()
let storageBroken = false

const storageAvailable = (): boolean => {
  if (storageBroken) return false
  try {
    return typeof localStorage !== 'undefined' && localStorage !== null
  } catch {
    storageBroken = true
    return false
  }
}

export const safeGetItem = (key: string): string | null => {
  if (storageAvailable()) {
    try {
      return localStorage.getItem(key)
    } catch {
      storageBroken = true
    }
  }
  return memoryFallback.get(key) ?? null
}

export const safeSetItem = (key: string, value: string): void => {
  memoryFallback.set(key, value)
  if (storageAvailable()) {
    try {
      localStorage.setItem(key, value)
    } catch {
      // Quota exceeded or a per-call failure: the in-memory copy above still
      // holds the value, and later reads stay on it for this session.
      storageBroken = true
    }
  }
}

export const safeRemoveItem = (key: string): void => {
  memoryFallback.delete(key)
  if (storageAvailable()) {
    try {
      localStorage.removeItem(key)
    } catch {
      storageBroken = true
    }
  }
}
