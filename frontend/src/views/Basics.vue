<template>
  <v-row style="margin-bottom: 10px;">
    <v-col cols="12" justify="center" align="center">
      <v-btn variant="outlined" color="warning" @click="saveConfig" :loading="loading" :disabled="stateChange">
        {{ $t('actions.save') }}
      </v-btn>
    </v-col>
  </v-row>
  <v-expansion-panels>
    <v-expansion-panel :title="$t('basic.log.title')">
      <v-expansion-panel-text>
        <v-row>
          <v-col cols="12" sm="6" md="3" lg="2">
            <v-switch v-model="appConfig.log.disabled" color="primary" :label="$t('disable')" hide-details></v-switch>
          </v-col>
          <v-col cols="12" sm="6" md="3" lg="2">
            <v-select
              hide-details
              :label="$t('basic.log.level')"
              :items="levels"
              clearable
              @click:clear="delete appConfig.log.level"
              v-model="appConfig.log.level">
            </v-select>
          </v-col>
          <v-col cols="12" sm="6" md="3" lg="2">
            <v-text-field
              v-model="appConfig.log.output"
              hide-details
              placeholder="box.log"
              persistent-placeholder
              :label="$t('basic.log.output')"
            >
              <template v-slot:append-inner><SettingInfo :text="$t('basic.hint.logOutput')" /></template>
            </v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="3" lg="2">
            <v-switch v-model="appConfig.log.timestamp" color="primary" :label="$t('basic.log.timestamp')" hide-details></v-switch>
          </v-col>
        </v-row>
      </v-expansion-panel-text>
    </v-expansion-panel>
    <v-expansion-panel title="NTP">
      <v-expansion-panel-text>
        <v-row>
          <v-col cols="12" sm="6" md="3" lg="2">
            <v-switch v-model="enableNtp" color="primary" :label="$t('enable')" hide-details></v-switch>
          </v-col>
          <v-col cols="12" sm="6" md="3" lg="2" v-if="appConfig.ntp?.enabled">
            <v-text-field
              v-model="appConfig.ntp.server"
              hide-details
              placeholder="time.apple.com"
              persistent-placeholder
              :label="$t('out.addr')"
            >
              <template v-slot:append-inner><SettingInfo :text="$t('basic.hint.ntpServer')" /></template>
            </v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="3" lg="2" v-if="appConfig.ntp?.enabled">
            <v-text-field
              v-model="appConfig.ntp.server_port"
              hide-details
              type="number"
              clearable
              placeholder="123"
              persistent-placeholder
              @click:clear="delete appConfig.ntp?.server_port"
              :label="$t('out.port')"
            >
              <template v-slot:append><SettingInfo :text="$t('basic.hint.ntpPort')" /></template>
            </v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="3" lg="2" v-if="appConfig.ntp?.enabled">
            <v-text-field
              v-model="ntpInterval"
              hide-details
              :suffix="$t('date.m')"
              min="0"
              type="number"
              placeholder="30"
              persistent-placeholder
              :label="$t('ruleset.interval')"
            >
              <template v-slot:append-inner><SettingInfo :text="$t('basic.hint.ntpInterval')" /></template>
            </v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="3" lg="2" v-if="appConfig.ntp?.enabled">
            <v-switch v-model="appConfig.ntp.write_to_system" color="primary" :label="$t('singbox.writeSystemClock')" hide-details></v-switch>
          </v-col>
          <v-col cols="12" v-if="appConfig.ntp?.write_to_system">
            <v-alert density="compact" type="warning" variant="tonal">
              {{ $t('singbox.writeSystemClockWarning') }}
            </v-alert>
          </v-col>
        </v-row>
        <Dial :dial="appConfig.ntp" v-if="appConfig.ntp?.enabled" />
      </v-expansion-panel-text>
    </v-expansion-panel>
    <v-expansion-panel :title="$t('singbox.certificateTrust')">
      <v-expansion-panel-text>
        <v-row>
          <v-col cols="12" sm="6" md="3" lg="2">
            <v-select v-model="certificateMode" hide-details :label="$t('singbox.preset')" :items="certificateModes"></v-select>
          </v-col>
          <v-col cols="12" sm="6" md="3" lg="2" v-if="appConfig.certificate">
            <v-select
              v-model="appConfig.certificate.store"
              hide-details
              clearable
              @click:clear="delete appConfig.certificate?.store"
              :label="$t('tls.store')"
              :items="certificateStores">
            </v-select>
          </v-col>
        </v-row>
        <v-row v-if="appConfig.certificate && (certificateMode == 'file' || certificateMode == 'custom')">
          <v-col cols="12" sm="8">
            <v-textarea v-model="certificatePathText" rows="2" auto-grow hide-details :label="$t('singbox.caFilePaths')"></v-textarea>
          </v-col>
        </v-row>
        <v-row v-if="appConfig.certificate && (certificateMode == 'directory' || certificateMode == 'custom')">
          <v-col cols="12" sm="8">
            <v-textarea v-model="certificateDirectoryText" rows="2" auto-grow hide-details :label="$t('singbox.caDirectoryPaths')"></v-textarea>
          </v-col>
        </v-row>
        <v-row v-if="appConfig.certificate && (certificateMode == 'pem' || certificateMode == 'custom')">
          <v-col cols="12">
            <v-textarea v-model="certificateText" rows="5" auto-grow hide-details :label="$t('singbox.pemCertificates')"></v-textarea>
          </v-col>
        </v-row>
      </v-expansion-panel-text>
    </v-expansion-panel>
    <v-expansion-panel title="Experimental">
      <v-expansion-panel-text>
        <v-row>
          <v-col class="v-card-subtitle">{{ $t('singbox.cacheFile') }}</v-col>
        </v-row>
        <v-row>
          <v-col cols="12" sm="6" md="3" lg="2">
            <v-switch v-model="enableCacheFile" color="primary" :label="$t('enable')" hide-details></v-switch>
          </v-col>
          <v-col cols="12" sm="6" md="3" lg="2" v-if="appConfig.experimental.cache_file">
            <v-text-field
              v-model="appConfig.experimental.cache_file.path"
              hide-details
              placeholder="cache.db"
              persistent-placeholder
              :label="$t('transport.path')"
            >
              <template v-slot:append-inner><SettingInfo :text="$t('basic.hint.cachePath')" /></template>
            </v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="3" lg="2" v-if="appConfig.experimental.cache_file">
            <v-text-field
              v-model="appConfig.experimental.cache_file.cache_id"
              hide-details
              :label="$t('singbox.cacheId')"
            >
              <template v-slot:append-inner><SettingInfo :text="$t('basic.hint.cacheId')" /></template>
            </v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="3" lg="2" v-if="appConfig.experimental.cache_file">
            <v-switch v-model="appConfig.experimental.cache_file.store_fakeip"
              color="primary"
              :label="$t('basic.exp.storeFakeIp')"
              hide-details></v-switch>
          </v-col>
          <v-col cols="12" sm="6" md="3" lg="2" v-if="appConfig.experimental.cache_file">
            <v-switch v-model="appConfig.experimental.cache_file.store_rdrc"
              color="primary"
              :label="$t('singbox.storeRdrc')"
              hide-details></v-switch>
          </v-col>
          <v-col cols="12" sm="6" md="3" lg="2" v-if="appConfig.experimental.cache_file?.store_rdrc">
            <v-text-field
              v-model="appConfig.experimental.cache_file.rdrc_timeout"
              hide-details
              placeholder="7d"
              :label="$t('singbox.rdrcTimeout')">
            </v-text-field>
          </v-col>
        </v-row>
        <v-row>
          <v-col class="v-card-subtitle">{{ $t('singbox.debug') }}</v-col>
        </v-row>
        <v-row>
          <v-col cols="12" sm="6" md="3" lg="2">
            <v-switch v-model="enableDebug" color="primary" :label="$t('enable')" hide-details></v-switch>
          </v-col>
          <template v-if="appConfig.experimental.debug">
            <v-col cols="12" sm="6" md="3" lg="2">
              <v-text-field v-model="appConfig.experimental.debug.listen" hide-details placeholder="127.0.0.1:8080" persistent-placeholder :label="$t('objects.listen')">
                <template v-slot:append-inner><SettingInfo :text="$t('basic.hint.debugListen')" /></template>
              </v-text-field>
            </v-col>
            <v-col cols="12" sm="6" md="3" lg="2">
              <v-text-field v-model.number="appConfig.experimental.debug.gc_percent" type="number" hide-details :label="$t('singbox.gcPercent')">
                <template v-slot:append-inner><SettingInfo :text="$t('basic.hint.gcPercent')" /></template>
              </v-text-field>
            </v-col>
            <v-col cols="12" sm="6" md="3" lg="2">
              <v-text-field v-model="appConfig.experimental.debug.memory_limit" hide-details placeholder="256MiB" persistent-placeholder :label="$t('singbox.memoryLimit')">
                <template v-slot:append-inner><SettingInfo :text="$t('basic.hint.memoryLimit')" /></template>
              </v-text-field>
            </v-col>
            <v-col cols="12" sm="6" md="3" lg="2">
              <v-text-field v-model.number="appConfig.experimental.debug.max_stack" type="number" hide-details :label="$t('singbox.maxStack')">
                <template v-slot:append-inner><SettingInfo :text="$t('basic.hint.maxStack')" /></template>
              </v-text-field>
            </v-col>
            <v-col cols="12" sm="6" md="3" lg="2">
              <v-text-field v-model.number="appConfig.experimental.debug.max_threads" type="number" hide-details :label="$t('singbox.maxThreads')">
                <template v-slot:append-inner><SettingInfo :text="$t('basic.hint.maxThreads')" /></template>
              </v-text-field>
            </v-col>
            <v-col cols="12" sm="6" md="3" lg="2">
              <v-switch v-model="appConfig.experimental.debug.panic_on_fault" color="primary" :label="$t('singbox.panicOnFault')" hide-details></v-switch>
            </v-col>
            <v-col cols="12" sm="6" md="3" lg="2">
              <v-select
                v-model="appConfig.experimental.debug.trace_back"
                hide-details clearable
                @click:clear="delete appConfig.experimental.debug?.trace_back"
                :label="$t('singbox.traceback')"
                :items="['none','single','all','system','crash']">
              </v-select>
            </v-col>
          </template>
        </v-row>
        <v-row>
          <v-col class="v-card-subtitle">Clash API</v-col>
        </v-row>
        <v-row>
          <v-col cols="12" sm="6" md="3" lg="2">
            <v-switch v-model="enableClashApi" color="primary" :label="$t('enable')" hide-details></v-switch>
          </v-col>
          <template v-if="appConfig.experimental.clash_api">
            <v-col cols="12" sm="6" md="3" lg="2">
              <v-text-field
                v-model="appConfig.experimental.clash_api.external_controller"
                hide-details
                placeholder="127.0.0.1:9090"
                persistent-placeholder
                :label="$t('basic.exp.extController')"
              >
                <template v-slot:append-inner><SettingInfo :text="$t('basic.hint.clashController')" /></template>
              </v-text-field>
            </v-col>
            <v-col cols="12" sm="6" md="3" lg="2">
              <v-text-field
                v-model="appConfig.experimental.clash_api.secret"
                hide-details
                :label="$t('basic.exp.secret')"
              >
                <template v-slot:append-inner><SettingInfo :text="$t('basic.hint.clashSecret')" /></template>
              </v-text-field>
            </v-col>
          </template>
        </v-row>
        <v-row v-if="appConfig.experimental.clash_api">
          <v-col cols="12" sm="6" md="3" lg="2">
            <v-text-field
              v-model="appConfig.experimental.clash_api.external_ui"
              hide-details
              :label="$t('basic.exp.extUi')"
            >
              <template v-slot:append-inner><SettingInfo :text="$t('basic.hint.clashExtUi')" /></template>
            </v-text-field>
          </v-col>
          <v-col cols="12" sm="8" md="4">
            <v-text-field
              v-model="appConfig.experimental.clash_api.external_ui_download_url"
              hide-details
              :label="$t('basic.exp.extUiDownloadUrl')"
            >
              <template v-slot:append-inner><SettingInfo :text="$t('basic.hint.clashExtUiUrl')" /></template>
            </v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="3" lg="2">
            <v-select
              v-model="appConfig.experimental.clash_api.external_ui_download_detour"
              hide-details
              :items="outboundTags"
              clearable
              @click:clear="delete appConfig.experimental.clash_api.external_ui_download_detour"
              :label="$t('basic.exp.extUiDownloadDetour')"
            ></v-select>
          </v-col>
        </v-row>
        <v-row v-if="appConfig.experimental.clash_api">
          <v-col cols="12" sm="6" md="3" lg="2">
            <v-select
              v-model="appConfig.experimental.clash_api.default_mode"
              hide-details
              clearable
              :items="['rule','global','direct']"
              placeholder="rule"
              persistent-placeholder
              :label="$t('basic.exp.defaultMode')"
            >
              <template v-slot:append><SettingInfo :text="$t('basic.hint.clashDefaultMode')" /></template>
            </v-select>
          </v-col>
          <v-col cols="12" sm="8" md="4">
            <v-text-field
              v-model="origin"
              hide-details
              :label="$t('basic.exp.allowOrigin') + ' ' + $t('commaSeparated')"
            >
              <template v-slot:append-inner><SettingInfo :text="$t('basic.hint.clashAllowOrigin')" /></template>
            </v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="3" lg="2">
            <v-switch v-model="appConfig.experimental.clash_api.access_control_allow_private_network" color="primary" :label="$t('basic.exp.allowPrivate')" hide-details></v-switch>
          </v-col>
        </v-row>
        <v-row>
          <v-col class="v-card-subtitle">V2Ray API</v-col>
        </v-row>
        <v-row>
          <v-col cols="12" sm="6" md="3" lg="2">
            <v-switch v-model="enableV2rayApi" color="primary" :label="$t('enable')" hide-details></v-switch>
          </v-col>
          <template v-if="appConfig.experimental.v2ray_api">
            <v-col cols="12" sm="6" md="3" lg="2">
              <v-text-field
                v-model="appConfig.experimental.v2ray_api.listen"
                hide-details
                placeholder="127.0.0.1:8080"
                persistent-placeholder
                :label="$t('objects.listen')"
              >
                <template v-slot:append-inner><SettingInfo :text="$t('basic.hint.v2rayListen')" /></template>
              </v-text-field>
            </v-col>
            <v-col cols="12" sm="6" md="3" lg="2">
              <v-switch v-model="appConfig.experimental.v2ray_api.stats.enabled"
                color="primary"
                :label="$t('stats.enable')"
                hide-details></v-switch>
            </v-col>
          </template>
        </v-row>
        <v-row v-if="appConfig.experimental.v2ray_api?.stats?.enabled">
          <v-col cols="12" sm="6">
            <v-select
              hide-details
              :label="$t('pages.inbounds')"
              multiple chips closable-chips
              :items="inboundTags"
              v-model="appConfig.experimental.v2ray_api.stats.inbounds">
            </v-select>
          </v-col>
          <v-col cols="12" sm="6">
            <v-select
              hide-details
              :label="$t('pages.outbounds')"
              multiple chips closable-chips
              :items="outboundTags"
              v-model="appConfig.experimental.v2ray_api.stats.outbounds">
            </v-select>
          </v-col>
          <v-col cols="12" sm="6">
            <v-select
              hide-details
              :label="$t('pages.clients')"
              multiple chips closable-chips
              :items="clientNames"
              v-model="appConfig.experimental.v2ray_api.stats.users">
            </v-select>
          </v-col>
        </v-row>
      </v-expansion-panel-text>
    </v-expansion-panel>
  </v-expansion-panels>
</template>

<script lang="ts" setup>
import Data from '@/store/modules/data'
import Dial from '@/components/Dial.vue'
import SettingInfo from '@/components/SettingInfo.vue'
import { computed, ref, onBeforeMount } from 'vue'
import { i18n } from '@/locales'
import { Config, Ntp } from '@/types/config'
import { FindDiff } from '@/plugins/utils'

const oldConfig = ref(<any>{})
const loading = ref(false)

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

const stateChange = computed(() => {
  return FindDiff.deepCompare(appConfig.value,oldConfig.value)
})

const saveConfig = async () => {
  loading.value = true
  const success = await Data().save("config", "set", appConfig.value)
  if (success) {
    resyncFromStore()
    loading.value = false
  }
}

const inboundTags = computed((): string[] => {
  return [...Data().inbounds?.map((i:any) => i.tag), ...Data().endpoints?.filter((e:any) => e.listen_port > 0).map((e:any) => e.tag)]
})

const clientNames = computed((): string[] => {
  const clients = <any[]>Data().clients
  return clients?.map(c => c.name)
})

const outboundTags = computed((): string[] => {
  return [...Data().outbounds?.map((o:any) => o.tag), ...Data().endpoints?.map((e:any) => e.tag)]
})

const levels = ["trace", "debug", "info", "warn", "error", "fatal", "panic"]
const certificateModes = [
  { title: i18n.global.t('singbox.off'), value: 'off' },
  { title: 'System', value: 'system' },
  { title: 'Mozilla', value: 'mozilla' },
  { title: 'Chrome', value: 'chrome' },
  { title: i18n.global.t('singbox.customCaFile'), value: 'file' },
  { title: i18n.global.t('singbox.customCaDirectory'), value: 'directory' },
  { title: i18n.global.t('singbox.pastePem'), value: 'pem' },
  { title: i18n.global.t('singbox.advanced'), value: 'custom' },
]
const certificateStores = ['system', 'mozilla', 'chrome', 'none']

function textToList(value: string): string[] | undefined {
  const items = value.split('\n').map(item => item.trim()).filter(item => item.length > 0)
  return items.length > 0 ? items : undefined
}

const enableNtp = computed({
  get() { return appConfig.value.ntp?.enabled?? false },
  set(v:boolean) { 
    if (v){
      appConfig.value.ntp = <Ntp>{ enabled: true, server: 'time.apple.com', server_port: 123, interval: '30m'}
    } else { delete appConfig.value.ntp }
  }
})

const ntpInterval = computed({
  get():any { return appConfig.value.ntp?.interval? parseInt(appConfig.value.ntp?.interval.replace('m','')) : null },
  set(v:number) { if (appConfig.value.ntp) v>0 ? appConfig.value.ntp.interval =  v + 'm' : delete appConfig.value.ntp.interval }
})

const enableCacheFile = computed({
  get() { return appConfig.value.experimental.cache_file?.enabled?? false },
  set(v:boolean) { 
    if (v){
      appConfig.value.experimental.cache_file = { enabled: true }
    } else { delete appConfig.value.experimental.cache_file  }
  }
})

const enableDebug = computed({
  get() { return appConfig.value.experimental.debug != undefined },
  set(v:boolean) { v ? appConfig.value.experimental.debug = {} : delete appConfig.value.experimental.debug }
})

const certificateMode = computed({
  get(): string {
    const cert = appConfig.value.certificate
    if (!cert) return 'off'
    if (cert.certificate && cert.certificate.length > 0) return 'pem'
    if (cert.certificate_path && cert.certificate_path.length > 0) return 'file'
    if (cert.certificate_directory_path && cert.certificate_directory_path.length > 0) return 'directory'
    return cert.store || 'system'
  },
  set(v:string) {
    if (v == 'off') {
      delete appConfig.value.certificate
      return
    }
    appConfig.value.certificate = {}
    if (['system', 'mozilla', 'chrome'].includes(v)) {
      appConfig.value.certificate.store = v as 'system' | 'mozilla' | 'chrome'
    } else if (v == 'file') {
      appConfig.value.certificate.certificate_path = []
    } else if (v == 'directory') {
      appConfig.value.certificate.certificate_directory_path = []
    } else if (v == 'pem') {
      appConfig.value.certificate.certificate = []
    } else if (v == 'custom') {
      appConfig.value.certificate.store = 'none'
    }
  }
})

const certificateText = computed({
  get(): string { return appConfig.value.certificate?.certificate?.join('\n') ?? '' },
  set(v:string) {
    if (!appConfig.value.certificate) appConfig.value.certificate = {}
    const values = textToList(v)
    values ? appConfig.value.certificate.certificate = values : delete appConfig.value.certificate.certificate
  }
})

const certificatePathText = computed({
  get(): string { return appConfig.value.certificate?.certificate_path?.join('\n') ?? '' },
  set(v:string) {
    if (!appConfig.value.certificate) appConfig.value.certificate = {}
    const values = textToList(v)
    values ? appConfig.value.certificate.certificate_path = values : delete appConfig.value.certificate.certificate_path
  }
})

const certificateDirectoryText = computed({
  get(): string { return appConfig.value.certificate?.certificate_directory_path?.join('\n') ?? '' },
  set(v:string) {
    if (!appConfig.value.certificate) appConfig.value.certificate = {}
    const values = textToList(v)
    values ? appConfig.value.certificate.certificate_directory_path = values : delete appConfig.value.certificate.certificate_directory_path
  }
})

const enableClashApi = computed({
  get() { return appConfig.value.experimental.clash_api != undefined },
  set(v:boolean) { appConfig.value.experimental.clash_api = v ? { external_controller: '127.0.0.1:9090' } : undefined }
})

const enableV2rayApi = computed({
  get() { return appConfig.value.experimental.v2ray_api != undefined },
  set(v:boolean) { appConfig.value.experimental.v2ray_api = v ? { listen: '127.0.0.1:8080', stats: { enabled: false, inbounds: [], outbounds: [], users: [] }} : undefined }
})

const origin = computed({
  get() { return appConfig.value.experimental.clash_api?.access_control_allow_origin &&
    appConfig.value.experimental.clash_api.access_control_allow_origin.length>0 ? appConfig.value.experimental.clash_api.access_control_allow_origin.join(',') : '' },
  set(v:string) {
    if (appConfig.value.experimental.clash_api?.access_control_allow_origin)
      appConfig.value.experimental.clash_api.access_control_allow_origin = v.length> 0 ? v.split(',') : undefined
    }
})
</script>
