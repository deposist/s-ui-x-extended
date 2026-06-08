export const UI_MODES = ['classic', 'nexus'] as const

export type UiMode = (typeof UI_MODES)[number]

export const DEFAULT_UI_MODE: UiMode = 'classic'
export const UI_MODE_KEY = 'sui:ui:mode'

export const isUiMode = (value: unknown): value is UiMode =>
  typeof value === 'string' && UI_MODES.some(mode => mode === value)
