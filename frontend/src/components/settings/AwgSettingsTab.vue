<template>
  <v-alert
    class="mb-4"
    :type="statusOk ? 'success' : 'warning'"
    variant="tonal"
    density="comfortable"
  >
    {{ $t('setting.awg.status.summary', statusArgs) }}
  </v-alert>

  <v-card variant="outlined" class="settings-section-card">
    <v-card-title class="settings-section-title">{{ $t('setting.awg.title') }}</v-card-title>
    <v-card-text class="pa-4 pt-2">
      <div class="text-body-2 text-medium-emphasis mb-4">{{ $t('setting.awg.hint') }}</div>
      <v-btn variant="tonal" prepend-icon="lucide:rotate-cw" :loading="loading" @click="loadStatus">
        {{ $t('actions.refresh') }}
      </v-btn>
    </v-card-text>
  </v-card>
</template>

<script setup lang="ts">
import HttpUtils from '@/plugins/httputil'
import { i18n } from '@/locales'
import { computed, onMounted, ref } from 'vue'

const loading = ref(false)
const status = ref<any>(null)

const statusOk = computed(() =>
  (status.value?.managedEndpoints ?? 0) > 0 && !!status.value?.coreReachable && !!status.value?.encryptionKeyAvailable)

const statusArgs = computed(() => ({
  servers: status.value?.managedEndpoints ?? 0,
  core: status.value?.coreReachable
    ? i18n.global.t('setting.awg.status.coreOnline')
    : i18n.global.t('setting.awg.status.coreOffline'),
  desired: status.value?.desired ?? 0,
  provisioned: status.value?.provisioned ?? 0,
  pending: status.value?.pending ?? 0,
  errors: status.value?.errors ?? 0,
}))

const loadStatus = async () => {
  loading.value = true
  try {
    const msg = await HttpUtils.get('api/awg/status')
    if (msg.success) status.value = msg.obj
  } finally {
    loading.value = false
  }
}

onMounted(loadStatus)
</script>
