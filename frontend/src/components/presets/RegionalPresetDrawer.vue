<template>
  <v-navigation-drawer
    v-model="visible"
    class="regional-preset-drawer"
    location="right"
    temporary
    :width="drawerWidth"
  >
    <div class="regional-preset-drawer__shell">
      <header class="regional-preset-drawer__header">
        <div>
          <h2>{{ t('regionalPresets.title') }}</h2>
          <p>{{ t('regionalPresets.subtitle') }}</p>
        </div>
        <v-btn
          :aria-label="t('actions.close')"
          icon="mdi-close"
          size="small"
          variant="text"
          @click="closeDrawer"
        />
      </header>

      <main class="regional-preset-drawer__body">
        <template v-if="step === 'selection'">
          <v-alert density="compact" type="info" variant="tonal" class="mb-4">
            {{ t('regionalPresets.security.note') }}
          </v-alert>

          <v-row class="mb-2">
            <v-col cols="12">
              <v-select
                v-model="directOutbound"
                density="compact"
                hide-details
                :items="outboundItems"
                :label="t('regionalPresets.directOutbound')"
                variant="outlined"
              />
            </v-col>
            <v-col cols="12">
              <v-select
                v-model="downloadOutbound"
                density="compact"
                :hint="t('regionalPresets.ruleSetDownload.hint')"
                :items="downloadOutboundItems"
                :label="t('regionalPresets.ruleSetDownload.label')"
                persistent-hint
                variant="outlined"
              />
            </v-col>
          </v-row>

          <v-alert v-if="!hasOutbounds" density="compact" type="warning" variant="tonal" class="mb-4">
            {{ t('regionalPresets.selectOutbounds') }}
          </v-alert>

          <section class="regional-preset-drawer__cards">
            <!-- Region Card Loop -->
            <v-card
              v-for="r in regions"
              :key="r.key"
              variant="outlined"
              class="mb-4"
              rounded="lg"
            >
              <!-- Card header with switch -->
              <div class="pa-4 d-flex justify-space-between align-center">
                <div>
                  <h3 class="text-subtitle-1 font-weight-bold mb-1">{{ r.title }}</h3>
                  <p class="text-caption text-medium-emphasis mb-2">{{ r.description }}</p>
                  <v-chip size="x-small" class="font-weight-medium" variant="tonal">
                    {{ r.status }}
                  </v-chip>
                </div>
                <v-switch
                  v-model="r.state.enabled"
                  color="primary"
                  density="compact"
                  hide-details
                />
              </div>

              <!-- Card body inside expand transition -->
              <v-expand-transition>
                <div v-show="r.state.enabled">
                  <v-divider />
                  <div class="pa-4">
                    <div class="pa-3 bg-surface-variant rounded-lg text-caption text-medium-emphasis d-flex align-center">
                      <v-icon icon="mdi-information-outline" size="small" class="mr-2" />
                      <span>{{ r.dnsText }}</span>
                    </div>
                  </div>
                </div>
              </v-expand-transition>
            </v-card>
          </section>

          <!-- AWG endpoint RU->direct preset (stage 5): appears only when a
               managed endpoint with a listen_port participates in routing. -->
          <v-card v-if="awgEndpointItems.length > 0" variant="outlined" class="mb-4" rounded="lg">
            <div class="pa-4 d-flex justify-space-between align-center">
              <div>
                <h3 class="text-subtitle-1 font-weight-bold mb-1">{{ t('regionalPresets.awgRuDirect.title') }}</h3>
                <p class="text-caption text-medium-emphasis mb-2">{{ t('regionalPresets.awgRuDirect.description') }}</p>
                <v-chip size="x-small" class="font-weight-medium" variant="tonal">
                  {{ awgRuStatus }}
                </v-chip>
              </div>
              <v-switch
                v-model="awgRuState.enabled"
                color="primary"
                density="compact"
                hide-details
              />
            </div>
            <v-expand-transition>
              <div v-show="awgRuState.enabled">
                <v-divider />
                <div class="pa-4">
                  <v-select
                    v-model="awgRuState.endpointTag"
                    density="compact"
                    hide-details
                    :items="awgEndpointItems"
                    :label="t('regionalPresets.awgRuDirect.endpoint')"
                    variant="outlined"
                  />
                </div>
              </div>
            </v-expand-transition>
          </v-card>

          <div class="regional-preset-drawer__manual-link">
            <span>{{ t('regionalPresets.needFullControl') }}</span>
            <span>{{ t('regionalPresets.editRulesManually') }}</span>
          </div>
        </template>

        <template v-else-if="step === 'preview'">
          <v-alert density="compact" type="info" variant="tonal" class="mb-4">
            {{ t('regionalPresets.previewGroups.securityNote') }}
          </v-alert>

          <!-- AWG RU->direct preview -->
          <v-card v-if="awgPreviewVisible" variant="outlined" class="mb-4" rounded="lg">
            <div class="pa-4">
              <h3 class="text-subtitle-1 font-weight-bold mb-3">{{ t('regionalPresets.awgRuDirect.title') }}</h3>
              <div
                v-for="section in [
                  { key: 'willAdd', title: t('regionalPresets.previewGroups.willAdd'), items: awgPreview.willAdd, color: 'success' },
                  { key: 'willKeep', title: t('regionalPresets.previewGroups.willKeep'), items: awgPreview.willKeep, color: 'info' },
                  { key: 'willRemove', title: t('regionalPresets.previewGroups.willRemove'), items: awgPreview.willRemove, color: 'error' }
                ]"
                :key="section.key"
              >
                <div v-if="section.items.length > 0" class="mt-3">
                  <div class="text-caption font-weight-bold d-flex align-center" :class="`text-${section.color}`">
                    <span class="mr-1">•</span>
                    {{ section.title }} ({{ section.items.length }})
                  </div>
                  <ul class="text-caption pl-4 mt-1 text-medium-emphasis">
                    <li v-for="item in section.items" :key="item">{{ item }}</li>
                  </ul>
                </div>
              </div>
            </div>
          </v-card>

          <!-- Preview Card Loop -->
          <div class="regional-preset-drawer__preview-cards">
            <v-card
              v-for="p in [
                { key: 'ru', title: t('regionalPresets.region.ru.title'), state: ruState, group: preview.ru },
                { key: 'zh', title: t('regionalPresets.region.zh.title'), state: zhState, group: preview.zh }
              ]"
              :key="p.key"
              variant="outlined"
              class="mb-4"
              rounded="lg"
            >
              <div class="pa-4">
                <div class="d-flex justify-space-between align-center mb-3">
                  <h3 class="text-subtitle-1 font-weight-bold">{{ p.title }}</h3>
                  <v-chip
                    v-if="p.state.enabled"
                    color="primary"
                    size="small"
                    variant="flat"
                  >
                    {{ t(`regionalPresets.direction.${p.state.direction}.title`) }}
                  </v-chip>
                  <v-chip
                    v-else
                    color="grey"
                    size="small"
                    variant="tonal"
                  >
                    {{ t('regionalPresets.previewGroups.noChanges') }}
                  </v-chip>
                </div>

                <!-- Preview Actions: Will Add, Will Change, Will Keep, Will Remove -->
                <!-- (regional preview card body continues below) -->
                <div v-if="p.state.enabled || p.group.willRemove.length > 0">
                  <div
                    v-for="section in [
                      { key: 'willAdd', title: t('regionalPresets.previewGroups.willAdd'), items: p.group.willAdd, color: 'success' },
                      { key: 'willChange', title: t('regionalPresets.previewGroups.willChange'), items: p.group.willChange, color: 'warning' },
                      { key: 'willKeep', title: t('regionalPresets.previewGroups.willKeep'), items: p.group.willKeep, color: 'info' },
                      { key: 'willRemove', title: t('regionalPresets.previewGroups.willRemove'), items: p.group.willRemove, color: 'error' }
                    ]"
                    :key="section.key"
                  >
                    <div v-if="section.items.length > 0" class="mt-3">
                      <div class="text-caption font-weight-bold d-flex align-center" :class="`text-${section.color}`">
                        <span class="mr-1">•</span>
                        {{ section.title }} ({{ section.items.length }})
                      </div>
                      <ul class="text-caption pl-4 mt-1 text-medium-emphasis">
                        <li v-for="item in section.items" :key="item">{{ item }}</li>
                      </ul>
                    </div>
                  </div>
                </div>

                <!-- Security Warnings -->
                <div class="mt-4 pt-3 border-t border-opacity-25">
                  <div v-if="p.group.securityWarnings.length > 0" class="bg-warning-lighten-5 pa-3 rounded-lg border border-warning border-opacity-25">
                    <div class="text-caption font-weight-bold text-warning d-flex align-center mb-1">
                      <v-icon icon="mdi-alert" size="small" class="mr-1" />
                      {{ t('regionalPresets.previewGroups.securityWarnings') }}
                    </div>
                    <ul class="text-caption pl-4 text-warning-darken-2">
                      <li v-for="warning in p.group.securityWarnings" :key="warning">
                        {{ t(warning) }}
                      </li>
                    </ul>
                  </div>
                  <div v-else class="text-caption text-medium-emphasis style-italic">
                    {{ t('regionalPresets.previewGroups.noWarnings') }}
                  </div>
                </div>
              </div>
            </v-card>
          </div>
        </template>

        <template v-else-if="step === 'success'">
          <div class="regional-preset-drawer__result">
            <v-icon color="success" icon="mdi-check-circle" size="56" />
            <h3>{{ t('regionalPresets.applied') }}</h3>
            <p>{{ t('regionalPresets.result.customItemsKept') }}</p>
            <div class="regional-preset-drawer__result-summary mt-4">
              <span>{{ t('regionalPresets.region.ru.title') }}: {{ resultLabel(ruState) }}</span>
              <span>{{ t('regionalPresets.region.zh.title') }}: {{ resultLabel(zhState) }}</span>
            </div>
          </div>
        </template>

        <template v-else-if="step === 'error'">
          <div class="regional-preset-drawer__result">
            <v-icon color="error" icon="mdi-alert-circle-outline" size="56" />
            <h3>{{ t('regionalPresets.result.failed') }}</h3>
            <p>{{ errorMessage }}</p>
          </div>
        </template>
      </main>

      <footer class="regional-preset-drawer__footer">
        <template v-if="step === 'selection'">
          <v-btn variant="text" @click="closeDrawer">{{ t('regionalPresets.cancel') }}</v-btn>
          <v-btn color="primary" :disabled="!canPreview" variant="flat" @click="openPreview">
            {{ t('regionalPresets.preview') }}
          </v-btn>
        </template>
        <template v-else-if="step === 'preview'">
          <v-btn :disabled="applying" variant="text" @click="step = 'selection'">{{ t('regionalPresets.back') }}</v-btn>
          <v-btn color="primary" :loading="applying" variant="flat" @click="applySelectedPresets">
            {{ t('regionalPresets.apply') }}
          </v-btn>
        </template>
        <template v-else-if="step === 'success'">
          <v-btn color="primary" variant="flat" @click="closeDrawer">{{ t('regionalPresets.done') }}</v-btn>
        </template>
        <template v-else>
          <v-btn color="primary" variant="tonal" @click="step = 'selection'">{{ t('regionalPresets.back') }}</v-btn>
        </template>
      </footer>
    </div>
  </v-navigation-drawer>
</template>

<script lang="ts" setup>
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'

import type { Config } from '@/types/config'
import HttpUtils from '@/plugins/httputil'
import {
  applyPresets,
  computePreview,
  detectPresetState,
  type PresetPreviewGroup,
  type PresetRegion,
  type PresetRegionKey,
  type RegionalPresetState,
  requiredRuleSetSources,
  validatePresetCatalogShape,
} from './routingDnsPresets'
import {
  applyAWGRuDirectState,
  computeAWGRuDirectPreview,
  detectAWGRuDirect,
  type AWGRuDirectState,
} from './awgRuDirectPreset'

const props = defineProps<{
  modelValue: boolean
  config: Config
  outboundTags: string[]
  awgEndpointTags?: string[]
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  apply: [config: Config]
}>()

const { t } = useI18n()
const drawerWidth = 520
const step = ref<'selection' | 'preview' | 'success' | 'error'>('selection')
const directOutbound = ref('direct')
const errorMessage = ref('')
const applying = ref(false)

// Which network path the panel uses to fetch the .srs files. Sentinel value
// rather than '' so the select always has a visible choice; a censored server
// needs one of its own outbounds here, since sources like
// raw.githubusercontent.com are commonly unreachable directly.
const DIRECT_DOWNLOAD = '__direct__'
const downloadOutbound = ref(DIRECT_DOWNLOAD)

const downloadOutboundItems = computed(() => [
  { title: t('regionalPresets.ruleSetDownload.direct'), value: DIRECT_DOWNLOAD },
  ...[...new Set(props.outboundTags.filter(Boolean))].map(tag => ({ title: tag, value: tag })),
])

const ruState = reactive<RegionalPresetState>({ region: 'RU', enabled: false, direction: 'direct', exceptions: [] })
const zhState = reactive<RegionalPresetState>({ region: 'ZH', enabled: false, direction: 'direct', exceptions: [] })
const awgRuState = reactive<AWGRuDirectState>({ enabled: false, endpointTag: '' })

const visible = computed({
  get: () => props.modelValue,
  set: value => emit('update:modelValue', value),
})

const outboundItems = computed(() => {
  const tags = new Set(['direct', ...props.outboundTags.filter(Boolean)])
  return [...tags].map(tag => ({ title: tag, value: tag }))
})

const hasOutbounds = computed(() => directOutbound.value.length > 0)
const hasEnabledRegion = computed(() => ruState.enabled || zhState.enabled)

const awgEndpointItems = computed(() =>
  (props.awgEndpointTags ?? []).filter(Boolean).map(tag => ({ title: tag, value: tag })))

const awgRuStatus = computed(() => {
  const existing = detectAWGRuDirect(props.config)
  return existing.length > 0
    ? t('regionalPresets.region.status.enabled')
    : t('regionalPresets.region.status.notConfigured')
})

const awgChanged = computed(() => {
  const existing = detectAWGRuDirect(props.config)
  const current = awgRuState.enabled && awgRuState.endpointTag ? [awgRuState.endpointTag] : []
  return JSON.stringify(existing) !== JSON.stringify(current)
})

const awgPreview = computed(() => computeAWGRuDirectPreview(props.config, awgRuState, directOutbound.value))
const awgPreviewVisible = computed(() =>
  awgPreview.value.willAdd.length > 0 || awgPreview.value.willKeep.length > 0 || awgPreview.value.willRemove.length > 0)

const hasChanges = computed(() => {
  const detected = detectPresetState(props.config)
  const ruChanged = ruState.enabled !== detected.ru.enabled ||
                    ruState.direction !== detected.ru.direction ||
                    JSON.stringify(ruState.exceptions) !== JSON.stringify(detected.ru.exceptions)
  const zhChanged = zhState.enabled !== detected.zh.enabled ||
                    zhState.direction !== detected.zh.direction ||
                    JSON.stringify(zhState.exceptions) !== JSON.stringify(detected.zh.exceptions)
  return ruChanged || zhChanged
})

const canPreview = computed(() => {
  return hasOutbounds.value && (hasEnabledRegion.value || hasChanges.value || awgChanged.value)
})

const preview = computed(() => {
  if (!hasOutbounds.value) {
    return {
      ru: emptyPreviewGroup(),
      zh: emptyPreviewGroup(),
    }
  }
  return computePreview(props.config, ruState, zhState, {
    directOutbound: directOutbound.value,
  })
})

const regions = computed(() => [
  {
    key: 'ru' as PresetRegionKey,
    region: 'RU' as PresetRegion,
    state: ruState,
    title: t('regionalPresets.region.ru.title'),
    description: t('regionalPresets.region.ru.description'),
    status: regionStatus('RU'),
    dnsText: dnsText(ruState, 'RU'),
  },
  {
    key: 'zh' as PresetRegionKey,
    region: 'ZH' as PresetRegion,
    state: zhState,
    title: t('regionalPresets.region.zh.title'),
    description: t('regionalPresets.region.zh.description'),
    status: regionStatus('ZH'),
    dnsText: dnsText(zhState, 'ZH'),
  }
])

watch(() => props.modelValue, open => {
  if (!open) return
  resetFromConfig()
  void loadDownloadChannel()
})

// Show the channel that is actually configured, so re-opening the drawer does
// not silently reset a previously chosen outbound back to direct.
const loadDownloadChannel = async () => {
  const msg = await HttpUtils.get('api/settings')
  if (!msg.success) return
  const settings = (msg.obj ?? {}) as Record<string, unknown>
  const mode = String(settings.ruleSetDownloadMode ?? 'direct')
  const tag = String(settings.ruleSetDownloadOutbound ?? '')
  downloadOutbound.value = mode === 'outbound' && tag ? tag : DIRECT_DOWNLOAD
}

const emptyPreviewGroup = (): PresetPreviewGroup => ({
  willAdd: [],
  willChange: [],
  willKeep: [],
  willRemove: [],
  securityWarnings: [],
})

const assignState = (target: RegionalPresetState, source: RegionalPresetState) => {
  target.region = source.region
  target.enabled = source.enabled
  target.direction = source.direction
  target.exceptions = [...source.exceptions]
}

const resetFromConfig = () => {
  const detected = detectPresetState(props.config)
  assignState(ruState, detected.ru)
  assignState(zhState, detected.zh)
  const awgExisting = detectAWGRuDirect(props.config)
  awgRuState.enabled = awgExisting.length > 0
  awgRuState.endpointTag = awgExisting[0] ?? ((props.awgEndpointTags ?? [])[0] ?? '')
  directOutbound.value = outboundItems.value.some(item => item.value === 'direct') ? 'direct' : (props.outboundTags[0] ?? '')
  errorMessage.value = ''
  step.value = 'selection'
}

const closeDrawer = () => {
  visible.value = false
}

const dnsText = (state: RegionalPresetState, label: string) => t('regionalPresets.dns.behavior', {
  mode: t(`regionalPresets.direction.${state.direction}.title`),
  region: label,
})

const hasCustomRegionalConfig = (region: PresetRegion) => {
  const needle = region === 'RU' ? ['ru', 'russia', 'blocked', 'private'] : ['cn', 'china', 'zh']
  const raw = JSON.stringify(props.config?.route ?? {}).toLowerCase()
  return needle.some(item => raw.includes(item)) && !detectPresetState(props.config)[region === 'RU' ? 'ru' : 'zh'].enabled
}

const regionStatus = (region: PresetRegion) => {
  const state = region === 'RU' ? ruState : zhState
  if (state.enabled) return t('regionalPresets.region.status.enabled')
  if (hasCustomRegionalConfig(region)) return t('regionalPresets.region.status.customDetected')
  return t('regionalPresets.region.status.notConfigured')
}

const resultLabel = (state: RegionalPresetState) => state.enabled
  ? t(`regionalPresets.direction.${state.direction}.title`)
  : t('disable')

const openPreview = () => {
  if (!validatePresetCatalogShape()) {
    errorMessage.value = t('regionalPresets.result.regionalDataUnavailable')
    step.value = 'error'
    return
  }
  step.value = 'preview'
}

// Download the rule-set files before touching the config.
//
// The order matters and is deliberately fail-closed: if a download fails we
// leave the config untouched, because a config that points at a missing .srs
// stops sing-box from starting at all. Better to keep the current working
// routing than to save something that bricks the core on its next restart.
const applySelectedPresets = async () => {
  applying.value = true
  try {
    const sources = requiredRuleSetSources(ruState, zhState)
    const ruleSetPaths: Record<string, string> = {}

    if (sources.length > 0) {
      // The download channel lives in panel settings, not in the request, so it
      // has to be saved before asking for the download. Doing it in this order
      // means a server whose direct network is censored can pick an outbound
      // here and have the very next download use it.
      const useDirect = downloadOutbound.value === DIRECT_DOWNLOAD
      const saved = await HttpUtils.post('api/save', {
        object: 'settings',
        action: 'set',
        data: JSON.stringify({
          ruleSetDownloadMode: useDirect ? 'direct' : 'outbound',
          ruleSetDownloadOutbound: useDirect ? '' : downloadOutbound.value,
        }),
      })
      if (!saved.success) {
        errorMessage.value = saved.msg || t('regionalPresets.result.ruleSetDownloadFailed')
        step.value = 'error'
        return
      }

      // Requests go out as form-urlencoded, which cannot carry a nested array:
      // posting { sources } directly flattens into sources[0][tag]=... and the
      // API rejects it. The panel convention is a single JSON-encoded "data"
      // field, same as the api/save call above.
      const msg = await HttpUtils.post('api/rulesets/materialize', {
        data: JSON.stringify({ sources }),
      })
      if (!msg.success) {
        errorMessage.value = msg.msg || t('regionalPresets.result.ruleSetDownloadFailed')
        step.value = 'error'
        return
      }
      // The API answers with tag/path pairs; applyPresets keys off the source
      // URL, so map each returned tag back to the URL that was requested.
      const assets = (msg.obj as { tag: string, path: string }[] | null) ?? []
      const urlByTag = new Map(sources.map(source => [source.tag, source.url]))
      for (const asset of assets) {
        const url = urlByTag.get(asset.tag)
        if (url && asset.path) ruleSetPaths[url] = asset.path
      }
    }

    const result = applyPresets(props.config, ruState, zhState, {
      directOutbound: directOutbound.value,
      ruleSetPaths,
      requireLocalAssets: true,
    })
    applyAWGRuDirectState(result.config, awgRuState, directOutbound.value)
    emit('apply', result.config)
    step.value = 'success'
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : t('regionalPresets.result.regionalDataUnavailable')
    step.value = 'error'
  } finally {
    applying.value = false
  }
}
</script>

<style scoped>
.regional-preset-drawer {
  border-inline-start: 1px solid rgb(var(--v-theme-on-surface) / 12%);
}

.regional-preset-drawer__shell {
  display: grid;
  grid-template-rows: auto 1fr auto;
  height: 100%;
  min-height: 0;
}

.regional-preset-drawer__header,
.regional-preset-drawer__footer {
  background: rgb(var(--v-theme-surface));
  border-block-end: 1px solid rgb(var(--v-theme-on-surface) / 12%);
  display: flex;
  gap: 12px;
  justify-content: space-between;
  padding: 16px;
}

.regional-preset-drawer__footer {
  border-block-end: 0;
  border-block-start: 1px solid rgb(var(--v-theme-on-surface) / 12%);
}

.regional-preset-drawer__header h2,
.regional-preset-drawer__result h3 {
  font-size: 1.1rem;
  font-weight: 600;
  line-height: 1.3;
  margin: 0;
}

.regional-preset-drawer__header p,
.regional-preset-drawer__result p,
.regional-preset-drawer__manual-link {
  color: rgb(var(--v-theme-on-surface) / 68%);
  font-size: 0.85rem;
  line-height: 1.45;
  margin: 0;
}

.regional-preset-drawer__body {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-height: 0;
  overflow-y: auto;
  padding: 16px;
}

.regional-preset-drawer__manual-link {
  display: flex;
  gap: 6px;
  justify-content: center;
  margin-top: 8px;
}

.regional-preset-drawer__result {
  align-items: center;
  align-self: center;
  display: grid;
  gap: 12px;
  justify-items: center;
  text-align: center;
  padding: 24px 0;
}

.regional-preset-drawer__result-summary {
  background: rgb(var(--v-theme-on-surface) / 5%);
  border-radius: 12px;
  display: grid;
  gap: 6px;
  min-width: min(320px, 100%);
  padding: 12px;
  text-align: start;
}

.style-italic {
  font-style: italic;
}
</style>
