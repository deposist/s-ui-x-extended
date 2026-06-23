<template>
  <!-- Nexus: right-side drawer; Classic: centered dialog. One shell so every
       entity form is presented identically per mode without duplicating bodies. -->
  <entity-drawer
    v-if="mode === 'nexus'"
    :dirty="dirty"
    :loading="loading"
    :model-value="modelValue"
    :save-disabled="saveDisabled"
    :saving="loading"
    :title="title"
    :width="width"
    @close="emit('close')"
    @save="emit('save')"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <slot />
  </entity-drawer>

  <v-dialog
    v-else
    :model-value="modelValue"
    transition="dialog-bottom-transition"
    :width="dialogWidth"
    @update:model-value="onClassicModel"
  >
    <v-card class="rounded-lg" :loading="loading">
      <v-card-title>{{ title }}</v-card-title>
      <v-divider />
      <v-card-text style="padding: 0 16px; overflow-y: auto;">
        <slot />
      </v-card-text>
      <v-card-actions>
        <v-spacer />
        <v-btn color="primary" variant="outlined" @click="requestClose">
          {{ $t('actions.close') }}
        </v-btn>
        <v-btn color="primary" variant="tonal" :disabled="saveDisabled" :loading="loading" @click="emit('save')">
          {{ $t('actions.save') }}
        </v-btn>
      </v-card-actions>

      <!-- Self-contained unsaved-changes guard for Classic mode. The Nexus
           branch delegates to EntityDrawer (which uses the shell's ConfirmHost);
           ConfirmHost is not mounted in Classic, so this dialog confirms here.
           v-dialog teleports to <body>, so nesting it keeps a single root node
           (preserving the host modal's v-model attribute fallthrough). -->
      <v-dialog v-model="discardConfirm" max-width="420">
        <v-card class="rounded-xl">
          <v-card-title>{{ $t('form.leaveTitle') }}</v-card-title>
          <v-card-text>{{ $t('form.leaveConfirm') }}</v-card-text>
          <v-card-actions>
            <v-spacer />
            <v-btn variant="text" @click="discardConfirm = false">{{ $t('actions.close') }}</v-btn>
            <v-btn color="error" variant="flat" @click="confirmDiscard">{{ $t('form.discard') }}</v-btn>
          </v-card-actions>
        </v-card>
      </v-dialog>
    </v-card>
  </v-dialog>
</template>

<script lang="ts" setup>
import { ref } from 'vue'

import { useUiMode } from '@/uiMode/useUiMode'
import EntityDrawer from './EntityDrawer.vue'

const props = withDefaults(defineProps<{
  // Optional like v-dialog's: the host modal's v-model falls through to here,
  // so callers need not bind it explicitly (and vue-tsc won't demand it).
  modelValue?: boolean
  title: string
  width?: number
  dialogWidth?: number | string
  loading?: boolean
  dirty?: boolean
  saveDisabled?: boolean
}>(), {
  modelValue: false,
  width: 720,
  dialogWidth: 800,
})

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  save: []
  close: []
}>()

const { mode } = useUiMode()

// Classic-mode unsaved-changes guard (Nexus is handled by EntityDrawer).
const discardConfirm = ref(false)

const doClose = () => {
  emit('update:modelValue', false)
  emit('close')
}

const requestClose = () => {
  if (props.dirty) discardConfirm.value = true
  else doClose()
}

const confirmDiscard = () => {
  discardConfirm.value = false
  doClose()
}

// The classic v-dialog model only ever closes (false) via scrim/Esc; route that
// through the dirty guard, but never block opening.
const onClassicModel = (value: boolean) => {
  if (value) {
    emit('update:modelValue', true)
    return
  }
  requestClose()
}
</script>
