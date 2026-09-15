<template>
  <v-alert type="info" variant="tonal" class="mb-4">{{ $t('collectionForms.httpHint') }}</v-alert>
  <CollectionFields :data="data" :fields="[{ key: 'engine', items: ['go', 'apple'] }, { key: 'disable_version_fallback', kind: 'boolean' }]" />
  <v-select :model-value="data.version ?? 0" :items="[0, 1, 2, 3]" :label="$t('collectionForms.fields.version')" :hint="$t('collectionForms.versionHint')" persistent-hint @update:model-value="changeVersion" />
  <CollectionFields v-if="data.version !== 1" :data="data" :fields="httpFields" />
  <CollectionFields v-if="data.version === 3" :data="data" :fields="[{ key: 'initial_packet_size', kind: 'number', max: 65527 }, { key: 'disable_path_mtu_discovery', kind: 'boolean' }]" />
  <v-divider class="my-4" />
  <h3>{{ $t('objects.headers') }}</h3>
  <v-row v-for="(header, index) in headers" :key="index">
    <v-col cols="12" sm="5"><v-text-field v-model="header.name" :label="$t('objects.key')" :rules="[headerName]" @update:model-value="saveHeaders" /></v-col>
    <v-col cols="10" sm="6"><v-text-field v-model="header.value" :label="$t('objects.value')" :rules="[headerValue]" @update:model-value="saveHeaders" /></v-col>
    <v-col cols="2" sm="1"><v-btn icon="mdi-delete" :aria-label="$t('actions.del')" @click="headers.splice(index, 1); saveHeaders()" /></v-col>
  </v-row>
  <v-btn prepend-icon="mdi-plus" @click="headers.push({ name: '', value: '' })">{{ $t('actions.add') }}</v-btn>
  <v-divider class="my-4" />
  <v-switch :model-value="data.tls != null" :label="$t('objects.tls')" color="primary" @update:model-value="toggleTls" />
  <OutTLS v-if="data.tls != null" :outbound="data" />
  <Dial :dial="data" />
</template>
<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Dial from '@/components/Dial.vue'
import OutTLS from '@/components/tls/OutTLS.vue'
import CollectionFields, { type CollectionField } from './CollectionFields.vue'
const props = defineProps<{ data: Record<string, any> }>()
const { t } = useI18n()
const httpFields: CollectionField[] = [
  { key: 'idle_timeout', kind: 'duration' }, { key: 'keep_alive_period', kind: 'duration' },
  { key: 'stream_receive_window' }, { key: 'connection_receive_window' }, { key: 'max_concurrent_streams', kind: 'number' },
]
const changeVersion = (version: number) => {
  props.data.version = version
  if (version !== 3) { delete props.data.initial_packet_size; delete props.data.disable_path_mtu_discovery }
  if (version === 1) for (const field of httpFields) delete props.data[field.key]
}
const headers = ref<{ name: string; value: string }[]>(Object.entries(props.data.headers ?? {}).flatMap(([name, value]) => (Array.isArray(value) ? value : [value]).map(value => ({ name, value: String(value) }))))
const headerName = (value: string) => /^[!#$%&'*+.^_`|~\w-]+$/.test(value) || t('collectionForms.headerNameError')
const headerValue = (value: string) => !/[\r\n\0]/.test(value) || t('collectionForms.headerValueError')
const saveHeaders = () => {
  const result: Record<string, string[]> = Object.create(null)
  for (const header of headers.value) {
    if (!header.name) continue
    ;(result[header.name] ??= []).push(header.value)
  }
  if (Object.keys(result).length) props.data.headers = result
  else delete props.data.headers
}
const toggleTls = (enabled: boolean | null) => {
  if (enabled) props.data.tls ??= { enabled: true }
  else delete props.data.tls
}
</script>
