import { isNexusEnabled } from './featureGate'
import { DEFAULT_UI_MODE, UI_MODE_KEY, isUiMode, type UiMode } from './types'

const applyUiMode = (): void => {
  let mode: UiMode = DEFAULT_UI_MODE

  if (isNexusEnabled()) {
    try {
      const raw = localStorage.getItem(UI_MODE_KEY)

      if (isUiMode(raw)) mode = raw
    } catch {
      // Keep the default when storage is unavailable.
    }
  }

  document.documentElement.dataset.uiMode = mode
}

applyUiMode()
