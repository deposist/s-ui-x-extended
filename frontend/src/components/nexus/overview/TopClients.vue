<template>
  <article class="nexus-overview-panel">
    <panel-header :title="$t('nexus.overview.clients.title')">
      <template #action>
        <span class="nexus-top-clients__count">
          {{ $t('nexus.overview.clients.shown', { count: clients.length }) }}
        </span>
      </template>
    </panel-header>

    <div v-if="loading" class="nexus-overview-panel__state">
      {{ $t('nexus.overview.clients.loading') }}
    </div>

    <div v-else-if="clients.length === 0" class="nexus-overview-panel__state">
      {{ $t('nexus.overview.clients.empty') }}
    </div>

    <dense-table v-else>
      <thead>
        <tr>
          <th>{{ $t('objects.client') }}</th>
          <th>{{ $t('nexus.overview.clients.state') }}</th>
          <th>{{ $t('stats.download') }}</th>
          <th>{{ $t('nexus.overview.clients.total') }}</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="client in clients" :key="client.id ?? client.name">
          <td>{{ client.name }}</td>
          <td>
            <status-badge
              :label="client.online ? $t('nexus.status.online') : $t('nexus.status.offline')"
              :tone="client.online ? 'success' : 'info'"
            />
          </td>
          <td>{{ formatOverviewSize(client.download) }}</td>
          <td>{{ formatOverviewSize(client.total) }}</td>
        </tr>
      </tbody>
    </dense-table>
  </article>
</template>

<script lang="ts" setup>
import DenseTable from '@/components/nexus/primitives/DenseTable.vue'
import PanelHeader from '@/components/nexus/primitives/PanelHeader.vue'
import StatusBadge from '@/components/nexus/primitives/StatusBadge.vue'
import { formatOverviewSize } from './overviewFormatters'
import type { TopClientRow } from './selectors/topClientsSelectors'

defineProps<{
  clients: TopClientRow[]
  loading: boolean
}>()
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

.nexus-top-clients__count {
  color: rgb(var(--v-theme-on-surface) / 68%);
  font-size: 0.76rem;
  letter-spacing: 0;
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

:deep(.nexus-dense-table td),
:deep(.nexus-dense-table th) {
  height: 32px;
  padding-block: var(--nexus-gap-1);
}

:deep(.nexus-dense-table tr) {
  line-height: 1.25;
}

@media (max-width: 600px) {
  .nexus-overview-panel :deep(th:nth-child(3)),
  .nexus-overview-panel :deep(td:nth-child(3)) {
    display: none;
  }
}
</style>
