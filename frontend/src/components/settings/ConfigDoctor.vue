<template>
  <section class="settings-config-doctor">
    <div class="settings-config-doctor__header">
      <div class="settings-config-doctor__heading">
        <v-icon color="primary" icon="lucide:activity" />
        <div>
          <h3>{{ $t('doctor.title') }}</h3>
          <p>{{ report?.summary ?? $t('doctor.idle') }}</p>
        </div>
      </div>
      <div class="settings-config-doctor__actions">
        <v-chip :color="statusColor" density="compact" label>
          {{ statusLabel }}
        </v-chip>
        <v-btn
          color="primary"
          prepend-icon="lucide:activity"
          :loading="loading"
          variant="tonal"
          @click="runDoctor"
        >
          {{ $t('doctor.run') }}
        </v-btn>
      </div>
    </div>

    <v-progress-linear
      v-if="loading && report"
      color="primary"
      indeterminate
      rounded
    />

    <v-skeleton-loader
      v-if="loading && !report"
      class="settings-config-doctor__skeleton"
      type="list-item-three-line, list-item-three-line, list-item-three-line"
    />

    <v-alert
      v-else-if="errorMessage"
      density="compact"
      type="warning"
      variant="tonal"
    >
      {{ errorMessage }}
    </v-alert>

    <div
      v-else-if="report?.items?.length"
      class="settings-config-doctor__items"
    >
      <section
        v-for="item in report.items"
        :key="item.id"
        class="settings-config-doctor__item"
      >
        <v-chip
          class="settings-config-doctor__severity"
          :color="severityColor(item.severity)"
          density="compact"
          label
        >
          {{ severityLabel(item.severity) }}
        </v-chip>
        <div class="settings-config-doctor__copy">
          <strong>{{ item.title }}</strong>
          <span>{{ item.message }}</span>
          <small v-if="item.action">{{ item.action }}</small>
          <ul
            v-if="detailLines(item.details).length"
            class="settings-config-doctor__details"
          >
            <li
              v-for="(detail, index) in detailLines(item.details)"
              :key="`${item.id}-${index}`"
            >
              {{ detail }}
            </li>
          </ul>
        </div>
      </section>
    </div>

    <v-empty-state
      v-else
      icon="lucide:activity"
      :text="$t('doctor.noReport')"
      :title="$t('doctor.notRun')"
    />
  </section>
</template>

<script lang="ts" setup>
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'

import HttpUtils from '@/plugins/httputil'
import type { DoctorReport, DoctorSeverity } from '@/types/doctor'

const { t } = useI18n()

const loading = ref(false)
const report = ref<DoctorReport>()
const errorMessage = ref('')

const severityColor = (severity: DoctorSeverity) => {
  if (severity === 'error') return 'error'
  if (severity === 'warn') return 'warning'
  return 'success'
}

const severityLabel = (severity: DoctorSeverity) => {
  if (severity === 'error') return t('doctor.error')
  if (severity === 'warn') return t('doctor.warn')
  return t('doctor.ok')
}

const statusColor = computed(() => report.value ? severityColor(report.value.status) : undefined)
const statusLabel = computed(() => report.value ? severityLabel(report.value.status) : t('doctor.notRun'))

const formatDetail = (detail: unknown): string => {
  if (typeof detail === 'string') return detail
  if (typeof detail === 'number' || typeof detail === 'boolean') return String(detail)
  try {
    return JSON.stringify(detail)
  } catch {
    return String(detail)
  }
}

const detailLines = (details: unknown): string[] => {
  if (details == null) return []
  if (Array.isArray(details)) return details.map(formatDetail)
  if (typeof details === 'object') {
    return Object.entries(details as Record<string, unknown>)
      .map(([key, value]) => `${key}: ${formatDetail(value)}`)
  }
  return [formatDetail(details)]
}

const runDoctor = async () => {
  loading.value = true
  errorMessage.value = ''
  try {
    const msg = await HttpUtils.post('api/doctor/run', {})
    if (msg.success) {
      report.value = msg.obj as DoctorReport
    } else {
      errorMessage.value = msg.msg || t('doctor.noReport')
    }
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.settings-config-doctor {
  border: 1px solid rgba(var(--v-theme-on-surface), 0.12);
  border-radius: 8px;
  display: grid;
  gap: 14px;
  min-width: 0;
  padding: 16px;
}

.settings-config-doctor__header {
  align-items: flex-start;
  display: flex;
  gap: 16px;
  justify-content: space-between;
  min-width: 0;
}

.settings-config-doctor__heading {
  align-items: flex-start;
  display: flex;
  gap: 12px;
  min-width: 0;
}

.settings-config-doctor__heading h3,
.settings-config-doctor__heading p {
  letter-spacing: 0;
  margin: 0;
  min-width: 0;
  overflow-wrap: anywhere;
}

.settings-config-doctor__heading h3 {
  font-size: 1rem;
  font-weight: 600;
  line-height: 1.4;
}

.settings-config-doctor__heading p {
  color: rgba(var(--v-theme-on-surface), 0.72);
  font-size: 0.875rem;
  line-height: 1.4;
  margin-top: 2px;
}

.settings-config-doctor__actions {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  justify-content: flex-end;
}

.settings-config-doctor__items {
  display: grid;
  gap: 10px;
  min-width: 0;
}

.settings-config-doctor__item {
  align-items: flex-start;
  background: rgba(var(--v-theme-surface-variant), 0.18);
  border: 1px solid rgba(var(--v-theme-on-surface), 0.1);
  border-radius: 8px;
  display: grid;
  gap: 12px;
  grid-template-columns: auto minmax(0, 1fr);
  min-width: 0;
  padding: 12px;
}

.settings-config-doctor__severity {
  min-width: 66px;
}

.settings-config-doctor__copy {
  display: grid;
  gap: 4px;
  min-width: 0;
}

.settings-config-doctor__copy strong,
.settings-config-doctor__copy span,
.settings-config-doctor__copy small,
.settings-config-doctor__details {
  letter-spacing: 0;
  min-width: 0;
  overflow-wrap: anywhere;
}

.settings-config-doctor__copy strong {
  font-size: 0.92rem;
  line-height: 1.35;
}

.settings-config-doctor__copy span {
  color: rgba(var(--v-theme-on-surface), 0.78);
  font-size: 0.86rem;
  line-height: 1.4;
}

.settings-config-doctor__copy small {
  color: rgb(var(--v-theme-warning));
  font-size: 0.8rem;
  line-height: 1.4;
}

.settings-config-doctor__details {
  color: rgba(var(--v-theme-on-surface), 0.74);
  font-family: ui-monospace, SFMono-Regular, Consolas, "Liberation Mono", monospace;
  font-size: 0.78rem;
  line-height: 1.45;
  margin: 4px 0 0;
  padding-inline-start: 18px;
}

.settings-config-doctor__skeleton {
  background: transparent;
}

@media (max-width: 600px) {
  .settings-config-doctor {
    padding: 12px;
  }

  .settings-config-doctor__header,
  .settings-config-doctor__actions {
    align-items: stretch;
    flex-direction: column;
  }

  .settings-config-doctor__actions {
    justify-content: flex-start;
  }

  .settings-config-doctor__item {
    grid-template-columns: minmax(0, 1fr);
  }

  .settings-config-doctor__severity {
    min-width: 0;
    width: fit-content;
  }
}
</style>
