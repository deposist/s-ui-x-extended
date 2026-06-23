<template>
  <div class="nexus-skeleton" :aria-label="$t('loading')" role="status">
    <div v-for="row in rows" :key="row" class="nexus-skeleton__row">
      <span
        v-for="col in columns"
        :key="col"
        class="nexus-skeleton__cell"
      />
    </div>
  </div>
</template>

<script lang="ts" setup>
withDefaults(defineProps<{
  rows?: number
  columns?: number
}>(), {
  rows: 5,
  columns: 5,
})
</script>

<style scoped>
.nexus-skeleton {
  display: flex;
  flex-direction: column;
}

.nexus-skeleton__row {
  align-items: center;
  border-block-end: 1px solid var(--nexus-border);
  display: grid;
  gap: var(--nexus-gap-3);
  grid-template-columns: repeat(v-bind(columns), 1fr);
  height: 44px;
  padding-inline: var(--nexus-gap-3);
}

.nexus-skeleton__row:last-child {
  border-block-end: 0;
}

.nexus-skeleton__cell {
  background: linear-gradient(
    90deg,
    var(--nexus-elevated) 25%,
    var(--nexus-surface-hover) 37%,
    var(--nexus-elevated) 63%
  );
  background-size: 400% 100%;
  border-radius: var(--nexus-radius-sm);
  height: 12px;
  width: 100%;
  animation: nexus-skeleton-shimmer 1.4s ease infinite;
}

@keyframes nexus-skeleton-shimmer {
  0% {
    background-position: 100% 50%;
  }

  100% {
    background-position: 0 50%;
  }
}

@media (prefers-reduced-motion: reduce) {
  .nexus-skeleton__cell {
    animation: none;
  }
}
</style>
