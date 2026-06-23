<template>
  <div class="nexus-toolbar">
    <div v-if="$slots.filters" class="nexus-toolbar__filters">
      <slot name="filters" />
    </div>

    <div v-if="$slots['secondary-actions']" class="nexus-toolbar__secondary-actions">
      <slot name="secondary-actions" />
    </div>

    <div v-if="$slots['primary-actions']" class="nexus-toolbar__primary-actions">
      <slot name="primary-actions" />
    </div>

    <div v-if="$slots.actions" class="nexus-toolbar__actions">
      <slot name="actions" />
    </div>
  </div>
</template>

<script lang="ts" setup>
// Layout shell for the content row above a table. Search lives solely in the
// topbar (PageHeader); primary actions get a stable mobile row.
</script>

<style scoped>
.nexus-toolbar {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: var(--nexus-gap-2);
  margin-block-end: var(--nexus-gap-4);
}

.nexus-toolbar__filters,
.nexus-toolbar__secondary-actions,
.nexus-toolbar__primary-actions,
.nexus-toolbar__actions {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: var(--nexus-gap-2);
}

.nexus-toolbar__filters + .nexus-toolbar__primary-actions,
.nexus-toolbar__primary-actions:first-child,
.nexus-toolbar__secondary-actions,
.nexus-toolbar__actions {
  margin-inline-start: auto;
}

@media (max-width: 600px) {
  .nexus-toolbar {
    align-items: stretch;
  }

  .nexus-toolbar__filters,
  .nexus-toolbar__secondary-actions,
  .nexus-toolbar__primary-actions,
  .nexus-toolbar__actions {
    flex: 1 1 100%;
    margin-inline-start: 0;
  }

  .nexus-toolbar__primary-actions {
    order: -1;
  }

  .nexus-toolbar__primary-actions :deep(.v-btn) {
    flex: 1 1 auto;
  }
}
</style>
