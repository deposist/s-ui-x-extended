import { computed, readonly, ref } from 'vue'

import { safeGetItem, safeSetItem } from '@/utils/safeStorage'

import { isNexusEnabled } from './featureGate'
import { DEFAULT_UI_MODE, UI_MODE_KEY, type UiMode } from './types'

const readPersisted = (): UiMode => {
  const raw = safeGetItem(UI_MODE_KEY)

  return raw === DEFAULT_UI_MODE ? DEFAULT_UI_MODE : DEFAULT_UI_MODE
}

const persisted = ref<UiMode>(readPersisted())

// When the Nexus feature gate is off we force the literal 'classic'. When the
// gate is on, Nexus is the only selectable mode and the persisted value is
// normalized to the Nexus default.
const effective = computed<UiMode>(() =>
  isNexusEnabled() ? DEFAULT_UI_MODE : 'classic',
)

const syncDocumentUiMode = (next: UiMode): void => {
  if (typeof document === 'undefined') return

  document.documentElement.dataset.uiMode = next
}

const setMode = (_next: UiMode): void => {
  persisted.value = DEFAULT_UI_MODE
  syncDocumentUiMode(effective.value)
  safeSetItem(UI_MODE_KEY, DEFAULT_UI_MODE)
}

export const useUiMode = () => ({
  mode: readonly(effective),
  persisted: readonly(persisted),
  setMode,
})
