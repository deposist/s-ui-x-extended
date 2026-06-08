<template>
  <article class="nexus-protocol-summaries">
    <panel-header :title="$t('nexus.overview.protocols.title')">
      <template #action>
        <span class="nexus-protocol-summaries__count">
          {{ $t('nexus.overview.protocols.groups', { count: summaries.length }) }}
        </span>
      </template>
    </panel-header>

    <div v-if="loading" class="nexus-protocol-summaries__state">
      {{ $t('nexus.overview.protocols.loading') }}
    </div>

    <div v-else-if="summaries.length === 0" class="nexus-protocol-summaries__state">
      {{ $t('nexus.overview.protocols.empty') }}
    </div>

    <dense-table v-else>
      <thead>
        <tr>
          <th>{{ $t('nexus.overview.protocols.type') }}</th>
          <th>{{ $t('nexus.overview.clients.state') }}</th>
          <th>{{ $t('nexus.overview.protocols.activeShort') }}</th>
          <th>{{ $t('nexus.overview.protocols.totalShort') }}</th>
          <th>{{ $t('nexus.overview.protocols.tags') }}</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="summary in summaries" :key="summary.type">
          <td>{{ summary.type }}</td>
          <td>
            <status-badge
              :label="summary.activeInbounds > 0 ? $t('nexus.status.online') : $t('nexus.status.idle')"
              :tone="summary.activeInbounds > 0 ? 'success' : 'info'"
            />
          </td>
          <td>{{ summary.activeInbounds }}</td>
          <td>{{ summary.totalInbounds }}</td>
          <td>
            <span class="nexus-protocol-row__tags">
              <span v-if="summary.tags.length === 0" class="nexus-protocol-row__no-tag">
                {{ $t('nexus.overview.protocols.noTag') }}
              </span>
              <template v-else>
                <span
                  v-for="tag in visibleTags(summary)"
                  :key="tag"
                  class="nexus-protocol-row__tag"
                >
                  {{ tag }}
                </span>
                <span
                  v-if="overflowCount(summary) > 0"
                  class="nexus-protocol-row__tag nexus-protocol-row__tag--overflow"
                >
                  +{{ overflowCount(summary) }}
                </span>
              </template>
            </span>
          </td>
        </tr>
      </tbody>
    </dense-table>
  </article>
</template>

<script lang="ts" setup>
import DenseTable from '@/components/nexus/primitives/DenseTable.vue'
import PanelHeader from '@/components/nexus/primitives/PanelHeader.vue'
import StatusBadge from '@/components/nexus/primitives/StatusBadge.vue'
import type { ProtocolSummary } from './selectors/protocolSummarySelectors'

defineProps<{
  loading: boolean
  summaries: ProtocolSummary[]
}>()

const MAX_TAGS = 8

const visibleTags = (summary: ProtocolSummary): string[] => {
  return summary.tags.slice(0, MAX_TAGS)
}

const overflowCount = (summary: ProtocolSummary): number => {
  return Math.max(0, summary.tags.length - MAX_TAGS)
}
</script>

<style scoped>
.nexus-protocol-summaries {
  background: var(--nexus-surface-1);
  border: 1px solid var(--nexus-border);
  border-radius: var(--nexus-radius-lg);
  display: grid;
  gap: var(--nexus-gap-3);
  min-width: 0;
  padding: var(--nexus-gap-4);
}

.nexus-protocol-summaries__count {
  color: rgb(var(--v-theme-on-surface) / 68%);
  font-size: 0.76rem;
  letter-spacing: 0;
}

.nexus-protocol-row__tags {
  display: flex;
  flex-wrap: wrap;
  gap: var(--nexus-gap-1);
  min-width: 0;
}

.nexus-protocol-row__tag,
.nexus-protocol-row__no-tag {
  background: var(--nexus-surface-2);
  border: 1px solid var(--nexus-border);
  border-radius: var(--nexus-radius-sm);
  color: rgb(var(--v-theme-on-surface) / 68%);
  font-size: 0.72rem;
  letter-spacing: 0;
  line-height: 1.3;
  max-width: 100%;
  overflow-wrap: anywhere;
  padding: 3px 6px;
}

.nexus-protocol-row__tag--overflow {
  border-color: var(--nexus-border-strong);
}

.nexus-protocol-summaries__state {
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
</style>
