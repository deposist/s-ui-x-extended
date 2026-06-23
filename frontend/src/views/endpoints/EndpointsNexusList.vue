<template>
  <div class="endpoints-nexus">
    <page-header
      :search="search"
      searchable
      :subtitle="subtitle"
      :title="$t('pages.endpoints')"
      @update:search="search = $event"
    />

    <page-toolbar>
      <template #actions>
        <v-btn color="primary" prepend-icon="lucide:plus" variant="flat" @click="emit('add')">
          {{ $t('actions.add') }}
        </v-btn>
      </template>
    </page-toolbar>

    <nexus-data-table :columns="columns" :items="filtered" :row-key="(item) => item.id">
      <template #col.status="{ item }">
        <status-badge v-if="onlines.includes(item.tag)" :label="$t('online')" tone="success" />
        <status-badge v-else :label="$t('nexus.status.offline')" tone="neutral" />
      </template>
      <template #col.tag="{ item }">
        <span class="endpoints-nexus__tag">{{ item.tag }}</span>
      </template>
      <template #col.address="{ item }">
        <span v-if="item.address && item.address.length" class="nexus-mono">{{ item.address[0] }}</span>
        <span v-else class="endpoints-nexus__muted">—</span>
      </template>
      <template #col.listen_port="{ item }">
        <span v-if="(item.listen_port ?? 0) > 0" class="nexus-mono">{{ item.listen_port }}</span>
        <span v-else class="endpoints-nexus__muted">—</span>
      </template>
      <template #col.peers="{ item }">
        <span>{{ item.peers?.length ?? '—' }}</span>
      </template>

      <template #actions="{ item }">
        <row-actions :actions="endpointActions(item)" @action="(key) => handleAction(key, item)" />
      </template>

      <template #empty>
        <empty-state icon="lucide:globe" :title="$t('table.noData')" />
      </template>
    </nexus-data-table>
  </div>
</template>

<script lang="ts" setup>
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'

import type { Column } from '@/components/nexus/data/dataTableColumns'
import NexusDataTable from '@/components/nexus/data/NexusDataTable.vue'
import RowActions from '@/components/nexus/data/RowActions.vue'
import type { RowAction } from '@/components/nexus/data/rowActions'
import EmptyState from '@/components/nexus/primitives/EmptyState.vue'
import PageHeader from '@/components/nexus/primitives/PageHeader.vue'
import PageToolbar from '@/components/nexus/primitives/PageToolbar.vue'
import StatusBadge from '@/components/nexus/primitives/StatusBadge.vue'
import { useConfirm } from '@/components/nexus/primitives/useConfirm'

interface EndpointRow {
  id: number
  tag: string
  type: string
  address?: string[]
  listen_port?: number
  peers?: unknown[]
  [key: string]: unknown
}

const props = defineProps<{
  endpoints: EndpointRow[]
  onlines: string[]
  enableTraffic: boolean
}>()

const emit = defineEmits<{
  add: []
  edit: [id: number]
  del: [tag: string]
  stats: [tag: string]
  qr: [id: number]
}>()

const { t } = useI18n()
const { confirm } = useConfirm()
const search = ref('')

const subtitle = computed(() => {
  const total = props.endpoints.length
  const online = props.endpoints.filter(item => props.onlines.includes(item.tag)).length

  return t('nexus.summary.endpoints', { total, online })
})

const columns: Column<EndpointRow>[] = [
  { key: 'status', labelKey: 'status' },
  { key: 'tag', labelKey: 'objects.tag', sortable: true },
  { key: 'type', labelKey: 'type', sortable: true },
  { key: 'address', labelKey: 'in.addr' },
  { key: 'listen_port', labelKey: 'in.port', sortable: true },
  { key: 'peers', labelKey: 'types.wg.peers' },
]

const filtered = computed<EndpointRow[]>(() => {
  const query = search.value.trim().toLowerCase()

  if (!query) return props.endpoints

  return props.endpoints.filter(item => String(item.tag).toLowerCase().includes(query))
})

const endpointActions = (item: EndpointRow): RowAction[] => [
  { key: 'edit', labelKey: 'actions.edit', icon: 'lucide:pencil', inline: true },
  { key: 'qr', labelKey: 'objects.config', icon: 'lucide:qr-code', inline: true, hidden: !(item.type === 'wireguard' && (item.peers?.length ?? 0) > 0) },
  { key: 'stats', labelKey: 'stats.graphTitle', icon: 'lucide:line-chart', inline: true, hidden: !props.enableTraffic },
  { key: 'del', labelKey: 'actions.del', icon: 'lucide:trash-2', tone: 'error', divider: true },
]

const handleAction = async (key: string, item: EndpointRow) => {
  switch (key) {
    case 'edit':
      emit('edit', item.id)
      break
    case 'qr':
      emit('qr', item.id)
      break
    case 'stats':
      emit('stats', item.tag)
      break
    case 'del': {
      const discard = await confirm({
        title: `${t('actions.del')} ${t('objects.endpoint')}`,
        message: item.tag,
        confirmLabel: t('actions.del'),
        tone: 'error',
      })

      if (discard) emit('del', item.tag)
      break
    }
  }
}
</script>

<style scoped>
.endpoints-nexus__tag {
  color: var(--nexus-text-primary);
  font-weight: 600;
}

.endpoints-nexus__muted {
  color: var(--nexus-text-muted);
}
</style>
