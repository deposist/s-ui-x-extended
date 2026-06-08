<template>
  <article class="nexus-overview-panel">
    <panel-header :title="$t('nexus.overview.events.title')">
      <template #action>
        <status-badge :label="stateLabel" :tone="stateTone" />
      </template>
    </panel-header>

    <div v-if="loading" class="nexus-overview-panel__state">
      {{ $t('nexus.overview.events.loading') }}
    </div>

    <div v-else-if="events.length === 0" class="nexus-overview-panel__state">
      {{ emptyCopy }}
    </div>

    <dense-list v-else class="nexus-recent-events__list">
      <li
        v-for="(event, index) in events"
        :key="`${event.id}-${event.timestamp}-${index}`"
      >
        <v-icon :icon="event.icon" :class="`nexus-recent-events__icon--${event.tone}`" />

        <div class="nexus-recent-events__copy">
          <strong>{{ event.text }}</strong>
          <span v-if="event.detail">{{ event.detail }}</span>
        </div>

        <time :datetime="dateTimeValue(event.timestamp)">
          {{ timestampLabel(event.timestamp) }}
        </time>
      </li>
    </dense-list>
  </article>
</template>

<script lang="ts" setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

import DenseList from '@/components/nexus/primitives/DenseList.vue'
import PanelHeader from '@/components/nexus/primitives/PanelHeader.vue'
import StatusBadge from '@/components/nexus/primitives/StatusBadge.vue'
import type { AuditDisplayItem } from './selectors/auditMapper'

const props = defineProps<{
  events: AuditDisplayItem[]
  loading: boolean
  offline: boolean
  unavailable: boolean
}>()

const { t } = useI18n()

const stateLabel = computed(() => {
  if (props.offline) return t('nexus.status.offline')
  if (props.unavailable) return t('nexus.status.unavailable')
  return t('nexus.overview.events.rows', { count: props.events.length })
})

const stateTone = computed(() => {
  if (props.offline) return 'error'
  return props.unavailable ? 'warning' : 'info'
})

const emptyCopy = computed(() => {
  if (props.offline) return t('nexus.overview.events.emptyOffline')
  if (props.unavailable) return t('nexus.overview.events.emptyUnavailable')
  return t('nexus.overview.events.empty')
})

const eventDate = (timestamp: number): Date | undefined => {
  if (!timestamp) return

  const date = new Date(timestamp * 1000)
  return Number.isNaN(date.getTime()) ? undefined : date
}

const timestampLabel = (timestamp: number): string => {
  return eventDate(timestamp)?.toLocaleString() ?? '-'
}

const dateTimeValue = (timestamp: number): string | undefined => {
  return eventDate(timestamp)?.toISOString()
}
</script>

<style scoped>
.nexus-overview-panel {
  background: var(--nexus-surface-1);
  border: 1px solid var(--nexus-border);
  border-radius: var(--nexus-radius-lg);
  display: grid;
  gap: var(--nexus-gap-3);
  min-width: 0;
  padding: var(--nexus-gap-4);
}

.nexus-overview-panel__state {
  align-items: center;
  background: var(--nexus-surface-2);
  border: 1px dashed var(--nexus-border-strong);
  border-radius: var(--nexus-radius-md);
  color: rgb(var(--v-theme-on-surface) / 68%);
  display: grid;
  font-size: 0.86rem;
  letter-spacing: 0;
  line-height: 1.4;
  min-height: auto;
  overflow-wrap: anywhere;
  padding: var(--nexus-gap-3);
  text-align: center;
}

.nexus-recent-events__list :deep(li) {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
}

.nexus-recent-events__copy {
  display: grid;
  gap: 2px;
  min-width: 0;
}

.nexus-recent-events__copy strong,
.nexus-recent-events__copy span,
.nexus-recent-events__list time {
  font-size: 0.78rem;
  letter-spacing: 0;
  line-height: 1.35;
  min-width: 0;
  overflow-wrap: anywhere;
}

.nexus-recent-events__copy span,
.nexus-recent-events__list time {
  color: rgb(var(--v-theme-on-surface) / 62%);
}

.nexus-recent-events__list time {
  margin-inline-start: auto;
  text-align: end;
  white-space: nowrap;
}

.nexus-recent-events__icon--info {
  color: var(--nexus-status-info);
}

.nexus-recent-events__icon--success {
  color: var(--nexus-status-success);
}

.nexus-recent-events__icon--warning {
  color: var(--nexus-status-warn);
}

.nexus-recent-events__icon--error {
  color: var(--nexus-status-error);
}

@media (min-width: 600px) {
  .nexus-recent-events__copy {
    align-items: baseline;
    display: flex;
    gap: var(--nexus-gap-2);
  }

  .nexus-recent-events__copy strong {
    flex: 0 0 auto;
    white-space: nowrap;
  }

  .nexus-recent-events__copy span {
    flex: 1 1 auto;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .nexus-recent-events__copy span::before {
    border-inline-start: 1px solid var(--nexus-border-strong);
    content: '';
    margin-inline-end: var(--nexus-gap-2);
  }
}

@media (max-width: 600px) {
  .nexus-recent-events__list :deep(li) {
    grid-template-columns: auto minmax(0, 1fr);
  }

  .nexus-recent-events__list time {
    grid-column: 2;
    margin-inline-start: 0;
    text-align: start;
  }
}
</style>
