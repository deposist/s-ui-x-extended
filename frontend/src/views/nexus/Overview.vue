<template>
  <section class="nexus-overview">
    <kpi-row
      :loading="dashboardLoading"
      :summary="kpiSummary"
      :traffic="trafficSparkSeries"
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
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'

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
import type { TrafficSeries } from '@/components/nexus/overview/selectors/trafficSelectors'
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
const SPARK_WINDOW = 24
const sparkSamples = ref<{ download: number; upload: number; ts: number }[]>([])

let statusInterval: ReturnType<typeof setInterval> | undefined
let statusRequestPending = false
let previousNetworkSample = overviewStatusNetworkSample()

const storeLoading = computed(() => data.lastLoad === 0)
const dashboardLoading = computed(() => storeLoading.value || statusLoading.value)
const systemStatus = computed(() => selectSystemStatus(statusPayload.value, nowSec.value))
const systemMetrics = computed(() => overviewStatusMetrics(statusPayload.value))
const trafficSparkSeries = computed<TrafficSeries>(() => ({
  range: 'realtime',
  labels: sparkSamples.value.map((sample) => String(sample.ts)),
  download: sparkSamples.value.map((sample) => sample.download),
  upload: sparkSamples.value.map((sample) => sample.upload),
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

const pushSparkSample = (rate: NetworkTrafficRate) => {
  const next = sparkSamples.value.slice(-SPARK_WINDOW + 1)
  next.push({ download: rate.downloadBps, upload: rate.uploadBps, ts: Date.now() })
  sparkSamples.value = next
}

const loadStatus = async () => {
  if (statusRequestPending) return

  if (!browserOnline.value) {
    statusLoading.value = false
    statusUnavailable.value = true
    previousNetworkSample = undefined
    liveTraffic.value = { downloadBps: 0, uploadBps: 0 }
    sparkSamples.value = []
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
      pushSparkSample(rate)
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
  const msg = await HttpUtils.get('api/security/audit', { limit: 6 })

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
}

const setOffline = () => {
  browserOnline.value = false
  statusUnavailable.value = true
  auditUnavailable.value = true
  previousNetworkSample = undefined
  liveTraffic.value = { downloadBps: 0, uploadBps: 0 }
  sparkSamples.value = []
}

onMounted(() => {
  if (data.lastLoad === 0) {
    void data.loadData()
  }

  window.addEventListener('online', setOnline)
  window.addEventListener('offline', setOffline)
  void loadStatus()
  void loadAuditEvents()
  statusInterval = setInterval(() => {
    void loadStatus()
  }, 10000)
})

onBeforeUnmount(() => {
  if (statusInterval) clearInterval(statusInterval)
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
  display: grid;
  gap: var(--nexus-gap-4);
  min-width: 0;
  grid-template-columns:
    minmax(0, 1.15fr)
    minmax(0, 1.4fr)
    minmax(320px, 1fr);
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
