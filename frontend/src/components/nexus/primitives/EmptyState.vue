<template>
  <div class="nexus-empty" :class="{ 'nexus-empty--compact': compact }">
    <v-icon class="nexus-empty__icon" :icon="icon" :size="compact ? 18 : 48" />
    <p class="nexus-empty__title">{{ title }}</p>
    <p v-if="description" class="nexus-empty__description">{{ description }}</p>
    <div v-if="$slots.action" class="nexus-empty__action">
      <slot name="action" />
    </div>
  </div>
</template>

<script lang="ts" setup>
withDefaults(defineProps<{
  title: string
  description?: string
  icon?: string
  // Compact = a quiet, left-aligned single muted line for an empty sub-section
  // on a page that stacks multiple tables (Rules/DNS), so it doesn't read like
  // the whole page is empty. Default false keeps the big centered full-page state.
  compact?: boolean
}>(), {
  icon: 'lucide:inbox',
  compact: false,
})
</script>

<style scoped>
.nexus-empty {
  align-items: center;
  display: flex;
  flex-direction: column;
  gap: var(--nexus-gap-2);
  padding-block: var(--nexus-gap-6);
  padding-inline: var(--nexus-gap-4);
  text-align: center;
}

.nexus-empty__icon {
  color: var(--nexus-text-muted);
  margin-block-end: var(--nexus-gap-1);
}

.nexus-empty__title {
  color: var(--nexus-text-primary);
  font-size: 0.95rem;
  font-weight: 600;
}

.nexus-empty__description {
  color: var(--nexus-text-secondary);
  font-size: 0.825rem;
  max-width: 42ch;
}

.nexus-empty__action {
  margin-block-start: var(--nexus-gap-2);
}

/* Compact: a quiet muted line under a section header, not a centered hero. */
.nexus-empty--compact {
  align-items: center;
  flex-direction: row;
  gap: var(--nexus-gap-2);
  justify-content: flex-start;
  padding-block: var(--nexus-gap-2);
  padding-inline: var(--nexus-gap-1);
  text-align: start;
}

.nexus-empty--compact .nexus-empty__icon {
  margin-block-end: 0;
}

.nexus-empty--compact .nexus-empty__title {
  color: var(--nexus-text-muted);
  font-size: 0.8rem;
  font-weight: 400;
}
</style>
