<template>
  <DnsVue
    v-model="dnsModal.visible"
    :visible="dnsModal.visible"
    :index="dnsModal.index"
    :data="dnsModal.data"
    :tsTags="tsTags"
    :rslvdTags="rslvdTags"
    @close="closeDnsModal"
    @save="saveDnsModal"
  />
  <DnsRuleVue
    v-model="dnsRuleModal.visible"
    :visible="dnsRuleModal.visible"
    :index="dnsRuleModal.index"
    :data="dnsRuleModal.data"
    :clients="clients"
    :inTags="inboundTags"
    :serverTags="dnsServerTags"
    :ruleSets="ruleSets"
    @close="closeDnsRuleModal"
    @save="saveDnsRuleModal"
  />
  <RegionalPresetDrawer
    v-model="regionalPresetDrawer"
    :config="appConfig"
    :outbound-tags="outboundTags"
    @apply="applyPresetConfig"
  />
  <page-header
    v-if="nexus"
    :search="search"
    searchable
    :subtitle="subtitle"
    :title="$t('pages.dns')"
    @update:search="search = $event"
  />

  <page-toolbar v-if="nexus">
    <template #secondary-actions>
      <v-btn color="primary" prepend-icon="lucide:plus" variant="flat" @click="showDnsModal(-1)">{{ $t('dns.add') }}</v-btn>
      <v-btn prepend-icon="lucide:plus" variant="text" @click="showDnsRuleModal(-1)">{{ $t('dns.rule.add') }}</v-btn>
      <v-btn prepend-icon="mdi-routes" variant="text" @click="regionalPresetDrawer = true">{{ $t('regionalPresets.open') }}</v-btn>
    </template>
    <template #primary-actions>
      <v-btn variant="tonal" color="warning" @click="saveConfig" :loading="loading" :disabled="stateChange">
        {{ $t('actions.save') }}
      </v-btn>
    </template>
  </page-toolbar>
  <v-row v-else>
    <v-col cols="12" justify="center" align="center">
      <v-btn color="primary" @click="showDnsModal(-1)" style="margin: 0 5px;">{{ $t('dns.add') }}</v-btn>
      <v-btn color="primary" @click="showDnsRuleModal(-1)" style="margin: 0 5px;">{{ $t('dns.rule.add') }}</v-btn>
      <v-btn color="primary" @click="regionalPresetDrawer = true" style="margin: 0 5px;">{{ $t('regionalPresets.open') }}</v-btn>
      <v-btn variant="outlined" color="warning" @click="saveConfig" :loading="loading" :disabled="stateChange">
        {{ $t('actions.save') }}
      </v-btn>
    </v-col>
  </v-row>
  <v-row>
    <v-col class="v-card-subtitle" cols="12">{{ $t('pages.basics') }}</v-col>
    <v-col cols="12">
      <v-row>
        <v-col cols="12" sm="6" md="3" lg="2">
          <v-select
            hide-details
            :label="$t('dns.final')"
            :items="[ {title: $t('dns.firstServer'), value: ''}, ...dnsServerTags]"
            v-model="finalDns">
          </v-select>
        </v-col>
        <v-col cols="12" sm="6" md="3" lg="2">
          <v-select
            hide-details
            :label="$t('dns.domainStrategy')"
            clearable
            @click:clear="delete dns.strategy"
            :items="['prefer_ipv4','prefer_ipv6','ipv4_only','ipv6_only']"
            v-model="dns.strategy">
          </v-select>
        </v-col>
        <v-col cols="12" sm="6" md="3" lg="2">
          <v-text-field
            v-model="dns.client_subnet" hide-details
            clearable @click:clear="delete dns.client_subnet"
            :label="$t('dns.rule.action.clientSubnet')"></v-text-field>
        </v-col>
        <v-col cols="auto">
          <v-text-field
            v-model.number="dns.cache_capacity"
            type="number" min="1024" hide-details
            clearable @click:clear="delete dns.cache_capacity"
            :label="$t('dns.cacheCapacity')"></v-text-field>
        </v-col>
        <v-col cols="auto">
          <v-checkbox v-model="dns.disable_cache" hide-details :label="$t('dns.disableCache')" />
        </v-col>
        <v-col cols="auto">
          <v-checkbox v-model="dns.disable_expire" hide-details :label="$t('dns.disableExpire')" />
        </v-col>
        <v-col cols="auto">
          <v-checkbox v-model="dns.independent_cache" hide-details :label="$t('dns.independentCache')" />
        </v-col>
        <v-col cols="auto">
          <v-checkbox v-model="dns.reverse_mapping" hide-details :label="$t('dns.reverseMapping')" />
        </v-col>
      </v-row>
    </v-col>
  </v-row>
  <template v-if="nexus">
    <div class="dns-nexus__section">{{ $t('dns.title') }}</div>
    <nexus-data-table :columns="serverColumns" :items="dnsServerRows" :row-key="(item) => item._index">
      <template #col.tag="{ item }"><span class="dns-nexus__tag">{{ item.tag }}</span></template>
      <template #col.server="{ item }">
        <span v-if="item.server" class="nexus-mono">{{ item.server }}</span>
        <span v-else class="dns-nexus__muted">—</span>
      </template>
      <template #col.server_port="{ item }">
        <span v-if="item.server_port" class="nexus-mono">{{ item.server_port }}</span>
        <span v-else class="dns-nexus__muted">—</span>
      </template>
      <template #col.tls="{ item }">
        <nexus-badge
          v-if="Object.hasOwn(item, 'tls')"
          :label="item.tls?.enabled ? $t('nexus.on') : $t('nexus.off')"
          :variant="item.tls?.enabled ? 'success' : 'secondary'"
        />
        <span v-else class="dns-nexus__muted">—</span>
      </template>
      <template #col.source="{ item }">
        <nexus-badge :label="presetSourceLabel(item)" :variant="isPresetManagedItem(item) ? 'success' : 'secondary'" />
      </template>
      <template #actions="{ item }">
        <row-actions :actions="serverActions()" @action="(key) => handleServerAction(key, item)" />
      </template>
      <template #empty><empty-state compact icon="lucide:network" :title="$t('dns.empty')" /></template>
    </nexus-data-table>
  </template>
  <v-row v-else>
    <v-col class="v-card-subtitle" cols="12">{{ $t('dns.title') }}</v-col>
    <v-col cols="12" sm="4" md="3" lg="2" v-for="(item, index) in <any[]>dns.servers" :key="item.id">
      <v-card rounded="xl" elevation="5" min-width="200" :title="item.tag">
        <v-card-subtitle style="margin-top: -15px;">
          <v-row>
            <v-col>{{ item.type }}</v-col>
            <v-col cols="auto" v-if="isPresetManagedItem(item)">
              <v-chip color="primary" label size="x-small" variant="tonal">{{ $t('presets.presetManaged') }}</v-chip>
            </v-col>
          </v-row>
        </v-card-subtitle>
        <v-card-text>
          <v-row>
            <v-col>{{ $t('dns.server') }}</v-col>
            <v-col>
              {{ item.server?? '-' }}
            </v-col>
          </v-row>
          <v-row>
            <v-col>{{ $t('in.port') }}</v-col>
            <v-col>
              {{ item.server_port?? '-' }}
            </v-col>
          </v-row>
          <v-row>
            <v-col>{{ $t('objects.tls') }}</v-col>
            <v-col>
              {{ Object.hasOwn(item,'tls') ? $t(item.tls?.enabled ? 'enable' : 'disable') : '-'  }}
            </v-col>
          </v-row>
        </v-card-text>
        <v-divider></v-divider>
        <v-card-actions style="padding: 0;">
          <v-btn icon="mdi-file-edit" @click="showDnsModal(index)">
            <v-icon />
            <v-tooltip activator="parent" location="top" :text="$t('actions.edit')"></v-tooltip>
          </v-btn>
          <v-btn icon="mdi-file-remove" style="margin-inline-start:0;" color="warning" @click="delDnsOverlay[index] = true">
            <v-icon />
            <v-tooltip activator="parent" location="top" :text="$t('actions.del')"></v-tooltip>
          </v-btn>
          <v-overlay
            v-model="delDnsOverlay[index]"
            contained
            class="align-center justify-center"
          >
            <v-card :title="$t('actions.del')" rounded="lg">
              <v-divider></v-divider>
              <v-card-text>{{ $t('confirm') }}</v-card-text>
              <v-card-actions>
                <v-btn color="error" variant="outlined" @click="delDns(index)">{{ $t('yes') }}</v-btn>
                <v-btn color="success" variant="outlined" @click="delDnsOverlay[index] = false">{{ $t('no') }}</v-btn>
              </v-card-actions>
            </v-card>
          </v-overlay>
        </v-card-actions>
      </v-card>
    </v-col>
  </v-row>
  <template v-if="nexus">
    <div class="dns-nexus__section">{{ $t('dns.rule.title') }}</div>
    <nexus-data-table :columns="ruleColumns" :items="dnsRuleRows" :row-key="(item) => item._index" :paginated="false">
      <template #col._index="{ item }">{{ item._index + 1 }}</template>
      <template #col.type="{ item }">{{ item.type != undefined ? $t('rule.logical') + ' (' + item.mode + ')' : $t('rule.simple') }}</template>
      <template #col.server="{ item }">{{ item.server ?? '-' }}</template>
      <template #col.invert="{ item }">{{ $t((item.invert ?? false) ? 'yes' : 'no') }}</template>
      <template #col.source="{ item }">
        <nexus-badge :label="presetSourceLabel(item)" :variant="isPresetManagedItem(item) ? 'success' : 'secondary'" />
      </template>
      <template #actions="{ item }">
        <row-actions :actions="ruleActions(item)" @action="(key) => handleRuleAction(key, item)" />
      </template>
      <template #empty><empty-state compact icon="lucide:filter" :title="$t('dns.rule.empty')" /></template>
    </nexus-data-table>
  </template>
  <v-row v-else>
    <v-col class="v-card-subtitle" cols="12">{{ $t('dns.rule.title') }}</v-col>
    <v-col cols="12" sm="4" md="3" lg="2" v-for="(item, index) in <any[]>dnsRules"
      :key="item.id"
      :draggable="true"
      @dragstart="onDragStart(index)"
      @dragover.prevent
      @drop="onDrop(index)"
      >
      <v-card rounded="xl" elevation="5" min-width="200" :title="index+1">
        <v-card-subtitle style="margin-top: -15px;">
          <v-row>
            <v-col>{{ item.type != undefined ? $t('rule.logical') + ' (' + item.mode + ')' : $t('rule.simple') }}</v-col>
            <v-col cols="auto" v-if="isPresetManagedItem(item)">
              <v-chip color="primary" label size="x-small" variant="tonal">{{ $t('presets.presetManaged') }}</v-chip>
            </v-col>
          </v-row>
        </v-card-subtitle>
        <v-card-text>
          <v-row>
            <v-col>{{ $t('admin.action') }}</v-col>
            <v-col>
              {{ item.action }}
            </v-col>
          </v-row>
          <v-row>
            <v-col>{{ $t('dns.server') }}</v-col>
            <v-col>
              {{ item.server?? '-' }}
            </v-col>
          </v-row>
          <v-row>
            <v-col>{{ $t('pages.rules') }}</v-col>
            <v-col>
              {{ item.rules ? item.rules.length : Object.keys(item).filter(r => !actionDnsRuleKeys.includes(r)).length }}
            </v-col>
          </v-row>
          <v-row>
            <v-col>{{ $t('rule.invert') }}</v-col>
            <v-col>
              {{ $t( (item.invert?? false)? 'yes' : 'no') }}
            </v-col>
          </v-row>
        </v-card-text>
        <v-divider></v-divider>
        <v-card-actions style="padding: 0;">
          <v-btn icon="mdi-file-edit" @click="showDnsRuleModal(index)">
            <v-icon />
            <v-tooltip activator="parent" location="top" :text="$t('actions.edit')"></v-tooltip>
          </v-btn>
          <v-btn icon="mdi-file-remove" style="margin-inline-start:0;" color="warning" @click="delDnsRuleOverlay[index] = true">
            <v-icon />
            <v-tooltip activator="parent" location="top" :text="$t('actions.del')"></v-tooltip>
          </v-btn>
          <v-overlay
            v-model="delDnsRuleOverlay[index]"
            contained
            class="align-center justify-center"
          >
            <v-card :title="$t('actions.del')" rounded="lg">
              <v-divider></v-divider>
              <v-card-text>{{ $t('confirm') }}</v-card-text>
              <v-card-actions>
                <v-btn color="error" variant="outlined" @click="delDnsRule(index)">{{ $t('yes') }}</v-btn>
                <v-btn color="success" variant="outlined" @click="delDnsRuleOverlay[index] = false">{{ $t('no') }}</v-btn>
              </v-card-actions>
            </v-card>
          </v-overlay>
        </v-card-actions>
      </v-card>
    </v-col>
  </v-row>
</template>

<script lang="ts" setup>
import Data from '@/store/modules/data'
import { computed, ref, onBeforeMount } from 'vue'
import { useI18n } from 'vue-i18n'
import DnsVue from '@/layouts/modals/Dns.vue'
import DnsRuleVue from '@/layouts/modals/DnsRule.vue'
import RegionalPresetDrawer from '@/components/presets/RegionalPresetDrawer.vue'
import { isPresetManagedItem } from '@/components/presets/routingDnsPresets'
import { Config } from '@/types/config'
import { actionDnsRuleKeys, dnsRule } from '@/types/dns'
import { FindDiff } from '@/plugins/utils'
import type { Column } from '@/components/nexus/data/dataTableColumns'
import NexusDataTable from '@/components/nexus/data/NexusDataTable.vue'
import RowActions from '@/components/nexus/data/RowActions.vue'
import type { RowAction } from '@/components/nexus/data/rowActions'
import NexusBadge from '@/components/nexus/primitives/Badge.vue'
import EmptyState from '@/components/nexus/primitives/EmptyState.vue'
import PageHeader from '@/components/nexus/primitives/PageHeader.vue'
import PageToolbar from '@/components/nexus/primitives/PageToolbar.vue'
import { useConfirm } from '@/components/nexus/primitives/useConfirm'
import { useUiMode } from '@/uiMode/useUiMode'

const { t } = useI18n()
const { confirm } = useConfirm()
const { mode } = useUiMode()
const nexus = computed(() => mode.value === 'nexus')
const presetSourceLabel = (item: any) => isPresetManagedItem(item) ? t('presets.presetManaged') : t('presets.custom')

const oldConfig = ref(<any>{})
const loading = ref(false)
const search = ref('')
const regionalPresetDrawer = ref(false)

// Edit a LOCAL clone of the store config. A background reload (data.ts setNewData
// replaces Data().config wholesale, driven by the 10s poll / WS events) must not wipe
// unsaved edits, so the form binds to this clone instead of the live store object.
const cloneStoreConfig = (): Config => JSON.parse(JSON.stringify(Data().config ?? {}))
const ensureDnsShape = (cfg: Config) => {
  // fix old configs
  if (!cfg.dns) cfg.dns = { servers: [], rules: [] }
  if (!cfg.dns.servers) cfg.dns.servers = []
  if (!cfg.dns.rules) cfg.dns.rules = []
}
const appConfig = ref<Config>((() => { const c = cloneStoreConfig(); ensureDnsShape(c); return c })())

const resyncFromStore = () => {
  const c = cloneStoreConfig()
  ensureDnsShape(c)
  appConfig.value = c
  oldConfig.value = JSON.parse(JSON.stringify(c))
}

onBeforeMount( async () => {
  loading.value = true
  while (Data().lastLoad == 0) {
    await new Promise(resolve => setTimeout(resolve, 100))
  }
  resyncFromStore()
  loading.value = false
})

const tsTags = computed((): string[] => {
  return Data().endpoints?.filter((e:any) => e.type == "tailscale").map((e:any) => e.tag)
})

const rslvdTags = computed((): string[] => {
  return Data().services?.filter((e:any) => e.type == "resolved").map((e:any) => e.tag)
})

const clients = computed((): string[] => {
  return Data().clients.map((c:any) => c.name)
})

const stateChange = computed(() => {
  return FindDiff.deepCompare(appConfig.value.dns,oldConfig.value.dns)
})

const saveConfig = async () => {
  loading.value = true
  const success = await Data().save("config", "set", appConfig.value)
  if (success) {
    resyncFromStore()
  }
  loading.value = false
}

const applyPresetConfig = (config: Config) => {
  appConfig.value = config
}

const inboundTags = computed((): string[] => {
  return [...Data().inbounds?.map((o:any) => o.tag), ...Data().endpoints?.filter((e:any) => e.listen_port > 0).map((e:any) => e.tag)]
})

const outboundTags = computed((): string[] => [
  ...Data().outbounds?.map((o:any) => o.tag),
  ...Data().endpoints?.map((e:any) => e.tag),
])

const dns = computed((): any => {
  return appConfig.value.dns
})

const dnsServerTags = computed((): string[] => {
  return dns.value?.servers?.filter((s:any) => s.tag && s.tag != "")?.map((s:any) => s.tag) ?? []
})

const finalDns = computed({
  get() { return dns.value?.final?? '' },
  set(v:string) { dns.value.final = v.length>0 ? v : undefined }
})


const dnsRules = computed((): dnsRule[] => {
  return <dnsRule[]>dns.value.rules
})

const ruleSets = computed((): string[] => {
  return appConfig.value?.route?.rule_set?.map((r:any) => r.tag) ?? []
})

let delDnsOverlay = ref(new Array<boolean>)
let delDnsRuleOverlay = ref(new Array<boolean>)

const dnsModal = ref({
  visible: false,
  index: -1,
  data: "",
})

const showDnsModal = (index: number) => {
  dnsModal.value.index = index
  dnsModal.value.data = index == -1 ? '' : JSON.stringify(dns.value.servers[index])
  dnsModal.value.visible = true
}

const closeDnsModal = () => {
  dnsModal.value.visible = false
}

const saveDnsModal = (data:any) => {
  // New or Edit
  if (dnsModal.value.index == -1) {
    dns.value.servers.push(data)
  } else {
    dns.value.servers[dnsModal.value.index] = data
  }
  dnsModal.value.visible = false
}

const delDns = (index: number) => {
  dns.value.servers.splice(index,1)
  delDnsOverlay.value[index] = false
}

const dnsRuleModal = ref({
  visible: false,
  index: -1,
  data: "",
})

const showDnsRuleModal = (index: number) => {
  dnsRuleModal.value.index = index
  dnsRuleModal.value.data = index == -1 ? '' : JSON.stringify(dnsRules.value[index])
  dnsRuleModal.value.visible = true
}

const closeDnsRuleModal = () => {
  dnsRuleModal.value.visible = false
}

const saveDnsRuleModal = (data:dnsRule) => {
  // New or Edit
  if (dnsRuleModal.value.index == -1) {
    dnsRules.value.push(data)
  } else {
    dnsRules.value[dnsRuleModal.value.index] = data
  }
  dnsRuleModal.value.visible = false
}

const delDnsRule = (index: number) => {
  dnsRules.value.splice(index,1)
  delDnsRuleOverlay.value[index] = false
}

// ---- Nexus table projections (read-only; actions carry the array index) ----
// _index keeps the ORIGINAL array index (edit/delete operate by index), so filter
// AFTER mapping. Search matches tag/type/server (servers) and action/server (rules).
const matchesSearch = (text: string): boolean => {
  const q = search.value.trim().toLowerCase()
  return !q || text.toLowerCase().includes(q)
}

const dnsServerRows = computed(() =>
  (dns.value?.servers ?? [])
    .map((s: any, i: number) => ({ ...s, _index: i }))
    .filter((s: any) => matchesSearch(`${s.tag ?? ''} ${s.type ?? ''} ${s.server ?? ''}`)))

const dnsRuleRows = computed(() =>
  dnsRules.value
    .map((r: any, i: number) => ({
      ...r,
      _index: i,
      _rulesCount: r.rules ? r.rules.length : Object.keys(r).filter((k: string) => !actionDnsRuleKeys.includes(k)).length,
    }))
    .filter((r: any) => matchesSearch(`${r.action ?? ''} ${r.server ?? ''}`)))

const serverColumns: Column<any>[] = [
  { key: 'tag', labelKey: 'objects.tag', sortable: true },
  { key: 'type', labelKey: 'type', sortable: true },
  { key: 'server', labelKey: 'dns.server' },
  { key: 'server_port', labelKey: 'in.port' },
  { key: 'tls', labelKey: 'objects.tls' },
  { key: 'source', labelKey: 'presets.source' },
]

const ruleColumns: Column<any>[] = [
  { key: '_index', label: '#' },
  { key: 'type', labelKey: 'type' },
  { key: 'action', labelKey: 'admin.action' },
  { key: 'server', labelKey: 'dns.server' },
  { key: '_rulesCount', labelKey: 'pages.rules' },
  { key: 'invert', labelKey: 'rule.invert' },
  { key: 'source', labelKey: 'presets.source' },
]

const subtitle = computed(() => {
  const servers = dns.value?.servers?.length ?? 0
  const rules = dnsRules.value?.length ?? 0

  return t('nexus.summary.dns', { servers, rules })
})

const serverActions = (): RowAction[] => [
  { key: 'edit', labelKey: 'actions.edit', icon: 'lucide:pencil', inline: true },
  { key: 'del', labelKey: 'actions.del', icon: 'lucide:trash-2', tone: 'error', inline: true },
]

const ruleActions = (item: any): RowAction[] => [
  { key: 'up', labelKey: 'table.moveUp', icon: 'lucide:arrow-up', inline: true, hidden: item._index === 0 },
  { key: 'down', labelKey: 'table.moveDown', icon: 'lucide:arrow-down', inline: true, hidden: item._index === dnsRules.value.length - 1 },
  { key: 'edit', labelKey: 'actions.edit', icon: 'lucide:pencil', inline: true },
  { key: 'del', labelKey: 'actions.del', icon: 'lucide:trash-2', tone: 'error', inline: true },
]

const handleServerAction = async (key: string, item: any) => {
  if (key === 'edit') { showDnsModal(item._index); return }
  if (key === 'del') {
    const ok = await confirm({ title: `${t('actions.del')} ${t('objects.dnsserver')}`, message: item.tag, confirmLabel: t('actions.del'), tone: 'error' })
    if (ok) delDns(item._index)
  }
}

const moveDnsRule = (index: number, dir: number) => {
  const target = index + dir
  if (target < 0 || target >= dnsRules.value.length) return
  const arr = dnsRules.value
  const [moved] = arr.splice(index, 1)
  arr.splice(target, 0, moved)
}

const handleRuleAction = async (key: string, item: any) => {
  if (key === 'edit') { showDnsRuleModal(item._index); return }
  if (key === 'up') { moveDnsRule(item._index, -1); return }
  if (key === 'down') { moveDnsRule(item._index, 1); return }
  if (key === 'del') {
    const ok = await confirm({ title: `${t('actions.del')} ${t('dns.rule.title')}`, message: String(item._index + 1), confirmLabel: t('actions.del'), tone: 'error' })
    if (ok) delDnsRule(item._index)
  }
}

const draggedItemIndex = ref(null)

const onDragStart = (index: any) => {
  draggedItemIndex.value = index
}

const onDrop = (index: any) => {
  if (draggedItemIndex.value !== null) {
    // Swap the dragged item with the dropped one
    const draggedItem = dnsRules.value[draggedItemIndex.value]
    dnsRules.value.splice(draggedItemIndex.value, 1)
    dnsRules.value.splice(index, 0, draggedItem)
    draggedItemIndex.value = null
  }
}
</script>

<style scoped>
.dns-nexus__section {
  color: var(--nexus-text-secondary);
  font-size: 0.78rem;
  font-weight: 650;
  letter-spacing: 0.4px;
  margin-block: var(--nexus-gap-4) var(--nexus-gap-2);
  text-transform: uppercase;
}

.dns-nexus__tag {
  color: var(--nexus-text-primary);
  font-weight: 600;
}

.dns-nexus__muted {
  color: var(--nexus-text-muted);
}
</style>
