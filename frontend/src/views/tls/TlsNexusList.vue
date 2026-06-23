<template>
  <div class="tls-nexus">
    <page-header
      :search="search"
      searchable
      :subtitle="subtitle"
      :title="$t('pages.tls')"
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
      <template #col.name="{ item }">
        <span class="tls-nexus__name">{{ item.name }}</span>
      </template>

      <template #col.server_name="{ item }">
        <span v-if="item.server?.server_name?.length" class="nexus-mono">{{ item.server.server_name }}</span>
        <span v-else class="tls-nexus__muted">—</span>
      </template>

      <template #col.inbounds="{ item }">
        <span v-if="usedBy(item.id).length">
          <v-tooltip activator="parent" dir="ltr" location="bottom">
            <span v-for="tag in usedBy(item.id)" :key="tag">{{ tag }}<br /></span>
          </v-tooltip>
          {{ usedBy(item.id).length }}
        </span>
        <span v-else class="tls-nexus__muted">—</span>
      </template>

      <template #col.acme="{ item }">
        <nexus-badge v-if="item.server?.acme != undefined" :label="$t('yes')" variant="success" />
        <span v-else class="tls-nexus__muted">—</span>
      </template>
      <template #col.ech="{ item }">
        <nexus-badge v-if="item.server?.ech != undefined" :label="$t('yes')" variant="success" />
        <span v-else class="tls-nexus__muted">—</span>
      </template>
      <template #col.reality="{ item }">
        <nexus-badge v-if="item.server?.reality != undefined" :label="$t('yes')" variant="success" />
        <span v-else class="tls-nexus__muted">—</span>
      </template>

      <template #actions="{ item }">
        <row-actions :actions="tlsActions(item)" @action="(key) => handleAction(key, item)" />
      </template>

      <template #empty>
        <empty-state icon="lucide:lock" :title="$t('table.noData')" />
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

interface TlsRow {
  id: number
  name: string
  server?: { server_name?: string; acme?: unknown; ech?: unknown; reality?: unknown }
  [key: string]: unknown
}

interface InboundRef { tls_id?: number; tag: string }

const props = defineProps<{
  tlsConfigs: TlsRow[]
  inbounds: InboundRef[]
}>()

const emit = defineEmits<{
  add: []
  edit: [id: number]
  clone: [item: TlsRow]
  del: [id: number]
}>()

const { t } = useI18n()
const { confirm } = useConfirm()
const search = ref('')

const subtitle = computed(() => {
  const total = props.tlsConfigs.length
  const acme = props.tlsConfigs.filter(c => c.server?.acme != undefined).length
  const reality = props.tlsConfigs.filter(c => c.server?.reality != undefined).length

  return t('nexus.summary.tls', { total, acme, reality })
})

const columns: Column<TlsRow>[] = [
  { key: 'name', labelKey: 'client.name', sortable: true },
  { key: 'server_name', labelKey: 'setting.domain' },
  { key: 'inbounds', labelKey: 'pages.inbounds' },
  { key: 'acme', labelKey: 'ACME' },
  { key: 'ech', labelKey: 'ECH' },
  { key: 'reality', labelKey: 'Reality' },
]

const usedBy = (id: number): string[] =>
  props.inbounds.filter(i => i.tls_id === id).map(i => i.tag)

const filtered = computed<TlsRow[]>(() => {
  const query = search.value.trim().toLowerCase()

  if (!query) return props.tlsConfigs

  return props.tlsConfigs.filter(item => String(item.name).toLowerCase().includes(query))
})

const tlsActions = (item: TlsRow): RowAction[] => [
  { key: 'edit', labelKey: 'actions.edit', icon: 'lucide:pencil', inline: true },
  { key: 'clone', labelKey: 'actions.clone', icon: 'lucide:copy', inline: true },
  // Delete-guard: a TLS config bound to an inbound cannot be deleted.
  { key: 'del', labelKey: 'actions.del', icon: 'lucide:trash-2', tone: 'error', divider: true, hidden: usedBy(item.id).length > 0 },
]

const handleAction = async (key: string, item: TlsRow) => {
  switch (key) {
    case 'edit':
      emit('edit', item.id)
      break
    case 'clone':
      emit('clone', item)
      break
    case 'del': {
      const discard = await confirm({
        title: `${t('actions.del')} ${t('objects.tls')}`,
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
.tls-nexus__name {
  color: var(--nexus-text-primary);
  font-weight: 600;
}

.tls-nexus__muted {
  color: var(--nexus-text-muted);
}
</style>
