<template>
  <overview-panel :title="$t('nexus.overview.clients.title')">
      <template #action>
        <span class="nexus-top-clients__count">
          {{ $t('nexus.overview.clients.shown', { count: clients.length }) }}
        </span>
      </template>

    <overview-state v-if="loading">
      {{ $t('nexus.overview.clients.loading') }}
    </overview-state>

    <overview-state v-else-if="clients.length === 0">
      {{ $t('nexus.overview.clients.empty') }}
    </overview-state>

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
  </overview-panel>
</template>

<script lang="ts" setup>
import DenseTable from '@/components/nexus/primitives/DenseTable.vue'
import StatusBadge from '@/components/nexus/primitives/StatusBadge.vue'
import OverviewPanel from './OverviewPanel.vue'
import OverviewState from './OverviewState.vue'
import { formatOverviewSize } from './overviewFormatters'
import type { TopClientRow } from './selectors/topClientsSelectors'

defineProps<{
  clients: TopClientRow[]
  loading: boolean
}>()
</script>

<style scoped>
.nexus-top-clients__count {
  color: rgb(var(--v-theme-on-surface) / 68%);
  font-size: 0.76rem;
  letter-spacing: 0;
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
