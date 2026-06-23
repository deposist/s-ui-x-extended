<template>
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
import { ref } from 'vue'

interface ModalControl {
  visible: boolean
}

const backupModal = ref<ModalControl>({ visible: false })
</script>
