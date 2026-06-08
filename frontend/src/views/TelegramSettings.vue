<template>
  <v-card :loading="loading">
    <v-card-title>{{ $t('telegram.title') }}</v-card-title>
    <v-divider></v-divider>
    <v-card-text>
      <v-alert type="warning" variant="tonal" density="compact" class="mb-4">
        {{ $t('telegram.securityWarning') }}
      </v-alert>
      <v-row align="center">
        <v-col cols="12" sm="6" md="4">
          <v-switch color="primary" v-model="telegramEnabled" :label="$t('telegram.enabled')" hide-details />
        </v-col>
        <v-col cols="12" sm="6" md="4">
          <v-switch color="primary" v-model="telegramNotifyCpu" :label="$t('telegram.notifyCpu')" hide-details />
        </v-col>
        <v-col cols="12" sm="6" md="4">
          <v-switch color="primary" v-model="telegramReport" :label="$t('telegram.report')" hide-details />
        </v-col>
      </v-row>
      <v-row>
        <v-col cols="12" sm="6" md="4">
          <SettingsSecretField
            v-model="settings.telegramBotToken"
            :has-secret="settings.telegramBotTokenHasSecret"
            :label="$t('telegram.botToken')"
            hide-details
          />
        </v-col>
        <v-col cols="12" sm="6" md="4">
          <v-text-field v-model="settings.telegramChatID" :label="$t('telegram.chatId')" hide-details />
        </v-col>
        <v-col cols="12" sm="6" md="4">
          <v-text-field
            v-model.number="telegramCpuThreshold"
            type="number"
            min="1"
            max="100"
            :label="$t('telegram.cpuThreshold')"
            suffix="%"
            hide-details
          />
        </v-col>
      </v-row>
      <v-row>
        <v-col cols="12" sm="6" md="4">
          <v-select
            v-model="settings.telegramTransportMode"
            :items="transportModes"
            item-title="title"
            item-value="value"
            label="Transport"
            hide-details
          />
        </v-col>
        <v-col v-if="settings.telegramTransportMode === 'outbound'" cols="12" sm="6" md="8">
          <v-select
            v-model="settings.telegramOutboundTag"
            :items="outboundOptions"
            item-title="title"
            item-value="value"
            label="Outbound (sing-box) — requires core running"
            :hint="outboundOptions.length === 0 ? 'No outbounds configured' : ''"
            persistent-hint
          />
        </v-col>
      </v-row>
      <v-row v-if="settings.telegramTransportMode !== 'outbound'">
        <v-col cols="12" sm="6" md="4">
          <SettingsSecretField
            v-model="settings.telegramProxyURL"
            :has-secret="settings.telegramProxyURLHasSecret"
            :label="$t('telegram.proxyUrl')"
            hide-details
          />
        </v-col>
        <v-col cols="12" sm="6" md="4">
          <SettingsSecretField
            v-model="settings.telegramProxyUsername"
            :has-secret="settings.telegramProxyUsernameHasSecret"
            :label="$t('telegram.proxyUsername')"
            hide-details
          />
        </v-col>
        <v-col cols="12" sm="6" md="4">
          <SettingsSecretField
            v-model="settings.telegramProxyPassword"
            :has-secret="settings.telegramProxyPasswordHasSecret"
            :label="$t('telegram.proxyPassword')"
            hide-details
          />
        </v-col>
      </v-row>
      <v-row>
        <v-col cols="12" md="8">
          <v-text-field v-model="settings.telegramReportCron" :label="$t('telegram.reportCron')" hide-details />
        </v-col>
      </v-row>
      <v-divider class="my-4"></v-divider>
      <section :class="{ 'telegram-backup-disabled': !telegramEnabled }">
        <div class="text-subtitle-1 mb-2">{{ $t('telegram.backup.title') }}</div>
        <v-row align="center">
          <v-col cols="12" sm="6" md="4">
            <v-switch
              color="primary"
              v-model="telegramBackupEnabled"
              :label="$t('telegram.backup.enabled')"
              :disabled="!telegramEnabled"
              hide-details
            />
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-text-field
              v-model.number="telegramBackupMaxSizeMB"
              type="number"
              min="1"
              max="50"
              :label="$t('telegram.backup.maxSize')"
              suffix="MB"
              :disabled="!telegramEnabled"
              hide-details
            />
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-btn
              variant="outlined"
              color="primary"
              :loading="backupRunLoading"
              :disabled="!telegramEnabled"
              @click="sendTelegramBackupNow"
            >
              <v-icon icon="mdi-cloud-upload-outline" class="me-2" />
              {{ $t('telegram.backup.sendNow') }}
            </v-btn>
          </v-col>
        </v-row>
        <v-row>
          <v-col cols="12" md="6">
            <SettingsSecretField
              v-model="settings.telegramBackupPassphrase"
              :has-secret="settings.telegramBackupPassphraseHasSecret"
              :label="$t('telegram.backup.passphrase')"
              :disabled="!telegramEnabled"
              :error-messages="telegramBackupPassphraseErrors"
              :hide-details="telegramBackupPassphraseErrors.length === 0"
            />
            <div class="text-caption text-medium-emphasis mt-1">
              {{ $t('telegram.backup.passphraseHint') }}
            </div>
          </v-col>
          <v-col cols="12" md="6">
            <v-row>
              <v-col cols="12" md="6">
                <v-select
                  v-model="telegramBackupScheduleMode"
                  :items="telegramBackupScheduleOptions"
                  item-title="title"
                  item-value="value"
                  :label="$t('telegram.backup.schedule.title')"
                  :disabled="!telegramEnabled"
                  hide-details
                  @update:model-value="handleTelegramBackupScheduleModeChange"
                />
              </v-col>
              <v-col v-if="telegramBackupScheduleMode === 'custom'" cols="12" md="3">
                <v-text-field
                  v-model.number="telegramBackupCustomValue"
                  type="number"
                  min="1"
                  :max="telegramBackupCustomMax"
                  :label="$t('telegram.backup.schedule.customValue')"
                  :disabled="!telegramEnabled"
                  :error-messages="telegramBackupScheduleErrors"
                  :hide-details="telegramBackupScheduleErrors.length === 0"
                  @update:model-value="updateTelegramBackupCronFromSchedule"
                />
              </v-col>
              <v-col v-if="telegramBackupScheduleMode === 'custom'" cols="12" md="3">
                <v-select
                  v-model="telegramBackupCustomUnit"
                  :items="telegramBackupScheduleUnitOptions"
                  item-title="title"
                  item-value="value"
                  :label="$t('telegram.backup.schedule.customUnit')"
                  :disabled="!telegramEnabled"
                  hide-details
                  @update:model-value="updateTelegramBackupCronFromSchedule"
                />
              </v-col>
              <v-col v-if="telegramBackupScheduleMode === 'advanced'" cols="12">
                <v-text-field
                  v-model="telegramBackupAdvancedCron"
                  :label="$t('telegram.backup.schedule.advancedCron')"
                  :disabled="!telegramEnabled"
                  :error-messages="telegramBackupScheduleErrors"
                  :hide-details="telegramBackupScheduleErrors.length === 0"
                  @update:model-value="updateTelegramBackupCronFromSchedule"
                />
              </v-col>
            </v-row>
          </v-col>
        </v-row>
        <v-row>
          <v-col cols="12">
            <div class="text-caption text-medium-emphasis mb-1">{{ $t('telegram.backup.excludeTables') }}</div>
          </v-col>
          <v-col v-for="table in telegramBackupExcludeTableOptions" :key="table" cols="12" sm="6" md="3">
            <v-checkbox
              v-model="telegramBackupExcludeTables"
              :value="table"
              :label="$t('telegram.backup.tables.' + table)"
              :disabled="!telegramEnabled"
              hide-details
            />
          </v-col>
        </v-row>
        <v-row v-if="backupRunStatus">
          <v-col cols="12" md="6">
            <v-chip :color="backupRunStatus.success ? 'success' : 'warning'" label>
              {{ backupRunStatus.timestamp }} · {{ backupRunStatus.success ? $t('success') : backupRunStatus.errorClass }}
            </v-chip>
          </v-col>
        </v-row>
      </section>
      <v-row align="center">
        <v-col cols="auto">
          <v-btn
            color="primary"
            :loading="loading"
            :disabled="!stateChange || telegramBackupScheduleErrors.length > 0 || telegramBackupPassphraseErrors.length > 0"
            @click="save"
          >
            {{ $t('actions.save') }}
          </v-btn>
        </v-col>
        <v-col cols="auto">
          <v-btn variant="outlined" color="primary" :loading="testLoading" @click="testTelegram">
            <v-icon icon="mdi-send-check-outline" class="me-2" />
            {{ $t('actions.test') }}
          </v-btn>
        </v-col>
        <v-col cols="12" md="6" v-if="testResult">
          <v-chip :color="testResult.success ? 'success' : 'warning'" label>
            {{ testResult.success ? $t('success') : testResult.errorClass }}
          </v-chip>
        </v-col>
      </v-row>
    </v-card-text>
  </v-card>
</template>

<script lang="ts" setup>
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { i18n } from '@/locales'
import HttpUtils from '@/plugins/httputil'
import { FindDiff } from '@/plugins/utils'
import { push } from 'notivue'
import SettingsSecretField from '@/components/SettingsSecretField.vue'
import { normalizeSecretFields, stripSecretPlaceholders } from '@/components/settingsSecretField'
import {
  parseTelegramBackupSchedule,
  serializeTelegramBackupSchedule,
  validateTelegramBackupSchedule,
  type TelegramBackupScheduleMode,
  type TelegramBackupScheduleUnit,
} from '@/views/telegramBackupSchedule'
import {
  hasWeakTelegramBackupPassphrase,
  pickTelegramSettings,
  type TelegramSettingsMap,
} from '@/views/telegramSettingsPayload'

type TelegramResult = {
  success: boolean
  errorClass?: string
}

type BackupRunStatus = {
  success: boolean
  timestamp: string
  errorClass?: string
}

const defaultTelegramSettings: TelegramSettingsMap = {
  telegramEnabled: 'false',
  telegramBotToken: '',
  telegramBotTokenHasSecret: 'false',
  telegramChatID: '',
  telegramProxyURL: '',
  telegramProxyURLHasSecret: 'false',
  telegramProxyUsername: '',
  telegramProxyUsernameHasSecret: 'false',
  telegramProxyPassword: '',
  telegramProxyPasswordHasSecret: 'false',
  telegramTransportMode: 'proxy',
  telegramOutboundTag: '',
  telegramCpuThreshold: '90',
  telegramNotifyCpu: 'false',
  telegramReport: 'false',
  telegramReportCron: '',
  telegramBackupEnabled: 'false',
  telegramBackupPassphrase: '',
  telegramBackupPassphraseHasSecret: 'false',
  telegramBackupCron: '',
  telegramBackupExcludeTables: 'stats,client_ips,audit_events,changes',
  telegramBackupMaxSizeMB: '45',
}

const loading = ref(false)
const testLoading = ref(false)
const backupRunLoading = ref(false)
const settings = ref<TelegramSettingsMap>({ ...defaultTelegramSettings })
const oldSettings = ref<TelegramSettingsMap>({ ...defaultTelegramSettings })
const testResult = ref<TelegramResult | null>(null)
const backupRunStatus = ref<BackupRunStatus | null>(null)
const backupRunController = ref<AbortController | null>(null)
const telegramBackupExcludeTableOptions = ['stats', 'client_ips', 'audit_events', 'changes']
const telegramBackupScheduleMode = ref<TelegramBackupScheduleMode>('manual')
const telegramBackupCustomValue = ref(15)
const telegramBackupCustomUnit = ref<TelegramBackupScheduleUnit>('minutes')
const telegramBackupAdvancedCron = ref('')

const loadData = async () => {
  loading.value = true
  const msg = await HttpUtils.get('api/settings')
  if (msg.success) {
    setData(msg.obj ?? {})
  }
  loading.value = false
}

const transportModes = [
  { title: 'Proxy', value: 'proxy' },
  { title: 'Outbound (sing-box)', value: 'outbound' },
]
const outboundOptions = ref<{ title: string; value: string }[]>([])
const loadOutbounds = async () => {
  const msg = await HttpUtils.get('api/outbounds')
  const list = msg?.obj?.outbounds
  if (msg.success && Array.isArray(list)) {
    outboundOptions.value = list.map((o: any) => ({ title: `${o.tag} (${o.type})`, value: o.tag }))
  }
}

onMounted(() => {
  loadData()
  loadOutbounds()
})
onUnmounted(() => {
  backupRunController.value?.abort()
})

const setData = (data: TelegramSettingsMap) => {
  const normalized = normalizeSecretFields({ ...defaultTelegramSettings, ...data })
  settings.value = pickTelegramSettings(normalized)
  syncTelegramBackupScheduleFromCron(settings.value.telegramBackupCron)
  oldSettings.value = { ...settings.value }
}

const boolSetting = (key: string) => computed({
  get: () => settings.value[key] === 'true',
  set: (value: boolean) => { settings.value[key] = value ? 'true' : 'false' },
})

const telegramEnabled = boolSetting('telegramEnabled')
const telegramNotifyCpu = boolSetting('telegramNotifyCpu')
const telegramReport = boolSetting('telegramReport')
const telegramBackupEnabled = boolSetting('telegramBackupEnabled')

const telegramCpuThreshold = computed({
  get: () => Number(settings.value.telegramCpuThreshold || 90),
  set: (value: number) => {
    const normalized = Number.isFinite(value) && value > 0 ? Math.min(Math.trunc(value), 100) : 90
    settings.value.telegramCpuThreshold = normalized.toString()
  },
})

const telegramBackupMaxSizeMB = computed({
  get: () => Number(settings.value.telegramBackupMaxSizeMB || 45),
  set: (value: number) => {
    const normalized = Number.isFinite(value) ? Math.min(Math.max(Math.trunc(value), 1), 50) : 45
    settings.value.telegramBackupMaxSizeMB = normalized.toString()
  },
})

const telegramBackupExcludeTables = computed({
  get: () => settings.value.telegramBackupExcludeTables
    .split(',')
    .map(item => item.trim())
    .filter(item => telegramBackupExcludeTableOptions.includes(item)),
  set: (value: string[]) => {
    settings.value.telegramBackupExcludeTables = telegramBackupExcludeTableOptions
      .filter(item => value.includes(item))
      .join(',')
  },
})

const telegramBackupScheduleOptions = computed(() => [
  { title: i18n.global.t('telegram.backup.schedule.manual'), value: 'manual' },
  { title: i18n.global.t('telegram.backup.schedule.every15m'), value: 'every15m' },
  { title: i18n.global.t('telegram.backup.schedule.every30m'), value: 'every30m' },
  { title: i18n.global.t('telegram.backup.schedule.hourly'), value: 'hourly' },
  { title: i18n.global.t('telegram.backup.schedule.every6h'), value: 'every6h' },
  { title: i18n.global.t('telegram.backup.schedule.every12h'), value: 'every12h' },
  { title: i18n.global.t('telegram.backup.schedule.daily3'), value: 'daily3' },
  { title: i18n.global.t('telegram.backup.schedule.custom'), value: 'custom' },
  { title: i18n.global.t('telegram.backup.schedule.advanced'), value: 'advanced' },
])

const telegramBackupScheduleUnitOptions = computed(() => [
  { title: i18n.global.t('telegram.backup.schedule.minutes'), value: 'minutes' },
  { title: i18n.global.t('telegram.backup.schedule.hours'), value: 'hours' },
])

const telegramBackupCustomMax = computed(() => telegramBackupCustomUnit.value === 'hours' ? 23 : 59)

const telegramBackupScheduleState = computed(() => ({
  mode: telegramBackupScheduleMode.value,
  customValue: Number(telegramBackupCustomValue.value),
  customUnit: telegramBackupCustomUnit.value,
  advancedCron: telegramBackupAdvancedCron.value,
}))

const telegramBackupScheduleErrors = computed(() => {
  return validateTelegramBackupSchedule(telegramBackupScheduleState.value)
    .map(error => i18n.global.t('telegram.backup.schedule.errors.' + error))
})

const telegramBackupPassphraseErrors = computed(() => {
  if (!hasWeakTelegramBackupPassphrase(settings.value.telegramBackupPassphrase)) {
    return []
  }
  return [i18n.global.t('telegram.backup.passphraseMinLength')]
})

const syncTelegramBackupScheduleFromCron = (cron: string) => {
  const schedule = parseTelegramBackupSchedule(cron)
  telegramBackupScheduleMode.value = schedule.mode
  telegramBackupCustomValue.value = schedule.customValue
  telegramBackupCustomUnit.value = schedule.customUnit
  telegramBackupAdvancedCron.value = schedule.advancedCron
}

const updateTelegramBackupCronFromSchedule = () => {
  settings.value.telegramBackupCron = serializeTelegramBackupSchedule(telegramBackupScheduleState.value)
}

const handleTelegramBackupScheduleModeChange = () => {
  if (telegramBackupScheduleMode.value === 'advanced' && !telegramBackupAdvancedCron.value.trim()) {
    telegramBackupAdvancedCron.value = settings.value.telegramBackupCron.trim()
  }
  updateTelegramBackupCronFromSchedule()
}

const save = async () => {
  if (telegramBackupScheduleErrors.value.length > 0 || telegramBackupPassphraseErrors.value.length > 0) {
    return
  }
  loading.value = true
  const payload = stripSecretPlaceholders(pickTelegramSettings(settings.value))
  if (payload.telegramEnabled !== 'true') {
    delete payload.telegramBackupEnabled
    delete payload.telegramBackupPassphrase
    delete payload.telegramBackupPassphraseHasSecret
    delete payload.telegramBackupCron
    delete payload.telegramBackupExcludeTables
    delete payload.telegramBackupMaxSizeMB
  }
  const msg = await HttpUtils.post('api/save', { object: 'settings', action: 'set', data: JSON.stringify(payload) })
  if (msg.success) {
    push.success({
      title: i18n.global.t('success'),
      duration: 5000,
      message: i18n.global.t('actions.set') + ' ' + i18n.global.t('telegram.title'),
    })
    setData(msg.obj.settings)
  }
  loading.value = false
}

const testTelegram = async () => {
  testLoading.value = true
  testResult.value = null
  const msg = await HttpUtils.post('api/telegram/test', {})
  if (msg.success) {
    testResult.value = msg.obj as TelegramResult
  }
  testLoading.value = false
}

const sendTelegramBackupNow = async () => {
  backupRunController.value?.abort()
  const controller = new AbortController()
  backupRunController.value = controller
  backupRunLoading.value = true
  const msg = await HttpUtils.post('api/telegram/backup/run', {}, { signal: controller.signal })
  backupRunStatus.value = {
    success: msg.success,
    timestamp: new Date().toLocaleString(),
    errorClass: msg.success ? undefined : String(msg.obj?.errorClass ?? msg.msg),
  }
  backupRunLoading.value = false
  backupRunController.value = null
}

const stateChange = computed(() => {
  return !FindDiff.deepCompare(settings.value, oldSettings.value)
})
</script>

<style scoped>
.telegram-backup-disabled {
  opacity: 0.62;
}
</style>
