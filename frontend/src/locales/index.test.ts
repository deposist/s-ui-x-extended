import { beforeEach, describe, expect, it, vi } from 'vitest'

const storage = new Map<string, string>()

const stubLocalStorage = () => {
  vi.stubGlobal('localStorage', {
    getItem: (key: string) => storage.get(key) ?? null,
    setItem: (key: string, value: string) => storage.set(key, value),
    removeItem: (key: string) => storage.delete(key),
    clear: () => storage.clear(),
  })
}

describe('locale loading', () => {
  beforeEach(() => {
    storage.clear()
    vi.resetModules()
    vi.unstubAllGlobals()
    stubLocalStorage()
  })

  it('loads only default messages on default startup', async () => {
    const { i18n, loadInitialLocaleMessages } = await import('./index')

    await loadInitialLocaleMessages()

    expect(i18n.global.availableLocales).toContain('en')
    expect(i18n.global.availableLocales).not.toContain('ru')
  })

  it('loads stored locale with english fallback on startup', async () => {
    storage.set('locale', 'ru')
    const { i18n, loadInitialLocaleMessages } = await import('./index')

    await loadInitialLocaleMessages()

    expect(i18n.global.availableLocales).toEqual(expect.arrayContaining(['en', 'ru']))
    expect(i18n.global.availableLocales).not.toContain('fa')
  })

  it('loads and stores locales when changed', async () => {
    const { i18n, setI18nLocale } = await import('./index')

    const selectedLocale = await setI18nLocale('zhHans')

    expect(selectedLocale).toBe('zhHans')
    expect(storage.get('locale')).toBe('zhHans')
    expect(i18n.global.locale.value).toBe('zhHans')
    expect(i18n.global.availableLocales).toEqual(expect.arrayContaining(['en', 'zhHans']))
  })

  it('falls back to english for unsupported locales', async () => {
    const { i18n, setI18nLocale } = await import('./index')

    const selectedLocale = await setI18nLocale('missing')

    expect(selectedLocale).toBe('en')
    expect(storage.get('locale')).toBe('en')
    expect(i18n.global.locale.value).toBe('en')
    expect(i18n.global.availableLocales).toEqual(['en'])
  })
})
