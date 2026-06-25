import { readonly, ref } from 'vue'

export const UI_PALETTES = ['technical', 'navy', 'emerald', 'dracula'] as const

export type UiPalette = (typeof UI_PALETTES)[number]

export const DEFAULT_UI_PALETTE: UiPalette = 'technical'
export const UI_PALETTE_KEY = 'sui:ui:palette'

export const isUiPalette = (value: unknown): value is UiPalette =>
  typeof value === 'string' && UI_PALETTES.some(palette => palette === value)

const readPersisted = (): UiPalette => {
  try {
    const raw = localStorage.getItem(UI_PALETTE_KEY)

    return isUiPalette(raw) ? raw : DEFAULT_UI_PALETTE
  } catch {
    return DEFAULT_UI_PALETTE
  }
}

const persisted = ref<UiPalette>(readPersisted())

const setPalette = (next: UiPalette): void => {
  if (!isUiPalette(next)) return

  persisted.value = next

  try {
    localStorage.setItem(UI_PALETTE_KEY, next)
  } catch {
    // Keep the reactive preference when storage is unavailable.
  }
}

export const useUiPalette = () => ({
  palette: readonly(persisted),
  setPalette,
})
