<template>
  <div class="nexus-overview-kpis">
    <kpi-card
      :delta="$t('nexus.overview.kpi.liveTrafficDelta')"
      :label="$t('nexus.overview.kpi.liveTraffic')"
      :value="loading ? '-' : formatOverviewRate(summary.liveTrafficBps)"
    >
      <template #trend>
        <area-spark
          :aria-label="$t('nexus.overview.kpi.trafficTrend')"
          :labels="traffic.labels"
          :values="trafficTrend"
        />
      </template>
    </kpi-card>

    <kpi-card
      :delta="wsStateLabel"
      :label="$t('nexus.overview.kpi.onlineClients')"
      :value="formatOverviewCount(summary.onlineClients)"
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

import AreaSpark from '@/components/nexus/primitives/AreaSpark.vue'
import KpiCard from '@/components/nexus/primitives/KpiCard.vue'
import type { WsConnectionState } from '@/store/ws'
import {
  formatOverviewCount,
  formatOverviewRate,
} from './overviewFormatters'
import type { KpiSummary } from './selectors/kpiSelectors'
import type { TrafficSeries } from './selectors/trafficSelectors'

const props = defineProps<{
  loading: boolean
  summary: KpiSummary
  traffic: TrafficSeries
  wsState: WsConnectionState
}>()

const { t } = useI18n()

const trafficTrend = computed(() => {
  return props.traffic.download.map((download, index) => {
    return download + (props.traffic.upload[index] ?? 0)
  })
})

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
  grid-template-columns: repeat(3, minmax(0, 1fr));
  min-width: 0;
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
}
</style>
