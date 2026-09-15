<template>
  <v-row>
    <v-col v-for="field in fields" :key="field.key" cols="12" :sm="field.kind === 'lines' ? 12 : 6">
      <v-select v-if="field.items" :model-value="data[field.key]" :items="field.items" :label="label(field)" :hint="hint(field)" persistent-hint clearable :rules="rules(field)" @update:model-value="set(field, $event)" />
      <v-switch v-else-if="field.kind === 'boolean'" :model-value="data[field.key]" color="primary" :label="label(field)" :hint="hint(field)" persistent-hint @update:model-value="set(field, $event)" />
      <v-combobox v-else-if="field.kind === 'list'" :model-value="list(data[field.key])" multiple chips closable-chips clearable :label="label(field)" :hint="hint(field)" persistent-hint :rules="rules(field)" @update:model-value="set(field, $event)" />
      <v-textarea v-else-if="field.kind === 'lines'" :model-value="data[field.key]" rows="3" auto-grow :label="label(field)" :hint="hint(field)" persistent-hint :rules="rules(field)" @update:model-value="set(field, $event)" />
      <v-text-field v-else :model-value="data[field.key]" :type="field.kind === 'secret' ? 'password' : field.kind === 'number' ? 'number' : 'text'" :autocomplete="field.kind === 'secret' ? 'new-password' : 'off'" :label="label(field)" :hint="hint(field)" persistent-hint clearable :rules="rules(field)" @update:model-value="set(field, $event)" />
    </v-col>
  </v-row>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
export interface CollectionField {
  key: string
  kind?: 'boolean' | 'list' | 'number' | 'secret' | 'lines' | 'duration'
  items?: (string | number)[]
  required?: boolean
  max?: number
}
const props = defineProps<{ data: Record<string, any>; fields: CollectionField[] }>()
const { t } = useI18n()
const label = (field: CollectionField) => t(`collectionForms.fields.${field.key}`)
const hint = (field: CollectionField) => field.kind === 'duration' ? t('collectionForms.durationHint') : field.key
const list = (value: unknown) => value == null ? [] : Array.isArray(value) ? value : [value]
const set = (field: CollectionField, value: any) => {
  if (value == null || value === '' || (Array.isArray(value) && !value.length)) delete props.data[field.key]
  else props.data[field.key] = field.kind === 'number' ? Number(value) : value
}
const rules = (field: CollectionField) => [(value: any) => {
  const empty = value == null || value === '' || (Array.isArray(value) && !value.length)
  if (empty) return !field.required || t('collectionForms.required')
  if (typeof value === 'string' && !value.trim()) return t('collectionForms.required')
  if (field.kind === 'number' && (!Number.isInteger(Number(value)) || Number(value) < 0 || Number(value) > (field.max ?? Number.MAX_SAFE_INTEGER))) return t('collectionForms.numberRange', { max: field.max ?? Number.MAX_SAFE_INTEGER })
  if (field.kind === 'duration' && !/^(?:0|(?:\d+(?:\.\d+)?(?:ns|us|µs|μs|ms|s|m|h))+)$/u.test(value)) return t('collectionForms.durationHint')
  if (field.key === 'email' && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value)) return t('collectionForms.emailError')
  if (field.key === 'server_url') {
    try { if (!['http:', 'https:'].includes(new URL(value).protocol)) return t('collectionForms.urlError') } catch { return t('collectionForms.urlError') }
  }
  return true
}]
</script>
