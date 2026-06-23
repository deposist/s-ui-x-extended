<template>
  <div class="inbounds-nexus">
    <page-header
      :search="search"
      searchable
      :subtitle="subtitle"
      :title="$t('pages.inbounds')"
      @update:search="search = $event"
    />

    <page-toolbar>
      <template #actions>
        <v-btn
          color="primary"
          prepend-icon="lucide:plus"
          variant="flat"
          @click="emit('add')"
        >
          {{ $t('actions.add') }}
        </v-btn>
      </template>
    </page-toolbar>

    <nexus-data-table
      :columns="columns"
      :items="filtered"
      :row-key="(item) => item.id"
    >
      <template #col.status="{ item }">
        <status-badge
          v-if="onlines.includes(item.tag)"
          :label="$t('online')"
          tone="success"
        />
        <status-badge v-else :label="$t('nexus.status.offline')" tone="neutral" />
      </template>

      <template #col.tag="{ item }">
        <span class="inbounds-nexus__tag">{{ item.tag }}</span>
      </template>

      <template #col.listen="{ item }">
        <span class="nexus-mono">{{ item.listen }}</span>
      </template>

      <template #col.listen_port="{ item }">
        <span class="nexus-mono">{{ item.listen_port }}</span>
      </template>

      <template #col.tls="{ item }">
        <nexus-badge
          :label="item.tls_id > 0 ? $t('nexus.on') : $t('nexus.off')"
          :variant="item.tls_id > 0 ? 'success' : 'secondary'"
        />
      </template>

      <template #col.clients="{ item }">
        <span v-if="item.users && item.users.length">
          <v-tooltip activator="parent" dir="ltr" location="bottom">
            <span v-for="user in item.users" :key="user">{{ user }}<br /></span>
          </v-tooltip>
          {{ item.users.length }}
        </span>
        <span v-else class="inbounds-nexus__muted">—</span>
      </template>

      <template #actions="{ item }">
        <row-actions :actions="inboundActions(item)" @action="(key) => handleAction(key, item)" />
      </template>

      <template #empty>
        <empty-state icon="lucide:zap" :title="$t('table.noData')" />
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
import NexusBadge from '@/components/nexus/primitives/Badge.vue'
import EmptyState from '@/components/nexus/primitives/EmptyState.vue'
import PageHeader from '@/components/nexus/primitives/PageHeader.vue'
import PageToolbar from '@/components/nexus/primitives/PageToolbar.vue'
import StatusBadge from '@/components/nexus/primitives/StatusBadge.vue'
import { useConfirm } from '@/components/nexus/primitives/useConfirm'

interface InboundRow {
  id: number
  tag: string
  type: string
  listen: string
  listen_port: number
  tls_id: number
  users?: string[]
  [key: string]: unknown
}

const props = defineProps<{
  inbounds: InboundRow[]
  onlines: string[]
  enableTraffic: boolean
}>()

const emit = defineEmits<{
  add: []
  edit: [id: number]
  clone: [id: number]
  del: [id: number]
  stats: [tag: string]
}>()

const { t } = useI18n()
const { confirm } = useConfirm()
const search = ref('')

const subtitle = computed(() => {
  const total = props.inbounds.length
  const online = props.inbounds.filter(item => props.onlines.includes(item.tag)).length

  return t('nexus.summary.inbounds', { total, online })
})

const columns: Column<InboundRow>[] = [
  { key: 'status', labelKey: 'status' },
  { key: 'tag', labelKey: 'objects.tag', sortable: true },
  { key: 'type', labelKey: 'type', sortable: true },
  { key: 'listen', labelKey: 'in.addr' },
  { key: 'listen_port', labelKey: 'in.port', sortable: true },
  { key: 'tls', labelKey: 'objects.tls' },
  { key: 'clients', labelKey: 'pages.clients', align: 'end' },
]

const filtered = computed<InboundRow[]>(() => {
  const query = search.value.trim().toLowerCase()

  if (!query) return props.inbounds

  return props.inbounds.filter(item => String(item.tag).toLowerCase().includes(query))
})

// Named to NOT collide with the <row-actions> (RowActions) component: a
// camelCase `rowActions` binding would shadow the component in the template and
// Vue would render this function's return as text ([object Object]).
const inboundActions = (item: InboundRow): RowAction[] => [
  { key: 'edit', labelKey: 'actions.edit', icon: 'lucide:pencil', inline: true },
  { key: 'clone', labelKey: 'actions.clone', icon: 'lucide:copy', inline: true },
  { key: 'stats', labelKey: 'stats.graphTitle', icon: 'lucide:line-chart', inline: true, hidden: !props.enableTraffic },
  { key: 'del', labelKey: 'actions.del', icon: 'lucide:trash-2', tone: 'error', divider: true },
]

const handleAction = async (key: string, item: InboundRow) => {
  switch (key) {
    case 'edit':
      emit('edit', item.id)
      break
    case 'clone':
      emit('clone', item.id)
      break
    case 'stats':
      emit('stats', item.tag)
      break
    case 'del': {
      const discard = await confirm({
        title: `${t('actions.del')} ${t('objects.inbound')}`,
        message: item.tag,
        confirmLabel: t('actions.del'),
        tone: 'error',
      })

      if (discard) emit('del', item.id)
      break
    }
  }
}
</script>

<style scoped>
.inbounds-nexus__tag {
  color: var(--nexus-text-primary);
  font-weight: 600;
}

.inbounds-nexus__muted {
  color: var(--nexus-text-muted);
}
</style>
