<template>
  <div class="nexus-overview-kpis">
    <kpi-card
      class="nexus-overview-kpis__traffic"
      :delta="trafficDeltaLabel"
      :label="$t('nexus.overview.kpi.trafficStats')"
      :value="loading ? '-' : formatOverviewSize(trafficTotal)"
    >
      <template #meta>
        <div class="nexus-overview-kpis__window-selector">
          <v-menu>
            <template #activator="{ props }">
              <v-btn
                variant="text"
                density="compact"
                size="small"
                class="nexus-overview-kpis__window-btn"
                v-bind="props"
              >
                {{ trafficRangeLabel }}
                <v-icon icon="lucide:chevron-down" size="14" class="ms-1" />
              </v-btn>
            </template>
            <v-list density="compact" min-width="120">
              <v-list-item
                v-for="opt in trafficRangeOptions"
                :key="opt.value"
                :active="opt.value === trafficRange"
                @click="trafficRange = opt.value"
              >
                <v-list-item-title class="text-caption">{{ opt.label }}</v-list-item-title>
              </v-list-item>
            </v-list>
          </v-menu>
        </div>
      </template>

      <template #trend>
        <area-series
          compact
          :aria-label="$t('nexus.overview.kpi.trafficTrend')"
          :labels="traffic.labels"
          :series="trafficSeries"
          :value-formatter="formatOverviewSize"
        />
      </template>
    </kpi-card>

    <kpi-card
      :delta="wsStateLabel"
      :label="$t('nexus.overview.kpi.onlineClients')"
      :value="formatOverviewCount(summary.onlineClients)"
      class="nexus-overview-kpis__online-clients"
    >
      <template #trend>
        <div class="nexus-overview-kpis__signal">
          {{ $t('nexus.overview.kpi.clientSignal') }}
        </div>
      </template>
    </kpi-card>

    <kpi-card
      :delta="$t('nexus.overview.kpi.activeInbounds', { count: formatOverviewCount(summary.activeInbounds) })"
      :label="$t('nexus.overview.kpi.enabledInbounds')"
      :value="formatOverviewCount(summary.totalInbounds)"
      class="nexus-overview-kpis__enabled-inbounds"
    >
      <template #trend>
        <div class="nexus-overview-kpis__signal">
          {{ $t('nexus.overview.kpi.inboundOnlineTags', { count: formatOverviewCount(summary.activeInbounds) }) }}
        </div>
      </template>
    </kpi-card>
  </div>
</template>

<script lang="ts" setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

import AreaSeries from '@/components/nexus/primitives/AreaSeries.vue'
import KpiCard from '@/components/nexus/primitives/KpiCard.vue'
import type { WsConnectionState } from '@/store/ws'
import {
  formatOverviewCount,
  formatOverviewSize,
} from './overviewFormatters'
import type { KpiSummary } from './selectors/kpiSelectors'
import type { SystemStatus } from './selectors/systemStatusSelectors'
import type { TrafficRange, TrafficSeries } from './selectors/trafficSelectors'

const props = defineProps<{
  loading: boolean
  summary: KpiSummary
  status: SystemStatus
  traffic: TrafficSeries
  wsState: WsConnectionState
  trafficRange: TrafficRange
}>()

const emit = defineEmits<{
  'update:trafficRange': [value: TrafficRange]
}>()

const trafficRange = computed({
  get: () => props.trafficRange,
  set: (val) => emit('update:trafficRange', val),
})

const { n, t } = useI18n()

const trafficRangeOptions = computed<{ value: TrafficRange; label: string }[]>(() => [
  { value: '1h', label: `${n(1)}${t('date.h')}` },
  { value: '6h', label: `${n(6)}${t('date.h')}` },
  { value: '12h', label: `${n(12)}${t('date.h')}` },
  { value: '24h', label: `${n(24)}${t('date.h')}` },
  { value: '7d', label: `${n(7)}${t('date.d')}` },
  { value: '30d', label: `${n(30)}${t('date.d')}` },
])

const trafficRangeLabel = computed(() => {
  const activeOpt = trafficRangeOptions.value.find(opt => opt.value === trafficRange.value)
  return activeOpt ? activeOpt.label : `${n(24)}${t('date.h')}`
})

const trafficTotal = computed(() => {
  return props.traffic.download.reduce((sum, value) => sum + value, 0)
    + props.traffic.upload.reduce((sum, value) => sum + value, 0)
})

const trafficDeltaLabel = computed(() => {
  const download = props.traffic.download.reduce((sum, value) => sum + value, 0)
  const upload = props.traffic.upload.reduce((sum, value) => sum + value, 0)

  return t('nexus.overview.kpi.trafficStatsDelta', {
    download: formatOverviewSize(download),
    upload: formatOverviewSize(upload),
  })
})

const trafficSeries = computed(() => [
  {
    label: t('stats.download'),
    values: props.traffic.download,
  },
  {
    label: t('stats.upload'),
    values: props.traffic.upload,
  },
])

const wsStateLabel = computed(() => {
  if (props.wsState === 'connected') return t('nexus.status.realtime')
  if (props.wsState === 'reconnecting') return t('nexus.status.reconnecting')
  return t('nexus.status.pollFallback')
})
</script>

<style scoped>
.nexus-overview-kpis {
  display: grid;
  gap: var(--nexus-gap-4);
  grid-template-columns: repeat(4, minmax(0, 1fr));
  min-width: 0;
}

.nexus-overview-kpis__traffic {
  grid-column: span 2;
}

.nexus-overview-kpis__online-clients :deep(.nexus-kpi-card__value),
.nexus-overview-kpis__enabled-inbounds :deep(.nexus-kpi-card__value),
.nexus-overview-kpis__traffic :deep(.nexus-kpi-card__value) {
  font-family: var(--nexus-font-mono);
}

.nexus-overview-kpis__window-selector {
  flex: 0 0 auto;
}

.nexus-overview-kpis__window-btn {
  text-transform: none;
  font-size: 0.72rem !important;
  color: var(--nexus-text-secondary);
  height: 24px !important;
  border: 1px solid var(--nexus-border);
  border-radius: var(--nexus-radius-sm);
  background: var(--nexus-surface-2);
}

.nexus-overview-kpis__window-btn:hover {
  color: var(--nexus-text-primary);
  border-color: var(--nexus-border-strong);
}

.nexus-overview-kpis__signal {
  color: rgb(var(--v-theme-on-surface) / 68%);
  font-size: 0.78rem;
  letter-spacing: 0;
  line-height: 1.35;
  min-width: 0;
  overflow-wrap: anywhere;
}

@media (max-width: 1264px) {
  .nexus-overview-kpis {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 600px) {
  .nexus-overview-kpis {
    grid-template-columns: minmax(0, 1fr);
  }

  .nexus-overview-kpis__traffic {
    grid-column: auto;
  }
}
</style>
