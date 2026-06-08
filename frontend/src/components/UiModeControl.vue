<template>
  <v-btn
    v-if="enabled && variant === 'quick'"
    icon
    :aria-label="quickLabel"
    :title="quickLabel"
    variant="text"
    @click="toggleMode"
  >
    <v-icon :icon="quickIcon" />
  </v-btn>

  <v-select
    v-else-if="enabled"
    :items="modeItems"
    :label="t('nexus.mode.label')"
    :model-value="mode"
    hide-details
    item-title="title"
    item-value="value"
    @update:model-value="selectMode"
  />
</template>

<script lang="ts" setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

import { isNexusEnabled } from '@/uiMode/featureGate'
import { DEFAULT_UI_MODE, isUiMode, type UiMode } from '@/uiMode/types'
import { useUiMode } from '@/uiMode/useUiMode'

withDefaults(defineProps<{
  variant?: 'quick' | 'select'
}>(), {
  variant: 'quick',
})

const { t } = useI18n()
const { mode, setMode } = useUiMode()
const enabled = isNexusEnabled()
const nexusMode: UiMode = 'nexus'

const modeItems = computed(() => [
  { title: t('nexus.mode.options.classic'), value: DEFAULT_UI_MODE },
  { title: t('nexus.mode.options.nexus'), value: nexusMode },
])

const nextMode = computed<UiMode>(() => mode.value === nexusMode ? DEFAULT_UI_MODE : nexusMode)
const quickIcon = computed(() =>
  mode.value === nexusMode ? 'mdi-view-dashboard-outline' : 'mdi-view-dashboard-edit-outline',
)
const quickLabel = computed(() =>
  t('nexus.mode.switchTo', { mode: t(`nexus.mode.options.${nextMode.value}`) }),
)

const toggleMode = () => setMode(nextMode.value)

const selectMode = (value: unknown) => {
  if (isUiMode(value)) {
    setMode(value)
  }
}
</script>
