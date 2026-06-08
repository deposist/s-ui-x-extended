<template>
  <component :is="ShellComponent">
    <router-view />
  </component>
</template>

<script lang="ts" setup>
import { computed, defineAsyncComponent } from 'vue'

import { useUiMode } from '@/uiMode/useUiMode'

const ClassicShell = defineAsyncComponent(
  () => import('@/layouts/default/Default.vue'),
)
const NexusShell = defineAsyncComponent(
  () => import('@/layouts/nexus/NexusShell.vue'),
)

const { mode } = useUiMode()
const ShellComponent = computed(() =>
  mode.value === 'nexus' ? NexusShell : ClassicShell,
)
</script>
