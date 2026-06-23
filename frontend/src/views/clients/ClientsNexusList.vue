<template>
  <div class="clients-nexus">
    <page-header
      :search="search"
      searchable
      :subtitle="subtitle"
      :title="$t('pages.clients')"
      @update:search="search = $event"
    />

    <page-toolbar>
      <template #filters>
        <v-select
          density="compact"
          hide-details
          :items="stateItems"
          :label="$t('type')"
          :model-value="filterState"
          variant="outlined"
          @update:model-value="filterState = $event"
        />
        <v-select
          density="compact"
          hide-details
          :items="groupItems"
          :label="$t('client.group')"
          :model-value="filterGroup"
          variant="outlined"
          @update:model-value="filterGroup = $event"
        />
      </template>
      <template #actions>
        <v-btn color="primary" prepend-icon="lucide:plus" variant="flat" @click="emit('add')">
          {{ $t('actions.add') }}
        </v-btn>
        <v-menu>
          <template #activator="{ props }">
            <v-btn :aria-label="$t('actions.action')" icon="lucide:wrench" variant="text" v-bind="props" />
          </template>
          <v-list density="compact">
            <v-list-item prepend-icon="lucide:user-plus" :title="$t('actions.addbulk')" @click="emit('addBulk')" />
            <v-list-item prepend-icon="lucide:user-check" :title="$t('actions.editbulk')" @click="emit('editBulk')" />
          </v-list>
        </v-menu>
      </template>
    </page-toolbar>

    <nexus-data-table :columns="columns" :items="filtered" :row-key="(item) => item.id">
      <template #col.name="{ item }">
        <span class="clients-nexus__name">{{ item.name }}</span>
      </template>
      <template #col.inbounds="{ item }">
        <span v-if="item.inbounds && item.inbounds.length">
          <v-tooltip activator="parent" dir="ltr" location="start">
            <span v-for="i in item.inbounds" :key="i">{{ inboundTag(i) }}<br /></span>
          </v-tooltip>
          {{ item.inbounds.length }}
        </span>
        <span v-else class="clients-nexus__muted">—</span>
      </template>
      <template #col.volume="{ item }">
        <v-chip :color="volumeColor(item)" label size="small">
          {{ humanVolume(item) }}
        </v-chip>
        <v-progress-linear
          v-if="item.volume > 0"
          :color="percentColor(item)"
          :model-value="percent(item)"
        />
      </template>
      <template #col.expiry="{ item }">
        <v-chip :color="expiryColor(item)" label size="small">
          {{ remainedDays(item.expiry) }}
        </v-chip>
      </template>
      <template #col.online="{ item }">
        <status-badge v-if="onlines.includes(item.name)" :label="$t('online')" tone="success" />
        <status-badge
          v-else-if="item.enable === false"
          :label="$t('disable')"
          tone="neutral"
        />
        <status-badge v-else :label="$t('nexus.status.offline')" tone="neutral" />
      </template>
      <template #col.lastIpCount="{ item }">
        <v-chip label size="small" @click="emit('showIps', item.name)">
          {{ item.lastIpCount ?? 0 }}
        </v-chip>
      </template>

      <template #actions="{ item }">
        <row-actions :actions="clientActions()" @action="(key) => handleAction(key, item)" />
      </template>

      <template #empty>
        <empty-state icon="lucide:users" :title="$t('table.noData')" />
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
import { HumanReadable } from '@/plugins/utils'

interface ClientRow {
  id: number
  name: string
  desc?: string
  group?: string
  enable?: boolean
  inbounds?: number[]
  up: number
  down: number
  volume: number
  expiry: number
  lastIpCount?: number
  [key: string]: unknown
}

const props = defineProps<{
  clients: ClientRow[]
  inbounds: { id: number; tag: string }[]
  groups: string[]
  onlines: string[]
  enableTraffic: boolean
}>()

const emit = defineEmits<{
  add: []
  addBulk: []
  editBulk: []
  edit: [id: number]
  del: [id: number]
  qr: [id: number]
  diagnose: [id: number]
  stats: [name: string]
  showIps: [name: string]
}>()

const { t } = useI18n()
const { confirm } = useConfirm()
const search = ref('')
const filterState = ref('')
const filterGroup = ref('-')

const subtitle = computed(() => {
  const total = props.clients.length
  const online = props.clients.filter(c => props.onlines.includes(c.name)).length

  return t('nexus.summary.clients', { total, online })
})

const columns: Column<ClientRow>[] = [
  { key: 'name', labelKey: 'client.name', sortable: true },
  { key: 'desc', labelKey: 'client.desc' },
  { key: 'group', labelKey: 'client.group', sortable: true },
  { key: 'inbounds', labelKey: 'pages.inbounds' },
  { key: 'volume', labelKey: 'stats.volume' },
  { key: 'expiry', labelKey: 'date.expiry' },
  { key: 'online', labelKey: 'status' },
  { key: 'lastIpCount', labelKey: 'client.lastIpCount' },
]

const stateItems = computed(() => [
  { title: t('none'), value: '' },
  { title: t('disable'), value: 'disable' },
  { title: t('date.expired'), value: 'expired' },
  { title: t('online'), value: 'online' },
])
const groupItems = computed(() => [
  { title: t('all'), value: '-' },
  ...props.groups.map(g => ({ title: g.length > 0 ? g : t('none'), value: g })),
])

// Mirrors the classic doFilter() criteria; kept presentational so the parent's
// store-bound filter is untouched for Classic mode.
const filtered = computed<ClientRow[]>(() => {
  let rows = props.clients.slice()
  const query = search.value.trim().toLowerCase()

  if (query) rows = rows.filter(c => c.name.toLowerCase().includes(query) || (c.desc ?? '').toLowerCase().includes(query))
  if (filterGroup.value !== '-') rows = rows.filter(c => c.group === filterGroup.value)

  if (filterState.value === 'disable') rows = rows.filter(c => c.enable === false)
  else if (filterState.value === 'expired') rows = rows.filter(c => c.expiry > 0 && c.expiry < Date.now() / 1000)
  else if (filterState.value === 'online') rows = rows.filter(c => props.onlines.includes(c.name))

  return rows
})

const inboundTag = (id: number) => props.inbounds.find(i => i.id === id)?.tag ?? id
const percent = (c: ClientRow) => (c.volume > 0 ? Math.round((c.up + c.down) * 100 / c.volume) : 0)
const percentColor = (c: ClientRow) => ((c.up + c.down) >= c.volume ? 'error' : percent(c) > 90 ? 'warning' : 'success')
const humanVolume = (c: ClientRow) =>
  HumanReadable.sizeFormat(c.up + c.down) + ' / ' + (c.volume === 0 ? t('unlimited') : HumanReadable.sizeFormat(c.volume))
const volumeColor = (c: ClientRow) => (c.volume === 0 ? 'success' : c.volume <= (c.up + c.down) ? 'error' : undefined)
const remainedDays = (expiry: number) => HumanReadable.remainedDays(expiry)
const expiryColor = (c: ClientRow) => (c.expiry === 0 ? 'success' : c.expiry <= Date.now() / 1000 ? 'error' : undefined)

const clientActions = (): RowAction[] => [
  { key: 'edit', labelKey: 'actions.edit', icon: 'lucide:pencil', inline: true },
  { key: 'qr', labelKey: 'objects.config', icon: 'lucide:qr-code', inline: true },
  { key: 'diagnose', labelKey: 'actions.diagnose', icon: 'lucide:activity', inline: true },
  { key: 'stats', labelKey: 'stats.graphTitle', icon: 'lucide:line-chart', inline: true, hidden: !props.enableTraffic },
  { key: 'del', labelKey: 'actions.del', icon: 'lucide:trash-2', tone: 'error', divider: true },
]

const handleAction = async (key: string, item: ClientRow) => {
  switch (key) {
    case 'edit':
      emit('edit', item.id)
      break
    case 'qr':
      emit('qr', item.id)
      break
    case 'diagnose':
      emit('diagnose', item.id)
      break
    case 'stats':
      emit('stats', item.name)
      break
    case 'del': {
      const discard = await confirm({
        title: `${t('actions.del')} ${t('objects.client')}`,
        message: item.name,
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
.clients-nexus__name {
  color: var(--nexus-text-primary);
  font-weight: 600;
}

.clients-nexus__muted {
  color: var(--nexus-text-muted);
}
</style>
