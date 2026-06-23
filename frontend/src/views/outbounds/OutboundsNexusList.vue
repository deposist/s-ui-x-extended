<template>
  <div class="outbounds-nexus">
    <page-header
      :search="search"
      searchable
      :subtitle="subtitle"
      :title="$t('pages.outbounds')"
      @update:search="search = $event"
    />

    <page-toolbar>
      <template #actions>
        <v-btn color="primary" prepend-icon="lucide:plus" variant="flat" @click="emit('add')">
          {{ $t('actions.add') }}
        </v-btn>
        <v-btn prepend-icon="lucide:plus" variant="tonal" @click="emit('addBulk')">
          {{ $t('actions.addbulk') }}
        </v-btn>
        <v-btn
          append-icon="lucide:gauge"
          :disabled="testingAll || outbounds.length === 0"
          :loading="testingAll"
          variant="outlined"
          @click="emit('testAll')"
        >
          {{ $t('actions.testAll') }}
        </v-btn>
      </template>
    </page-toolbar>

    <nexus-data-table :columns="columns" :items="filtered" :row-key="(item) => item.id">
      <template #col.status="{ item }">
        <status-badge v-if="onlines.includes(item.tag)" :label="$t('online')" tone="success" />
        <status-badge v-else :label="$t('nexus.status.offline')" tone="neutral" />
      </template>

      <template #col.tag="{ item }">
        <span class="outbounds-nexus__tag">{{ item.tag }}</span>
      </template>

      <template #col.server="{ item }">
        <template v-if="item.type === 'failover'">
          <div class="outbounds-nexus__failover">
            <span v-if="failoverActive(item.tag)" class="nexus-mono outbounds-nexus__active">→ {{ failoverActive(item.tag) }}</span>
            <span v-if="failoverMembers(item.tag).length" class="outbounds-nexus__dots">
              <span
                v-for="m in failoverMembers(item.tag)"
                :key="m.tag"
                class="outbounds-nexus__dot"
                :class="{
                  'outbounds-nexus__dot--up': m.healthy,
                  'outbounds-nexus__dot--down': !m.healthy,
                  'outbounds-nexus__dot--active': m.tag === failoverActive(item.tag),
                }"
                :title="m.tag + ': ' + (m.healthy ? $t('online') : $t('nexus.status.offline'))"
                :aria-label="m.tag + ': ' + (m.healthy ? $t('online') : $t('nexus.status.offline'))"
              />
            </span>
            <nexus-badge v-if="failoverAllDown(item.tag)" :label="$t('types.failover.allDown')" variant="secondary" class="ml-1" />
            <span
              v-if="!failoverActive(item.tag) && !failoverAllDown(item.tag) && !failoverMembers(item.tag).length"
              class="outbounds-nexus__muted"
            >—</span>
          </div>
        </template>
        <span v-else-if="item.server" class="nexus-mono">{{ item.server }}</span>
        <span v-else class="outbounds-nexus__muted">—</span>
      </template>

      <template #col.server_port="{ item }">
        <span v-if="item.server_port" class="nexus-mono">{{ item.server_port }}</span>
        <span v-else class="outbounds-nexus__muted">—</span>
      </template>

      <template #col.tls="{ item }">
        <nexus-badge
          v-if="item.tls"
          :label="item.tls.enabled ? $t('nexus.on') : $t('nexus.off')"
          :variant="item.tls.enabled ? 'success' : 'secondary'"
        />
        <span v-else class="outbounds-nexus__muted">—</span>
      </template>

      <template #col.delay="{ item }">
        <div class="outbounds-nexus__delay">
          <v-progress-circular v-if="checkResults[item.tag]?.loading" indeterminate size="18" />
          <template v-else>
            <v-btn
              :aria-label="$t('actions.test')"
              density="comfortable"
              icon="lucide:gauge"
              size="small"
              :title="$t('actions.test')"
              variant="text"
              @click="emit('test', item.tag)"
            />
            <v-chip
              v-if="checkResults[item.tag] && checkResults[item.tag].success"
              color="success"
              density="compact"
              size="small"
              variant="flat"
            >
              {{ checkResults[item.tag].data?.Delay }}{{ $t('date.ms') }}
            </v-chip>
            <v-icon
              v-else-if="checkResults[item.tag] && checkResults[item.tag].loading === false"
              color="error"
              icon="lucide:x-circle"
              size="small"
              :title="checkResults[item.tag].errorMessage || $t('failed')"
            />
          </template>
        </div>
      </template>

      <template #actions="{ item }">
        <row-actions :actions="outboundActions(item)" @action="(key) => handleAction(key, item)" />
      </template>

      <template #empty>
        <empty-state icon="lucide:arrow-up-right" :title="$t('table.noData')" />
      </template>
    </nexus-data-table>
  </div>
</template>

<script lang="ts" setup>
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'

import api from '@/plugins/api'

import type { FailoverStatusEntry, FailoverMemberStatus } from '@/types/outbounds'

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

interface CheckResult {
  loading?: boolean
  success: boolean
  data?: { OK?: boolean; Delay?: number; Error?: string } | null
  errorMessage?: string
}

interface OutboundRow {
  id: number
  tag: string
  type: string
  server?: string
  server_port?: number
  tls?: { enabled?: boolean }
  [key: string]: unknown
}

const props = defineProps<{
  outbounds: OutboundRow[]
  onlines: string[]
  failover: Record<string, FailoverStatusEntry>
  enableTraffic: boolean
  checkResults: Record<string, CheckResult>
  testingAll: boolean
}>()

const emit = defineEmits<{
  add: []
  addBulk: []
  testAll: []
  test: [tag: string]
  edit: [id: number]
  del: [tag: string]
  stats: [tag: string]
}>()

const { t } = useI18n()
const { confirm } = useConfirm()
const search = ref('')

// Failover groups expose their active member + per-member health live through the
// onlines realtime push (the `failover` prop). A single fetch on mount seeds the
// crash-safe state (from failover_state) for the brief window after a panel
// restart, before the manager's first probe repopulates the live snapshot.
const initialFailover = ref<Record<string, FailoverStatusEntry>>({})

const failoverByTag = computed<Record<string, FailoverStatusEntry>>(() => {
  return Object.keys(props.failover ?? {}).length ? props.failover : initialFailover.value
})

const failoverActive = (tag: string): string => failoverByTag.value[tag]?.active || ''
const failoverAllDown = (tag: string): boolean => failoverByTag.value[tag]?.allDown === true
const failoverMembers = (tag: string): FailoverMemberStatus[] => failoverByTag.value[tag]?.members ?? []

onMounted(async () => {
  if (!props.outbounds.some(item => item.type === 'failover')) return
  try {
    const resp = await api.get('api/failover-status')
    const entries = (resp?.data?.obj ?? []) as FailoverStatusEntry[]
    initialFailover.value = Object.fromEntries(entries.map(entry => [entry.tag, entry]))
  } catch {
    // Silent: the realtime onlines.failover push will populate shortly.
  }
})

const subtitle = computed(() => {
  const total = props.outbounds.length
  const online = props.outbounds.filter(item => props.onlines.includes(item.tag)).length

  return t('nexus.summary.outbounds', { total, online })
})

const columns: Column<OutboundRow>[] = [
  { key: 'status', labelKey: 'status' },
  { key: 'tag', labelKey: 'objects.tag', sortable: true },
  { key: 'type', labelKey: 'type', sortable: true },
  { key: 'server', labelKey: 'in.addr' },
  { key: 'server_port', labelKey: 'in.port', sortable: true },
  { key: 'tls', labelKey: 'objects.tls' },
  { key: 'delay', labelKey: 'out.delay' },
]

const filtered = computed<OutboundRow[]>(() => {
  const query = search.value.trim().toLowerCase()

  if (!query) return props.outbounds

  return props.outbounds.filter(item => String(item.tag).toLowerCase().includes(query))
})

const outboundActions = (item: OutboundRow): RowAction[] => [
  { key: 'edit', labelKey: 'actions.edit', icon: 'lucide:pencil', inline: true },
  { key: 'stats', labelKey: 'stats.graphTitle', icon: 'lucide:line-chart', inline: true, hidden: !props.enableTraffic },
  { key: 'del', labelKey: 'actions.del', icon: 'lucide:trash-2', tone: 'error', divider: true },
]

const handleAction = async (key: string, item: OutboundRow) => {
  switch (key) {
    case 'edit':
      emit('edit', item.id)
      break
    case 'stats':
      emit('stats', item.tag)
      break
    case 'del': {
      const discard = await confirm({
        title: `${t('actions.del')} ${t('objects.outbound')}`,
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
.outbounds-nexus__tag {
  color: var(--nexus-text-primary);
  font-weight: 600;
}

.outbounds-nexus__muted {
  color: var(--nexus-text-muted);
}

.outbounds-nexus__active {
  color: var(--nexus-text-primary);
  font-weight: 600;
}

.outbounds-nexus__failover {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: var(--nexus-gap-2);
}

.outbounds-nexus__dots {
  align-items: center;
  display: inline-flex;
  gap: 3px;
}

.outbounds-nexus__dot {
  border-radius: 50%;
  display: inline-block;
  height: 8px;
  width: 8px;
}

.outbounds-nexus__dot--up {
  background: rgb(var(--v-theme-success));
}

.outbounds-nexus__dot--down {
  background: rgb(var(--v-theme-error));
}

.outbounds-nexus__dot--active {
  outline: 1.5px solid rgb(var(--v-theme-primary));
  outline-offset: 1px;
}

.outbounds-nexus__delay {
  align-items: center;
  display: flex;
  gap: var(--nexus-gap-2);
}
</style>
