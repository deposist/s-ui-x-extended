<template>
  <div class="services-nexus">
    <page-header
      :search="search"
      searchable
      :subtitle="subtitle"
      :title="$t('pages.services')"
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
      <template #col.tag="{ item }">
        <span class="services-nexus__tag">{{ item.tag }}</span>
      </template>
      <template #col.listen="{ item }">
        <span v-if="item.type !== 'oom-killer' && item.listen" class="nexus-mono">{{ item.listen }}</span>
        <span v-else class="services-nexus__muted">—</span>
      </template>
      <template #col.listen_port="{ item }">
        <span v-if="item.type !== 'oom-killer' && item.listen_port" class="nexus-mono">{{ item.listen_port }}</span>
        <span v-else class="services-nexus__muted">—</span>
      </template>
      <template #col.tls="{ item }">
        <nexus-badge
          v-if="item.type !== 'oom-killer'"
          :label="(item.tls_id ?? 0) > 0 ? $t('nexus.on') : $t('nexus.off')"
          :variant="(item.tls_id ?? 0) > 0 ? 'success' : 'secondary'"
        />
        <span v-else class="services-nexus__muted">—</span>
      </template>
      <template #col.memory="{ item }">
        <span>{{ item.type === 'oom-killer' ? (item.memory_limit || '—') : '—' }}</span>
      </template>

      <template #actions="{ item }">
        <row-actions :actions="serviceActions()" @action="(key) => handleAction(key, item)" />
      </template>

      <template #empty>
        <!-- Distinguishes "search matched nothing" from "section is genuinely
             empty": the two states need different copy and different actions. -->
        <empty-state
          v-if="showNoMatches"
          :description="$t('table.noResultsHint')"
          icon="lucide:filter-x"
          :title="$t('table.noResults')"
        >
          <template #action>
            <v-btn prepend-icon="lucide:filter-x" variant="tonal" @click="search = ''">
              {{ $t('table.clearFilters') }}
            </v-btn>
          </template>
        </empty-state>
        <empty-state
          v-else
          :description="$t('emptyState.servicesHint')"
          icon="lucide:server"
          :title="$t('emptyState.services')"
        >
          <template #action>
            <v-btn color="primary" prepend-icon="lucide:plus" variant="flat" @click="emit('add')">
              {{ $t('actions.add') }}
            </v-btn>
          </template>
        </empty-state>
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
import { useConfirm } from '@/components/nexus/primitives/useConfirm'

interface ServiceRow {
  id: number
  tag: string
  type: string
  listen?: string
  listen_port?: number
  tls_id?: number
  memory_limit?: string
  [key: string]: unknown
}

const props = defineProps<{ services: ServiceRow[] }>()

const emit = defineEmits<{
  add: []
  edit: [id: number]
  del: [id: number]
}>()

const { t } = useI18n()
const { confirm } = useConfirm()
const search = ref('')

const subtitle = computed(() => t('nexus.summary.services', { total: props.services.length }))

const columns: Column<ServiceRow>[] = [
  { key: 'tag', labelKey: 'objects.tag', sortable: true },
  { key: 'type', labelKey: 'type', sortable: true },
  { key: 'listen', labelKey: 'in.addr' },
  { key: 'listen_port', labelKey: 'in.port', sortable: true },
  { key: 'tls', labelKey: 'objects.tls' },
  { key: 'memory', labelKey: 'types.oom.memoryLimit' },
]

const filtered = computed<ServiceRow[]>(() => {
  const query = search.value.trim().toLowerCase()

  if (!query) return props.services

  return props.services.filter(item => String(item.tag).toLowerCase().includes(query))
})

// A true-empty section takes precedence: telling someone to clear filters when
// they simply have no services yet would send them down the wrong path.
const showNoMatches = computed(
  () => search.value.trim().length > 0 && props.services.length > 0,
)

const serviceActions = (): RowAction[] => [
  { key: 'edit', labelKey: 'actions.edit', icon: 'lucide:pencil', inline: true },
  { key: 'del', labelKey: 'actions.del', icon: 'lucide:trash-2', tone: 'error', divider: true },
]

const handleAction = async (key: string, item: ServiceRow) => {
  if (key === 'edit') {
    emit('edit', item.id)
    return
  }

  const discard = await confirm({
    title: `${t('actions.del')} ${t('objects.service')}`,
    message: item.tag,
    confirmLabel: t('actions.del'),
    tone: 'error',
  })

  if (discard) emit('del', item.id)
}
</script>

<style scoped>
.services-nexus__tag {
  color: var(--nexus-text-primary);
  font-weight: 600;
}

.services-nexus__muted {
  color: var(--nexus-text-muted);
}
</style>
