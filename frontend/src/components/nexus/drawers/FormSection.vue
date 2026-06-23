<template>
  <v-expansion-panels
    class="nexus-form-section"
    multiple
    :model-value="open"
    @update:model-value="open = ($event as number[])"
  >
    <v-expansion-panel
      :class="{ 'nexus-form-section--invalid': invalid }"
      :value="0"
    >
      <v-expansion-panel-title>
        <v-icon v-if="icon" class="nexus-form-section__icon" :icon="icon" size="20" />
        <span class="nexus-form-section__title">{{ title }}</span>
      </v-expansion-panel-title>
      <v-expansion-panel-text>
        <slot />
      </v-expansion-panel-text>
    </v-expansion-panel>
  </v-expansion-panels>
</template>

<script lang="ts" setup>
import { ref } from 'vue'

const props = withDefaults(defineProps<{
  title: string
  // Optional accent icon shown before the section title (reference form sections
  // use a cyan ⚡/🌐/🔒 glyph). Pass a Lucide name, e.g. "lucide:zap".
  icon?: string
  defaultOpen?: boolean
  invalid?: boolean
}>(), {
  defaultOpen: true,
})

const open = ref<number[]>(props.defaultOpen ? [0] : [])
</script>

<style scoped>
.nexus-form-section {
  margin-block-end: var(--nexus-gap-3);
}

.nexus-form-section :deep(.v-expansion-panel) {
  background: var(--nexus-surface-2);
  border: 1px solid var(--nexus-border);
}

.nexus-form-section--invalid {
  border-color: var(--nexus-status-error);
}

.nexus-form-section__icon {
  color: var(--nexus-accent-primary);
  margin-inline-end: var(--nexus-gap-2);
}

.nexus-form-section__title {
  color: var(--nexus-text-primary);
  font-size: 1rem;
  font-weight: 600;
}
</style>
