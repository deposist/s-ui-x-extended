<template>
  <overview-panel :title="$t('nexus.overview.system.title')">
      <template #action>
        <status-badge :label="statusLabel" :tone="statusTone" />
      </template>

    <dense-list class="nexus-system-status__list">
      <li>
        <span class="nexus-system-status__key">S-UI</span>
        <strong dir="ltr">{{ status.appVersion || '-' }}</strong>
      </li>
      <li>
        <span class="nexus-system-status__key">sing-box</span>
        <span class="nexus-system-status__value">
          {{ status.singboxRunning ? $t('nexus.status.running') : $t('nexus.status.notRunning') }}
          <span v-if="status.singboxVersion" dir="ltr">
            {{ status.singboxVersion }}
          </span>
        </span>
      </li>
      <li>
        <span class="nexus-system-status__key">{{ $t('nexus.overview.system.hostUptime') }}</span>
        <strong>{{ formatOverviewDuration(status.uptimeSec) }}</strong>
      </li>
      <li>
        <span class="nexus-system-status__key">{{ $t('nexus.overview.system.singboxUptime') }}</span>
        <strong>{{ formatOverviewDuration(status.singboxUptimeSec) }}</strong>
      </li>
      <li>
        <span class="nexus-system-status__key">{{ $t('nexus.overview.system.cpu') }}</span>
        <strong>{{ formatOverviewPercent(metrics.cpuPercent) }}</strong>
      </li>
      <li>
        <span class="nexus-system-status__key">{{ $t('nexus.overview.system.memory') }}</span>
        <span class="nexus-system-status__value">
          {{ capacityLabel(metrics.memory) }}
        </span>
      </li>
      <li>
        <span class="nexus-system-status__key">{{ $t('nexus.overview.system.disk') }}</span>
        <span class="nexus-system-status__value">
          {{ capacityLabel(metrics.disk) }}
        </span>
      </li>
      <li>
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
.nexus-system-status__list :deep(li) {
  display: grid;
  grid-template-columns: minmax(96px, 0.72fr) minmax(0, 1fr);
}

.nexus-system-status__key {
  color: rgb(var(--v-theme-on-surface) / 68%);
  min-width: 0;
}

.nexus-system-status__value,
.nexus-system-status__list strong {
  font-size: 0.8rem;
  font-weight: 650;
  letter-spacing: 0;
  min-width: 0;
  overflow-wrap: anywhere;
}

</style>
