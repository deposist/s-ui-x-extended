<template>
  <FormShell
    :loading="loading"
    :save-disabled="loading"
    :title="$t('actions.' + title) + ' ' + $t('objects.client')"
    @close="closeModal"
    @save="saveChanges"
  >
      <v-skeleton-loader
          class="mx-auto border"
          width="95%"
          type="card, text, divider, list-item-two-line"
          v-if="loading"
        ></v-skeleton-loader>
      <v-card-text style="padding: 0 16px; overflow-y: scroll;">
        <v-container style="padding: 0;" :hidden="loading">
          <v-tabs
            v-model="tab"
            align-tabs="center"
          >
            <v-tab value="t1">{{ $t('client.basics') }}</v-tab>
            <v-tab value="t2">{{ $t('client.config') }}</v-tab>
            <v-tab value="t3">{{ $t('client.links') }}</v-tab>
            <v-tab value="t4" v-if="awgEndpointItems.length > 0">{{ $t('client.awg.devices') }}</v-tab>
          </v-tabs>
          <v-window v-model="tab">
            <v-window-item value="t1">
              <v-row>
                <v-col cols="12" sm="6" md="4">
                  <v-switch color="primary" v-model="client.enable" :label="$t('enable')" hide-details></v-switch>
                </v-col>
                <v-col cols="12" sm="6" md="4">
                  <v-combobox v-model="client.group" :items="groups" :label="$t('client.group')" hide-details></v-combobox>
                </v-col>
              </v-row>
              <v-row>
                <v-col cols="12" sm="6" md="4">
                  <v-text-field v-model="client.name" :label="$t('client.name')" hide-details></v-text-field>
                </v-col>
                <v-col cols="12" sm="6" md="4">
                  <v-text-field v-model="client.desc" :label="$t('client.desc')" hide-details></v-text-field>
                </v-col>
              </v-row>
              <v-row>
                <v-col cols="12" sm="6" md="4">
                  <v-text-field v-model.number="Volume" type="number" min="0" :label="$t('stats.volume')" suffix="GiB" hide-details></v-text-field>
                </v-col>
                <v-col cols="12" sm="6" md="4" v-if="!(client.delayStart && !client.autoReset)">
                  <DatePick :expiry="expDate" @submit="setDate" />
                </v-col>
                <v-col cols="12" sm="6" md="4" v-if="client.autoReset || client.delayStart">
                  <v-text-field v-model.number="resetDays" type="number" min="1" :label="$t('client.resetDays')" hide-details></v-text-field>
                </v-col>
              </v-row>
              <v-row>
                <v-col cols="12" sm="6" md="4">
                  <v-switch color="primary"
                    :disabled="client.up+client.down>0"
                    v-model="delayStart"
                    :label="$t('client.delayStart')" hide-details>
                  </v-switch>
                </v-col>
                <v-col cols="12" sm="6" md="4">
                  <v-switch color="primary" v-model="autoReset" :label="$t('client.autoReset')" hide-details></v-switch>
                </v-col>
              </v-row>
              <v-row v-if="id > 0">
                <v-col cols="12" sm="6" md="4" class="d-flex flex-column">
                  <div class="d-flex justify-space-between align-center">
                    <div>
                      {{ $t('stats.usage') }}: {{ total }}<sup dir="ltr" v-if="percent>0">({{ percent }}%)</sup>
                    </div>
                    <v-btn density="compact" variant="text" icon="mdi-restore" @click="resetUsage">
                      <v-tooltip activator="parent" location="top">
                        {{ $t('reset') }}
                      </v-tooltip>
                      <v-icon />
                    </v-btn>
                  </div>
                  <v-progress-linear
                    v-model="percent"
                    :color="percentColor"
                    v-if="client.volume>0"
                    bottom
                  >
                  </v-progress-linear>
                </v-col>
                <v-col cols="12" sm="6" md="4">
                  <v-icon icon="mdi-upload" color="orange" /><span class="text-orange">{{ up }}</span>
                  / 
                  <v-icon icon="mdi-download" color="success" /><span class="text-success">{{ down }}</span>
                </v-col>
              </v-row>
              <v-row v-if="id >0 && client.autoReset">
                <v-col cols="12" sm="6" md="4">
                  <div class="text-medium-emphasis">{{ $t('client.nextReset') }}</div>
                  <div dir="ltr">{{ nextResetFormatted }}</div>
                </v-col>
                <v-col cols="12" sm="6" md="4">
                  <div class="text-medium-emphasis">{{ $t('main.stats.totalUsage') }}</div>
                  <div>
                    <v-icon icon="mdi-upload" color="orange" /><span class="text-orange">{{ totalUp }}</span>
                    /
                    <v-icon icon="mdi-download" color="success" /><span class="text-success">{{ totalDown }}</span>
                  </div>
                </v-col>
              </v-row>
              <v-row>
                <v-col cols="12" sm="6" md="4">
                  <v-text-field
                    v-model.number="client.limitIp"
                    type="number"
                    min="0"
                    :label="$t('client.limitIp')"
                    hide-details
                  ></v-text-field>
                </v-col>
                <v-col cols="12" sm="6" md="4">
                  <v-select
                    v-model="client.ipLimitMode"
                    :items="ipLimitModes"
                    :label="$t('client.ipLimitMode')"
                    hide-details
                  ></v-select>
                </v-col>
                <v-col cols="12" sm="6" md="4" v-if="client.ipLimitMode === 'enforce'">
                  <v-alert density="compact" type="warning" variant="tonal">
                    {{ $t('client.ipLimitWarn') }}
                  </v-alert>
                </v-col>
              </v-row>
              <v-row>
                <v-col>
                  <v-select
                    v-model="clientInbounds"
                    :items="inboundTags"
                    :label="$t('client.inboundTags')"
                    clearable
                    multiple
                    chips
                    hide-details>
                    <template v-slot:append>
                      <v-icon @click="setAllInbounds" icon="mdi-set-all" v-tooltip:top="$t('all')" />
                    </template>
                  </v-select>
                </v-col>
              </v-row>
              <v-row v-if="awgEndpointItems.length > 0">
                <v-col>
                  <v-select
                    v-model="clientAwgEndpoints"
                    :items="awgEndpointItems"
                    :label="$t('client.awg.endpoints')"
                    clearable
                    multiple
                    chips
                    hide-details
                    data-testid="client-awg-endpoints">
                  </v-select>
                </v-col>
              </v-row>
            </v-window-item>
            <v-window-item value="t2">
              <v-row>
                <v-col cols="12" sm="6" md="4">
                  <v-btn variant="tonal" @click="shuffle()">{{ $t('reset') + ' - ' + $t('all') }}<v-icon icon="mdi-refresh" /></v-btn>
                </v-col>
              </v-row>
              <v-row v-for="key in Object.keys(clientConfig)">
                <v-col cols="12" md="3" align="end" align-self="center">
                    {{ key }}
                    <v-icon @click="shuffle(key)" icon="mdi-refresh" v-tooltip:top="$t('reset')" />
                </v-col>
                <v-col>
                  <v-text-field
                    v-if="clientConfig[key].password != undefined"
                    label="Password"
                    v-model="clientConfig[key].password"
                    hide-details>
                  </v-text-field>
                  <v-text-field
                    v-if="clientConfig[key].uuid != undefined"
                    label="UUID"
                    v-model="clientConfig[key].uuid"
                    hide-details>
                  </v-text-field>
                  <v-select
                    v-if="key == 'vless'"
                    label="Flow"
                    :items="vlessFlows"
                    v-model="clientConfig[key].flow"
                    hide-details>
                  </v-select>
                  <v-text-field
                    v-if="key == 'hysteria'"
                    label="Auth"
                    v-model="clientConfig[key].auth_str"
                    hide-details>
                  </v-text-field>
                  <v-text-field
                    v-if="clientConfig[key].secret != undefined"
                    label="Secret"
                    v-model="clientConfig[key].secret"
                    hide-details>
                  </v-text-field>
                </v-col>
              </v-row>
            </v-window-item>
            <v-window-item value="t4">
              <v-alert v-if="id == 0 || (client.awgEndpoints ?? []).length == 0" type="info" variant="tonal" class="ma-2">
                {{ $t('client.awg.saveFirst') }}
              </v-alert>
              <template v-else>
                <v-card v-for="access in awgAccesses" :key="access.endpointId" border class="ma-2">
                  <v-card-subtitle style="padding-top: 8px;">
                    {{ access.tag }} · {{ access.publicEndpoint }} · {{ awgDevicesFor(access.endpointId).filter(d => d.desiredEnabled).length }}/{{ access.deviceLimit }}
                  </v-card-subtitle>
                  <v-card-text>
                    <v-row v-for="device in awgDevicesFor(access.endpointId)" :key="device.id" align="center" density="compact">
                      <template v-if="device.desiredEnabled">
                        <v-col cols="12" sm="4">
                          {{ device.name }}
                          <v-chip size="small" :color="device.provisioned ? 'success' : 'warning'" label class="ml-2">
                            {{ device.provisioned ? $t('client.awg.connected') : $t('client.awg.pending') }}
                          </v-chip>
                          <v-chip v-if="awgDeviceExpiryChip(device)" size="small" :color="awgDeviceExpiryChip(device)!.color" label class="ml-2">
                            {{ awgDeviceExpiryChip(device)!.text }}
                          </v-chip>
                          <div class="text-caption">{{ device.ipv4Address }}</div>
                        </v-col>
                        <v-col cols="12" sm="8" class="d-flex flex-wrap ga-1">
                          <v-btn size="small" variant="tonal" prepend-icon="mdi-qrcode" @click="showAwgQr(access.endpointId, device)">{{ $t('client.awg.qr') }}</v-btn>
                          <v-btn size="small" variant="tonal" prepend-icon="mdi-download" @click="downloadAwgConfig(access.endpointId, device)">{{ $t('client.awg.config') }}</v-btn>
                          <v-btn size="small" variant="tonal" prepend-icon="mdi-refresh" @click="rotateAwgDevice(access.endpointId, device)">{{ $t('client.awg.rotate') }}</v-btn>
                          <v-btn size="small" variant="tonal" color="error" prepend-icon="mdi-delete" @click="revokeAwgDevice(access.endpointId, device)">{{ $t('client.awg.revoke') }}</v-btn>
                        </v-col>
                      </template>
                    </v-row>
                    <v-row v-if="awgDevicesFor(access.endpointId).filter(d => d.desiredEnabled).length == 0">
                      <v-col class="text-medium-emphasis">{{ $t('client.awg.none') }}</v-col>
                    </v-row>
                    <v-row>
                      <v-col cols="12" sm="4">
                        <v-text-field v-model="awgNewDeviceNames[access.endpointId]" :label="$t('client.awg.deviceName')" hide-details density="compact" />
                      </v-col>
                      <v-col cols="12" sm="4">
                        <v-text-field
                          v-model="awgNewDeviceExpiry[access.endpointId]"
                          type="date"
                          :label="$t('client.awg.expiresAt')"
                          hide-details density="compact"
                          clearable />
                      </v-col>
                      <v-col cols="12" sm="4">
                        <v-btn color="primary" :loading="awgBusy" prepend-icon="mdi-plus" @click="createAwgDevice(access.endpointId)">{{ $t('client.awg.addDevice') }}</v-btn>
                      </v-col>
                    </v-row>
                  </v-card-text>
                </v-card>
              </template>
              <v-dialog v-model="awgQrDialog" width="420">
                <v-card class="rounded-lg pa-4 text-center">
                  <v-card-title>{{ awgQrDeviceName }}</v-card-title>
                  <img v-if="awgQrUrl" :src="awgQrUrl" alt="AWG QR" style="width: 100%; max-width: 380px; margin: 0 auto;" />
                  <v-card-actions>
                    <v-spacer />
                    <v-btn variant="tonal" @click="awgQrDialog = false">{{ $t('actions.close') }}</v-btn>
                  </v-card-actions>
                </v-card>
              </v-dialog>
            </v-window-item>
            <v-window-item value="t3">
              <v-row v-for="(lnk, index) in links">
                <v-col cols="auto">{{ index + 1 }}</v-col>
                <v-col style="direction: ltr; overflow-y: hidden;">{{ lnk.uri }}</v-col>
              </v-row>
              <v-row>
                <v-col>
                  <v-btn color="primary" @click="extLinks.push({ type: 'external', uri: ''})">{{ $t('actions.add') }} {{ $t('client.external') }}</v-btn>
                </v-col>
              </v-row>
              <v-row v-for="(lnk, index) in extLinks">
                <v-col>
                  <v-text-field
                  dir="ltr"
                  :label="$t('client.external') + ' ' + (index+1)"
                  append-icon="mdi-delete"
                  @click:append="extLinks.splice(index,1)"
                  placeholder="<protocol>://<data>"
                  v-model="lnk.uri" />
                </v-col>
              </v-row>
              <v-row>
                <v-col>
                  <v-btn color="primary" @click="subLinks.push({ type: 'sub', uri: ''})">{{ $t('actions.add') }} {{ $t('client.sub') }}</v-btn>
                </v-col>
              </v-row>
              <v-row v-for="(lnk, index) in subLinks">
                <v-col>
                  <v-text-field
                  dir="ltr"
                  :label="$t('client.sub') + ' ' + (index+1)"
                  append-icon="mdi-delete"
                  @click:append="subLinks.splice(index,1)"
                  placeholder="http[s]://<domain>[:]<port>/<path>"
                  v-model="lnk.uri" />
                </v-col>
              </v-row>
            </v-window-item>
          </v-window>
        </v-container>
      </v-card-text>
  </FormShell>
</template>

<script lang="ts">
import { createClient, randomConfigs, updateConfigs, Link, shuffleConfigs } from '@/types/clients'
import HttpUtils from '@/plugins/httputil'
import api from '@/plugins/api'
import { push } from 'notivue'
import DatePick from '@/components/DateTime.vue'
import { HumanReadable } from '@/plugins/utils'
import Data from '@/store/modules/data'
import { locale } from '@/locales'
import { vlessFlows } from '@/types/recommended'
import FormShell from '@/components/nexus/drawers/FormShell.vue'

export default {
  props: ['visible', 'id', 'inboundTags', 'groups'],
  emits: ['close'],
  data() {
    return {
      client: createClient(),
      title: "add",
      loading: false,
      tab: "t1",
      clientConfig: <any>[],
      links: <Link[]>[],
      extLinks: <Link[]>[],
      subLinks: <Link[]>[],
      ipLimitModes: ['monitor', 'enforce'],
      vlessFlows,
      awgAccesses: <any[]>[],
      awgDevices: <Record<number, any[]>>{},
      awgNewDeviceNames: <Record<number, string>>{},
      awgNewDeviceExpiry: <Record<number, string>>{},
      awgBusy: false,
      awgQrDialog: false,
      awgQrUrl: '',
      awgQrDeviceName: '',
    }
  },
  methods: {
    async updateData(id: number) {
      this.awgAccesses = []
      this.awgDevices = {}
      this.awgNewDeviceNames = {}
      this.awgNewDeviceExpiry = {}
      if (this.awgQrUrl) { URL.revokeObjectURL(this.awgQrUrl); this.awgQrUrl = '' }
      if (id > 0) {
        this.loading = true
        const newData = await Data().loadClients(id)
        this.client = createClient(newData)
        this.title = "edit"
        this.clientConfig = this.client.config
        this.loading = false
        this.loadAwgState()
      }
      else {
        this.client = createClient()
        this.title = "add"
        this.clientConfig = randomConfigs('client')
      }
      this.links = this.client.links?.filter(l => l.type == 'local')?? []
      this.extLinks = this.client.links?.filter(l => l.type == 'external')?? []
      this.subLinks = this.client.links?.filter(l => l.type == 'sub')?? []
      this.tab = "t1"
      this.loading = false
    },
    closeModal() {
      this.updateData(0) // reset
      this.$emit('close')
    },
    async saveChanges() {
      // Guard against double-submit: ignore re-entry while a save is in flight
      // (the button is also :disabled while loading).
      if (!this.$props.visible || this.loading) return
      // check duplicate name
      const isDuplicateName = Data().checkClientName(this.$props.id, this.client.name)
      if (isDuplicateName) return

      // check if delayStart is true and autoReset is false, set expiry to 0
      if (this.client.delayStart && !this.client.autoReset) this.client.expiry = 0

      // save data
      this.loading = true
      try {
        this.client.config = updateConfigs(this.clientConfig, this.client.name)
        this.client.links = [
                          ...this.extLinks.filter(l => l.uri != ''),
                          ...this.subLinks.filter(l => l.uri != '')]
        const success = await Data().save("clients", this.$props.id == 0 ? "new" : "edit", this.client)
        if (success) this.closeModal()
      } finally {
        this.loading = false
      }
    },
    setDate(newDate:number){
      this.client.expiry = newDate
    },
    setAllInbounds(){
      this.client.inbounds = this.inboundTags.map((i:any) => i.value).sort()
    },
    shuffle(k?:string) {
      shuffleConfigs(this.clientConfig, k)
    },
    resetUsage(){
      this.client.totalUp = (this.client.totalUp ?? 0) + this.client.up
      this.client.totalDown = (this.client.totalDown ?? 0) + this.client.down
      this.client.up = 0
      this.client.down = 0
    },
    awgDevicesFor(endpointId: number): any[] {
      return this.awgDevices[endpointId] ?? []
    },
    async loadAwgState() {
      if (this.$props.id == 0) return
      const accessMsg = await HttpUtils.get(`api/awg/clients/${this.$props.id}/access`)
      if (!accessMsg.success) return
      this.awgAccesses = accessMsg.obj ?? []
      for (const access of this.awgAccesses) {
        const devicesMsg = await HttpUtils.get(`api/awg/clients/${this.$props.id}/devices`, { endpointId: access.endpointId })
        if (devicesMsg.success) this.awgDevices[access.endpointId] = devicesMsg.obj ?? []
      }
    },
    async createAwgDevice(endpointId: number) {
      const name = (this.awgNewDeviceNames[endpointId] ?? '').trim()
      if (name.length == 0) return
      // Optional expiry: the date input yields YYYY-MM-DD; expiry is the end
      // of that day (exclusive boundary, local time). Empty = never expires.
      let expiresAt = 0
      const expiryDate = (this.awgNewDeviceExpiry[endpointId] ?? '').trim()
      if (expiryDate.length > 0) {
        const parsed = new Date(expiryDate + 'T23:59:59')
        if (isNaN(parsed.getTime()) || parsed.getTime() <= Date.now()) {
          push.error({ message: this.$t('client.awg.expiryInvalid') })
          return
        }
        expiresAt = Math.floor(parsed.getTime() / 1000)
      }
      this.awgBusy = true
      try {
        const msg = await HttpUtils.post(`api/awg/clients/${this.$props.id}/devices`, { endpointId, name, expiresAt })
        if (msg.success) {
          this.awgNewDeviceNames[endpointId] = ''
          this.awgNewDeviceExpiry[endpointId] = ''
          await this.loadAwgState()
        }
      } finally {
        this.awgBusy = false
      }
    },
    awgDeviceExpiryChip(device: any): { text: string, color: string } | null {
      const expiresAt = device?.expiresAt ?? 0
      if (!expiresAt) return null
      const secondsLeft = expiresAt - Math.floor(Date.now() / 1000)
      if (secondsLeft <= 0) {
        return { text: this.$t('client.awg.expired'), color: 'error' }
      }
      const days = Math.ceil(secondsLeft / 86400)
      return {
        text: this.$t('client.awg.expiresIn', { days }),
        color: days <= 3 ? 'warning' : 'secondary',
      }
    },
    async rotateAwgDevice(endpointId: number, device: any) {
      this.awgBusy = true
      try {
        const msg = await HttpUtils.post(`api/awg/clients/${this.$props.id}/devices/${device.id}/rotate?endpointId=${endpointId}`, null)
        if (msg.success) await this.loadAwgState()
      } finally {
        this.awgBusy = false
      }
    },
    async revokeAwgDevice(endpointId: number, device: any) {
      this.awgBusy = true
      try {
        const msg = await HttpUtils.post(`api/awg/clients/${this.$props.id}/devices/${device.id}/revoke?endpointId=${endpointId}`, null)
        if (msg.success) await this.loadAwgState()
      } finally {
        this.awgBusy = false
      }
    },
    async downloadAwgConfig(endpointId: number, device: any) {
      const response = await api.get(`api/awg/clients/${this.$props.id}/devices/${device.id}/config`, {
        params: { endpointId }, responseType: 'blob',
      })
      const url = URL.createObjectURL(response.data)
      const link = document.createElement('a')
      link.href = url
      link.download = `${device.name || 'awg-device'}.conf`
      link.click()
      URL.revokeObjectURL(url)
    },
    async showAwgQr(endpointId: number, device: any) {
      const response = await api.get(`api/awg/clients/${this.$props.id}/devices/${device.id}/qr`, {
        params: { endpointId }, responseType: 'blob',
      })
      if (this.awgQrUrl) URL.revokeObjectURL(this.awgQrUrl)
      this.awgQrUrl = URL.createObjectURL(response.data)
      this.awgQrDeviceName = device.name
      this.awgQrDialog = true
    },
  },
  computed: {
    clientInbounds: {
      get() { return this.client.inbounds.length>0 ? this.client.inbounds.sort() : [] },
      set(v:number[]) { this.client.inbounds = v.length == 0 ?  [] : v.sort() }
    },
    awgEndpointItems(): any[] {
      const endpoints = Data().endpoints ?? []
      return endpoints
        .filter((e: any) => e.awgManaged === true)
        .map((e: any) => ({ title: e.tag, value: e.id }))
    },
    clientAwgEndpoints: {
      get(): number[] { return (this.client.awgEndpoints ?? []).slice().sort((a: number, b: number) => a - b) },
      set(v: number[]) { this.client.awgEndpoints = (v ?? []).slice().sort((a: number, b: number) => a - b) }
    },
    expDate: {
      get() { return this.client.expiry},
      set(v:any) { this.client.expiry = v }
    },
    Volume: {
      get() { return this.client.volume == 0 ? 0 : (this.client.volume / (1024 ** 3)) },
      set(v:number) { this.client.volume = v > 0 ? v*(1024 ** 3) : 0 }
    },
    delayStart: {
      get() { return this.client.delayStart?? false },
      set(v:boolean) {
        this.client.delayStart = v
        this.client.resetDays = v ? 1 : 0
        if (v && !this.autoReset) this.client.expiry = 0
      }
    },
    autoReset: {
      get() { return this.client.autoReset?? false },
      set(v:boolean) {
        this.client.autoReset = v
        this.client.resetDays = v ? 1 : 0
        if (!v) this.client.nextReset = 0
      }
    },
    resetDays: {
      get() { return this.client.resetDays?? 1 },
      set(v:number|null) {
        if (!v) v = 1
        if (this.client.nextReset && this.client.nextReset > 0) {
          this.client.nextReset += (v-(this.client.resetDays?? 0))*24*60*60
        }
        this.client.resetDays = v
      }
    },
    up() :string { return HumanReadable.sizeFormat(this.client.up) },
    down() :string { return HumanReadable.sizeFormat(this.client.down) },
    total() :string { return HumanReadable.sizeFormat(this.client.down + this.client.up) },
    totalUp() :string { return HumanReadable.sizeFormat((this.client.totalUp ?? 0) + this.client.up) },
    totalDown() :string { return HumanReadable.sizeFormat((this.client.totalDown ?? 0) + this.client.down) },
    nextResetFormatted() :string {
      const ts = this.client.nextReset?? 0
      if (ts == 0) return '-'
      const date = new Date(ts*1000)
      return date.toLocaleString(locale)
    },
    percent() :number { return this.client.volume>0 ? Math.round((this.client.up + this.client.down) *100 / this.client.volume) : 0 },
    percentColor() :string { return (this.client.up+this.client.down) >= this.client.volume ? 'error' : this.percent>90 ? 'warning' : 'success' },
  },
  watch: {
    visible(newValue) {
      if (newValue) {
        this.updateData(this.$props.id)
      }
    },
  },
  components: { DatePick, FormShell },
}

</script>
