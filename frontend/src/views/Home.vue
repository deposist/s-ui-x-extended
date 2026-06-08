<template>
  <component :is="HomeVariant" />
</template>

<script lang="ts" setup>
import { computed, defineAsyncComponent } from 'vue'

import ClassicMain from '@/components/Main.vue'
import { useUiMode } from '@/uiMode/useUiMode'

const NexusOverview = defineAsyncComponent(
  () => import('@/views/nexus/Overview.vue'),
)

const { mode } = useUiMode()
const HomeVariant = computed(() =>
  mode.value === 'nexus' ? NexusOverview : ClassicMain,
)
</script>
