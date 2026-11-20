
<template>
  <ClientModal 
    v-model="modal.visible"
    :visible="modal.visible"
    :id="modal.id"
    :groups="groups"
    :inboundTags="inboundTags"
    @close="closeModal"
  />
  <ClientAddBulk 
    v-model="addBulkModal"
    :visible="addBulkModal"
    :groups="groups"
    :inboundTags="inboundTags"
    @close="closeAddBulk"
  />
  <ClientEditBulk 
    v-model="editBulkModal"
    :visible="editBulkModal"
    :inboundTags="inboundTags"
    :clients="Data().clients"
    @close="closeEditBulk"
  />
  <QrCode
    v-model="qrcode.visible"
    :visible="qrcode.visible"
    :id="qrcode.id"
    @close="closeQrCode"
  />
  <ClientDoctor
    v-model="doctor.visible"
    :visible="doctor.visible"
    :id="doctor.id"
    @close="closeDoctor"
  />
  <Stats
    v-model="stats.visible"
    :visible="stats.visible"
    :resource="stats.resource"
    :tag="stats.tag"
    @close="closeStats"
  />
  <IpHistoryModal
    v-model:visible="ipModal.visible"
    :client="ipModal.client"
    :is-admin="true"
    @cleared="onClientIpsCleared"
  />

  <ClientsNexusList
    v-if="mode === 'nexus'"
    :clients="<any[]>clients"
    :inbounds="<any[]>inbounds"
    :groups="groups"
    :onlines="<string[]>(Data().onlines.user ?? [])"
    :enable-traffic="enableTraffic"
    :total="pageData.total"
    :filtered-count="pageData.filtered"
    :online-count="pageData.online"
    :loading="loading"
    @query="nexusQuery = $event"
    @add="showModal(0)"
    @add-bulk="addBulk"
    @del="delClient"
    @diagnose="showDoctor"
    @edit="showModal"
    @edit-bulk="editBulk"
    @qr="showQrCode"
    @show-ips="showClientIps"
    @stats="showStats"
  />

  <template v-else>
  <v-row justify="center" align="center">
    <v-col cols="auto">
      <v-btn color="primary" @click="showModal(0)">{{ $t('actions.add') }}</v-btn>
    </v-col>
    <v-col cols="auto">
      <v-menu v-model="actionMenu" :close-on-content-click="false" location="bottom center">
        <template v-slot:activator="{ props }">
          <v-btn v-bind="props" hide-details variant="text" icon>
            <v-icon icon="mdi-tools" color="primary" />
          </v-btn>
        </template>
        <v-list density="compact" nav>
          <v-list-item link @click="addBulk">
            <template v-slot:prepend>
              <v-icon icon="mdi-account-multiple-plus"></v-icon>
            </template>
            <v-list-item-title v-text="$t('actions.addbulk')"></v-list-item-title>
          </v-list-item>
          <v-list-item link @click="editBulk">
            <template v-slot:prepend>
              <v-icon icon="mdi-account-multiple-check"></v-icon>
            </template>
            <v-list-item-title v-text="$t('actions.editbulk')"></v-list-item-title>
          </v-list-item>
        </v-list>
      </v-menu>
    </v-col>
    <v-col cols="auto">
      <v-menu v-model="filterMenu" :close-on-content-click="false" location="bottom center">
        <template v-slot:activator="{ props }">
          <v-btn v-bind="props" hide-details variant="text" icon>
            <v-icon :icon="filterSettings.enabled ? 'mdi-filter-check-outline' : 'mdi-filter-menu-outline'" :color="filterSettings.enabled ? 'primary' : ''" />
          </v-btn>
        </template>
        <v-card>
          <v-container>
            <v-row>
              <v-col>
                <v-select
                variant="underlined"
                density="compact"
                :label="$t('type')"
                :items="filterItems"
                v-model="filterSettings.state">
                </v-select>
              </v-col>
            </v-row>
            <v-row>
              <v-col>
                <v-select
                variant="underlined"
                density="compact"
                :label="$t('client.group')"
                :items="[ {title: $t('all'), value: '-'}, ...groups.map(g => ({ title: g.length>0 ? g : $t('none'), value: g}))]"
                v-model="filterSettings.group">
                </v-select>
              </v-col>
            </v-row>
            <v-row>
              <v-col>
                <v-text-field
                variant="underlined"
                density="compact"
                :label="$t('client.name')"
                v-model="filterSettings.text">
                </v-text-field>
              </v-col>
            </v-row>
          </v-container>
          <v-card-actions>
            <v-card-actions>
              <v-spacer></v-spacer>
              <v-btn
                color="blue-darken-1"
                variant="outlined"
                @click="clearFilter"
              >
                {{ $t('actions.del') }}
              </v-btn>
              <v-btn
                color="blue-darken-1"
                variant="tonal"
                @click="doFilter"
              >
                {{ $t('actions.update') }}
              </v-btn>
            </v-card-actions>
          </v-card-actions>
        </v-card>
      </v-menu>
    </v-col>
  </v-row>
  <v-row>
    <v-col cols="12">
      <v-data-table-server
        :headers="headers"
        :items="clients"
        :items-length="pageData.filtered"
        :loading="loading"
        :page="classicPage"
        :items-per-page="itemPerPage"
        :items-per-page-options="[10, 25, 50, 100]"
        @update:page="classicPage = $event"
        @update:sort-by="classicSort = $event"
        @update:items-per-page="setItemPerPage($event)"
        hide-no-data
        fixed-header
        item-value="name"
        :mobile="smAndDown"
        mobile-breakpoint="sm"
        width="100%"
        class="elevation-3 rounded"
        >
        <template v-slot:item.inbounds="{ item }">
          <span>
          <v-tooltip activator="parent" dir="ltr" location="start" v-if="item.inbounds?.length">
            <span v-for="i in item.inbounds">{{ inbounds.find(inb => inb.id == i)?.tag }}<br /></span>
          </v-tooltip>
          {{ item.inbounds?.length }}
          </span>
        </template>
        <template v-slot:item.volume="{ item }">
          <div class="text-start" v-tooltip:top="'↓' + HumanReadable.sizeFormat(item.down) + ' - ' + HumanReadable.sizeFormat(item.up) + '↑'">
            <v-chip
              size="small"
              :color="item.volume==0 ? 'success' : item.volume<=(item.up + item.down)? 'error': ''"
              label
            >{{ HumanReadable.sizeFormat(item.up + item.down) + ' / ' + (item.volume == 0 ? $t('unlimited') : HumanReadable.sizeFormat(item.volume)) }}</v-chip>
          </div>
          <v-progress-linear
            :model-value="percent(item)"
            :color="percentColor(item)"
            v-if="item.volume>0"
            bottom
          >
          </v-progress-linear>
        </template>
        <template v-slot:item.expiry="{ item }">
          <div class="text-start">
            <v-tooltip v-if="item.expiry>0" activator="parent" location="top" :text="new Date(item.expiry * 1000).toLocaleString(locale)" />
            <v-chip
              size="small"
              :color="item.expiry==0 ? 'success' : item.expiry<=Date.now()/1000? 'error': ''"
              label
            >{{ HumanReadable.remainedDays(item.expiry) }}</v-chip>
          </div>
        </template>
        <template v-slot:item.online="{ item }">
          <div class="text-start">
            <template v-if="onlineUsers.has(item.name)">
              <v-chip density="comfortable" size="small" color="success" variant="flat">{{ $t('online') }}</v-chip>
            </template>
            <template v-else>-</template>
          </div>
        </template>
        <template v-slot:item.lastIpCount="{ item }">
          <v-chip size="small" label @click="showClientIps(item.name)">
            {{ item.lastIpCount ?? 0 }}
          </v-chip>
        </template>
        <template v-slot:item.actions="{ item }">
        <v-icon
          class="me-2"
          @click="showModal(item.id)"
        >
          mdi-pencil
        </v-icon>
        <v-menu
          v-model="delOverlay[clients.findIndex(c => c.id == item.id)]"
          :close-on-content-click="false"
          location="top center"
        >
          <template v-slot:activator="{ props }">
            <v-icon
              class="me-2"
              color="error"
              v-bind="props"
            >
              mdi-delete
            </v-icon>
          </template>
          <v-card :title="$t('actions.del')" rounded="lg">
            <v-divider></v-divider>
            <v-card-text>{{ $t('confirm') }}</v-card-text>
            <v-card-actions>
              <v-btn color="error" variant="outlined" @click="delClient(item.id)">{{ $t('yes') }}</v-btn>
              <v-btn color="success" variant="outlined" @click="delOverlay[clients.findIndex(c => c.id == item.id)] = false">{{ $t('no') }}</v-btn>
            </v-card-actions>
          </v-card>
        </v-menu>
        <v-icon
          class="me-2"
          @click="showQrCode(item.id)"
        >
          mdi-qrcode
        </v-icon>
        <v-icon class="me-2" icon="lucide:activity" @click="showDoctor(item.id)">
          <v-tooltip activator="parent" location="top" :text="$t('actions.diagnose')"></v-tooltip>
        </v-icon>
        <v-icon icon="mdi-chart-line" @click="showStats(item.name)" v-if="Data().enableTraffic">
          <v-tooltip activator="parent" location="top" :text="$t('stats.graphTitle')"></v-tooltip>
        </v-icon>
      </template>
      </v-data-table-server>
    </v-col>
  </v-row>
  </template>
</template>
<style>
.v-data-table__tr--mobile td {
  height: fit-content;
  min-height: 36px !important;
}
.v-data-table__tr--mobile td div {
  width:max-content;
}
</style>
<script lang="ts" setup>
import Data from '@/store/modules/data'
import ClientModal from '@/layouts/modals/Client.vue'
import ClientAddBulk from '@/layouts/modals/ClientAddBulk.vue'
import ClientEditBulk from '@/layouts/modals/ClientEditBulk.vue'
import QrCode from '@/layouts/modals/QrCode.vue'
import ClientDoctor from '@/layouts/modals/ClientDoctor.vue'
import Stats from '@/layouts/modals/Stats.vue'
import IpHistoryModal from '@/components/IpHistoryModal.vue'
import { Client } from '@/types/clients'
import { computed, defineAsyncComponent, ref, watch } from 'vue'
import HttpUtils from '@/plugins/httputil'
import type { SortState } from '@/components/nexus/data/dataTableColumns'
import { HumanReadable } from '@/plugins/utils'
import { i18n, locale } from '@/locales'
import { safeGetItem, safeSetItem } from '@/utils/safeStorage'
import { useDisplay } from 'vuetify'
import { useUiMode } from '@/uiMode/useUiMode'

const { smAndDown } = useDisplay()

const { mode } = useUiMode()
const ClientsNexusList = defineAsyncComponent(
  () => import('@/views/clients/ClientsNexusList.vue'),
)
const enableTraffic = computed((): boolean => Data().enableTraffic)

type PageQuery = {
  page: number
  size: number
  sort: SortState | null
  q: string
  group: string
  state: string
}
const data = Data()
const pageData = ref({ clients: [] as (Client & { id: number })[], groups: [] as string[], filtered: 0, total: 0, online: 0, page: 1 })
const loading = ref(false)
const clients = computed(() => pageData.value.clients)
const nexusQuery = ref<PageQuery | null>(null)
const classicPage = ref(1)
const classicSort = ref<readonly { key: string; order?: boolean | 'asc' | 'desc' }[]>([])
const classicFilters = ref({ group: '-', q: '', state: '' })

// One reactive Set per render instead of a throwaway computed per table row:
// O(1) membership lookup, no per-row reactive effect churn on each onlines update.
const onlineUsers = computed(() => new Set<string>(Data().onlines?.user ?? []))

const inbounds = computed((): any[] => {
  return Data().inbounds?? []
})

const inboundTags = computed((): any[] => {
  if (!inbounds.value) return []
  return inbounds.value?.filter(i => i.tag != "" && i.users).map(i => { return { title: i.tag, value: i.id } })
})

const groups = computed(() => pageData.value.groups)

const actionMenu = ref(false)
const filterMenu = ref(false)
const filterSettings = ref({
  enabled: false,
  state: '',
  group: '-',
  text: '',
})

const filterItems = [
  { title: i18n.global.t('none'), value: '' },
  { title: i18n.global.t('disable'), value: 'disable' },
  { title: i18n.global.t('date.expired'), value: 'expired' },
  { title: i18n.global.t('online'), value: 'online' },
]

const headers = [
  { title: i18n.global.t('client.name'), key: 'name' },
  { title: i18n.global.t('client.desc'), key: 'desc' },
  { title: i18n.global.t('client.group'), key: 'group' },
  { title: i18n.global.t('pages.inbounds'), key: 'inbounds', width: 10 },
  { title: i18n.global.t('actions.action'), key: 'actions', sortable: false },
  { title: i18n.global.t('stats.volume'), key: 'volume' },
  { title: i18n.global.t('date.expiry'), key: 'expiry' },
  { title: i18n.global.t('online'), key: 'online' },
  { title: i18n.global.t('client.lastIpCount'), key: 'lastIpCount' },
  { key: 'data-table-group', width: 0 },
]

const itemPerPage = ref(Math.min(100, Math.max(1, Number(safeGetItem('items-per-page')) || 10)))

const setItemPerPage = (items: number) => {
  classicPage.value = 1
  itemPerPage.value = items
  safeSetItem('items-per-page', items.toString())
}

const activeQuery = computed<PageQuery | null>(() => {
  if (mode.value === 'nexus') return nexusQuery.value
  const current = classicSort.value[0]
  return {
    ...classicFilters.value, page: classicPage.value, size: itemPerPage.value,
    sort: current ? { key: current.key, direction: current.order === 'desc' ? 'desc' : 'asc' } : null,
  }
})

watch([activeQuery, () => data.lastLoad, () => data.loadGeneration, () => JSON.stringify(data.onlines?.user ?? [])], ([query], _, onCleanup) => {
  if (!query) return
  let current = true
  loading.value = true
  const timer = setTimeout(async () => {
    try {
      const msg = await HttpUtils.get('api/clients', {
        page: query.page, size: Math.min(100, query.size), q: query.q, group: query.group, state: query.state,
        sort: query.sort?.key ?? '', dir: query.sort?.direction ?? '', fold: mode.value === 'nexus' ? 1 : 0,
      })
      if (current && msg.success && msg.obj.clientPage) {
        pageData.value = msg.obj.clientPage
        if (mode.value === 'classic') classicPage.value = pageData.value.page
      }
    } finally {
      if (current) loading.value = false
    }
  }, 100)
  onCleanup(() => { current = false; clearTimeout(timer) })
}, { immediate: true })

const modal = ref({
  visible: false,
  id: 0,
})

const delOverlay = ref(new Array<boolean>(clients.value.length).fill(false))

const showModal = async (id: number) => {
  modal.value.id = id
  modal.value.visible = true
}
const closeModal = () => {
  modal.value.visible = false
}

const delClient = async (id: number) => {
  const index = clients.value.findIndex(c => c.id === id)
  const success = await Data().save("clients", "del", id)
  if (success) delOverlay.value[index] = false
}

const qrcode = ref({
  visible: false,
  id: 0,
})

const showQrCode = (id: number) => {
  qrcode.value.id = id
  qrcode.value.visible = true
}
const closeQrCode = () => {
  qrcode.value.visible = false
}

const doctor = ref({
  visible: false,
  id: 0,
})

const showDoctor = (id: number) => {
  doctor.value.id = id
  doctor.value.visible = true
}
const closeDoctor = () => {
  doctor.value.visible = false
}

const stats = ref({
  visible: false,
  resource: "user",
  tag: "",
})

const ipModal = ref({
  visible: false,
  client: '',
})

const showClientIps = (clientName: string) => {
  ipModal.value.visible = true
  ipModal.value.client = clientName
}

const onClientIpsCleared = () => {
  Data().loadData()
}

const showStats = (tag: string) => {
  stats.value.tag = tag
  stats.value.visible = true
}
const closeStats = () => {
  stats.value.visible = false
}

const doFilter = () => {
  classicPage.value = 1
  classicFilters.value = { group: filterSettings.value.group, q: filterSettings.value.text, state: filterSettings.value.state }
  filterSettings.value.enabled = true
  filterMenu.value = false
}

const clearFilter = () => {
  classicPage.value = 1
  classicFilters.value = { group: '-', q: '', state: '' }
  filterSettings.value = {
    enabled: false,
    state: '',
    group: '-',
    text: '',
    }
  filterMenu.value = false
}

const addBulkModal = ref(false)

const addBulk = () => {
  addBulkModal.value = true
  actionMenu.value = false
}

const closeAddBulk = () => {
  addBulkModal.value = false
}

const editBulkModal = ref(false)

const editBulk = async () => {
  if (await data.loadClientCatalog()) editBulkModal.value = true
  actionMenu.value = false
}

const closeEditBulk = () => {
  editBulkModal.value = false
}

const percent = (c: Client) => { return c.volume>0 ? Math.round((c.up+c.down) *100 / c.volume) : 0 }
const percentColor = (c: Client) => { return (c.up+c.down) >= c.volume ? 'error' : percent(c)>90 ? 'warning' : 'success' }

</script>
