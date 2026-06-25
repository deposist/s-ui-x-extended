<template>
  <overview-panel class="nexus-top-clients" :title="$t('nexus.overview.clients.title')">
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

    <div v-else class="nexus-top-clients__content">
      <!-- Summary metrics zone using real data -->
      <div class="nexus-top-clients__summary">
        <div class="nexus-top-clients__summary-item">
          <span class="nexus-top-clients__summary-label">{{ $t('stats.usage') }}</span>
          <strong class="nexus-top-clients__summary-value nexus-mono">{{ formatOverviewSize(totalTraffic) }}</strong>
        </div>
        <div class="nexus-top-clients__summary-item">
          <span class="nexus-top-clients__summary-label">{{ $t('nexus.status.online') }}</span>
          <strong class="nexus-top-clients__summary-value">{{ onlineCount }}</strong>
        </div>
        <div class="nexus-top-clients__summary-item">
          <span class="nexus-top-clients__summary-label">{{ $t('nexus.overview.clients.total') }}</span>
          <strong class="nexus-top-clients__summary-value">{{ clients.length }}</strong>
        </div>
      </div>

      <div class="nexus-top-clients__table-scroll">
        <dense-table>
          <thead>
            <tr>
              <th>{{ $t('objects.client') }}</th>
              <th>{{ $t('nexus.overview.clients.state') }}</th>
              <th>{{ $t('stats.download') }}</th>
              <th>{{ $t('nexus.overview.clients.total') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="client in clients" :key="client.id ?? client.name" class="nexus-dense-table__row-hoverable">
              <td>{{ client.name }}</td>
              <td>
                <status-badge
                  :label="client.online ? $t('nexus.status.online') : $t('nexus.status.offline')"
                  :tone="client.online ? 'success' : 'neutral'"
                />
              </td>
              <td class="nexus-mono">{{ formatOverviewSize(client.download) }}</td>
              <td class="nexus-mono">{{ formatOverviewSize(client.total) }}</td>
            </tr>
          </tbody>
        </dense-table>
      </div>

      <div class="nexus-top-clients__footer">
        <v-btn
          to="/clients"
          variant="text"
          density="compact"
          size="small"
          class="nexus-top-clients__view-all-btn"
        >
          {{ $t('nexus.overview.clients.viewAll') }}
          <v-icon icon="lucide:arrow-right" size="14" class="ms-1" />
        </v-btn>
      </div>
    </div>
  </overview-panel>
</template>

<script lang="ts" setup>
import { computed } from 'vue'
import DenseTable from '@/components/nexus/primitives/DenseTable.vue'
import StatusBadge from '@/components/nexus/primitives/StatusBadge.vue'
import OverviewPanel from './OverviewPanel.vue'
import OverviewState from './OverviewState.vue'
import { formatOverviewSize } from './overviewFormatters'
import type { TopClientRow } from './selectors/topClientsSelectors'

const props = defineProps<{
  clients: TopClientRow[]
  loading: boolean
}>()

const totalTraffic = computed(() => {
  return props.clients.reduce((acc, client) => acc + client.total, 0)
})

const onlineCount = computed(() => {
  return props.clients.filter(client => client.online).length
})
</script>

<style scoped>
.nexus-top-clients {
  height: var(--nexus-overview-primary-panel-height);
  min-height: 0;
  overflow: hidden;
}

.nexus-top-clients__count {
  color: rgb(var(--v-theme-on-surface) / 68%);
  font-size: 0.76rem;
  letter-spacing: 0;
}

.nexus-top-clients__content {
  display: flex;
  flex-direction: column;
  gap: var(--nexus-gap-3);
  height: 100%;
  min-height: 0;
}

.nexus-top-clients__summary {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: var(--nexus-gap-2);
  background: var(--nexus-surface-2);
  border: 1px solid var(--nexus-border);
  border-radius: var(--nexus-radius-md);
  padding: var(--nexus-gap-2) var(--nexus-gap-3);
}

.nexus-top-clients__summary-item {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.nexus-top-clients__summary-label {
  font-size: 0.68rem;
  color: var(--nexus-text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.5px;
  line-height: 1.2;
}

.nexus-top-clients__summary-value {
  font-size: 0.95rem;
  font-weight: 700;
  color: var(--nexus-text-primary);
  line-height: 1.3;
}

.nexus-top-clients__table-scroll {
  border-radius: var(--nexus-radius-md);
  flex: 1 1 auto;
  min-height: 0;
  overflow: auto;
  scrollbar-width: thin;
}

.nexus-top-clients__footer {
  margin-top: auto;
  display: flex;
  flex: 0 0 auto;
  justify-content: flex-end;
  padding-top: var(--nexus-gap-1);
}

.nexus-top-clients__view-all-btn {
  color: var(--nexus-accent-primary);
  font-weight: 600;
  text-transform: none;
  font-size: 0.78rem;
}

.nexus-top-clients__view-all-btn:hover {
  color: var(--nexus-text-primary);
}

:deep(.nexus-dense-table td),
:deep(.nexus-dense-table th) {
  height: 32px;
  padding-block: var(--nexus-gap-1);
}

:deep(.nexus-dense-table tr) {
  line-height: 1.25;
}

.nexus-top-clients__table-scroll :deep(.nexus-dense-table__table th) {
  background: var(--nexus-surface-2);
  position: sticky;
  top: 0;
  z-index: 1;
}

:deep(.nexus-dense-table tbody tr.nexus-dense-table__row-hoverable:hover td) {
  background: var(--nexus-surface-hover);
  cursor: pointer;
}

@media (max-width: 600px) {
  .nexus-overview-panel :deep(th:nth-child(3)),
  .nexus-overview-panel :deep(td:nth-child(3)) {
    display: none;
  }
}
</style>
