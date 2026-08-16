import { createI18n } from 'vue-i18n'
import { safeGetItem, safeSetItem } from '@/utils/safeStorage'

type LocaleCode = 'en' | 'fa' | 'vi' | 'zhHans' | 'zhHant' | 'ru'
type LocaleMessages = Record<string, unknown>

const DEFAULT_LOCALE: LocaleCode = 'en'

const localeLoaders: Record<LocaleCode, () => Promise<{ default: LocaleMessages }>> = {
  en: () => import('./en'),
  fa: () => import('./fa'),
  vi: () => import('./vi'),
  zhHans: () => import('./zhcn'),
  zhHant: () => import('./zhtw'),
  ru: () => import('./ru'),
}

const supportedLocales = new Set<LocaleCode>(Object.keys(localeLoaders) as LocaleCode[])
const loadedLocales = new Set<LocaleCode>()

const normalizeLocale = (value?: string | null): LocaleCode => {
  if (value && supportedLocales.has(value as LocaleCode)) {
    return value as LocaleCode
  }
  return DEFAULT_LOCALE
}

const storedLocale = () => normalizeLocale(safeGetItem('locale'))

const initialLocale = storedLocale()

export const i18n = createI18n({
  legacy: false,
  locale: initialLocale,
  fallbackLocale: DEFAULT_LOCALE,
  messages: {},
})

const loadMessages = async (localeCode: LocaleCode) => {
  if (loadedLocales.has(localeCode)) {
    return
  }
  const messages = await localeLoaders[localeCode]()
  i18n.global.setLocaleMessage(localeCode, messages.default)
  loadedLocales.add(localeCode)
}

export const loadLocaleMessages = async (localeCode: string) => {
  const normalized = normalizeLocale(localeCode)
  await loadMessages(DEFAULT_LOCALE)
  if (normalized !== DEFAULT_LOCALE) {
    await loadMessages(normalized)
  }
  return normalized
}

export const loadInitialLocaleMessages = () => loadLocaleMessages(initialLocale)

export const setI18nLocale = async (localeCode: string) => {
  const normalized = await loadLocaleMessages(localeCode)
  i18n.global.locale.value = normalized
  safeSetItem('locale', normalized)
  applyDocumentLocale()
  return normalized
}

// Maps an internal locale code to a BCP47 tag usable outside vue-i18n
// (Vuetify locale names, document.documentElement.lang).
const bcp47Locale = (value: LocaleCode): string => {
  switch (value) {
    case 'zhHans':
      return 'zh-cn'
    case 'zhHant':
      return 'zh-tw'
    default:
      return value
  }
}

// Keeps <html lang> in sync with the active locale so screen readers pick the
// right pronunciation; index.html ships a neutral "en" until this runs.
export const applyDocumentLocale = () => {
  if (typeof document === 'undefined') return
  document.documentElement.lang = bcp47Locale(normalizeLocale(i18n.global.locale.value))
}

export const locale = bcp47Locale(initialLocale)

export const languages = [
  { title: 'English', value: 'en' },
  { title: 'فارسی', value: 'fa' },
  { title: 'Tiếng Việt', value: 'vi' },
  { title: '简体中文', value: 'zhHans' },
  { title: '繁體中文', value: 'zhHant' },
  { title: 'Русский', value: 'ru' },
]
