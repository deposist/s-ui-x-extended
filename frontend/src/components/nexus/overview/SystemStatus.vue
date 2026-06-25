<template>
  <overview-panel class="nexus-system-status" :title="$t('nexus.overview.system.title')">
      <template #action>
        <status-badge :label="statusLabel" :tone="statusTone" />
      </template>

    <dense-list class="nexus-system-status__list">
      <li class="nexus-system-status__item">
        <span class="nexus-system-status__key">S-UI-X</span>
        <strong dir="ltr" class="nexus-mono">{{ status.appVersion || '-' }}</strong>
      </li>
      <li class="nexus-system-status__item">
        <span class="nexus-system-status__key">sing-box</span>
        <span class="nexus-system-status__value">
          <status-badge
            :label="status.singboxRunning ? $t('nexus.status.running') : $t('nexus.status.notRunning')"
            :tone="status.singboxRunning ? 'success' : 'error'"
            class="me-1"
          />
          <span v-if="status.singboxVersion" dir="ltr" class="nexus-mono">
            {{ status.singboxVersion }}
          </span>
        </span>
      </li>
      <li class="nexus-system-status__item">
        <span class="nexus-system-status__key">{{ $t('nexus.overview.system.hostUptime') }}</span>
        <strong class="nexus-mono">{{ formatOverviewDuration(status.uptimeSec) }}</strong>
      </li>
      <li class="nexus-system-status__item">
        <span class="nexus-system-status__key">{{ $t('nexus.overview.system.singboxUptime') }}</span>
        <strong class="nexus-mono">{{ formatOverviewDuration(status.singboxUptimeSec) }}</strong>
      </li>
      <li class="nexus-system-status__item">
        <span class="nexus-system-status__key">{{ $t('nexus.overview.system.cpu') }}</span>
        <strong class="nexus-mono">{{ formatOverviewPercent(metrics.cpuPercent) }}</strong>
      </li>
      <li class="nexus-system-status__item">
        <span class="nexus-system-status__key">{{ $t('nexus.overview.system.memory') }}</span>
        <span class="nexus-system-status__value nexus-mono">
          {{ capacityLabel(metrics.memory) }}
        </span>
      </li>
      <li class="nexus-system-status__item">
        <span class="nexus-system-status__key">{{ $t('nexus.overview.system.disk') }}</span>
        <span class="nexus-system-status__value nexus-mono">
          {{ capacityLabel(metrics.disk) }}
        </span>
      </li>
      <li class="nexus-system-status__item">
        <span class="nexus-system-status__key">{{ $t('nexus.overview.system.realtime') }}</span>
        <strong>{{ wsLabel }}</strong>
      </li>
    </dense-list>
  </overview-panel>
</template>

<script lang="ts" setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

import DenseList from '@/components/nexus/primitives/DenseList.vue'
import StatusBadge from '@/components/nexus/primitives/StatusBadge.vue'
import OverviewPanel from './OverviewPanel.vue'
import type { WsConnectionState } from '@/store/ws'
import {
  formatOverviewDuration,
  formatOverviewPercent,
  formatOverviewSize,
} from './overviewFormatters'
import type {
  OverviewCapacityMetric,
  OverviewStatusMetrics,
} from './overviewPayloads'
import type { SystemStatus } from './selectors/systemStatusSelectors'

const props = defineProps<{
  loading: boolean
  metrics: OverviewStatusMetrics
  offline: boolean
  status: SystemStatus
  unavailable: boolean
  wsState: WsConnectionState
}>()

const { t } = useI18n()

const statusLabel = computed(() => {
  if (props.offline) return t('nexus.status.offline')
  if (props.loading) return t('nexus.status.loading')
  if (props.unavailable) return t('nexus.status.statusMissing')
  return props.status.singboxRunning ? t('nexus.status.running') : t('nexus.status.coreDown')
})

const statusTone = computed(() => {
  if (props.offline || (!props.loading && !props.status.singboxRunning)) return 'error'
  if (props.unavailable) return 'warning'
  return props.loading ? 'info' : 'success'
})

const wsLabel = computed(() => {
  if (props.wsState === 'connected') return t('nexus.status.connected')
  if (props.wsState === 'reconnecting') return t('nexus.status.reconnecting')
  return t('nexus.status.pollFallback')
})

const capacityLabel = (metric: OverviewCapacityMetric): string => {
  if (metric.current === undefined && metric.total === undefined) return '-'

  return `${formatOverviewSize(metric.current)} / ${formatOverviewSize(metric.total)} (${formatOverviewPercent(metric.percent)})`
}
</script>

<style scoped>
.nexus-system-status.nexus-overview-panel {
  height: var(--nexus-overview-primary-panel-height);
  min-height: 0;
  overflow: hidden;
}

.nexus-system-status__list {
  flex: 1 1 auto;
  min-height: 0;
}

.nexus-system-status__list :deep(li) {
  display: grid;
  grid-template-columns: minmax(96px, 0.72fr) minmax(0, 1fr);
  min-height: 31px;
  padding-block: 5px;
  transition: background var(--nexus-transition-fast);
}

.nexus-system-status__list :deep(li.nexus-system-status__item:hover) {
  background: var(--nexus-surface-hover);
  cursor: pointer;
}

.nexus-system-status__key {
  color: var(--nexus-text-secondary);
  font-size: 0.8rem;
  min-width: 0;
}

.nexus-system-status__value,
.nexus-system-status__list strong {
  font-size: 0.8rem;
  font-weight: 650;
  letter-spacing: 0;
  min-width: 0;
  overflow-wrap: anywhere;
  color: var(--nexus-text-primary);
}

</style>
