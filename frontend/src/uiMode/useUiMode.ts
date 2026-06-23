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

// When the Nexus feature gate is off we force the literal 'classic' rather
// than DEFAULT_UI_MODE: the default is now 'nexus', so deriving from it would
// keep Nexus active even when the gate is meant to disable it.
const effective = computed<UiMode>(() =>
  isNexusEnabled() ? persisted.value : 'classic',
)

const syncDocumentUiMode = (next: UiMode): void => {
  if (typeof document === 'undefined') return

  document.documentElement.dataset.uiMode = next
}

const setMode = (next: UiMode): void => {
  if (!isUiMode(next)) return

  persisted.value = next
  syncDocumentUiMode(effective.value)

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
