import { isNexusEnabled } from './featureGate'
import { DEFAULT_UI_PALETTE, UI_PALETTE_KEY, isUiPalette, type UiPalette } from './palette'
import { DEFAULT_UI_MODE, UI_MODE_KEY, isUiMode, type UiMode } from './types'

const applyUiMode = (): void => {
  const mode: UiMode = isNexusEnabled() ? DEFAULT_UI_MODE : 'classic'

  if (isNexusEnabled()) {
    try {
      const raw = localStorage.getItem(UI_MODE_KEY)

      if (raw !== DEFAULT_UI_MODE) {
        localStorage.setItem(UI_MODE_KEY, DEFAULT_UI_MODE)
      }
    } catch {
      // Keep the default when storage is unavailable.
    }
  }

  document.documentElement.dataset.uiMode = mode
}

const applyUiPalette = (): void => {
  // Set the palette attribute before mount so the Nexus token blocks pick the
  // right surfaces/accent immediately and avoid a flash of the wrong palette.
  let palette: UiPalette = DEFAULT_UI_PALETTE

  try {
    const raw = localStorage.getItem(UI_PALETTE_KEY)

    if (isUiPalette(raw)) palette = raw
  } catch {
    // Keep the default when storage is unavailable.
  }

  document.documentElement.dataset.uiPalette = palette
}

applyUiMode()
applyUiPalette()
