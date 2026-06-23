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
            <v-col cols="12" sm="6">
              <v-select
                v-model="proxyOutbound"
                density="compact"
                hide-details
                :items="outboundItems"
                :label="t('regionalPresets.proxyOutbound')"
                variant="outlined"
              />
            </v-col>
            <v-col cols="12" sm="6">
              <v-select
                v-model="directOutbound"
                density="compact"
                hide-details
                :items="outboundItems"
                :label="t('regionalPresets.directOutbound')"
                variant="outlined"
              />
            </v-col>
          </v-row>

          <v-alert v-if="!hasOutbounds" density="compact" type="warning" variant="tonal" class="mb-4">
            {{ t('regionalPresets.selectOutbounds') }}
          </v-alert>
          <v-alert v-else-if="sameOutbound" density="compact" type="warning" variant="tonal" class="mb-4">
            {{ t('regionalPresets.sameOutboundWarning') }}
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
                    <!-- Direction Choice -->
                    <div class="text-caption font-weight-bold mb-2">
                      {{ t('regionalPresets.proxyOutbound') }} / {{ t('regionalPresets.directOutbound') }}
                    </div>
                    <v-radio-group
                      v-model="r.state.direction"
                      hide-details
                      class="mt-0"
                    >
                      <v-radio value="direct" class="mb-2">
                        <template #label>
                          <div>
                            <div class="text-body-2 font-weight-bold text-high-emphasis">{{ t('regionalPresets.direction.direct.title') }}</div>
                            <div class="text-caption text-medium-emphasis">{{ t('regionalPresets.direction.direct.description') }}</div>
                          </div>
                        </template>
                      </v-radio>
                      <v-radio value="proxy">
                        <template #label>
                          <div>
                            <div class="text-body-2 font-weight-bold text-high-emphasis">{{ t('regionalPresets.direction.proxy.title') }}</div>
                            <div class="text-caption text-medium-emphasis">{{ t('regionalPresets.direction.proxy.description') }}</div>
                          </div>
                        </template>
                      </v-radio>
                    </v-radio-group>

                    <!-- DNS Text -->
                    <div class="mt-4 pa-3 bg-surface-variant rounded-lg text-caption text-medium-emphasis d-flex align-center">
                      <v-icon icon="mdi-information-outline" size="small" class="mr-2" />
                      <span>{{ r.dnsText }}</span>
                    </div>

                    <!-- Exceptions Panel -->
                    <div class="mt-4">
                      <v-expansion-panels variant="accordion" class="border border-opacity-25 rounded-lg">
                        <v-expansion-panel elevation="0">
                          <v-expansion-panel-title class="text-caption font-weight-bold py-2 px-3 min-height-0">
                            {{ t('regionalPresets.advanced.title') }}
                          </v-expansion-panel-title>
                          <v-expansion-panel-text class="px-0 pt-2">
                            <div class="text-caption text-medium-emphasis mb-3">
                              {{ t('regionalPresets.advanced.exceptionsHelp') }}
                            </div>
                            
                            <div class="d-flex align-start">
                              <v-text-field
                                v-model="exceptionInputs[r.key]"
                                density="compact"
                                :error-messages="exceptionErrors[r.key]"
                                hide-details="auto"
                                :label="t('regionalPresets.advanced.exceptions')"
                                variant="outlined"
                                @keydown.enter="handleAddException(r.key)"
                              />
                              <v-btn variant="tonal" height="40" class="ml-2" @click="handleAddException(r.key)">
                                {{ t('regionalPresets.advanced.addDomain') }}
                              </v-btn>
                            </div>

                            <div v-if="r.state.exceptions.length === 0" class="text-caption text-medium-emphasis mt-3 text-center style-italic">
                              {{ t('regionalPresets.advanced.noExceptions') }}
                            </div>
                            <div v-else class="d-flex flex-wrap gap-2 mt-3">
                              <v-chip
                                v-for="(item, index) in r.state.exceptions"
                                :key="item"
                                closable
                                size="small"
                                variant="tonal"
                                class="mr-2 mb-2"
                                @click:close="removeException(r.key, index)"
                              >
                                {{ item }}
                              </v-chip>
                            </div>
                          </v-expansion-panel-text>
                        </v-expansion-panel>
                      </v-expansion-panels>
                    </div>
                  </div>
                </div>
              </v-expand-transition>
            </v-card>
          </section>

          <div class="regional-preset-drawer__manual-link">
            <span>{{ t('regionalPresets.needFullControl') }}</span>
            <span>{{ t('regionalPresets.editRulesManually') }}</span>
          </div>
        </template>

        <template v-else-if="step === 'preview'">
          <v-alert density="compact" type="info" variant="tonal" class="mb-4">
            {{ t('regionalPresets.previewGroups.securityNote') }}
          </v-alert>

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
          <v-btn variant="text" @click="step = 'selection'">{{ t('regionalPresets.back') }}</v-btn>
          <v-btn color="primary" variant="flat" @click="applySelectedPresets">
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
import {
  applyPresets,
  computePreview,
  detectPresetState,
  isPresetManagedItem,
  type PresetDirection,
  type PresetPreviewGroup,
  type PresetRegion,
  type PresetRegionKey,
  type RegionalPresetState,
  validatePresetCatalogShape,
} from './routingDnsPresets'

const props = defineProps<{
  modelValue: boolean
  config: Config
  outboundTags: string[]
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  apply: [config: Config]
}>()

const { t } = useI18n()
const drawerWidth = 520
const step = ref<'selection' | 'preview' | 'success' | 'error'>('selection')
const proxyOutbound = ref('')
const directOutbound = ref('direct')
const errorMessage = ref('')

const ruState = reactive<RegionalPresetState>({ region: 'RU', enabled: false, direction: 'direct', exceptions: [] })
const zhState = reactive<RegionalPresetState>({ region: 'ZH', enabled: false, direction: 'direct', exceptions: [] })
const exceptionErrors = reactive<Record<PresetRegionKey, string>>({ ru: '', zh: '' })
const exceptionInputs = reactive<Record<PresetRegionKey, string>>({ ru: '', zh: '' })

const visible = computed({
  get: () => props.modelValue,
  set: value => emit('update:modelValue', value),
})

const outboundItems = computed(() => {
  const tags = new Set(['direct', ...props.outboundTags.filter(Boolean)])
  return [...tags].map(tag => ({ title: tag, value: tag }))
})

const hasOutbounds = computed(() => proxyOutbound.value.length > 0 && directOutbound.value.length > 0)
const sameOutbound = computed(() => hasOutbounds.value && proxyOutbound.value === directOutbound.value)
const hasEnabledRegion = computed(() => ruState.enabled || zhState.enabled)

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
  return hasOutbounds.value && (hasEnabledRegion.value || hasChanges.value)
})

const preview = computed(() => {
  if (!hasOutbounds.value) {
    return {
      ru: emptyPreviewGroup(),
      zh: emptyPreviewGroup(),
    }
  }
  return computePreview(props.config, ruState, zhState, {
    proxyOutbound: proxyOutbound.value,
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
  if (open) resetFromConfig()
})

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
  proxyOutbound.value = props.outboundTags.find(tag => tag && tag !== 'direct') ?? ''
  directOutbound.value = outboundItems.value.some(item => item.value === 'direct') ? 'direct' : (props.outboundTags[0] ?? '')
  exceptionErrors.ru = ''
  exceptionErrors.zh = ''
  exceptionInputs.ru = ''
  exceptionInputs.zh = ''
  errorMessage.value = ''
  step.value = 'selection'
}

const closeDrawer = () => {
  visible.value = false
}

const isValidDomain = (value: string) => {
  const domain = value.trim().replace(/^\.+|\.+$/g, '').toLowerCase()
  if (!domain || domain.includes('/') || domain.includes('*') || domain.includes(' ')) return false
  return /^[a-z0-9-]+(\.[a-z0-9-]+)+$/i.test(domain)
}

const handleAddException = (region: PresetRegionKey) => {
  const value = exceptionInputs[region] || ''
  const normalized = value.trim().replace(/^\.+|\.+$/g, '').toLowerCase()
  exceptionErrors[region] = ''
  if (!isValidDomain(normalized)) {
    exceptionErrors[region] = t('regionalPresets.advanced.invalidDomain')
    return
  }
  const target = region === 'ru' ? ruState : zhState
  if (!target.exceptions.includes(normalized)) {
    target.exceptions.push(normalized)
  }
  exceptionInputs[region] = ''
}

const removeException = (region: PresetRegionKey, index: number) => {
  const target = region === 'ru' ? ruState : zhState
  target.exceptions.splice(index, 1)
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

const applySelectedPresets = () => {
  try {
    const result = applyPresets(props.config, ruState, zhState, {
      proxyOutbound: proxyOutbound.value,
      directOutbound: directOutbound.value,
    })
    emit('apply', result.config)
    step.value = 'success'
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : t('regionalPresets.result.regionalDataUnavailable')
    step.value = 'error'
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
