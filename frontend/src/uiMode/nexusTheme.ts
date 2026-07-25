import { computed, watch } from 'vue'
import type { ComputedRef } from 'vue'
import { useTheme } from 'vuetify'

import type { UiPalette } from './palette'
import { useUiPalette } from './palette'

export const resolveNexusThemeName = (
  activeThemeName: string,
  currentIsDark: boolean,
  palette: UiPalette,
): string => {
  const systemIsDark = activeThemeName === 'system' && currentIsDark
  const isDark = activeThemeName === 'dark' || systemIsDark

  if (palette === 'navy') return isDark ? 'nexusDark' : 'nexusLight'
  if (palette === 'emerald') return isDark ? 'emeraldDark' : 'emeraldLight'
  if (palette === 'dracula') return isDark ? 'draculaDark' : 'draculaLight'

  return isDark ? 'technicalDark' : 'technicalLight'
}

export const useNexusTheme = (): ComputedRef<string> => {
  const theme = useTheme()
  const { palette } = useUiPalette()

  watch(palette, (next) => {
    if (typeof document !== 'undefined') {
      document.documentElement.dataset.uiPalette = next
    }
  }, { immediate: true })

  return computed(() => resolveNexusThemeName(
    theme.global.name.value,
    theme.global.current.value.dark,
    palette.value,
  ))
}
