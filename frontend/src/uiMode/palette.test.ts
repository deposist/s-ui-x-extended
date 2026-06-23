import { beforeEach, describe, expect, it, vi } from 'vitest'

import { UI_PALETTE_KEY } from './palette'

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

describe('useUiPalette', () => {
  beforeEach(() => {
    storage.clear()
    getItem.mockClear()
    setItem.mockClear()
    vi.resetModules()
    vi.unstubAllGlobals()
    stubLocalStorage()
  })

  it('defaults to technical for a missing key without writing the default', async () => {
    const { useUiPalette } = await import('./palette')

    expect(useUiPalette().palette.value).toBe('technical')
    expect(getItem).toHaveBeenCalledWith(UI_PALETTE_KEY)
    expect(setItem).not.toHaveBeenCalled()
  })

  it('reads a persisted palette', async () => {
    storage.set(UI_PALETTE_KEY, 'navy')
    const { useUiPalette } = await import('./palette')

    expect(useUiPalette().palette.value).toBe('navy')
  })

  it('falls back to technical for an invalid stored value', async () => {
    storage.set(UI_PALETTE_KEY, 'NAVY')
    const { useUiPalette } = await import('./palette')

    expect(useUiPalette().palette.value).toBe('technical')
  })

  it('persists a valid value', async () => {
    const { useUiPalette } = await import('./palette')

    useUiPalette().setPalette('navy')

    expect(storage.get(UI_PALETTE_KEY)).toBe('navy')
    expect(setItem).toHaveBeenCalledWith(UI_PALETTE_KEY, 'navy')
  })

  it('updates reactively across consumers', async () => {
    const { useUiPalette } = await import('./palette')

    const first = useUiPalette()
    const second = useUiPalette()

    first.setPalette('navy')

    expect(first.palette.value).toBe('navy')
    expect(second.palette.value).toBe('navy')
  })

  it('keeps the in-memory palette when storage persistence throws', async () => {
    setItem.mockImplementationOnce(() => {
      throw new Error('storage unavailable')
    })
    const { useUiPalette } = await import('./palette')

    const { palette, setPalette } = useUiPalette()

    setPalette('navy')

    expect(palette.value).toBe('navy')
  })

  it('ignores invalid palette values passed to setPalette', async () => {
    const { useUiPalette } = await import('./palette')

    const { palette, setPalette } = useUiPalette()

    setPalette('teal' as never)

    expect(palette.value).toBe('technical')
    expect(setItem).not.toHaveBeenCalled()
  })
})
