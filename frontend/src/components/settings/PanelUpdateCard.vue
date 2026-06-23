<template>
  <section class="panel-update">
    <div class="panel-update__header">
      <div class="panel-update__heading">
        <v-icon color="primary" icon="mdi-cloud-download-outline" />
        <div>
          <h3>{{ $t('update.title') }}</h3>
          <p>{{ $t('update.subtitle') }}</p>
        </div>
      </div>
    </div>

    <div class="panel-update__row">
      <v-btn-toggle
        v-model="channel"
        color="primary"
        density="comfortable"
        divided
        mandatory
        variant="outlined"
      >
        <v-btn value="main">{{ $t('update.channelMain') }}</v-btn>
        <v-btn value="beta">{{ $t('update.channelBeta') }}</v-btn>
      </v-btn-toggle>

      <v-btn
        color="primary"
        prepend-icon="mdi-refresh"
        :loading="checking"
        variant="tonal"
        @click="checkUpdates"
      >
        {{ $t('update.check') }}
      </v-btn>
    </div>

    <div class="panel-update__versions">
      <span>{{ $t('update.current') }}: <strong>{{ status?.current || 'N/A' }}</strong></span>
      <v-icon icon="mdi-arrow-right" size="small" />
      <span>
        {{ $t('update.available') }}:
        <strong>{{ status?.latest || 'N/A' }}</strong>
      </span>
      <v-chip
        v-if="status?.latest"
        :color="status?.prerelease ? 'warning' : 'success'"
        density="compact"
        label
        size="small"
      >
        {{ status?.prerelease ? $t('update.beta') : $t('update.stable') }}
      </v-chip>
    </div>

    <v-alert
      v-if="status?.checkError"
      density="compact"
      type="warning"
      variant="tonal"
    >
      {{ $t('update.checkFailed') }}
    </v-alert>

    <v-alert
      v-else-if="status && !status.updateAvailable && status.latest"
      density="compact"
      type="success"
      variant="tonal"
    >
      {{ $t('update.upToDate') }}
    </v-alert>

    <!-- Release notes are external content: render a small Markdown subset without v-html. -->
    <v-sheet
      v-if="status?.updateAvailable && status?.releaseNotes"
      class="panel-update__notes"
      rounded
    >
      <div class="panel-update__notes-title">{{ $t('update.releaseNotes') }}</div>
      <div class="panel-update__notes-body">
        <template v-for="(block, index) in releaseNoteBlocks" :key="index">
          <component
            :is="headingTag(block.level)"
            v-if="block.type === 'heading'"
            class="panel-update__notes-heading"
          >
            <template v-for="(segment, segmentIndex) in block.inline" :key="segmentIndex">
              <code v-if="segment.type === 'code'">{{ segment.text }}</code>
              <strong v-else-if="segment.type === 'strong'">{{ segment.text }}</strong>
              <span v-else>{{ segment.text }}</span>
            </template>
          </component>
          <p v-else-if="block.type === 'paragraph'" class="panel-update__notes-paragraph">
            <template v-for="(segment, segmentIndex) in block.inline" :key="segmentIndex">
              <code v-if="segment.type === 'code'">{{ segment.text }}</code>
              <strong v-else-if="segment.type === 'strong'">{{ segment.text }}</strong>
              <span v-else>{{ segment.text }}</span>
            </template>
          </p>
          <component
            :is="block.ordered ? 'ol' : 'ul'"
            v-else-if="block.type === 'list'"
            class="panel-update__notes-list"
          >
            <li v-for="(item, itemIndex) in block.items" :key="itemIndex">
              <template v-for="(segment, segmentIndex) in item" :key="segmentIndex">
                <code v-if="segment.type === 'code'">{{ segment.text }}</code>
                <strong v-else-if="segment.type === 'strong'">{{ segment.text }}</strong>
                <span v-else>{{ segment.text }}</span>
              </template>
            </li>
          </component>
          <pre v-else-if="block.type === 'code'" class="panel-update__notes-code"><code>{{ block.text }}</code></pre>
          <v-divider v-else-if="block.type === 'rule'" class="my-3" />
        </template>
      </div>
    </v-sheet>

    <div v-if="jobActive" class="panel-update__progress">
      <v-progress-linear color="primary" indeterminate rounded />
      <span>{{ $t('update.stage.' + (status?.job?.stage || 'idle')) }}</span>
    </div>

    <v-alert
      v-else-if="status?.job?.stage === 'failed'"
      density="compact"
      type="error"
      variant="tonal"
    >
      {{ $t('update.failed') }}<span v-if="status?.job?.error">: {{ status.job.error }}</span>
    </v-alert>

    <div class="panel-update__actions">
      <v-btn
        color="primary"
        :disabled="!canUpdate"
        prepend-icon="mdi-arrow-up-bold"
        @click="openConfirm"
      >
        {{ $t('update.update') }}
      </v-btn>
    </div>

    <v-dialog v-model="confirm" max-width="460">
      <v-card>
        <v-card-title>{{ $t('update.confirmTitle') }}</v-card-title>
        <v-card-text>
          <v-alert
            class="mb-3"
            density="compact"
            type="warning"
            variant="tonal"
          >
            {{ $t('update.restartWarning') }}
          </v-alert>
          <p class="mb-3">{{ $t('update.confirmTo', { version: status?.latest }) }}</p>
          <v-text-field
            v-model="password"
            autocomplete="current-password"
            density="comfortable"
            :label="$t('update.password')"
            type="password"
            variant="outlined"
          />
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn variant="text" @click="confirm = false">{{ $t('cancel') }}</v-btn>
          <v-btn
            color="primary"
            :disabled="!password"
            :loading="applying"
            @click="runUpdate"
          >
            {{ $t('update.update') }}
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </section>
</template>

<script lang="ts" setup>
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'

import HttpUtils from '@/plugins/httputil'
import { parseMarkdownBlocks } from '@/plugins/markdown'

interface UpdateJob {
  stage: string
  error?: string
}

interface UpdateStatus {
  current: string
  channel: string
  latest?: string
  prerelease?: boolean
  updateAvailable?: boolean
  assetAvailable?: boolean
  releaseNotes?: string
  checkError?: string
  job?: UpdateJob
}

const RUNNING_STAGES = ['downloading', 'verifying', 'applying', 'restarting']

const status = ref<UpdateStatus>()
const channel = ref<'main' | 'beta'>('main')
const checking = ref(false)
const applying = ref(false)
const confirm = ref(false)
const password = ref('')
let suppressChannelCheck = false
let pollTimer: ReturnType<typeof setInterval> | undefined

const jobActive = computed(() => RUNNING_STAGES.includes(status.value?.job?.stage || ''))
const canUpdate = computed(() =>
  !!status.value?.updateAvailable && !!status.value?.assetAvailable && !jobActive.value && !applying.value,
)
const releaseNoteBlocks = computed(() => parseMarkdownBlocks(status.value?.releaseNotes || ''))
const headingTag = (level = 3) => `h${Math.min(Math.max(level + 2, 4), 6)}`

const applyStatus = (obj: unknown) => {
  status.value = obj as UpdateStatus
  const incoming = status.value?.channel
  // Only arm the watch-suppression when the incoming channel actually differs
  // from the current selection. Arming it on an identical value (e.g. the common
  // 'main' === 'main' case) leaves the flag stuck. Vue does not fire the watch
  // for a no-op assignment, which would silently swallow the user's first real
  // channel toggle (no auto-check, channel not persisted).
  if ((incoming === 'main' || incoming === 'beta') && incoming !== channel.value) {
    suppressChannelCheck = true
    channel.value = incoming
  }
}

const loadStatus = async () => {
  const msg = await HttpUtils.get('api/update/status')
  if (msg.success) applyStatus(msg.obj)
}

const checkUpdates = async () => {
  checking.value = true
  try {
    const msg = await HttpUtils.post('api/update/check', { channel: channel.value })
    if (msg.success) applyStatus(msg.obj)
  } finally {
    checking.value = false
  }
}

const openConfirm = () => {
  password.value = ''
  confirm.value = true
}

const runUpdate = async () => {
  applying.value = true
  try {
    const msg = await HttpUtils.post('api/update/apply', {
      channel: channel.value,
      targetVersion: status.value?.latest ?? '',
      password: password.value,
    })
    password.value = ''
    if (msg.success) {
      confirm.value = false
      applyStatus(msg.obj)
      startPolling()
    }
  } finally {
    applying.value = false
  }
}

const startPolling = () => {
  stopPolling()
  pollTimer = setInterval(async () => {
    // While the panel restarts into the new binary, requests fail. That is the
    // expected end state; keep polling so the UI recovers once it returns.
    await loadStatus()
    if (!jobActive.value) stopPolling()
  }, 2000)
}

const stopPolling = () => {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = undefined
  }
}

watch(channel, () => {
  if (suppressChannelCheck) {
    suppressChannelCheck = false
    return
  }
  checkUpdates()
})

onMounted(loadStatus)
onUnmounted(stopPolling)
</script>

<style scoped>
.panel-update {
  border: 1px solid rgba(var(--v-theme-on-surface), 0.12);
  border-radius: 8px;
  display: grid;
  gap: 14px;
  min-width: 0;
  padding: 16px;
}

.panel-update__heading {
  align-items: flex-start;
  display: flex;
  gap: 12px;
  min-width: 0;
}

.panel-update__heading h3 {
  font-size: 1rem;
  font-weight: 600;
  margin: 0;
}

.panel-update__heading p {
  color: rgba(var(--v-theme-on-surface), 0.72);
  font-size: 0.875rem;
  margin: 2px 0 0;
}

.panel-update__row {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  justify-content: space-between;
}

.panel-update__versions {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  font-size: 0.9rem;
}

.panel-update__notes {
  background: rgba(var(--v-theme-surface-variant), 0.18);
  max-height: 220px;
  overflow: auto;
  padding: 12px;
}

.panel-update__notes-title {
  font-size: 0.8rem;
  font-weight: 600;
  margin-bottom: 6px;
  opacity: 0.8;
}

.panel-update__notes-body {
  font-size: 0.84rem;
  line-height: 1.5;
  word-break: break-word;
}

.panel-update__notes-heading {
  font-size: 0.92rem;
  font-weight: 650;
  margin: 10px 0 4px;
}

.panel-update__notes-heading:first-child,
.panel-update__notes-paragraph:first-child,
.panel-update__notes-list:first-child,
.panel-update__notes-code:first-child {
  margin-top: 0;
}

.panel-update__notes-paragraph {
  margin: 0 0 8px;
}

.panel-update__notes-list {
  margin: 0 0 8px 18px;
  padding: 0;
}

.panel-update__notes-code {
  background: rgba(var(--v-theme-on-surface), 0.08);
  border-radius: 6px;
  margin: 0 0 8px;
  overflow: auto;
  padding: 8px;
}

.panel-update__notes-body code {
  font-family: ui-monospace, SFMono-Regular, Consolas, monospace;
  font-size: 0.82em;
}

.panel-update__progress {
  align-items: center;
  display: grid;
  gap: 6px;
}

.panel-update__actions {
  display: flex;
  justify-content: flex-end;
}
</style>
