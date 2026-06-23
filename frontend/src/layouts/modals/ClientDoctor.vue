<template>
  <v-dialog transition="dialog-bottom-transition" width="720">
    <v-card class="rounded-lg" :loading="loading">
      <v-card-title>
        <v-row>
          <v-col>{{ $t('doctor.clientTitle') }}</v-col>
          <v-spacer />
          <v-col cols="auto">
            <v-icon icon="mdi-close-box" @click="$emit('close')" />
          </v-col>
        </v-row>
      </v-card-title>
      <v-divider />
      <v-card-text>
        <v-row>
          <v-col cols="12" sm="8">
            <v-text-field
              v-model="target"
              clearable
              density="compact"
              hide-details
              :label="$t('doctor.testTarget')"
              variant="outlined"
            />
          </v-col>
          <v-col cols="12" sm="4">
            <v-btn
              block
              color="primary"
              prepend-icon="lucide:activity"
              :loading="loading"
              variant="tonal"
              @click="run"
            >
              {{ $t('doctor.run') }}
            </v-btn>
          </v-col>
        </v-row>

        <v-alert
          v-if="report"
          class="mt-4"
          density="compact"
          :type="alertType"
          variant="tonal"
        >
          {{ report.summary }}
        </v-alert>

        <v-skeleton-loader
          v-if="loading && !report"
          class="mt-4"
          type="list-item-three-line, list-item-three-line, list-item-three-line"
        />

        <v-list v-else-if="report?.items?.length" class="doctor-client__list" density="compact">
          <v-list-item v-for="item in report.items" :key="item.id" class="doctor-client__item">
            <template #prepend>
              <v-chip
                class="doctor-client__severity"
                :color="severityColor(item.severity)"
                label
                size="small"
                variant="flat"
              >
                {{ severityLabel(item.severity) }}
              </v-chip>
            </template>
            <v-list-item-title>{{ item.title }}</v-list-item-title>
            <v-list-item-subtitle>{{ item.message }}</v-list-item-subtitle>
            <v-list-item-subtitle v-if="item.action" class="doctor-client__action">
              {{ item.action }}
            </v-list-item-subtitle>
          </v-list-item>
        </v-list>

        <v-empty-state
          v-else
          icon="lucide:activity"
          :text="$t('doctor.noReport')"
          :title="$t('doctor.notRun')"
        />
      </v-card-text>
    </v-card>
  </v-dialog>
</template>

<script lang="ts" setup>
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'

import HttpUtils from '@/plugins/httputil'
import type { DoctorReport, DoctorSeverity } from '@/types/doctor'

const props = defineProps<{
  id: number
  visible: boolean
}>()

defineEmits<{
  close: []
}>()

const { t } = useI18n()
const loading = ref(false)
const report = ref<DoctorReport>()
const target = ref('https://www.gstatic.com/generate_204')

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

const alertType = computed(() => {
  if (report.value?.status === 'error') return 'error'
  if (report.value?.status === 'warn') return 'warning'
  return 'success'
})

const run = async () => {
  if (!props.id) return
  loading.value = true
  const msg = await HttpUtils.post('api/doctor/client', {
    data: JSON.stringify({ clientId: props.id, target: target.value }),
  })
  if (msg.success) {
    report.value = msg.obj as DoctorReport
  }
  loading.value = false
}

watch(() => props.visible, (visible) => {
  if (visible) {
    report.value = undefined
    void run()
  }
})
</script>

<style scoped>
.doctor-client__list {
  background: transparent;
  margin-top: 12px;
}

.doctor-client__item {
  border-bottom: 1px solid rgb(var(--v-theme-on-surface) / 10%);
}

.doctor-client__severity {
  min-width: 56px;
  justify-content: center;
}

.doctor-client__action {
  color: rgb(var(--v-theme-warning));
  opacity: 1;
}
</style>
