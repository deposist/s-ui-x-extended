<template>
  <div class="nodes-page">
    <page-header
      :subtitle="$t('nodes.subtitle', { total: nodes.length, online: onlineCount })"
      :title="$t('pages.nodes')"
    />

    <node-panel-setup @role-change="panelRole = $event" />

    <template v-if="panelRole !== 'slave'">
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
          @click="loadNodes()"
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
          :label="item.syncState ? $t(`nodes.syncState.${item.syncState}`) : item.peerDisconnect ? $t('nodes.setup.disconnectPending') : item.enabled ? (item.online ? $t('nodes.online') : $t('nodes.offline')) : $t('nodes.disabled')"
          :variant="item.peerDisconnect ? 'warning' : item.enabled ? (item.online ? 'success' : 'secondary') : 'warning'"
        />
        <div v-if="item.applyState" class="nodes-page__muted">{{ $t(`nodes.applyState.${item.applyState}`) }}</div>
        <div v-if="item.blockedAssignments || item.peerBlocked" class="nodes-page__muted">{{ $t('nodes.setup.blockedAssignments', { count: item.blockedAssignments ?? item.peerBlocked }) }}</div>
      </template>
      <template #col.linkHost="{ item }">
        <span v-if="item.linkHost" class="nexus-mono">{{ item.linkHost }}</span>
        <span v-else class="nodes-page__muted">—</span>
      </template>
      <template #col.inventory="{ item }">
        <span v-if="item.paired && item.inbounds?.length">{{ item.inbounds.join(', ') }}</span>
        <span v-else-if="item.inbounds?.length">{{ item.inbounds.length }}</span>
        <span v-else class="nodes-page__muted">—</span>
      </template>
      <template #col.clientCount="{ item }">
        <span v-if="item.paired" class="nodes-page__muted">—</span>
        <v-btn
          v-else
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
          <v-row>
            <v-col cols="12" sm="6">
              <v-select
                v-model="form.limiterRoles.speedShared"
                item-title="title"
                item-value="value"
                :items="speedLimiterItems"
                :label="$t('nodes.sharedBandwidthLimiter')"
                variant="outlined"
              />
            </v-col>
            <v-col cols="12" sm="6">
              <v-select
                v-model="form.limiterRoles.speedDown"
                item-title="title"
                item-value="value"
                :items="speedLimiterItems"
                :label="$t('nodes.downloadBandwidthLimiter')"
                variant="outlined"
              />
            </v-col>
            <v-col cols="12" sm="6">
              <v-select
                v-model="form.limiterRoles.speedUp"
                item-title="title"
                item-value="value"
                :items="speedLimiterItems"
                :label="$t('nodes.uploadBandwidthLimiter')"
                variant="outlined"
              />
            </v-col>
            <v-col cols="12" sm="6">
              <v-select
                v-model="form.limiterRoles.connections"
                item-title="title"
                item-value="value"
                :items="connectionLimiterItems"
                :label="$t('nodes.connectionsLimiter')"
                variant="outlined"
              />
            </v-col>
          </v-row>
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
    <v-dialog v-model="clientsPanel.visible" :persistent="clientsPanel.saving" max-width="1050" scrollable>
      <v-card :title="$t('nodes.clientsOf', { name: clientsPanel.node?.name ?? '' })">
        <v-divider />
        <v-card-text class="nodes-page__client-workspace" :class="{ 'nodes-page__client-workspace--paired': clientsPanel.node?.paired }">
          <nav v-if="clientsPanel.node?.paired" class="nodes-page__inbounds" :aria-label="$t('nodes.setup.selectInbound')">
            <div class="text-subtitle-2">{{ $t('nodes.inbounds') }}</div>
            <v-btn
              v-for="inbound in clientsPanel.node.peerInbounds"
              :key="inbound.id"
              :variant="inbound.id === clientsPanel.inboundId ? 'tonal' : 'text'"
              :aria-pressed="inbound.id === clientsPanel.inboundId"
              :disabled="clientsPanel.saving"
              @click="selectPeerInbound(inbound.id)"
            >
              {{ inbound.tag }} ({{ inbound.type }})
            </v-btn>
            <div v-if="!clientsPanel.node.peerInbounds?.length" class="text-body-2">
              {{ $t('nodes.setup.noPeerInbounds') }}
            </div>
          </nav>
          <div class="nodes-page__client-list">
            <template v-if="currentPeerNode">
              <p v-if="currentPeerNode.applyState">{{ $t(`nodes.applyState.${currentPeerNode.applyState}`) }}</p>
              <p v-if="currentPeerNode.lastPeerSync">{{ $t('nodes.lastPeerSync', { time: formatTimestamp(currentPeerNode.lastPeerSync) }) }}</p>
              <p>{{ currentPeerNode.accessUntil ? $t('nodes.accessUntil', { time: formatTimestamp(currentPeerNode.accessUntil) }) : $t('nodes.accessUntilUnknown') }}</p>
              <v-btn v-if="canProbe" class="my-2" variant="tonal" :loading="startingProbe" :disabled="probeResult?.result === 'running'" @click="startProbe">{{ $t('nodes.probe.run') }}</v-btn>
              <div v-if="probeResult" class="mb-3" role="status">
                <div>{{ $t('nodes.probe.from', { time: formatTimestamp(probeResult.checkedAt) }) }}</div>
                <div>{{ $t(`nodes.probe.stage.${probeResult.stage}`) }} · {{ $t(`nodes.probe.result.${probeResult.result}`) }}
                  <span v-if="probeResult.result === 'success'"> ({{ probeResult.latencyMs ?? 0 }} ms)</span>
                </div>
              </div>
            </template>
            <template v-if="clientsPanel.node?.paired && canSelectClients">
              <v-alert type="info" variant="tonal" class="mb-3">{{ $t('nodes.setup.assignmentsNotApplied') }}</v-alert>
            </template>
            <v-alert v-if="assignmentResult" type="info" variant="tonal" class="mb-3" role="status">{{ assignmentResult }}</v-alert>
            <v-alert v-if="assignmentPreview" type="info" variant="tonal" class="mb-3">
              <p>{{ $t('nodes.preview.summary', { changes: assignmentPreview.changes, skipped: assignmentPreview.skipped }) }}</p>
              <p v-if="assignmentPreview.remaining">{{ $t('nodes.preview.remaining', { count: assignmentPreview.remaining }) }}</p>
              <ul class="nodes-page__preview">
                <li v-for="item in assignmentPreview.items" :key="item.clientId">
                  {{ item.name || item.clientId }} · {{ $t(`nodes.preview.effect.${item.effect}`) }}
                  <span v-if="item.reason"> · {{ $t(`nodes.preview.reason.${item.reason}`) }}</span>
                </li>
              </ul>
              <v-btn :loading="clientsPanel.saving" color="primary" variant="flat" @click="confirmAssignmentPreview">{{ $t('nodes.preview.apply') }}</v-btn>
              <v-btn :disabled="clientsPanel.saving" variant="text" @click="assignmentPreview = null">{{ $t('actions.cancel') }}</v-btn>
            </v-alert>
            <v-row align="center" class="mb-2">
              <v-col cols="12" sm="6">
                <v-text-field
                  v-model.trim="clientsPanel.query"
                  density="compact"
                  hide-details
                  :label="$t('nodes.searchClients')"
                  :disabled="!canSelectClients || clientsPanel.selecting || clientsPanel.saving"
                  prepend-inner-icon="lucide:search"
                  variant="outlined"
                  @keyup.enter="loadClients(1)"
                />
              </v-col>
              <v-col cols="auto">
                <v-btn variant="tonal" :disabled="!canSelectClients || clientsPanel.selecting || clientsPanel.saving" @click="loadClients(1)">{{ $t('nodes.apply') }}</v-btn>
              </v-col>
            </v-row>

            <div v-if="canSelectClients" class="nodes-page__bulk-tools mb-3">
              <v-btn
                variant="tonal"
                :disabled="!clientsPanel.total || clientsPanel.selecting || clientsPanel.saving"
                @click="selectMatching()"
              >
                {{ $t('nodes.selectAllResults') }}
              </v-btn>
              <v-select
                v-model="clientsPanel.group"
                class="nodes-page__group"
                :items="clientGroups"
                :label="$t('client.group')"
                density="compact"
                hide-details
                :disabled="clientsPanel.selecting || clientsPanel.saving"
              />
              <v-btn
                variant="tonal"
                :disabled="clientsPanel.group === null || clientsPanel.selecting || clientsPanel.saving"
                @click="selectMatching(clientsPanel.group ?? undefined)"
              >
                {{ $t('nodes.selectGroup') }}
              </v-btn>
            </div>
            <v-table v-if="clientsPanel.items.length" class="nodes-page__client-table" density="compact">
              <thead>
                <tr>
                  <th class="nodes-page__checkbox">
                    <v-checkbox-btn
                      :aria-label="$t('nodes.selectPage')"
                      :disabled="clientsPanel.selecting || clientsPanel.saving"
                      :indeterminate="!allOnPageSelected && pageSelectedCount > 0"
                      :model-value="allOnPageSelected"
                      @update:model-value="togglePage"
                    />
                  </th>
                  <th>{{ $t('nodes.clientName') }}</th>
                  <th class="nodes-page__assignment">{{ $t('nodes.assigned') }}</th>
                  <th class="nodes-page__desktop-column">{{ $t('enable') }}</th>
                  <th class="nodes-page__desktop-column">{{ $t('nodes.traffic') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="client in clientsPanel.items" :key="client.id">
                  <td class="nodes-page__checkbox">
                    <v-checkbox-btn
                      :aria-label="client.name"
                      :disabled="clientsPanel.selecting || clientsPanel.saving"
                      :model-value="selection.ids.has(client.id)"
                      @update:model-value="toggleClient(client.id)"
                    />
                  </td>
                  <td>
                    {{ client.name }}
                    <div v-if="client.group" class="nodes-page__muted text-caption">{{ client.group }}</div>
                    <div v-if="client.linkReason" class="nodes-page__muted text-caption">{{ $t(`nodes.linkReason.${client.linkReason}`) }}</div>
                    <div class="nodes-page__mobile-details text-caption">
                      <span>{{ client.enable ? $t('enable') : $t('disable') }}</span>
                      <span><span class="d-sr-only">{{ $t('nodes.traffic') }} </span>{{ formatBytes(client.up + client.down) }}</span>
                    </div>
                  </td>
                  <td>
                    <nexus-badge
                      :label="client.assigned ? $t('yes') : $t('no')"
                      :variant="client.assigned ? 'success' : 'secondary'"
                    />
                  </td>
                  <td class="nodes-page__desktop-column">{{ client.enable ? $t('enable') : $t('disable') }}</td>
                  <td class="nodes-page__desktop-column nexus-mono">{{ formatBytes(client.up + client.down) }}</td>
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
          </div>
        </v-card-text>
        <v-divider />
        <v-card-actions class="flex-wrap">
          <span role="status" class="mx-2">
            {{ clientsPanel.selecting ? $t('nodes.selectingClients') : $t('nodes.clientsSelected', { count: selection.ids.size }) }}
          </span>
          <v-btn
            variant="text"
            :disabled="!selection.ids.size || clientsPanel.selecting || clientsPanel.saving"
            @click="selection.ids.clear()"
          >
            {{ $t('nodes.clearSelection') }}
          </v-btn>
          <v-spacer />
          <v-btn
            color="error"
            :disabled="!selectionRemovable.length || clientsPanel.selecting || clientsPanel.saving"
            variant="tonal"
            @click="assignSelected(false)"
          >
            {{ $t('nodes.removeSelected', { count: selectionRemovable.length }) }}
          </v-btn>
          <v-btn
            color="primary"
            :disabled="!selectionAddable.length || clientsPanel.selecting || clientsPanel.saving"
            variant="flat"
            @click="assignSelected(true)"
          >
            {{ $t('nodes.addSelected', { count: selectionAddable.length }) }}
          </v-btn>
          <v-btn variant="text" :disabled="clientsPanel.saving" @click="clientsPanel.visible = false">{{ $t('actions.close') }}</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
    </template>
  </div>
</template>

<script lang="ts" setup>
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
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
import Data from '@/store/modules/data'
import NodePanelSetup from './NodePanelSetup.vue'

interface LimiterRoles {
  speedShared: string
  speedDown: string
  speedUp: string
  connections: string
}

interface NodeLimiter {
  tag: string
  type: string
  variant?: string
}

interface NodeRow {
  uuid: string
  paired?: boolean
  peerDisconnect?: boolean
  peerBlocked?: number
  blockedAssignments?: number
  lastPeerSync?: number
  accessUntil?: number
  syncState?: 'waiting' | 'online' | 'offline' | 'disabled' | 'disconnecting'
  applyState?: 'unknown' | 'pending' | 'applied'
  probeResults?: ProbeResult[]
  probeInboundIds?: number[]
  name: string
  linkHost: string
  enabled: boolean
  online: boolean
  inbounds: string[]
  peerInbounds?: { id: number; tag: string; type: string }[]
  limiters: NodeLimiter[]
  limiterRoles: LimiterRoles
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
  linkReason?: string
}

interface ProbeResult {
  inboundId: number
  stage: string
  result: string
  checkedAt: number
  latencyMs?: number
}

const { t } = useI18n()
const { confirm } = useConfirm()

const nodes = ref<NodeRow[]>([])
const panelRole = ref('')
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
  limiters: [] as NodeLimiter[],
  limiterRoles: { speedShared: '', speedDown: '', speedUp: '', connections: '' },
  saving: false,
})

const limiterItems = (type: string, selected: string) => [
  { title: t('nodes.noLimiter'), value: '' },
  ...[...new Set([...form.limiters.filter(item => item.type === type).map(item => item.tag), selected].filter(Boolean))]
    .map(value => ({ title: value, value })),
]
const speedLimiterItems = computed(() => limiterItems('bandwidth-limiter', form.limiterRoles.speedShared))
const connectionLimiterItems = computed(() => limiterItems('connection-limiter', form.limiterRoles.connections))

const clientsPanel = reactive({
  visible: false,
  node: null as NodeRow | null,
  inboundId: undefined as number | undefined,
  group: null as string | null,
  saving: false,
  selecting: false,
  items: [] as NodeClientRow[],
  total: 0,
  page: 1,
  limit: 25,
  query: '',
})

interface AssignmentPreview {
  changes: number
  skipped: number
  fingerprint: string
  items: { clientId: number; name?: string; effect: string; reason?: string }[]
  uuid: string
  inboundId: number
  ids: number[]
  assigned: boolean
  remaining: number
}

const assignmentPreview = ref<AssignmentPreview | null>(null)
const assignmentResult = ref('')
const currentPeerNode = computed(() => {
  const node = clientsPanel.node
  return node?.paired ? nodes.value.find(item => item.uuid === node.uuid) ?? node : null
})
const formatTimestamp = (timestamp: number) => new Date(timestamp * 1000).toLocaleString()
const startingProbe = ref(false)
const canProbe = computed(() => clientsPanel.inboundId && currentPeerNode.value?.probeInboundIds?.includes(clientsPanel.inboundId))
const probeResult = computed(() => currentPeerNode.value?.probeResults?.find(result => result.inboundId === clientsPanel.inboundId))

async function startProbe() {
  const node = currentPeerNode.value
  const inboundId = clientsPanel.inboundId
  if (!canProbe.value || !node || !inboundId || startingProbe.value) return
  startingProbe.value = true
  try {
    const response = await HttpUtils.post(`api/nodes/${node.uuid}/probe`, { inboundId }, { headers: { 'Content-Type': 'application/json' } })
    if (response.success && response.obj) {
      node.probeResults = [...(node.probeResults ?? []).filter(result => result.inboundId !== inboundId), response.obj]
      await loadNodes()
    }
  } finally {
    startingProbe.value = false
  }
}

const clientGroups = computed(() =>
  [...new Set<string>(Data().clients.map((client: { group?: string }) => client.group ?? ''))]
    .sort((a, b) => a.localeCompare(b))
    .map(group => ({ title: group || t('nodes.ungrouped'), value: group })),
)
const canSelectClients = computed(() => Boolean(
  clientsPanel.node && (!clientsPanel.node.paired || clientsPanel.inboundId),
))

// The selection is keyed by client id and survives paging, so a batch applies
// to everything the operator ticked and not only to the visible page.
const selection = reactive({
  ids: new Map<number, boolean>(),
})

// The two bulk buttons act on what the ticked rows currently are, so ticking a
// client already on the server means "remove" and ticking one off it means
// "add". No separate mode switch to get wrong.
const selectionAddable = computed(() => [...selection.ids].filter(([, assigned]) => !assigned).map(([id]) => id))
const selectionRemovable = computed(() => [...selection.ids].filter(([, assigned]) => assigned).map(([id]) => id))
let clientsRevision = 0
let nodesRevision = 0
let refreshingNodes = false
let nodesTimer: ReturnType<typeof setInterval>
onBeforeUnmount(() => {
  clearInterval(nodesTimer)
  clientsRevision++
  nodesRevision++
})

async function loadNodes(background = false) {
  const revision = ++nodesRevision
  if (background) refreshingNodes = true
  if (!background) loading.value = true
  try {
    const msg = await HttpUtils.get('api/nodes')
    if (!msg.success || revision !== nodesRevision) return
    nodes.value = (msg.obj ?? []) as NodeRow[]
  } finally {
    if (background) refreshingNodes = false
    if (revision === nodesRevision) loading.value = false
  }
}

function startCreate() {
  form.uuid = ''
  form.name = ''
  form.linkHost = ''
  form.enabled = true
  form.limiters = nodes.value.flatMap(node => node.limiters ?? [])
  Object.assign(form.limiterRoles, { speedShared: '', speedDown: '', speedUp: '', connections: '' })
  form.visible = true
}

function startEdit(node: NodeRow) {
  form.uuid = node.uuid
  form.name = node.name
  form.linkHost = node.linkHost
  form.enabled = node.enabled
  form.limiters = node.limiters ?? []
  Object.assign(form.limiterRoles, { speedShared: '', speedDown: '', speedUp: '', connections: '' })
  Object.assign(form.limiterRoles, node.limiterRoles ?? {})
  form.visible = true
}

async function saveNode() {
  if (!form.name.trim()) return
  form.saving = true
  const payload = {
    name: form.name,
    linkHost: form.linkHost,
    enabled: String(form.enabled),
    ...form.limiterRoles,
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

async function deleteNode(node: NodeRow, force = false) {
  const ok = await confirm({
    message: t(force ? 'nodes.setup.pairedDeleteConfirm' : node.paired ? 'nodes.setup.disconnectConfirm' : 'nodes.deleteConfirm', { name: node.name }),
    title: t(force ? 'nodes.setup.forceDisconnect' : node.paired ? 'nodes.setup.disconnect' : 'actions.del'),
    tone: 'error',
  })
  if (!ok) return
  const msg = await HttpUtils.del(`api/nodes/${node.uuid}${force ? '?force=true' : ''}`)
  if (msg.success) await loadNodes()
}

async function openClients(node: NodeRow) {
  assignmentPreview.value = null
  assignmentResult.value = ''
  clientsPanel.node = node
  clientsPanel.visible = true
  clientsPanel.query = ''
  clientsPanel.group = null
  await selectPeerInbound(node.peerInbounds?.length === 1 ? node.peerInbounds[0]?.id : undefined)
}

async function selectPeerInbound(inboundId?: number) {
  assignmentPreview.value = null
  assignmentResult.value = ''
  clientsRevision++
  clientsPanel.inboundId = inboundId
  clientsPanel.items = []
  clientsPanel.total = 0
  selection.ids.clear()
  await loadClients(1)
}

async function loadClients(page: number) {
  if (!clientsPanel.node || (clientsPanel.node.paired && !clientsPanel.inboundId)) return
  const revision = ++clientsRevision
  const nodeUUID = clientsPanel.node.uuid
  clientsPanel.page = page
  const offset = (page - 1) * clientsPanel.limit
  const msg = await HttpUtils.get(`api/nodes/${nodeUUID}/clients`, {
    limit: clientsPanel.limit,
    offset,
    q: clientsPanel.query,
    inboundId: clientsPanel.inboundId,
  })
  if (!msg.success || revision !== clientsRevision || nodeUUID !== clientsPanel.node?.uuid) return
  const body = msg.obj as { items: NodeClientRow[], total: number }
  clientsPanel.items = body?.items ?? []
  clientsPanel.total = body?.total ?? 0
  for (const client of clientsPanel.items) {
    if (selection.ids.has(client.id)) selection.ids.set(client.id, Boolean(client.assigned))
  }
}

async function selectMatching(group?: string) {
  const node = clientsPanel.node
  if (!node || !canSelectClients.value || clientsPanel.selecting || clientsPanel.saving) return
  const revision = clientsRevision
  const { inboundId, query } = clientsPanel
  const matches = new Map<number, boolean>()
  clientsPanel.selecting = true
  try {
    let offset = 0
    for (;;) {
      const msg = await HttpUtils.get(`api/nodes/${node.uuid}/clients`, { limit: 200, offset, q: query, inboundId })
      if (!msg.success || revision !== clientsRevision || !clientsPanel.visible || clientsPanel.query !== query) return
      const { items, total } = msg.obj as { items: NodeClientRow[], total: number }
      for (const item of items) {
        if (group === undefined || (item.group ?? '') === group) matches.set(item.id, Boolean(item.assigned))
      }
      offset += items.length
      if (!items.length || offset >= total) break
    }
    for (const [id, assigned] of matches) selection.ids.set(id, assigned)
  } finally {
    clientsPanel.selecting = false
  }
}

function onPageSize(size: number) {
  clientsPanel.limit = size
  loadClients(1)
}

function toggleClient(id: number) {
  if (selection.ids.has(id)) selection.ids.delete(id)
  else {
    const client = clientsPanel.items.find(item => item.id === id)
    if (client) selection.ids.set(id, Boolean(client.assigned))
  }
}

const pageSelectedCount = computed(
  () => clientsPanel.items.filter(client => selection.ids.has(client.id)).length,
)
const allOnPageSelected = computed(
  () => clientsPanel.items.length > 0 && pageSelectedCount.value === clientsPanel.items.length,
)

function togglePage(value: boolean | null) {
  for (const client of clientsPanel.items) {
    if (value) selection.ids.set(client.id, Boolean(client.assigned))
    else selection.ids.delete(client.id)
  }
}

function assignSelected(assigned: boolean) {
  return applyBatch(assigned ? selectionAddable.value : selectionRemovable.value, assigned)
}

async function applyBatch(ids: number[], assigned: boolean) {
  if (!clientsPanel.node || ids.length === 0 || clientsPanel.saving || clientsPanel.selecting) return
  if (clientsPanel.node.paired && !clientsPanel.inboundId) return
  const url = `api/nodes/${clientsPanel.node.uuid}/clients`
  const { inboundId } = clientsPanel
  if (clientsPanel.node.paired && inboundId) {
    const uuid = clientsPanel.node.uuid
    clientsPanel.saving = true
    assignmentPreview.value = null
    assignmentResult.value = ''
    try {
      const batch = ids.slice(0, 500)
      const response = await HttpUtils.post(`${url}/preview`, { inboundId, clientIds: batch, action: assigned ? 'add' : 'remove' }, {
        headers: { 'Content-Type': 'application/json' },
      })
      if (response.success && clientsPanel.node?.uuid === uuid && clientsPanel.inboundId === inboundId) {
        assignmentPreview.value = { ...response.obj, uuid, inboundId, ids: batch, assigned, remaining: ids.length - batch.length }
      }
    } finally {
      clientsPanel.saving = false
    }
    return
  }
  clientsPanel.saving = true
  try {
    const batchSize = 500
    let applied = false
    for (let start = 0; start < ids.length; start += batchSize) {
      const batch = ids.slice(start, start + batchSize)
      const body = { ids: JSON.stringify(batch), ...(inboundId ? { inboundId: String(inboundId) } : {}) }
      const msg = assigned ? await HttpUtils.post(url, new URLSearchParams(body)) : await HttpUtils.del(url, body)
      if (!msg.success) break
      applied = true
      for (const id of batch) selection.ids.delete(id)
    }
    if (applied) {
      await loadClients(clientsPanel.page)
      await loadNodes()
    }
  } finally {
    clientsPanel.saving = false
  }
}

async function confirmAssignmentPreview() {
  const preview = assignmentPreview.value
  if (!preview || clientsPanel.saving || clientsPanel.node?.uuid !== preview.uuid || clientsPanel.inboundId !== preview.inboundId) return
  clientsPanel.saving = true
  try {
    const url = `api/nodes/${preview.uuid}/clients`
    const body = { ids: JSON.stringify(preview.ids), inboundId: String(preview.inboundId), previewFingerprint: preview.fingerprint }
    const response = preview.assigned ? await HttpUtils.post(url, new URLSearchParams(body)) : await HttpUtils.del(url, body)
    if (response.obj?.reason === 'stale_preview') {
      assignmentPreview.value = null
      assignmentResult.value = t('nodes.preview.stale')
      return
    }
    if (!response.success) return
    assignmentPreview.value = null
    assignmentResult.value = t(preview.assigned ? 'nodes.preview.addResult' : 'nodes.preview.removeResult', { count: response.obj?.changed ?? 0 })
    for (const id of preview.ids) selection.ids.delete(id)
    await loadClients(clientsPanel.page)
    await loadNodes()
  } finally {
    clientsPanel.saving = false
  }
}

function nodeActions(node: NodeRow): RowAction[] {
  const actions: RowAction[] = [
    { icon: 'lucide:users', key: 'clients', labelKey: 'nodes.clients', inline: true },
  ]
  if (!node.paired) actions.push({ icon: 'lucide:pencil', key: 'edit', labelKey: 'actions.edit', inline: true })
  if (node.paired) {
    if (!node.peerDisconnect) actions.push({ divider: true, icon: 'lucide:unlink', key: 'delete', labelKey: 'nodes.setup.disconnect', tone: 'error' })
    actions.push({ icon: 'lucide:alert-triangle', key: 'force', labelKey: 'nodes.setup.forceDisconnect', tone: 'error' })
  } else if (node.uuid) {
    actions.push({ divider: true, icon: 'lucide:trash-2', key: 'delete', labelKey: 'actions.del', tone: 'error' })
  }
  return actions
}

function handleAction(key: string, node: NodeRow) {
  if (key === 'edit') startEdit(node)
  else if (key === 'delete') void deleteNode(node)
  else if (key === 'force') void deleteNode(node, true)
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

onMounted(() => {
  void loadNodes().then(() => {
    if (typeof window === 'undefined') return
    const uuid = new URLSearchParams(window.location.search).get('node')
    const node = nodes.value.find(item => item.uuid === uuid)
    if (node && !clientsPanel.visible) void openClients(node)
  })
  nodesTimer = setInterval(() => {
    if (panelRole.value === 'master' && !loading.value && !refreshingNodes) void loadNodes(true)
  }, 5000)
})
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

.nodes-page__client-workspace {
  display: grid;
  gap: 16px;
}

.nodes-page__client-workspace--paired {
  grid-template-columns: minmax(140px, 185px) minmax(0, 1fr);
}

.nodes-page__inbounds {
  align-self: start;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.nodes-page__client-list {
  min-width: 0;
}

.nodes-page__preview {
  max-height: 12rem;
  overflow-y: auto;
  padding-inline-start: 1.25rem;
  margin-block: 0.5rem;
}

.nodes-page__bulk-tools {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}

.nodes-page__group {
  flex: 1 1 160px;
  min-width: 0;
}

.nodes-page__mobile-details {
  display: none;
}

@media (max-width: 600px) {
  .nodes-page__client-workspace--paired {
    grid-template-columns: minmax(0, 1fr);
  }

  .nodes-page__client-table :deep(table) {
    table-layout: fixed;
    width: 100%;
  }

  .nodes-page__client-table :deep(th),
  .nodes-page__client-table :deep(td) {
    padding-inline: 6px;
    overflow-wrap: anywhere;
  }

  .nodes-page__client-table .nodes-page__checkbox {
    width: 44px;
  }

  .nodes-page__assignment {
    width: 34%;
  }

  .nodes-page__desktop-column {
    display: none;
  }

  .nodes-page__mobile-details {
    display: flex;
    flex-wrap: wrap;
    column-gap: 8px;
  }
}
</style>
