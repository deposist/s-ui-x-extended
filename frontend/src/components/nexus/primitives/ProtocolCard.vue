<template>
  <article class="nexus-protocol-card">
    <header class="nexus-protocol-card__header">
      <h3 class="nexus-protocol-card__type">{{ type }}</h3>
      <status-badge :label="status" :tone="statusTone" />
    </header>

    <div class="nexus-protocol-card__metrics">
      <strong class="nexus-protocol-card__connections">{{ connCount }}</strong>
      <span class="nexus-protocol-card__totals">{{ totals }}</span>
    </div>
  </article>
</template>

<script lang="ts" setup>
import StatusBadge from './StatusBadge.vue'

type ProtocolStatusTone = 'info' | 'success' | 'warning' | 'error'

withDefaults(defineProps<{
  type: string
  status: string
  connCount: number | string
  totals: string
  statusTone?: ProtocolStatusTone
}>(), {
  statusTone: 'info',
})
</script>

<style scoped>
.nexus-protocol-card {
  background: var(--nexus-surface-1);
  border: 1px solid var(--nexus-border);
  border-radius: var(--nexus-radius-md);
  display: grid;
  gap: var(--nexus-gap-3);
  min-height: 112px;
  min-width: 0;
  padding: var(--nexus-gap-3);
}

.nexus-protocol-card__header {
  align-items: flex-start;
  display: flex;
  flex-wrap: wrap;
  gap: var(--nexus-gap-2);
  justify-content: space-between;
  min-width: 0;
}

.nexus-protocol-card__type {
  font-size: 0.94rem;
  font-weight: 650;
  letter-spacing: 0;
  line-height: 1.25;
  margin: 0;
  min-width: 0;
  overflow-wrap: anywhere;
}

.nexus-protocol-card__metrics {
  align-items: end;
  display: grid;
  gap: var(--nexus-gap-1);
  min-width: 0;
}

.nexus-protocol-card__connections {
  font-size: 1.3rem;
  font-weight: 700;
  letter-spacing: 0;
  line-height: 1.15;
  min-width: 0;
  overflow-wrap: anywhere;
}

.nexus-protocol-card__totals {
  color: rgb(var(--v-theme-on-surface) / 68%);
  font-size: 0.78rem;
  letter-spacing: 0;
  line-height: 1.3;
  min-width: 0;
  overflow-wrap: anywhere;
}
</style>
