<template>
  <section class="nexus-overview">
    <kpi-row
      v-model:traffic-range="trafficRange"
      :loading="dashboardLoading"
      :summary="kpiSummary"
      :status="systemStatus"
      :traffic="trafficSeries"
      :ws-state="ws.state"
    />

    <div class="nexus-overview__primary">
      <top-clients :clients="topClients" :loading="storeLoading" />
      <recent-events
        :events="auditEvents"
        :loading="auditLoading"
        :offline="!browserOnline"
        :unavailable="auditUnavailable"
      />
      <system-status
        :loading="statusLoading"
        :metrics="systemMetrics"
        :offline="!browserOnline"
        :status="systemStatus"
        :unavailable="statusUnavailable"
        :ws-state="ws.state"
      />
    </div>

    <protocol-summaries
      :loading="storeLoading"
      :summaries="protocolSummaries"
    />
  </section>
</template>

<script lang="ts" setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'

import KpiRow from '@/components/nexus/overview/KpiRow.vue'
import ProtocolSummaries from '@/components/nexus/overview/ProtocolSummaries.vue'
import RecentEvents from '@/components/nexus/overview/RecentEvents.vue'
import SystemStatus from '@/components/nexus/overview/SystemStatus.vue'
import TopClients from '@/components/nexus/overview/TopClients.vue'
import { mapAuditDisplayItems } from '@/components/nexus/overview/selectors/auditMapper'
import { selectKpiSummary } from '@/components/nexus/overview/selectors/kpiSelectors'
import { selectProtocolSummaries } from '@/components/nexus/overview/selectors/protocolSummarySelectors'
import { selectSystemStatus } from '@/components/nexus/overview/selectors/systemStatusSelectors'
import { selectTopClients } from '@/components/nexus/overview/selectors/topClientsSelectors'
import {
  selectTrafficSeries,
  trafficRangeHours,
  type TrafficRange,
} from '@/components/nexus/overview/selectors/trafficSelectors'
import {
  auditEventsFromPayload,
  networkRateFromSamples,
  overviewStatusMetrics,
  overviewStatusNetworkSample,
  type NetworkTrafficRate,
} from '@/components/nexus/overview/overviewPayloads'
import HttpUtils from '@/plugins/httputil'
import Data from '@/store/modules/data'
import Ws from '@/store/ws'

const data = Data()
const ws = Ws()

const browserOnline = ref(typeof navigator === 'undefined' ? true : navigator.onLine)
const nowSec = ref(Math.floor(Date.now() / 1000))
const statusPayload = ref<unknown>()
const statusLoading = ref(true)
const statusLoaded = ref(false)
const statusUnavailable = ref(false)
const auditEvents = ref(mapAuditDisplayItems())
const auditLoading = ref(true)
const auditUnavailable = ref(false)
const liveTraffic = ref<NetworkTrafficRate>({
  downloadBps: 0,
  uploadBps: 0,
})
const trafficRange = ref<TrafficRange>('24h')
const trafficSummary = ref<unknown>()
const trafficLoading = ref(false)

let statusInterval: ReturnType<typeof setInterval> | undefined
let trafficInterval: ReturnType<typeof setInterval> | undefined
let statusRequestPending = false
let trafficRequestId = 0
let previousNetworkSample = overviewStatusNetworkSample()

const storeLoading = computed(() => data.lastLoad === 0)
const dashboardLoading = computed(() => storeLoading.value || statusLoading.value || trafficLoading.value)
const systemStatus = computed(() => selectSystemStatus(statusPayload.value, nowSec.value))
const systemMetrics = computed(() => overviewStatusMetrics(statusPayload.value))
const trafficSeries = computed(() => selectTrafficSeries({
  range: trafficRange.value,
  summary: trafficSummary.value,
}))
const topClients = computed(() => selectTopClients({
  clients: data.clients,
  onlines: data.onlines,
}))
const protocolSummaries = computed(() => selectProtocolSummaries({
  inbounds: data.inbounds,
  onlines: data.onlines,
}))
const kpiHealth = computed(() => {
  if (!browserOnline.value) {
    return {
      online: false,
      singboxRunning: statusLoaded.value ? systemStatus.value.singboxRunning : undefined,
    }
  }

  if (!statusLoaded.value) {
    return {
      online: undefined,
      singboxRunning: undefined,
    }
  }

  return {
    online: !statusUnavailable.value,
    singboxRunning: systemStatus.value.singboxRunning,
  }
})
const kpiSummary = computed(() => selectKpiSummary({
  inbounds: data.inbounds,
  onlines: data.onlines,
  liveTraffic: liveTraffic.value,
  health: kpiHealth.value,
}))

const loadTrafficStats = async () => {
  const requestId = ++trafficRequestId

  if (!browserOnline.value) {
    trafficSummary.value = undefined
    trafficLoading.value = false
    return
  }

  trafficLoading.value = trafficSummary.value === undefined

  const response = await HttpUtils.get('api/stats/traffic', {
    limit: trafficRangeHours[trafficRange.value],
    buckets: 48,
  })

  if (requestId === trafficRequestId) {
    trafficSummary.value = response.success ? response.obj : undefined
    trafficLoading.value = false
  }
}

const loadStatus = async () => {
  if (statusRequestPending) return

  if (!browserOnline.value) {
    statusLoading.value = false
    statusUnavailable.value = true
    previousNetworkSample = undefined
    liveTraffic.value = { downloadBps: 0, uploadBps: 0 }
    return
  }

  statusRequestPending = true
  statusLoading.value = !statusLoaded.value
  const msg = await HttpUtils.get('api/status', {
    r: 'sys,sbd,net,cpu,mem,dsk',
  })
  nowSec.value = Math.floor(Date.now() / 1000)

  if (msg.success) {
    statusPayload.value = msg.obj
    statusLoaded.value = true
    statusUnavailable.value = false

    const networkSample = overviewStatusNetworkSample(msg.obj)
    const rate = networkRateFromSamples(previousNetworkSample, networkSample)
    if (rate) {
      liveTraffic.value = rate
    }
    previousNetworkSample = networkSample
  } else {
    statusUnavailable.value = true
    previousNetworkSample = undefined
    liveTraffic.value = { downloadBps: 0, uploadBps: 0 }
  }

  statusLoading.value = false
  statusRequestPending = false
}

const loadAuditEvents = async () => {
  if (!browserOnline.value) {
    auditLoading.value = false
    auditUnavailable.value = true
    return
  }

  auditLoading.value = true
  const msg = await HttpUtils.get('api/security/audit', { limit: 10 })

  if (msg.success) {
    auditEvents.value = mapAuditDisplayItems(auditEventsFromPayload(msg.obj))
    auditUnavailable.value = false
  } else {
    auditEvents.value = []
    auditUnavailable.value = true
  }

  auditLoading.value = false
}

const setOnline = () => {
  browserOnline.value = true
  void loadStatus()
  void loadAuditEvents()
  void loadTrafficStats()
}

const setOffline = () => {
  browserOnline.value = false
  statusUnavailable.value = true
  auditUnavailable.value = true
  previousNetworkSample = undefined
  liveTraffic.value = { downloadBps: 0, uploadBps: 0 }
  trafficSummary.value = undefined
}

watch(trafficRange, () => {
  void loadTrafficStats()
})

// Pause polling while the browser tab is hidden; refresh immediately when visible.
const onVisible = () => {
  if (!document.hidden) {
    void loadStatus()
    void loadTrafficStats()
  }
}

onMounted(() => {
  if (data.lastLoad === 0) {
    void data.loadData()
  }

  window.addEventListener('online', setOnline)
  window.addEventListener('offline', setOffline)
  document.addEventListener('visibilitychange', onVisible)
  void loadStatus()
  void loadAuditEvents()
  void loadTrafficStats()
  statusInterval = setInterval(() => {
    if (document.hidden) return
    void loadStatus()
  }, 10000)
  trafficInterval = setInterval(() => {
    if (document.hidden) return
    void loadTrafficStats()
  }, 60000)
})

onBeforeUnmount(() => {
  if (statusInterval) clearInterval(statusInterval)
  if (trafficInterval) clearInterval(trafficInterval)
  document.removeEventListener('visibilitychange', onVisible)
  window.removeEventListener('online', setOnline)
  window.removeEventListener('offline', setOffline)
})
</script>

<style scoped>
.nexus-overview {
  display: grid;
  gap: var(--nexus-gap-4);
  min-width: 0;
}

.nexus-overview__primary {
  --nexus-overview-primary-panel-height: 320px;

  display: grid;
  gap: var(--nexus-gap-4);
  min-width: 0;
  grid-template-columns:
    minmax(0, 1.2fr)
    minmax(0, 1.2fr)
    minmax(320px, 1fr);
  align-items: stretch;
}

@media (max-width: 1264px) {
  .nexus-overview__primary {
    grid-auto-flow: dense;
    grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  }

  .nexus-overview__primary > :nth-child(3) {
    grid-column: 1 / -1;
  }
}

@media (max-width: 960px) {
  .nexus-overview__primary {
    grid-template-columns: minmax(0, 1fr);
  }

  .nexus-overview__primary > :nth-child(3) {
    grid-column: auto;
  }
}
</style>
