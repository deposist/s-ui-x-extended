<template>
  <page-header v-if="nexus" :title="$t('pages.settings')" />
  <v-card :loading="loading" :flat="nexus" :class="{ 'settings-nexus-card': nexus }">
    <v-tabs
    v-model="tab"
    color="primary"
    align-tabs="center"
    show-arrows
  >
    <v-tab value="t1">{{ $t('setting.interface') }}</v-tab>
    <v-tab value="t2">{{ $t('setting.sub') }}</v-tab>
    <v-tab value="t3">{{ $t('setting.jsonSub') }}</v-tab>
    <v-tab value="t4">{{ $t('setting.clashSub') }}</v-tab>
    <v-tab value="t5">{{ $t('setting.maintenance') }}</v-tab>
  </v-tabs>
  <v-card-text>
    <v-row
      v-if="tab !== 't5'"
      align="center"
      class="settings-actions"
      :class="{ 'settings-actions--nexus': nexus }"
      justify="center"
    >
      <v-col cols="auto">
        <v-btn color="primary" @click="save" :loading="loading" :disabled="!stateChange">
          {{ $t('actions.save') }}
        </v-btn>
      </v-col>
      <v-col cols="auto">
        <v-btn variant="outlined" color="warning" @click="restartApp" :loading="loading" :disabled="stateChange">
          {{ $t('actions.restartApp') }}
        </v-btn>
      </v-col>
    </v-row>
    <v-window v-model="tab">
      <v-window-item value="t1">
        <v-row v-if="showNexusControls">
          <v-col cols="12" sm="6" md="4">
            <ui-mode-control variant="select" />
          </v-col>
        </v-row>
        <v-row>
          <v-col cols="12" sm="6" md="4">
            <v-text-field v-model="settings.webListen" :label="$t('setting.addr')" placeholder="0.0.0.0" persistent-placeholder hide-details>
              <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.webListen')" /></template>
            </v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-text-field v-model.number="webPort" min="1" type="number" :label="$t('setting.port')" placeholder="2095" persistent-placeholder hide-details>
              <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.webPort')" /></template>
            </v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-text-field v-model="settings.webPath" :label="$t('setting.webPath')" placeholder="/app/" persistent-placeholder hide-details>
              <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.webPath')" /></template>
            </v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-text-field v-model="settings.webDomain" :label="$t('setting.domain')" placeholder="example.com" persistent-placeholder hide-details>
              <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.webDomain')" /></template>
            </v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-text-field v-model="settings.webKeyFile" :label="$t('setting.sslKey')" placeholder="/etc/s-ui/panel.key" persistent-placeholder hide-details>
              <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.sslKey')" /></template>
            </v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-text-field v-model="settings.webCertFile" :label="$t('setting.sslCert')" placeholder="/etc/s-ui/panel.crt" persistent-placeholder hide-details>
              <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.sslCert')" /></template>
            </v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-text-field v-model="settings.webURI" :label="$t('setting.webUri')" placeholder="https://panel.example.com/app/" persistent-placeholder hide-details>
              <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.webUri')" /></template>
            </v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-text-field
              type="number"
              v-model.number="sessionMaxAge"
              min="0"
              :label="$t('setting.sessionAge')"
              :suffix="$t('date.m')"
              placeholder="0"
              persistent-placeholder
              hide-details
              >
              <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.sessionAge')" /></template>
            </v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-text-field
              type="number"
              v-model.number="trafficAge"
              min="0"
              :label="$t('setting.trafficAge')"
              :suffix="$t('date.d')"
              placeholder="30"
              persistent-placeholder
              hide-details
              >
              <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.trafficAge')" /></template>
            </v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-autocomplete
              v-model="settings.timeLocation"
              :items="timezones"
              :label="$t('setting.timeLoc')"
              placeholder="Europe/Moscow"
              persistent-placeholder
              auto-select-first
              hide-details>
              <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.timeLoc')" /></template>
            </v-autocomplete>
          </v-col>
        </v-row>
      </v-window-item>

      <v-window-item value="t2">
        <v-row>
          <v-col cols="12" sm="6" md="4">
            <div class="d-flex align-center">
              <v-switch color="primary" v-model="subEncode" :label="$t('setting.subEncode')" hide-details />
              <SettingInfo :text="$t('setting.hint.subEncode')" class="ms-1" />
            </div>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <div class="d-flex align-center">
              <v-switch color="primary" v-model="subShowInfo" :label="$t('setting.subInfo')" hide-details />
              <SettingInfo :text="$t('setting.hint.subInfo')" class="ms-1" />
            </div>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <div class="d-flex align-center">
              <v-switch color="primary" v-model="subSecretRequired" :label="$t('setting.subSecretRequired')" hide-details />
              <SettingInfo :text="$t('setting.hint.subSecretRequired')" class="ms-1" />
            </div>
          </v-col>
        </v-row>
        <v-row>
          <v-col cols="12" sm="6" md="4">
            <div class="d-flex align-center">
              <v-switch color="primary" v-model="subLinkEnable" :label="$t('setting.subLinkEnable')" hide-details />
              <SettingInfo :text="$t('setting.hint.subLinkEnable')" class="ms-1" />
            </div>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <div class="d-flex align-center">
              <v-switch color="primary" v-model="subJsonEnable" :label="$t('setting.subJsonEnable')" hide-details />
              <SettingInfo :text="$t('setting.hint.subJsonEnable')" class="ms-1" />
            </div>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <div class="d-flex align-center">
              <v-switch color="primary" v-model="subClashEnable" :label="$t('setting.subClashEnable')" hide-details />
              <SettingInfo :text="$t('setting.hint.subClashEnable')" class="ms-1" />
            </div>
          </v-col>
        </v-row>
        <v-row>
          <v-col cols="12" sm="6" md="4">
            <v-text-field v-model="settings.subListen" :label="$t('setting.addr')" placeholder="0.0.0.0" persistent-placeholder hide-details>
              <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.subListen')" /></template>
            </v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-text-field
              type="number"
              v-model.number="subPort"
              min="1"
              :label="$t('setting.port')"
              placeholder="2096"
              persistent-placeholder
              hide-details>
              <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.subPort')" /></template>
            </v-text-field>
          </v-col>
        </v-row>
        <v-row>
          <v-col cols="12" sm="6" md="4">
            <v-text-field v-model="settings.subKeyFile" :label="$t('setting.sslKey')" placeholder="/etc/s-ui/sub.key" persistent-placeholder hide-details>
              <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.subKeyFile')" /></template>
            </v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-text-field v-model="settings.subCertFile" :label="$t('setting.sslCert')" placeholder="/etc/s-ui/sub.crt" persistent-placeholder hide-details>
              <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.subCertFile')" /></template>
            </v-text-field>
          </v-col>
        </v-row>
        <v-row>
          <v-col cols="12" sm="6" md="4">
            <v-text-field v-model="settings.subDomain" :label="$t('setting.domain')" placeholder="example.com" persistent-placeholder hide-details>
              <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.subDomain')" /></template>
            </v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-text-field v-model="settings.subPath" :label="$t('setting.path')" placeholder="/sub/" persistent-placeholder hide-details>
              <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.subPath')" /></template>
            </v-text-field>
          </v-col>
        </v-row>
        <v-row>
          <v-col cols="12" sm="6" md="4">
            <v-text-field
              type="number"
              v-model.number="subUpdates"
              min="0"
              :label="$t('setting.update')"
              :suffix="$t('date.h')"
              placeholder="12"
              persistent-placeholder
              hide-details
              >
              <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.update')" /></template>
            </v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-text-field v-model="settings.subURI" :label="$t('setting.subUri')" placeholder="https://sub.example.com/sub/" persistent-placeholder hide-details>
              <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.subUri')" /></template>
            </v-text-field>
          </v-col>
        </v-row>
        <v-row>
          <v-col cols="12" class="v-card-subtitle">{{ $t('setting.subAdvanced') }}</v-col>
          <v-col cols="12" sm="6" md="4">
            <v-text-field v-model="settings.subTitle" :label="$t('setting.subTitle')" placeholder="My VPN" persistent-placeholder hide-details>
              <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.subTitle')" /></template>
            </v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-text-field v-model="settings.subSupportUrl" :label="$t('setting.subSupportUrl')" placeholder="https://t.me/yoursupport" persistent-placeholder hide-details>
              <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.subSupportUrl')" /></template>
            </v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-text-field v-model="settings.subProfileUrl" :label="$t('setting.subProfileUrl')" placeholder="https://example.com" persistent-placeholder hide-details>
              <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.subProfileUrl')" /></template>
            </v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-text-field
              v-model.number="subRateLimitPerIP"
              min="0"
              type="number"
              :label="$t('setting.subRateLimitPerIP')"
              placeholder="60"
              persistent-placeholder
              hide-details
            >
              <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.subRateLimitPerIP')" /></template>
            </v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <div class="d-flex align-center">
              <v-switch color="primary" v-model="subNameInRemark" :label="$t('setting.subNameInRemark')" hide-details />
              <SettingInfo :text="$t('setting.hint.subNameInRemark')" class="ms-1" />
            </div>
          </v-col>
          <v-col cols="12">
            <v-textarea v-model="settings.subAnnounce" :label="$t('setting.subAnnounce')" rows="2" placeholder="Welcome!" persistent-placeholder hide-details>
              <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.subAnnounce')" /></template>
            </v-textarea>
          </v-col>
        </v-row>
      </v-window-item>

      <v-window-item value="t3">
        <v-row>
          <v-col cols="12" sm="6" md="4">
            <v-text-field v-model="settings.subJsonPath" :label="$t('setting.jsonPath')" placeholder="/json/" persistent-placeholder hide-details>
              <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.jsonPath')" /></template>
            </v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-text-field v-model="settings.subJsonURI" :label="$t('setting.jsonSub') + ' ' + $t('setting.subUri')" placeholder="https://sub.example.com/json/" persistent-placeholder hide-details>
              <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.subJsonURI')" /></template>
            </v-text-field>
          </v-col>
        </v-row>
        <SubJsonExtVue :settings="settings" />
      </v-window-item>

      <v-window-item value="t4">
        <v-row>
          <v-col cols="12" sm="6" md="4">
            <v-text-field v-model="settings.subClashPath" :label="$t('setting.clashPath')" placeholder="/clash/" persistent-placeholder hide-details>
              <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.clashPath')" /></template>
            </v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-text-field v-model="settings.subClashURI" :label="$t('setting.clashSub') + ' ' + $t('setting.subUri')" placeholder="https://sub.example.com/clash/" persistent-placeholder hide-details>
              <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.subClashURI')" /></template>
            </v-text-field>
          </v-col>
        </v-row>
        <SubClashExtVue :settings="settings" />
      </v-window-item>

      <v-window-item value="t5">
        <MaintenanceTab />
      </v-window-item>
    </v-window>
  </v-card-text>
</v-card>
</template>

<script lang="ts" setup>
import UiModeControl from '@/components/UiModeControl.vue'
import SettingInfo from '@/components/SettingInfo.vue'
import PageHeader from '@/components/nexus/primitives/PageHeader.vue'
import { isNexusEnabled } from '@/uiMode/featureGate'
import { useUiMode } from '@/uiMode/useUiMode'
import { i18n } from '@/locales'
import { Ref, computed, inject, onMounted, ref } from 'vue'
import HttpUtils from '@/plugins/httputil'
import { FindDiff } from '@/plugins/utils'
import SubJsonExtVue from '@/components/SubJsonExt.vue'
import SubClashExtVue from '@/components/SubClashExt.vue'
import MaintenanceTab from '@/components/settings/MaintenanceTab.vue'
import { normalizeSecretFields, stripSecretPlaceholders } from '@/components/settingsSecretField'
import { push } from 'notivue'
const tab = ref("t1")
// Full IANA timezone list for the timezone picker (a strictly-defined set);
// fall back to a common subset on engines without Intl.supportedValuesOf.
const timezones: string[] = (() => {
  try {
    const list = (Intl as any).supportedValuesOf?.('timeZone')
    if (Array.isArray(list) && list.length) return list
  } catch { /* older engine: use fallback */ }
  return ['UTC', 'Europe/Moscow', 'Europe/London', 'Europe/Berlin', 'America/New_York',
    'America/Los_Angeles', 'America/Sao_Paulo', 'Asia/Shanghai', 'Asia/Tokyo', 'Asia/Dubai',
    'Asia/Kolkata', 'Asia/Tehran', 'Australia/Sydney']
})()
const { mode } = useUiMode()
const nexus = computed(() => mode.value === 'nexus')
const showNexusControls = isNexusEnabled()
const loading:Ref = inject('loading')?? ref(false)
const oldSettings = ref({})

const settings = ref({
	webListen: "",
	webDomain: "",
	webPort: "2095",
	webCertFile: "",
	webKeyFile: "",
  webPath: "/app/",
  webURI: "",
	sessionMaxAge: "0",
  trafficAge: "30",
	timeLocation: "Asia/Shanghai",
  subListen: "",
	subPort: "2096",
	subPath: "/sub/",
	subDomain: "",
	subCertFile: "",
	subKeyFile: "",
	subUpdates: "12",
  subEncode: "true",
  subShowInfo: "false",
  subSecretRequired: "false",
  subRateLimitPerIP: "60",
  subLinkEnable: "true",
  subJsonEnable: "true",
  subClashEnable: "true",
	subURI: "",
  subJsonPath: "/json/",
  subClashPath: "/clash/",
  subJsonURI: "",
  subClashURI: "",
  subTitle: "",
  subSupportUrl: "",
  subProfileUrl: "",
  subAnnounce: "",
  subNameInRemark: "false",
  subJsonExt: "",
  subClashExt: "",
})

onMounted(async () => {
  loading.value = true
  await loadData()
  loading.value = false
})

const loadData = async () => {
  loading.value = true
  const msg = await HttpUtils.get('api/settings')
  loading.value = false
  if (msg.success) {
    setData(msg.obj)
  }
}

const setData = (data: any) => {
  const normalized = normalizeSecretFields(data)
  settings.value = normalized
  oldSettings.value = { ...normalized }
}

const save = async () => {
  loading.value = true
  const payload = stripSecretPlaceholders(settings.value)
  const restartRequired = subscriptionPathChanged()
  const msg = await HttpUtils.post('api/save', { object: 'settings', action: 'set', data: JSON.stringify(payload) })
  if (msg.success) {
    push.success({
      title: i18n.global.t('success'),
      duration: 5000,
      message: i18n.global.t('actions.set') + " " + i18n.global.t('pages.settings')
    })
    if (restartRequired) {
      push.warning({
        title: i18n.global.t('setting.restartRequired'),
        duration: 8000,
        message: i18n.global.t('setting.subPathRestartNotice')
      })
    }
    setData(msg.obj.settings)
  }
  loading.value = false
}

const sleep = (ms: number) => new Promise(resolve => setTimeout(resolve, ms))

const restartApp = async () => {
  loading.value = true
  const msg = await HttpUtils.post('api/restartApp',{})
  if (msg.success) {
    let url = settings.value.webURI
    if (url !== "") {
      const isTLS = settings.value.webCertFile !== "" || settings.value.webKeyFile !== ""
      url = buildURL(settings.value.webDomain,settings.value.webPort.toString(),isTLS, settings.value.webPath)
    }
    await sleep(3000)
    window.location.replace(url)
  }
  loading.value = false
}

const buildURL = (host: string, port: string, isTLS: boolean, path: string) => {
  if (!host || host.length == 0) host = window.location.hostname
  if (!port || port.length == 0) port = window.location.port

  const protocol = isTLS ? "https:" : "http:"

  if (port === "" || (isTLS && port === "443") || (!isTLS && port === "80")) {
      port = ""
  } else {
      port = `:${port}`
  }

  return `${protocol}//${host}${port}${path}settings`
}

const subEncode = computed({
  get: () => { return settings.value.subEncode == "true" },
  set: (v:boolean) => { settings.value.subEncode = v ? "true" : "false" }
})

const subShowInfo = computed({
  get: () => { return settings.value.subShowInfo == "true" },
  set: (v:boolean) => { settings.value.subShowInfo = v ? "true" : "false" }
})

const subSecretRequired = computed({
  get: () => { return settings.value.subSecretRequired == "true" },
  set: (v:boolean) => { settings.value.subSecretRequired = v ? "true" : "false" }
})

const subLinkEnable = computed({
  get: () => { return settings.value.subLinkEnable == "true" },
  set: (v:boolean) => { settings.value.subLinkEnable = v ? "true" : "false" }
})

const subJsonEnable = computed({
  get: () => { return settings.value.subJsonEnable == "true" },
  set: (v:boolean) => { settings.value.subJsonEnable = v ? "true" : "false" }
})

const subClashEnable = computed({
  get: () => { return settings.value.subClashEnable == "true" },
  set: (v:boolean) => { settings.value.subClashEnable = v ? "true" : "false" }
})

const subNameInRemark = computed({
  get: () => { return settings.value.subNameInRemark == "true" },
  set: (v:boolean) => { settings.value.subNameInRemark = v ? "true" : "false" }
})

const webPort = computed({
  get: () => { return settings.value.webPort.length>0 ? parseInt(settings.value.webPort) : 2095 },
  set: (v:number) => { settings.value.webPort = v>0 ? v.toString() : "2095" }
})

const sessionMaxAge = computed({
  get: () => { return settings.value.sessionMaxAge.length>0 ? parseInt(settings.value.sessionMaxAge) : 0 },
  set: (v:number) => { settings.value.sessionMaxAge = v>0 ? v.toString() : "0" }
})

const trafficAge = computed({
  get: () => { return settings.value.trafficAge.length>0 ? parseInt(settings.value.trafficAge) : 0 },
  set: (v:number) => { settings.value.trafficAge = v>0 ? v.toString() : "0" }
})

const subPort = computed({
  get: () => { return settings.value.subPort.length>0 ? parseInt(settings.value.subPort) : 2096 },
  set: (v:number) => { settings.value.subPort = v>0 ? v.toString() : "2096" }
})

const subUpdates = computed({
  get: () => { return settings.value.subUpdates.length>0 ? parseInt(settings.value.subUpdates) : 12 },
  set: (v:number) => { settings.value.subUpdates = v>0 ? v.toString() : "12" }
})

const subRateLimitPerIP = computed({
  get: () => { return settings.value.subRateLimitPerIP.length>0 ? parseInt(settings.value.subRateLimitPerIP) : 60 },
  set: (v:number) => { settings.value.subRateLimitPerIP = v>=0 ? v.toString() : "60" }
})

const subscriptionPathKeys = ['subPath', 'subJsonPath', 'subClashPath'] as const

const subscriptionPathChanged = () => {
  return subscriptionPathKeys.some((key) => settings.value[key] !== (oldSettings.value as any)[key])
}

const stateChange = computed(() => {
  return !FindDiff.deepCompare(settings.value,oldSettings.value)
})
</script>

<style scoped>
.settings-actions {
  margin-block-end: 10px;
}

.settings-nexus-card {
  background: var(--nexus-surface-1);
  border: 1px solid var(--nexus-border);
  border-radius: var(--nexus-radius-lg);
}

.settings-nexus-card :deep(.v-tabs) {
  border-block-end: 1px solid var(--nexus-border);
}

.settings-nexus-card :deep(.v-card-text) {
  padding: var(--nexus-gap-5);
  padding-block-start: var(--nexus-gap-4);
}

.settings-nexus-card :deep(.v-window) {
  padding-block-start: var(--nexus-gap-2);
}

.settings-nexus-card :deep(.v-row) {
  row-gap: var(--nexus-gap-2);
}

.settings-nexus-card :deep(.v-col) {
  min-width: 0;
}

.settings-nexus-card :deep(.v-field) {
  overflow: visible;
}

.settings-nexus-card :deep(.v-field-label) {
  max-width: calc(100% - var(--nexus-gap-4));
  overflow: hidden;
  text-overflow: ellipsis;
}

.settings-actions--nexus {
  gap: var(--nexus-gap-2);
  margin-block-end: var(--nexus-gap-4);
}

.settings-actions--nexus :deep(.v-col) {
  padding: var(--nexus-gap-1);
}

@media (max-width: 600px) {
  .settings-nexus-card :deep(.v-card-text) {
    padding: var(--nexus-gap-3);
  }

  .settings-actions--nexus {
    justify-content: stretch !important;
  }

  .settings-actions--nexus :deep(.v-col) {
    flex: 1 1 100%;
    max-width: 100%;
  }

  .settings-actions--nexus :deep(.v-btn) {
    width: 100%;
  }
}
</style>
