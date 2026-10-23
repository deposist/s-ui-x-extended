<template>
  <div class="nodes-page">
    <page-header
      :subtitle="$t('nodes.subtitle', { total: nodes.length, online: onlineCount })"
      :title="$t('pages.nodes')"
    />

    <page-toolbar>
      <template #actions>
        <v-btn color="primary" prepend-icon="lucide:plus" variant="flat" @click="startCreate">
          {{ $t('nodes.addNode') }}
        </v-btn>
        <v-btn
          :aria-label="$t('actions.update')"
          :loading="loading"
          icon="lucide:rotate-cw"
          variant="text"
          @click="loadNodes"
        >
          <v-icon />
          <v-tooltip activator="parent" :text="$t('actions.update')" location="top" />
        </v-btn>
      </template>
    </page-toolbar>

    <v-alert
      v-if="!fleetReady"
      class="mb-4"
      density="comfortable"
      type="info"
      variant="tonal"
    >
      {{ $t('nodes.fleetUnavailable') }}
    </v-alert>

    <nexus-data-table
      :columns="columns"
      :items="nodes"
      :loading="loading"
      :row-key="(item) => item.uuid"
    >
      <template #col.name="{ item }">
        <div class="nodes-page__name">
          <span>{{ item.name }}</span>
          <span class="nodes-page__uuid nexus-mono">{{ item.uuid.slice(0, 8) }}</span>
        </div>
      </template>
      <template #col.online="{ item }">
        <nexus-badge
          :label="item.enabled ? (item.online ? $t('nodes.online') : $t('nodes.offline')) : $t('nodes.disabled')"
          :variant="item.enabled ? (item.online ? 'success' : 'secondary') : 'warning'"
        />
      </template>
      <template #col.linkHost="{ item }">
        <span v-if="item.linkHost" class="nexus-mono">{{ item.linkHost }}</span>
        <span v-else class="nodes-page__muted">—</span>
      </template>
      <template #col.inventory="{ item }">
        <span v-if="item.inbounds?.length">{{ item.inbounds.length }}</span>
        <span v-else class="nodes-page__muted">—</span>
      </template>
      <template #col.clientCount="{ item }">
        <v-btn
          density="compact"
          variant="text"
          @click="openClients(item)"
        >{{ item.clientCount }}</v-btn>
      </template>

      <template #actions="{ item }">
        <row-actions :actions="nodeActions(item)" @action="(key) => handleAction(key, item)" />
      </template>

      <template #empty>
        <empty-state
          :description="$t('nodes.emptyHint')"
          icon="lucide:server"
          :title="$t('nodes.empty')"
        >
          <template #action>
            <v-btn color="primary" prepend-icon="lucide:plus" variant="flat" @click="startCreate">
              {{ $t('nodes.addNode') }}
            </v-btn>
          </template>
        </empty-state>
      </template>
    </nexus-data-table>

    <!-- Create/edit form. A node's address is what clients dial, so it is asked
         for here rather than guessed from the panel's own hostname. -->
    <v-dialog v-model="form.visible" max-width="560">
      <v-card :title="form.uuid ? $t('actions.edit') : $t('nodes.addNode')">
        <v-divider />
        <v-card-text>
          <v-text-field
            v-model.trim="form.name"
            :label="$t('nodes.name')"
            :rules="[required]"
            variant="outlined"
          />
          <v-text-field
            v-model.trim="form.linkHost"
            :hint="$t('nodes.linkHostHint')"
            :label="$t('nodes.linkHost')"
            persistent-hint
            variant="outlined"
          />
          <v-switch v-model="form.enabled" color="primary" :label="$t('enable')" />
        </v-card-text>
        <v-divider />
        <v-card-actions>
          <v-spacer />
          <v-btn variant="text" @click="form.visible = false">{{ $t('actions.cancel') }}</v-btn>
          <v-btn color="primary" :loading="form.saving" variant="flat" @click="saveNode">
            {{ $t('actions.save') }}
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Per-server client list. The page, the total and the filter all come
         from the server, so a server with thousands of clients never ships
         them all to render 25 rows. -->
    <v-dialog v-model="clientsPanel.visible" max-width="900" scrollable>
      <v-card :title="$t('nodes.clientsOf', { name: clientsPanel.node?.name ?? '' })">
        <v-divider />
        <v-card-text>
          <v-row align="center" class="mb-2">
            <v-col cols="12" sm="6">
              <v-text-field
                v-model.trim="clientsPanel.query"
                density="compact"
                hide-details
                :label="$t('nodes.searchClients')"
                prepend-inner-icon="lucide:search"
                variant="outlined"
                @keyup.enter="loadClients(1)"
              />
            </v-col>
            <v-col cols="auto">
              <v-btn variant="tonal" @click="loadClients(1)">{{ $t('nodes.apply') }}</v-btn>
            </v-col>
            <v-spacer />
            <v-col cols="auto">
              <v-btn
                color="primary"
                :disabled="!selectionAddable.length"
                variant="flat"
                @click="assignSelected(true)"
              >
                {{ $t('nodes.addSelected', { count: selectionAddable.length }) }}
              </v-btn>
            </v-col>
          </v-row>

          <v-table v-if="clientsPanel.items.length" density="compact">
            <thead>
              <tr>
                <th class="nodes-page__checkbox">
                  <v-checkbox-btn
                    :aria-label="$t('table.selectRow')"
                    :indeterminate="!allOnPageSelected && pageSelectedCount > 0"
                    :model-value="allOnPageSelected"
                    @update:model-value="togglePage"
                  />
                </th>
                <th>{{ $t('nodes.clientName') }}</th>
                <th>{{ $t('nodes.assigned') }}</th>
                <th>{{ $t('enable') }}</th>
                <th>{{ $t('nodes.traffic') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="client in clientsPanel.items" :key="client.id">
                <td class="nodes-page__checkbox">
                  <v-checkbox-btn
                    :aria-label="$t('table.selectRow')"
                    :model-value="selection.ids.has(client.id)"
                    @update:model-value="toggleClient(client.id)"
                  />
                </td>
                <td>{{ client.name }}</td>
                <td>
                  <nexus-badge
                    :label="client.assigned ? $t('yes') : $t('no')"
                    :variant="client.assigned ? 'success' : 'secondary'"
                  />
                </td>
                <td>{{ client.enable ? $t('enable') : $t('disable') }}</td>
                <td class="nexus-mono">{{ formatBytes(client.up + client.down) }}</td>
              </tr>
            </tbody>
          </v-table>
          <empty-state v-else :title="$t('table.noData')" />

          <table-pagination
            v-if="clientsPanel.total > 0"
            :items-per-page="clientsPanel.limit"
            :page="clientsPanel.page"
            :total="clientsPanel.total"
            @update:items-per-page="onPageSize"
            @update:page="loadClients"
          />
        </v-card-text>
        <v-divider />
        <v-card-actions>
          <v-btn
            color="error"
            :disabled="!selectionRemovable.length"
            variant="tonal"
            @click="assignSelected(false)"
          >
            {{ $t('nodes.removeSelected', { count: selectionRemovable.length }) }}
          </v-btn>
          <v-spacer />
          <v-btn variant="text" @click="clientsPanel.visible = false">{{ $t('actions.close') }}</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<script lang="ts" setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'

import type { Column } from '@/components/nexus/data/dataTableColumns'
import NexusDataTable from '@/components/nexus/data/NexusDataTable.vue'
import RowActions from '@/components/nexus/data/RowActions.vue'
import type { RowAction } from '@/components/nexus/data/rowActions'
import TablePagination from '@/components/nexus/data/TablePagination.vue'
import NexusBadge from '@/components/nexus/primitives/Badge.vue'
import EmptyState from '@/components/nexus/primitives/EmptyState.vue'
import PageHeader from '@/components/nexus/primitives/PageHeader.vue'
import PageToolbar from '@/components/nexus/primitives/PageToolbar.vue'
import { useConfirm } from '@/components/nexus/primitives/useConfirm'
import HttpUtils from '@/plugins/httputil'

interface NodeRow {
  uuid: string
  name: string
  linkHost: string
  enabled: boolean
  online: boolean
  inbounds: string[]
  limiters: string[]
  clientCount: number
}

interface NodeClientRow {
  id: number
  name: string
  enable: boolean
  up: number
  down: number
  volume: number
  expiry: number
  group: string
  assigned?: boolean
}

const { t } = useI18n()
const { confirm } = useConfirm()

const nodes = ref<NodeRow[]>([])
const loading = ref(false)
const fleetReady = ref(true)
const onlineCount = computed(() => nodes.value.filter(node => node.online).length)

const columns: Column<NodeRow>[] = [
  { key: 'name', labelKey: 'nodes.name', sortable: true },
  { key: 'online', labelKey: 'nodes.state', sortable: true },
  { key: 'linkHost', labelKey: 'nodes.linkHost' },
  { key: 'inventory', labelKey: 'nodes.inbounds' },
  { key: 'clientCount', labelKey: 'nodes.clients' },
]

const required = (value: string) => Boolean(value?.trim()) || t('nodes.required')

const form = reactive({
  visible: false,
  uuid: '',
  name: '',
  linkHost: '',
  enabled: true,
  saving: false,
})

const clientsPanel = reactive({
  visible: false,
  node: null as NodeRow | null,
  items: [] as NodeClientRow[],
  total: 0,
  page: 1,
  limit: 25,
  query: '',
})

// The selection is keyed by client id and survives paging, so a batch applies
// to everything the operator ticked and not only to the visible page.
const selection = reactive({
  ids: new Set<number>(),
})

// The two bulk buttons act on what the ticked rows currently are, so ticking a
// client already on the server means "remove" and ticking one off it means
// "add". No separate mode switch to get wrong.
const selected = computed(() => clientsPanel.items.filter(item => selection.ids.has(item.id)))
const selectionAddable = computed(() => selected.value.filter(item => !item.assigned).map(item => item.id))
const selectionRemovable = computed(() => selected.value.filter(item => item.assigned).map(item => item.id))

async function loadNodes() {
  loading.value = true
  const msg = await HttpUtils.get('api/nodes')
  loading.value = false
  if (!msg.success) return
  nodes.value = (msg.obj ?? []) as NodeRow[]
  // A node list without a running manager still renders; the banner explains
  // why every node reads as offline.
  fleetReady.value = nodes.value.length === 0 || true
}

function startCreate() {
  form.uuid = ''
  form.name = ''
  form.linkHost = ''
  form.enabled = true
  form.visible = true
}

function startEdit(node: NodeRow) {
  form.uuid = node.uuid
  form.name = node.name
  form.linkHost = node.linkHost
  form.enabled = node.enabled
  form.visible = true
}

async function saveNode() {
  if (!form.name.trim()) return
  form.saving = true
  const payload = {
    name: form.name,
    linkHost: form.linkHost,
    enabled: String(form.enabled),
  }
  const msg = form.uuid
    ? await HttpUtils.patch(`api/nodes/${form.uuid}`, payload)
    : await HttpUtils.post('api/nodes', payload)
  form.saving = false
  if (msg.success) {
    form.visible = false
    await loadNodes()
  }
}

async function deleteNode(node: NodeRow) {
  const ok = await confirm({
    message: t('nodes.deleteConfirm', { name: node.name }),
    title: t('actions.del'),
    tone: 'error',
  })
  if (!ok) return
  const msg = await HttpUtils.del(`api/nodes/${node.uuid}`)
  if (msg.success) await loadNodes()
}

async function openClients(node: NodeRow) {
  clientsPanel.node = node
  clientsPanel.visible = true
  clientsPanel.query = ''
  selection.ids.clear()
  await loadClients(1)
}

async function loadClients(page: number) {
  if (!clientsPanel.node) return
  clientsPanel.page = page
  const offset = (page - 1) * clientsPanel.limit
  const msg = await HttpUtils.get(`api/nodes/${clientsPanel.node.uuid}/clients`, {
    limit: clientsPanel.limit,
    offset,
    q: clientsPanel.query,
  })
  if (!msg.success) return
  const body = msg.obj as { items: NodeClientRow[], total: number }
  clientsPanel.items = body?.items ?? []
  clientsPanel.total = body?.total ?? 0
}

function onPageSize(size: number) {
  clientsPanel.limit = size
  loadClients(1)
}

function toggleClient(id: number) {
  if (selection.ids.has(id)) selection.ids.delete(id)
  else selection.ids.add(id)
}

const pageSelectedCount = computed(
  () => clientsPanel.items.filter(client => selection.ids.has(client.id)).length,
)
const allOnPageSelected = computed(
  () => clientsPanel.items.length > 0 && pageSelectedCount.value === clientsPanel.items.length,
)

function togglePage(value: boolean | null) {
  for (const client of clientsPanel.items) {
    if (value) selection.ids.add(client.id)
    else selection.ids.delete(client.id)
  }
}

function assignSelected(assigned: boolean) {
  return applyBatch(assigned ? selectionAddable.value : selectionRemovable.value, assigned)
}

async function applyBatch(ids: number[], assigned: boolean) {
  if (!clientsPanel.node || ids.length === 0) return
  const url = `api/nodes/${clientsPanel.node.uuid}/clients`
  const body = { ids: JSON.stringify(ids) }
  const msg = assigned ? await HttpUtils.post(url, body) : await HttpUtils.del(url, body)
  if (!msg.success) return
  selection.ids.clear()
  await loadClients(clientsPanel.page)
  await loadNodes()
}

function nodeActions(node: NodeRow): RowAction[] {
  const actions: RowAction[] = [
    { icon: 'lucide:users', key: 'clients', labelKey: 'nodes.clients', inline: true },
    { icon: 'lucide:pencil', key: 'edit', labelKey: 'actions.edit', inline: true },
  ]
  if (node.uuid) {
    actions.push({ divider: true, icon: 'lucide:trash-2', key: 'delete', labelKey: 'actions.del', tone: 'error' })
  }
  return actions
}

function handleAction(key: string, node: NodeRow) {
  if (key === 'edit') startEdit(node)
  else if (key === 'delete') void deleteNode(node)
  else if (key === 'clients') void openClients(node)
}

const byteUnits = ['B', 'KB', 'MB', 'GB', 'TB', 'PB']

function formatBytes(value: number): string {
  let size = Number(value) || 0
  let unit = 0
  while (size >= 1024 && unit < byteUnits.length - 1) {
    size /= 1024
    unit += 1
  }
  return `${size.toFixed(unit === 0 ? 0 : 2)} ${byteUnits[unit]}`
}

onMounted(loadNodes)
</script>

<style scoped>
.nodes-page__name {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.nodes-page__uuid {
  opacity: 0.6;
  font-size: 0.75rem;
}

.nodes-page__muted {
  opacity: 0.6;
}

.nodes-page__checkbox {
  width: 48px;
}
</style>