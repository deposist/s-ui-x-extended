import { computed, readonly, ref } from 'vue'

import { isNexusEnabled } from './featureGate'
import { DEFAULT_UI_MODE, UI_MODE_KEY, isUiMode, type UiMode } from './types'

const readPersisted = (): UiMode => {
  try {
    const raw = localStorage.getItem(UI_MODE_KEY)

    return isUiMode(raw) ? raw : DEFAULT_UI_MODE
  } catch {
    return DEFAULT_UI_MODE
  }
}

const persisted = ref<UiMode>(readPersisted())

const effective = computed<UiMode>(() =>
  isNexusEnabled() ? persisted.value : DEFAULT_UI_MODE,
)

const setMode = (next: UiMode): void => {
  if (!isUiMode(next)) return

  persisted.value = next

  try {
    localStorage.setItem(UI_MODE_KEY, next)
  } catch {
    // Keep the reactive preference when storage is unavailable.
  }
}

export const useUiMode = () => ({
  mode: readonly(effective),
  persisted: readonly(persisted),
  setMode,
})
