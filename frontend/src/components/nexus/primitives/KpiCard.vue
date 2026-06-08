<template>
  <article class="nexus-kpi-card">
    <span class="nexus-kpi-card__label">{{ label }}</span>

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

.nexus-kpi-card__label {
  color: rgb(var(--v-theme-on-surface) / 68%);
  font-size: 0.76rem;
  font-weight: 600;
  letter-spacing: 0;
  line-height: 1.3;
  overflow-wrap: anywhere;
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
</style>
