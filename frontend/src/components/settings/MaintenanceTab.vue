<template>
  <div v-if="nexus">
    <v-row class="settings-grid align-stretch">
      <v-col cols="12" md="6" class="d-flex flex-column">
        <v-card variant="outlined" class="settings-section-card d-flex flex-column flex-grow-1">
          <ConfigDoctor />
        </v-card>
      </v-col>

      <v-col cols="12" md="6" class="d-flex flex-column">
        <v-card variant="outlined" class="settings-section-card d-flex flex-column flex-grow-1">
          <PanelUpdateCard />
        </v-card>
      </v-col>
    </v-row>

    <v-row density="comfortable" class="mt-4">
      <v-col cols="12">
        <v-btn
          variant="tonal"
          prepend-icon="mdi-backup-restore"
          @click="backupModal.visible = true"
          color="primary"
        >
          {{ $t('main.backup.title') }}
        </v-btn>
      </v-col>
    </v-row>
  </div>

  <div v-else>
    <ConfigDoctor class="mb-4" />

    <PanelUpdateCard class="mb-4" />

    <v-row density="comfortable">
      <v-col cols="12" sm="6" md="4">
        <v-btn
          block
          variant="tonal"
          prepend-icon="mdi-backup-restore"
          @click="backupModal.visible = true"
        >
          {{ $t('main.backup.title') }}
        </v-btn>
      </v-col>
    </v-row>
  </div>

  <Backup
    v-model="backupModal.visible"
    :control="backupModal"
    :visible="backupModal.visible"
  />
</template>

<script setup lang="ts">
import ConfigDoctor from '@/components/settings/ConfigDoctor.vue'
import PanelUpdateCard from '@/components/settings/PanelUpdateCard.vue'
import Backup from '@/layouts/modals/Backup.vue'
import { ref, computed } from 'vue'
import { useUiMode } from '@/uiMode/useUiMode'

interface ModalControl {
  visible: boolean
}

const { mode } = useUiMode()
const nexus = computed(() => mode.value === 'nexus')
const backupModal = ref<ModalControl>({ visible: false })
</script>
