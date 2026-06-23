<template>
  <RuleVue
    v-model="ruleModal.visible"
    :visible="ruleModal.visible"
    :index="ruleModal.index"
    :data="ruleModal.data"
    :clients="clients"
    :inTags="inboundTags"
    :outTags="outboundTags"
    :rsTags="rulesetTags"
    @close="closeRuleModal"
    @save="saveRuleModal"
  />
  <RulesetVue
    v-model="rulesetModal.visible"
    :visible="rulesetModal.visible"
    :index="rulesetModal.index"
    :data="rulesetModal.data"
    :outTags="outboundTags"
    @close="closeRulesetModal"
    @save="saveRulesetModal"
  />
  <RuleImport
    v-model="importRulesModal.visible"
    :visible="importRulesModal.visible"
    :existingRulesCount="rules.length"
    :existingRulesetsCount="rulesets.length"
    :existingRulesetTags="rulesetTags"
    @save="saveImportRule"
    @close="closeImportRule"
  />
  <RulesetImport
    v-model="importRulesetsModal.visible"
    :visible="importRulesetsModal.visible"
    :outTags="outboundTags"
    :rsTags="rulesetTags"
    @save="saveImportRulesets"
    @close="closeImportRulesets"
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
    :title="$t('pages.rules')"
    @update:search="search = $event"
  />

  <page-toolbar v-if="nexus">
    <template #secondary-actions>
      <v-btn color="primary" prepend-icon="lucide:plus" variant="flat" @click="showRuleModal(-1)">{{ $t('rule.add') }}</v-btn>
      <v-btn prepend-icon="lucide:plus" variant="text" @click="showRulesetModal(-1)">{{ $t('ruleset.add') }}</v-btn>
      <v-btn prepend-icon="mdi-routes" variant="text" @click="regionalPresetDrawer = true">{{ $t('regionalPresets.open') }}</v-btn>
      <v-menu :close-on-content-click="false" location="bottom center">
        <template v-slot:activator="{ props }">
          <v-btn v-bind="props" :aria-label="$t('rule.import.title')" icon="lucide:wrench" variant="text" />
        </template>
        <v-list density="compact" nav>
          <v-list-item link prepend-icon="lucide:list" :title="$t('rule.import.rulesTitle')" @click="showImportRule" />
          <v-list-item link prepend-icon="lucide:download" :title="$t('rule.import.title')" @click="showImportRulesets" />
        </v-list>
      </v-menu>
    </template>
    <template #primary-actions>
      <v-btn variant="tonal" color="warning" @click="saveConfig" :loading="loading" :disabled="stateChange">
        {{ $t('actions.save') }}
      </v-btn>
    </template>
  </page-toolbar>

  <v-row v-else>
    <v-col cols="12" justify="center" align="center">
      <v-btn color="primary" @click="showRuleModal(-1)" style="margin: 0 5px;">{{ $t('rule.add') }}</v-btn>
      <v-btn color="primary" @click="showRulesetModal(-1)" style="margin: 0 5px;">{{ $t('ruleset.add') }}</v-btn>
      <v-btn color="primary" @click="regionalPresetDrawer = true" style="margin: 0 5px;">{{ $t('regionalPresets.open') }}</v-btn>
      <v-menu v-model="actionMenu" :close-on-content-click="false" location="bottom center">
        <template v-slot:activator="{ props }">
          <v-btn v-bind="props" hide-details variant="text" icon>
            <v-icon icon="mdi-tools" color="primary" />
          </v-btn>
        </template>
        <v-list density="compact" nav>
          <v-list-item link @click="showImportRule">
            <template v-slot:prepend>
              <v-icon icon="mdi-routes"></v-icon>
            </template>
            <v-list-item-title v-text="$t('rule.import.rulesTitle')"></v-list-item-title>
          </v-list-item>
          <v-list-item link @click="showImportRulesets">
            <template v-slot:prepend>
              <v-icon icon="mdi-download-multiple"></v-icon>
            </template>
            <v-list-item-title v-text="$t('rule.import.title')"></v-list-item-title>
          </v-list-item>
        </v-list>
      </v-menu>
      <v-btn variant="outlined" color="warning" @click="saveConfig" :loading="loading" :disabled="stateChange">
        {{ $t('actions.save') }}
      </v-btn>
    </v-col>
  </v-row>
  <v-row>
    <v-col class="v-card-subtitle" cols="12">{{ $t('basic.routing.title') }}</v-col>
    <v-col cols="12">
      <v-row>
        <v-col cols="12" sm="6" md="3" lg="2">
          <v-select hide-details :label="$t('basic.routing.defaultOut')" clearable
            @click:clear="delete route.final" :items="outboundTags" v-model="route.final"></v-select>
        </v-col>
        <v-col cols="12" sm="6" md="3" lg="2">
          <v-text-field v-model="route.default_interface" hide-details clearable
            @click:clear="delete route.default_interface" :label="$t('basic.routing.defaultIf')"></v-text-field>
        </v-col>
        <v-col cols="12" sm="6" md="3" lg="2">
          <v-text-field v-model.number="routeMark" hide-details type="number" min="0" :label="$t('basic.routing.defaultRm')"></v-text-field>
        </v-col>
        <v-col cols="12" sm="6" md="3" lg="2">
          <v-switch v-model="route.auto_detect_interface" color="primary" :label="$t('basic.routing.autoBind')" hide-details></v-switch>
        </v-col>
        <v-col cols="12" sm="6" md="3" lg="2">
          <v-select v-model="routePreset" hide-details :label="$t('singbox.routePreset')" :items="routePresets"></v-select>
        </v-col>
        <v-col cols="12" sm="6" md="3" lg="2">
          <v-switch v-model="findProcess" color="primary" :label="$t('singbox.findProcess')" hide-details></v-switch>
        </v-col>
        <v-col cols="12" sm="6" md="3" lg="2">
          <v-switch v-model="overrideAndroidVpn" color="primary" :label="$t('singbox.overrideAndroidVpn')" hide-details></v-switch>
        </v-col>
        <v-col cols="12" v-if="route.override_android_vpn">
          <v-alert density="compact" type="warning" variant="tonal">
            {{ $t('singbox.overrideAndroidVpnWarning') }}
          </v-alert>
        </v-col>
        <v-col cols="12" v-if="route.default_network_strategy && !route.auto_detect_interface">
          <v-alert density="compact" type="warning" variant="tonal">
            {{ $t('singbox.defaultNetworkStrategyRequired') }}
          </v-alert>
        </v-col>
        <v-col cols="12" v-if="route.default_network_strategy && route.default_interface">
          <v-alert density="compact" type="warning" variant="tonal">
            {{ $t('singbox.defaultNetworkStrategyConflict') }}
          </v-alert>
        </v-col>
      </v-row>
      <DomainResolver :data="route" field="default_domain_resolver" :label="$t('singbox.defaultDomainResolver')" />
      <v-row v-if="route.default_network_strategy">
        <v-col cols="12" sm="6" md="3" lg="2">
          <v-select
            v-model="routeDefaultNetworkStrategy"
            hide-details
            clearable
            @click:clear="routeDefaultNetworkStrategy = undefined"
            :label="$t('singbox.defaultNetworkStrategy')"
            :items="['fallback', 'hybrid']">
          </v-select>
        </v-col>
        <v-col cols="12" sm="6" md="3" lg="2">
          <v-select
            v-model="route.default_network_type"
            hide-details multiple chips closable-chips
            :label="$t('singbox.defaultNetworkType')"
            :items="networkTypes">
          </v-select>
        </v-col>
        <v-col cols="12" sm="6" md="3" lg="2" v-if="route.default_network_strategy == 'fallback'">
          <v-select
            v-model="route.default_fallback_network_type"
            hide-details multiple chips closable-chips
            :label="$t('singbox.defaultFallbackNetworkType')"
            :items="networkTypes">
          </v-select>
        </v-col>
        <v-col cols="12" sm="6" md="3" lg="2">
          <v-text-field
            v-model="defaultFallbackDelayMs"
            hide-details
            type="number"
            min="1"
            suffix="ms"
            :label="$t('singbox.defaultFallbackDelay')">
          </v-text-field>
        </v-col>
      </v-row>
    </v-col>
  </v-row>
  <template v-if="nexus">
    <div class="rules-nexus__section">{{ $t('rule.ruleset') }}</div>
    <nexus-data-table :columns="rulesetColumns" :items="rulesetRows" :row-key="(item) => item._index">
      <template #col.tag="{ item }"><span class="rules-nexus__tag">{{ item.tag }}</span></template>
      <template #col.type="{ item }">{{ $t('ruleset.' + item.type) }}</template>
      <template #col.download_detour="{ item }">{{ item.download_detour ?? '-' }}</template>
      <template #col.update_interval="{ item }">{{ item.update_interval ?? '-' }}</template>
      <template #col.source="{ item }">
        <nexus-badge :label="presetSourceLabel(item)" :variant="isPresetManagedItem(item) ? 'success' : 'secondary'" />
      </template>
      <template #actions="{ item }">
        <row-actions :actions="rulesetActions()" @action="(key) => handleRulesetAction(key, item)" />
      </template>
      <template #empty><empty-state compact icon="lucide:list" :title="$t('ruleset.empty')" /></template>
    </nexus-data-table>
  </template>
  <v-row v-else>
    <v-col class="v-card-subtitle" cols="12">{{ $t('rule.ruleset') }}</v-col>
    <v-col cols="12" sm="4" md="3" lg="2" v-for="(item, index) in <any[]>rulesets" :key="item.tag">
      <v-card rounded="xl" elevation="5" min-width="200" :title="item.tag">
        <v-card-subtitle style="margin-top: -15px;">
          <v-row>
            <v-col>{{ $t('ruleset.' + item.type) }}</v-col>
            <v-col cols="auto" v-if="isPresetManagedItem(item)">
              <v-chip color="primary" label size="x-small" variant="tonal">{{ $t('presets.presetManaged') }}</v-chip>
            </v-col>
          </v-row>
        </v-card-subtitle>
        <v-card-text>
          <v-row><v-col>{{ $t('ruleset.format') }}</v-col><v-col>{{ item.format }}</v-col></v-row>
          <v-row><v-col>{{ $t('objects.outbound') }}</v-col><v-col>{{ item.download_detour ?? '-' }}</v-col></v-row>
          <v-row><v-col>{{ $t('actions.update') }}</v-col><v-col>{{ item.update_interval ?? '-' }}</v-col></v-row>
        </v-card-text>
        <v-divider></v-divider>
        <v-card-actions style="padding: 0;">
          <v-btn icon="mdi-file-edit" @click="showRulesetModal(index)">
            <v-icon /><v-tooltip activator="parent" location="top" :text="$t('actions.edit')"></v-tooltip>
          </v-btn>
          <v-btn icon="mdi-file-remove" style="margin-inline-start:0;" color="warning" @click="delRulesetOverlay[index] = true">
            <v-icon /><v-tooltip activator="parent" location="top" :text="$t('actions.del')"></v-tooltip>
          </v-btn>
          <v-overlay v-model="delRulesetOverlay[index]" contained class="align-center justify-center">
            <v-card :title="$t('actions.del')" rounded="lg">
              <v-divider></v-divider>
              <v-card-text>{{ $t('confirm') }}</v-card-text>
              <v-card-actions>
                <v-btn color="error" variant="outlined" @click="delRuleset(index)">{{ $t('yes') }}</v-btn>
                <v-btn color="success" variant="outlined" @click="delRulesetOverlay[index] = false">{{ $t('no') }}</v-btn>
              </v-card-actions>
            </v-card>
          </v-overlay>
        </v-card-actions>
      </v-card>
    </v-col>
  </v-row>
  <template v-if="nexus">
    <div class="rules-nexus__section">{{ $t('pages.rules') }}</div>
    <nexus-data-table :columns="ruleColumns" :items="ruleRows" :row-key="(item) => item._index" :paginated="false">
      <template #col._index="{ item }">{{ item._index + 1 }}</template>
      <template #col.type="{ item }">{{ item.type != undefined ? $t('rule.logical') + ' (' + item.mode + ')' : $t('rule.simple') }}</template>
      <template #col.outbound="{ item }">{{ item.outbound ?? '-' }}</template>
      <template #col.invert="{ item }">{{ $t((item.invert ?? false) ? 'yes' : 'no') }}</template>
      <template #col.source="{ item }">
        <nexus-badge :label="presetSourceLabel(item)" :variant="isPresetManagedItem(item) ? 'success' : 'secondary'" />
      </template>
      <template #actions="{ item }">
        <row-actions :actions="ruleActions(item)" @action="(key) => handleRuleAction(key, item)" />
      </template>
      <template #empty><empty-state compact icon="lucide:list" :title="$t('rule.empty')" /></template>
    </nexus-data-table>
  </template>
  <v-row v-else>
    <v-col class="v-card-subtitle" cols="12">{{ $t('pages.rules') }}</v-col>
    <v-col cols="12" sm="4" md="3" lg="2" v-for="(item, index) in <any[]>rules"
        :key="item.id" :draggable="true"
        @dragstart="onDragStart(index)" @dragover.prevent @drop="onDrop(index)">
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
          <v-row><v-col>{{ $t('admin.action') }}</v-col><v-col>{{ item.action }}</v-col></v-row>
          <v-row><v-col>{{ $t('objects.outbound') }}</v-col><v-col>{{ item.outbound ?? '-' }}</v-col></v-row>
          <v-row><v-col>{{ $t('pages.rules') }}</v-col><v-col>{{ item.rules ? item.rules.length : Object.keys(item).filter(r => !actionKeys.includes(r)).length }}</v-col></v-row>
          <v-row><v-col>{{ $t('rule.invert') }}</v-col><v-col>{{ $t((item.invert ?? false) ? 'yes' : 'no') }}</v-col></v-row>
        </v-card-text>
        <v-divider></v-divider>
        <v-card-actions style="padding: 0;">
          <v-btn icon="mdi-file-edit" @click="showRuleModal(index)">
            <v-icon /><v-tooltip activator="parent" location="top" :text="$t('actions.edit')"></v-tooltip>
          </v-btn>
          <v-btn icon="mdi-file-remove" style="margin-inline-start:0;" color="warning" @click="delRuleOverlay[index] = true">
            <v-icon /><v-tooltip activator="parent" location="top" :text="$t('actions.del')"></v-tooltip>
          </v-btn>
          <v-overlay v-model="delRuleOverlay[index]" contained class="align-center justify-center">
            <v-card :title="$t('actions.del')" rounded="lg">
              <v-divider></v-divider>
              <v-card-text>{{ $t('confirm') }}</v-card-text>
              <v-card-actions>
                <v-btn color="error" variant="outlined" @click="delRule(index)">{{ $t('yes') }}</v-btn>
                <v-btn color="success" variant="outlined" @click="delRuleOverlay[index] = false">{{ $t('no') }}</v-btn>
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
import RuleVue from '@/layouts/modals/Rule.vue'
import RulesetVue from '@/layouts/modals/Ruleset.vue'
import RulesetImport from '@/layouts/modals/RulesetImport.vue'
import RuleImport from '@/layouts/modals/RuleImport.vue'
import DomainResolver from '@/components/DomainResolver.vue'
import RegionalPresetDrawer from '@/components/presets/RegionalPresetDrawer.vue'
import { isPresetManagedItem } from '@/components/presets/routingDnsPresets'
import { Config } from '@/types/config'
import { actionKeys, ruleset } from '@/types/rules'
import { FindDiff } from '@/plugins/utils'
import { i18n } from '@/locales'
import type { Column } from '@/components/nexus/data/dataTableColumns'
import NexusDataTable from '@/components/nexus/data/NexusDataTable.vue'
import RowActions from '@/components/nexus/data/RowActions.vue'
import type { RowAction } from '@/components/nexus/data/rowActions'
import EmptyState from '@/components/nexus/primitives/EmptyState.vue'
import NexusBadge from '@/components/nexus/primitives/Badge.vue'
import PageHeader from '@/components/nexus/primitives/PageHeader.vue'
import PageToolbar from '@/components/nexus/primitives/PageToolbar.vue'
import { useConfirm } from '@/components/nexus/primitives/useConfirm'
import { useUiMode } from '@/uiMode/useUiMode'

const { confirm } = useConfirm()
const { mode } = useUiMode()
const nexus = computed(() => mode.value === 'nexus')
const tt = (key: string) => i18n.global.t(key)
const presetSourceLabel = (item: any) => isPresetManagedItem(item) ? tt('presets.presetManaged') : tt('presets.custom')

const oldConfig = ref(<any>{})
const loading = ref(false)
const actionMenu = ref(false)
const search = ref('')
const regionalPresetDrawer = ref(false)
// Edit a LOCAL clone of the store config. A background reload (data.ts setNewData
// replaces Data().config wholesale, driven by the 10s poll / WS events) must not wipe
// unsaved edits, so the form binds to this clone instead of the live store object.
const cloneStoreConfig = (): Config => JSON.parse(JSON.stringify(Data().config ?? {}))
const appConfig = ref<Config>(cloneStoreConfig())

const resyncFromStore = () => {
  appConfig.value = cloneStoreConfig()
  oldConfig.value = cloneStoreConfig()
}

onBeforeMount(async () => {
  loading.value = true
  while (Data().lastLoad == 0) {
    await new Promise(resolve => setTimeout(resolve, 100))
  }
  resyncFromStore()
  loading.value = false
})

const routeMark = computed({
  get() { return route.value.default_mark ?? 0 },
  set(v:number) { v>0 ? route.value.default_mark = v : delete appConfig.value.route.default_mark }
})

const routePresets = [
  { title: i18n.global.t('singbox.defaultPreset'), value: 'default' },
  { title: i18n.global.t('singbox.mobileStable'), value: 'mobile' },
  { title: i18n.global.t('singbox.preferWifi'), value: 'wifi' },
  { title: i18n.global.t('singbox.processRules'), value: 'process' },
]

const networkTypes = ['wifi', 'cellular', 'ethernet', 'other']

const clearRouteNetworkDefaults = () => {
  delete route.value.default_network_strategy
  delete route.value.default_network_type
  delete route.value.default_fallback_network_type
  delete route.value.default_fallback_delay
}

const routePreset = computed({
  get(): string {
    if (route.value.default_network_strategy == 'fallback' &&
      JSON.stringify(route.value.default_network_type ?? []) == JSON.stringify(['wifi']) &&
      JSON.stringify(route.value.default_fallback_network_type ?? []) == JSON.stringify(['cellular'])) return 'wifi'
    if (route.value.default_network_strategy == 'fallback') return 'mobile'
    if (route.value.find_process) return 'process'
    return 'default'
  },
  set(v:string) {
    if (v == 'default') {
      clearRouteNetworkDefaults()
      delete route.value.find_process
    } else if (v == 'mobile') {
      delete route.value.default_interface
      route.value.auto_detect_interface = true
      route.value.default_network_strategy = 'fallback'
      delete route.value.default_network_type
      delete route.value.default_fallback_network_type
      delete route.value.default_fallback_delay
    } else if (v == 'wifi') {
      delete route.value.default_interface
      route.value.auto_detect_interface = true
      route.value.default_network_strategy = 'fallback'
      route.value.default_network_type = ['wifi']
      route.value.default_fallback_network_type = ['cellular']
      delete route.value.default_fallback_delay
    } else if (v == 'process') {
      route.value.find_process = true
    }
  }
})

const defaultFallbackDelayMs = computed({
  get(): number | undefined { return route.value.default_fallback_delay ? parseInt(route.value.default_fallback_delay.replace('ms', '')) : undefined },
  set(v:number | undefined) {
    if (typeof v == 'number' && !isNaN(v) && v > 0 && v != 300) route.value.default_fallback_delay = `${v}ms`
    else delete route.value.default_fallback_delay
  }
})
const routeDefaultNetworkStrategy = computed({
  get(): string | undefined { return route.value.default_network_strategy },
  set(v:string | undefined) {
    if (!v) {
      clearRouteNetworkDefaults()
      return
    }
    route.value.default_network_strategy = v
    if (v != 'fallback') {
      delete route.value.default_fallback_network_type
    }
  }
})
const findProcess = computed({
  get(): boolean { return route.value.find_process === true },
  set(v:boolean) { v ? route.value.find_process = true : delete route.value.find_process }
})
const overrideAndroidVpn = computed({
  get(): boolean { return route.value.override_android_vpn === true },
  set(v:boolean) { v ? route.value.override_android_vpn = true : delete route.value.override_android_vpn }
})

const stateChange = computed(() => FindDiff.deepCompare(appConfig.value, oldConfig.value))

const saveConfig = async () => {
  loading.value = true
  const success = await Data().save("config", "set", appConfig.value)
  if (success) {
    resyncFromStore()
    loading.value = false
  }
}

const applyPresetConfig = (config: Config) => {
  appConfig.value = config
}

const clients = computed((): string[] => Data().clients.map((c:any) => c.name))
const route = computed((): any => appConfig.value.route ?? {})

const rules = computed((): any[] => {
  const data = route.value
  if (!data) return []
  if (!('rules' in data) || !Array.isArray(data.rules)) data.rules = []
  return data.rules
})

const rulesets = computed((): any[] => {
  const data = route.value
  if (!data) return []
  if (!('rule_set' in data) || !Array.isArray(data.rule_set)) data.rule_set = []
  return data.rule_set
})

const rulesetTags = computed((): string[] => rulesets.value.map((rs:any) => rs.tag))

const outboundTags = computed((): string[] => [
  ...Data().outbounds?.map((o:any) => o.tag),
  ...Data().endpoints?.map((e:any) => e.tag)
])

const inboundTags = computed((): string[] => [
  ...Data().inbounds?.map((o:any) => o.tag),
  ...Data().endpoints?.filter((e:any) => e.listen_port > 0).map((e:any) => e.tag)
])

// ---- Nexus table projections (read-only; actions carry the array index) ----
// _index keeps the ORIGINAL array index (move/edit/delete operate by index), so
// filter AFTER mapping. Search matches tag/type/format (rulesets) and
// action/outbound (rules).
const matchesSearch = (text: string): boolean => {
  const q = search.value.trim().toLowerCase()
  return !q || text.toLowerCase().includes(q)
}

const rulesetRows = computed(() =>
  rulesets.value
    .map((rs: any, i: number) => ({ ...rs, _index: i }))
    .filter((rs: any) => matchesSearch(`${rs.tag ?? ''} ${rs.type ?? ''} ${rs.format ?? ''}`)))

const ruleRows = computed(() =>
  rules.value
    .map((r: any, i: number) => ({
      ...r,
      _index: i,
      _rulesCount: r.rules ? r.rules.length : Object.keys(r).filter((k: string) => !actionKeys.includes(k)).length,
    }))
    .filter((r: any) => matchesSearch(`${r.action ?? ''} ${r.outbound ?? ''}`)))

const rulesetColumns: Column<any>[] = [
  { key: 'tag', labelKey: 'objects.tag', sortable: true },
  { key: 'type', labelKey: 'type', sortable: true },
  { key: 'format', labelKey: 'ruleset.format' },
  { key: 'download_detour', labelKey: 'objects.outbound' },
  { key: 'update_interval', labelKey: 'actions.update' },
  { key: 'source', labelKey: 'presets.source' },
]

const ruleColumns: Column<any>[] = [
  { key: '_index', label: '#' },
  { key: 'type', labelKey: 'type' },
  { key: 'action', labelKey: 'admin.action' },
  { key: 'outbound', labelKey: 'objects.outbound' },
  { key: '_rulesCount', labelKey: 'pages.rules' },
  { key: 'invert', labelKey: 'rule.invert' },
  { key: 'source', labelKey: 'presets.source' },
]

const subtitle = computed(() =>
  i18n.global.t('nexus.summary.rules', { rulesets: rulesets.value.length, rules: rules.value.length }))

const rulesetActions = (): RowAction[] => [
  { key: 'edit', labelKey: 'actions.edit', icon: 'lucide:pencil', inline: true },
  { key: 'del', labelKey: 'actions.del', icon: 'lucide:trash-2', tone: 'error', inline: true },
]

const ruleActions = (item: any): RowAction[] => [
  { key: 'up', labelKey: 'table.moveUp', icon: 'lucide:arrow-up', inline: true, hidden: item._index === 0 },
  { key: 'down', labelKey: 'table.moveDown', icon: 'lucide:arrow-down', inline: true, hidden: item._index === rules.value.length - 1 },
  { key: 'edit', labelKey: 'actions.edit', icon: 'lucide:pencil', inline: true },
  { key: 'del', labelKey: 'actions.del', icon: 'lucide:trash-2', tone: 'error', inline: true },
]

const handleRulesetAction = async (key: string, item: any) => {
  if (key === 'edit') { showRulesetModal(item._index); return }
  if (key === 'del') {
    const ok = await confirm({ title: `${tt('actions.del')} ${tt('objects.ruleset')}`, message: item.tag, confirmLabel: tt('actions.del'), tone: 'error' })
    if (ok) delRuleset(item._index)
  }
}

const moveRule = (index: number, dir: number) => {
  const target = index + dir
  if (target < 0 || target >= rules.value.length) return
  const arr = rules.value
  const [moved] = arr.splice(index, 1)
  arr.splice(target, 0, moved)
}

const handleRuleAction = async (key: string, item: any) => {
  if (key === 'edit') { showRuleModal(item._index); return }
  if (key === 'up') { moveRule(item._index, -1); return }
  if (key === 'down') { moveRule(item._index, 1); return }
  if (key === 'del') {
    const ok = await confirm({ title: `${tt('actions.del')} ${tt('pages.rules')}`, message: String(item._index + 1), confirmLabel: tt('actions.del'), tone: 'error' })
    if (ok) delRule(item._index)
  }
}

let delRuleOverlay = ref(new Array<boolean>)
let delRulesetOverlay = ref(new Array<boolean>)

const ruleModal = ref({ visible: false, index: -1, data: "" })
const showRuleModal = (index: number) => {
  ruleModal.value.index = index
  ruleModal.value.data = index == -1 ? '' : JSON.stringify(rules.value[index])
  ruleModal.value.visible = true
}
const closeRuleModal = () => { ruleModal.value.visible = false }
const saveRuleModal = (data:any) => {
  if (ruleModal.value.index == -1) rules.value.push(data)
  else rules.value[ruleModal.value.index] = data
  ruleModal.value.visible = false
}
const delRule = (index: number) => { rules.value.splice(index, 1); delRuleOverlay.value[index] = false }

const rulesetModal = ref({ visible: false, index: -1, data: "" })
const showRulesetModal = (index: number) => {
  rulesetModal.value.index = index
  rulesetModal.value.data = index == -1 ? '' : JSON.stringify(rulesets.value[index])
  rulesetModal.value.visible = true
}
const closeRulesetModal = () => { rulesetModal.value.visible = false }
const saveRulesetModal = (data:ruleset) => {
  if (rulesetModal.value.index == -1) rulesets.value.push(data)
  else rulesets.value[rulesetModal.value.index] = data
  rulesetModal.value.visible = false
}
const delRuleset = (index: number) => { rulesets.value.splice(index, 1); delRulesetOverlay.value[index] = false }

const draggedItemIndex = ref(null)
const onDragStart = (index: any) => { draggedItemIndex.value = index }
const onDrop = (index: any) => {
  if (draggedItemIndex.value !== null) {
    const draggedItem = rules.value[draggedItemIndex.value]
    rules.value.splice(draggedItemIndex.value, 1)
    rules.value.splice(index, 0, draggedItem)
    draggedItemIndex.value = null
  }
}

const importRulesModal = ref({ visible: false })

function showImportRule() {
  importRulesModal.value.visible = true
}

function closeImportRule() {
  importRulesModal.value.visible = false
}

function saveImportRule(block: any, mode: 'merge' | 'replace', applyFinal: boolean) {
  if (mode === 'replace') {
    route.value.rules = block.rules ?? []
    route.value.rule_set = block.rule_set ?? []
  } else {
    const existingTags = new Set(rulesetTags.value)
    if (block.rules) rules.value.push(...block.rules)
    if (block.rule_set) {
      for (const rs of block.rule_set) {
        if (!existingTags.has(rs.tag)) rulesets.value.push(rs)
      }
    }
  }
  if (applyFinal && block.final) route.value.final = block.final
  importRulesModal.value.visible = false
}

const importRulesetsModal = ref({ visible: false })

function showImportRulesets() {
  importRulesetsModal.value.visible = true
}

function closeImportRulesets() {
  importRulesetsModal.value.visible = false
}

function saveImportRulesets(items: any[]) {
  rulesets.value.push(...items)
  importRulesetsModal.value.visible = false
}
</script>

<style scoped>
.rules-nexus__section {
  color: var(--nexus-text-secondary);
  font-size: 0.78rem;
  font-weight: 650;
  letter-spacing: 0.4px;
  margin-block: var(--nexus-gap-4) var(--nexus-gap-2);
  text-transform: uppercase;
}

.rules-nexus__tag {
  color: var(--nexus-text-primary);
  font-weight: 600;
}
</style>
