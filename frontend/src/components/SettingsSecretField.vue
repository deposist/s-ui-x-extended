<template>
  <v-text-field
    v-if="showStoredPlaceholder"
    v-bind="$attrs"
    :model-value="''"
    :placeholder="STORED_SECRET_PLACEHOLDER"
    readonly
    autocomplete="off"
    persistent-placeholder
  >
    <template #append-inner>
      <v-btn
        icon="mdi-pencil"
        size="small"
        variant="text"
        :aria-label="$t('actions.edit')"
        @click.stop="startEditing"
      >
        <v-icon />
        <v-tooltip activator="parent" location="top" :text="$t('actions.edit')" />
      </v-btn>
    </template>
  </v-text-field>
  <v-text-field
    v-else
    v-bind="$attrs"
    :model-value="modelValue ?? ''"
    type="password"
    autocomplete="new-password"
    clearable
    @update:model-value="updateValue"
  />
</template>

<script lang="ts" setup>
import { computed, ref, watch } from 'vue'
import { STORED_SECRET_PLACEHOLDER, hasStoredSecret } from '@/components/settingsSecretField'

defineOptions({ inheritAttrs: false })

const props = defineProps<{
  modelValue?: string | null
  hasSecret?: string | boolean | null
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const editing = ref(false)

const showStoredPlaceholder = computed(() => {
  return hasStoredSecret(props.hasSecret) && !editing.value && (!(props.modelValue ?? '') || props.modelValue === STORED_SECRET_PLACEHOLDER)
})

const startEditing = () => {
  editing.value = true
  emit('update:modelValue', '')
}

const updateValue = (value: string | null) => {
  emit('update:modelValue', value ?? '')
}

watch(() => props.hasSecret, () => {
  editing.value = false
})

watch(() => props.modelValue, (value, previous) => {
  if (hasStoredSecret(props.hasSecret) && !(value ?? '') && (previous ?? '')) {
    editing.value = false
  }
})
</script>
