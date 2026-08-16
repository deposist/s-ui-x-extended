import { readonly, ref } from 'vue'

import { safeGetItem, safeSetItem } from '@/utils/safeStorage'

export const UI_PALETTES = ['technical', 'navy', 'emerald', 'dracula'] as const

export type UiPalette = (typeof UI_PALETTES)[number]

export const DEFAULT_UI_PALETTE: UiPalette = 'technical'
export const UI_PALETTE_KEY = 'sui:ui:palette'

export const isUiPalette = (value: unknown): value is UiPalette =>
  typeof value === 'string' && UI_PALETTES.some(palette => palette === value)

const readPersisted = (): UiPalette => {
  const raw = safeGetItem(UI_PALETTE_KEY)

  return isUiPalette(raw) ? raw : DEFAULT_UI_PALETTE
}

const persisted = ref<UiPalette>(readPersisted())

const setPalette = (next: UiPalette): void => {
  if (!isUiPalette(next)) return

  persisted.value = next
  safeSetItem(UI_PALETTE_KEY, next)
}

export const useUiPalette = () => ({
  palette: readonly(persisted),
  setPalette,
})
