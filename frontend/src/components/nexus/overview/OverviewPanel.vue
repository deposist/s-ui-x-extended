<template>
  <article class="nexus-overview-panel">
    <panel-header :title="title">
      <template #range-tabs>
        <slot name="range-tabs" />
      </template>
      <template #action>
        <slot name="action" />
      </template>
    </panel-header>

    <slot />
  </article>
</template>

<script lang="ts" setup>
import PanelHeader from '@/components/nexus/primitives/PanelHeader.vue'

defineProps<{
  title: string
}>()
</script>

<style scoped>
.nexus-overview-panel {
  background: var(--nexus-surface-1);
  border: 1px solid var(--nexus-border);
  border-radius: var(--nexus-radius-lg);
  display: flex;
  flex-direction: column;
  gap: var(--nexus-gap-3);
  min-width: 0;
  padding: var(--nexus-gap-4);
  height: 100%;
}
</style>
