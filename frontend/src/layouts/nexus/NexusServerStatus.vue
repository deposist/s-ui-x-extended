<template>
  <div
    class="nexus-server-status"
    :class="`nexus-server-status--${statusTone}`"
    :title="$t(statusLabel)"
  >
    <v-icon
      class="nexus-server-status__icon"
      :icon="statusIcon"
      size="20"
    />
    <!-- Connection state was conveyed by icon colour and a title attribute only,
         so losing the connection was silent for screen readers. This single live
         region covers both rail and expanded modes. Metrics stay outside it on
         purpose: they refresh every 10s and would otherwise be announced
         constantly. -->
    <span class="d-sr-only" role="status">S-UI-X Extended: {{ $t(statusLabel) }}</span>
    <div v-if="!rail" class="nexus-server-status__copy">
      <div class="nexus-server-status__row">
        <strong aria-hidden="true">S-UI-X Extended</strong>
        <span class="nexus-server-status__label" aria-hidden="true">{{ $t(statusLabel) }}</span>
      </div>
      <div v-if="hasUsageMetrics" class="nexus-server-status__metrics nexus-mono">
        <span>CPU: {{ cpuPercent }}%</span>
        <span class="nexus-server-status__divider">|</span>
        <span>RAM: {{ ramPercent }}%</span>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, onMounted, onBeforeUnmount, ref } from 'vue'

import Ws from '@/store/ws'
import HttpUtils from '@/plugins/httputil'
import { isSelectorRecord, nonNegativeNumber } from '@/components/nexus/overview/selectors/selectorUtils'

defineProps<{
  rail: boolean
}>()

const ws = Ws()
const cpuPercent = ref<number | null>(null)
const ramPercent = ref<number | null>(null)
let metricsInterval: ReturnType<typeof setInterval> | undefined

const hasUsageMetrics = computed(() => cpuPercent.value !== null && ramPercent.value !== null)

const loadMetrics = async () => {
  try {
    const msg = await HttpUtils.get('api/status', { r: 'cpu,mem' })
    if (msg.success && msg.obj) {
      const status = isSelectorRecord(msg.obj) ? msg.obj : {}

      let cpu = nonNegativeNumber(status.cpu)

      if (cpu === undefined && isSelectorRecord(status.sys)) {
        cpu = nonNegativeNumber(status.sys.cpu)
      }

      if (cpu !== undefined) {
        cpuPercent.value = Math.round(Math.min(100, cpu))
      }

      let mem = isSelectorRecord(status.mem) ? status.mem : {}
      if (!mem.total && isSelectorRecord(status.sys) && isSelectorRecord(status.sys.mem)) {
        mem = status.sys.mem
      }

      const current = nonNegativeNumber(mem.current)
      const total = nonNegativeNumber(mem.total)
      if (current !== undefined && total && total > 0) {
        ramPercent.value = Math.round(Math.min(100, (current / total) * 100))
      }
    }
  } catch (_error) {
    // Sidebar status is best-effort. Keep the connection indicator visible if metrics fail.
  }
}

onMounted(() => {
  void loadMetrics()
  metricsInterval = setInterval(() => {
    void loadMetrics()
  }, 10000)
})

onBeforeUnmount(() => {
  if (metricsInterval) clearInterval(metricsInterval)
})

const statusLabel = computed(() => {
  if (ws.state === 'connected') {
    return 'nexus.status.online'
  }

  return ws.state === 'reconnecting' ? 'nexus.status.loading' : 'nexus.status.failed'
})

const statusIcon = computed(() => {
  if (ws.state === 'connected') {
    return 'mdi-server-network'
  }

  return ws.state === 'reconnecting' ? 'mdi-sync' : 'mdi-server-off'
})

const statusTone = computed(() => {
  if (ws.state === 'connected') {
    return 'success'
  }

  return ws.state === 'reconnecting' ? 'info' : 'error'
})
</script>

<style scoped>
.nexus-server-status {
  align-items: center;
  background: var(--nexus-surface-2);
  border: 1px solid var(--nexus-border);
  border-radius: var(--nexus-radius-md);
  display: flex;
  gap: var(--nexus-gap-2);
  margin-inline: var(--nexus-gap-2);
  min-height: 42px;
  min-width: 0;
  padding: var(--nexus-gap-2) var(--nexus-gap-3);
}

.nexus-server-status__copy {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
  flex: 1;
}

.nexus-server-status__row {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: var(--nexus-gap-2);
  min-width: 0;
}

.nexus-server-status__row strong,
.nexus-server-status__row span {
  letter-spacing: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.nexus-server-status__row strong {
  font-size: 0.78rem;
  color: var(--nexus-text-primary);
}

.nexus-server-status__label {
  color: rgb(var(--v-theme-on-surface) / 68%);
  font-size: 0.72rem;
}

.nexus-server-status__metrics {
  display: flex;
  align-items: center;
  gap: var(--nexus-gap-1);
  color: var(--nexus-text-secondary);
  font-size: 0.68rem;
  line-height: 1;
}

.nexus-server-status__divider {
  color: var(--nexus-border-strong);
  font-weight: 300;
}

.nexus-server-status__icon {
  flex: 0 0 auto;
}

.nexus-server-status--success .nexus-server-status__icon {
  color: var(--nexus-status-success);
}

.nexus-server-status--info .nexus-server-status__icon {
  color: var(--nexus-status-info);
}

.nexus-server-status--error .nexus-server-status__icon {
  color: var(--nexus-status-error);
}
</style>
