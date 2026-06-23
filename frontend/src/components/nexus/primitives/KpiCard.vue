<template>
  <article class="nexus-kpi-card">
    <div class="nexus-kpi-card__header">
      <span class="nexus-kpi-card__label">{{ label }}</span>
      <div v-if="$slots.meta" class="nexus-kpi-card__meta">
        <slot name="meta" />
      </div>
    </div>

    <div class="nexus-kpi-card__summary">
      <strong class="nexus-kpi-card__value">{{ value }}</strong>
      <span v-if="delta" class="nexus-kpi-card__delta">{{ delta }}</span>
    </div>

    <div class="nexus-kpi-card__trend">
      <slot name="trend" />
    </div>
  </article>
</template>

<script lang="ts" setup>
defineProps<{
  label: string
  value: string
  delta?: string
}>()
</script>

<style scoped>
.nexus-kpi-card {
  background: var(--nexus-surface-1);
  border: 1px solid var(--nexus-border);
  border-radius: var(--nexus-radius-lg);
  display: grid;
  gap: var(--nexus-gap-3);
  min-height: 144px;
  min-width: 0;
  padding: var(--nexus-gap-4);
}

.nexus-kpi-card__header {
  align-items: start;
  display: flex;
  gap: var(--nexus-gap-3);
  justify-content: space-between;
  min-width: 0;
}

.nexus-kpi-card__label {
  color: rgb(var(--v-theme-on-surface) / 68%);
  flex: 1 1 auto;
  font-size: 0.76rem;
  font-weight: 600;
  letter-spacing: 0;
  line-height: 1.3;
  min-width: 0;
  overflow-wrap: anywhere;
}

.nexus-kpi-card__meta {
  flex: 0 1 auto;
  max-inline-size: min(100%, 320px);
  min-inline-size: 0;
}

.nexus-kpi-card__summary {
  align-items: baseline;
  display: flex;
  flex-wrap: wrap;
  gap: var(--nexus-gap-2);
  min-width: 0;
}

.nexus-kpi-card__value {
  font-size: 1.7rem;
  font-weight: 700;
  letter-spacing: 0;
  line-height: 1.1;
  min-width: 0;
  overflow-wrap: anywhere;
}

.nexus-kpi-card__delta {
  background: var(--nexus-surface-2);
  border: 1px solid var(--nexus-border);
  border-radius: var(--nexus-radius-sm);
  color: var(--nexus-accent-secondary);
  font-size: 0.74rem;
  font-weight: 600;
  letter-spacing: 0;
  line-height: 1.25;
  max-width: 100%;
  overflow-wrap: anywhere;
  padding: 2px var(--nexus-gap-1);
}

.nexus-kpi-card__trend {
  align-items: end;
  display: grid;
  min-height: 40px;
  min-width: 0;
}

@media (max-width: 600px) {
  .nexus-kpi-card__header {
    align-items: stretch;
    flex-direction: column;
  }

  .nexus-kpi-card__meta {
    max-inline-size: 100%;
  }
}
</style>
