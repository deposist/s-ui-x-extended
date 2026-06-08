import { beforeEach, describe, expect, it, vi } from 'vitest'

import { UI_MODE_KEY } from './types'

const storage = new Map<string, string>()
const getItem = vi.fn((key: string) => storage.get(key) ?? null)
const setItem = vi.fn((key: string, value: string) => {
  storage.set(key, value)
})

const stubLocalStorage = () => {
  vi.stubGlobal('localStorage', {
    getItem,
    setItem,
  })
}

describe('useUiMode', () => {
  beforeEach(() => {
    storage.clear()
    getItem.mockClear()
    setItem.mockClear()
    vi.resetModules()
    vi.unstubAllEnvs()
    vi.unstubAllGlobals()
    stubLocalStorage()
  })

  it('returns classic for a missing key without writing the default', async () => {
    const { useUiMode } = await import('./useUiMode')

    expect(useUiMode().mode.value).toBe('classic')
    expect(getItem).toHaveBeenCalledWith(UI_MODE_KEY)
    expect(setItem).not.toHaveBeenCalled()
  })

  it('persists a valid value', async () => {
    const { useUiMode } = await import('./useUiMode')

    useUiMode().setMode('nexus')

    expect(storage.get(UI_MODE_KEY)).toBe('nexus')
    expect(setItem).toHaveBeenCalledWith(UI_MODE_KEY, 'nexus')
  })

  it('falls back to classic for an invalid stored value', async () => {
    storage.set(UI_MODE_KEY, 'NEXUS')
    const { useUiMode } = await import('./useUiMode')

    const { mode, persisted } = useUiMode()

    expect(mode.value).toBe('classic')
    expect(persisted.value).toBe('classic')
  })

  it('updates reactively when mode changes', async () => {
    const { useUiMode } = await import('./useUiMode')

    const first = useUiMode()
    const second = useUiMode()

    first.setMode('nexus')

    expect(first.mode.value).toBe('nexus')
    expect(second.mode.value).toBe('nexus')
    expect(second.persisted.value).toBe('nexus')
  })

  it('keeps the in-memory mode when storage persistence throws', async () => {
    setItem.mockImplementationOnce(() => {
      throw new Error('storage unavailable')
    })
    const { useUiMode } = await import('./useUiMode')

    const { mode, persisted, setMode } = useUiMode()

    setMode('nexus')

    expect(mode.value).toBe('nexus')
    expect(persisted.value).toBe('nexus')
  })

  it('forces classic behind the gate without erasing stored nexus', async () => {
    storage.set(UI_MODE_KEY, 'nexus')
    vi.stubEnv('VITE_ENABLE_NEXUS', 'false')
    const { useUiMode } = await import('./useUiMode')

    const { mode, persisted } = useUiMode()

    expect(mode.value).toBe('classic')
    expect(persisted.value).toBe('nexus')
    expect(storage.get(UI_MODE_KEY)).toBe('nexus')
    expect(setItem).not.toHaveBeenCalled()
  })
})
