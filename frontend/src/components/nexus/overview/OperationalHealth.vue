<template>
  <overview-panel class="nexus-operational-health" :title="$t('nexus.overview.operational.title')">
    <template #action>
      <status-badge :label="stateLabel" :tone="stateTone" />
    </template>

    <overview-state v-if="rows.length === 0">
      {{ $t('nexus.overview.operational.empty') }}
    </overview-state>

    <div v-else class="nexus-operational-health__scroll">
      <dense-list class="nexus-operational-health__list">
        <li
          v-for="row in rows"
          :key="`${row.kind}:${row.tag}`"
          class="nexus-operational-health__item"
        >
          <div class="nexus-operational-health__main">
            <span class="nexus-operational-health__kind">{{ row.kindLabel }}</span>
            <strong class="nexus-mono">{{ row.tag }}</strong>
            <span v-if="row.detail" class="nexus-operational-health__detail">{{ row.detail }}</span>
          </div>
          <status-badge :label="row.statusLabel" :tone="row.tone" />
        </li>
      </dense-list>
    </div>
  </overview-panel>
</template>

<script lang="ts" setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

import DenseList from '@/components/nexus/primitives/DenseList.vue'
import StatusBadge from '@/components/nexus/primitives/StatusBadge.vue'
import OverviewPanel from './OverviewPanel.vue'
import OverviewState from './OverviewState.vue'
import type { FailoverStatusEntry } from '@/types/outbounds'

type StatusTone = 'info' | 'success' | 'warning' | 'error' | 'neutral'

type ProviderHealth = {
  tag?: string
  status?: string
  updatedAt?: number
  outbounds?: number
  healthyOutbounds?: number
  outboundCount?: number
  healthyCount?: number
  lastError?: string
}

type HealthRow = {
  kind: 'failover' | 'provider'
  kindLabel: string
  tag: string
  status: 'healthy' | 'degraded' | 'down' | 'unknown'
  statusLabel: string
  tone: StatusTone
  detail: string
}

const props = defineProps<{
  failover?: Record<string, FailoverStatusEntry>
  providers?: Record<string, ProviderHealth>
}>()

const { t } = useI18n()

const normalizeStatus = (status?: string): HealthRow['status'] => {
  if (status === 'healthy' || status === 'degraded' || status === 'down') return status
  return 'unknown'
}

const toneForStatus = (status: HealthRow['status']): StatusTone => {
  if (status === 'healthy') return 'success'
  if (status === 'down') return 'error'
  if (status === 'degraded') return 'warning'
  return 'neutral'
}

const statusLabel = (status: HealthRow['status']): string => {
  if (status === 'healthy') return t('nexus.status.healthy')
  if (status === 'down') return t('nexus.status.down')
  if (status === 'degraded') return t('nexus.status.degraded')
  return t('nexus.status.unknown')
}

const providerCount = (provider: ProviderHealth, key: 'outbounds' | 'healthyOutbounds'): number => {
  if (key === 'outbounds') return provider.outbounds ?? provider.outboundCount ?? 0
  return provider.healthyOutbounds ?? provider.healthyCount ?? 0
}

const rows = computed<HealthRow[]>(() => {
  const out: HealthRow[] = []

  for (const [tag, entry] of Object.entries(props.failover ?? {})) {
    const members = entry.members ?? []
    const healthy = members.filter(member => member.healthy).length
    const status: HealthRow['status'] = members.length === 0
      ? 'unknown'
      : entry.allDown || healthy === 0
        ? 'down'
        : healthy < members.length
          ? 'degraded'
          : 'healthy'
    out.push({
      kind: 'failover',
      kindLabel: t('nexus.overview.operational.failover'),
      tag,
      status,
      statusLabel: statusLabel(status),
      tone: toneForStatus(status),
      detail: t('nexus.overview.operational.failoverDetail', {
        active: entry.active || t('none'),
        healthy,
        total: members.length,
        policy: entry.allDownPolicy || 'hold_current',
      }),
    })
  }

  for (const [tag, provider] of Object.entries(props.providers ?? {})) {
    const status = normalizeStatus(provider.status)
    const healthy = providerCount(provider, 'healthyOutbounds')
    const total = providerCount(provider, 'outbounds')
    out.push({
      kind: 'provider',
      kindLabel: t('nexus.overview.operational.provider'),
      tag: provider.tag || tag,
      status,
      statusLabel: statusLabel(status),
      tone: toneForStatus(status),
      detail: provider.lastError || t('nexus.overview.operational.providerDetail', { healthy, total }),
    })
  }

  return out.sort((a, b) => {
    const rank = { down: 0, degraded: 1, unknown: 2, healthy: 3 }
    return rank[a.status] - rank[b.status] || a.kind.localeCompare(b.kind) || a.tag.localeCompare(b.tag)
  }).slice(0, 8)
})

const stateTone = computed<StatusTone>(() => {
  if (rows.value.some(row => row.status === 'down')) return 'error'
  if (rows.value.some(row => row.status === 'degraded')) return 'warning'
  if (rows.value.some(row => row.status === 'unknown')) return 'neutral'
  return rows.value.length > 0 ? 'success' : 'info'
})

const stateLabel = computed(() => {
  if (rows.value.length === 0) return t('nexus.status.idle')
  return t('nexus.overview.operational.rows', { count: rows.value.length })
})
</script>

<style scoped>
.nexus-operational-health__scroll {
  border-radius: var(--nexus-radius-md);
  flex: 1 1 auto;
  min-height: 0;
  overflow: auto;
  scrollbar-width: thin;
}

.nexus-operational-health__list :deep(li) {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  transition: background var(--nexus-transition-fast);
}

.nexus-operational-health__list :deep(li.nexus-operational-health__item:hover) {
  background: var(--nexus-surface-hover);
  cursor: pointer;
}

.nexus-operational-health__main {
  display: grid;
  gap: 2px;
  min-width: 0;
}

.nexus-operational-health__kind,
.nexus-operational-health__detail {
  color: var(--nexus-text-secondary);
  font-size: 0.76rem;
  line-height: 1.3;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.nexus-operational-health__main strong {
  font-size: 0.8rem;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
