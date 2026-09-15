<template>
  <v-select :model-value="data.type" :items="['acme', 'tailscale', 'cloudflare-origin-ca']" :label="$t('type')" :rules="[value => ['acme', 'tailscale', 'cloudflare-origin-ca'].includes(value) || $t('collectionForms.required')]" @update:model-value="changeType" />
  <template v-if="data.type === 'tailscale'">
    <v-alert type="info" variant="tonal" class="mb-4">{{ $t('collectionForms.tailscaleHint') }}</v-alert>
    <v-combobox v-model="data.endpoint" :items="tailscaleTags" :label="$t('collectionForms.fields.endpoint')" :rules="[value => typeof value === 'string' && !!value.trim() || $t('collectionForms.required')]" />
  </template>
  <template v-else>
  <template v-if="data.type === 'acme'">
  <v-alert type="info" variant="tonal" class="mb-4">{{ $t('collectionForms.certificateHint') }}</v-alert>
  <CollectionFields :data="data" :fields="fields" />
  <v-switch :model-value="data.external_account != null" :label="$t('collectionForms.externalAccount')" color="primary" @update:model-value="toggle('external_account', $event)" />
  <CollectionFields v-if="data.external_account" :data="data.external_account" :fields="[{ key: 'key_id', required: true }, { key: 'mac_key', kind: 'secret', required: true }]" />
  <v-switch :model-value="data.dns01_challenge != null" :label="$t('collectionForms.dnsChallenge')" color="primary" @update:model-value="toggle('dns01_challenge', $event)" />
  <template v-if="data.dns01_challenge">
    <v-select :model-value="data.dns01_challenge.provider" :items="['alidns', 'cloudflare', 'acmedns']" :label="$t('collectionForms.fields.provider')" :rules="[value => !!value || $t('collectionForms.required')]" @update:model-value="changeDns" />
    <CollectionFields :data="data.dns01_challenge" :fields="dnsCommon" />
    <CollectionFields :data="data.dns01_challenge" :fields="dnsFields[data.dns01_challenge.provider] ?? []" />
  </template>
  </template>
  <template v-else-if="data.type === 'cloudflare-origin-ca'">
    <v-alert type="info" variant="tonal" class="mb-4">{{ $t('collectionForms.originHint') }}</v-alert>
    <CollectionFields :data="data" :fields="originFields" />
    <v-input :model-value="data.api_token || data.origin_ca_key" :rules="[value => !!value || $t('collectionForms.originCredentials')]" />
  </template>
  <v-select :model-value="httpMode" :items="[{ title: $t('collectionForms.defaultClient'), value: 'default' }, { title: $t('collectionForms.sharedClient'), value: 'shared' }, { title: $t('collectionForms.inlineClient'), value: 'inline' }]" :label="$t('basic.collections.httpClients')" @update:model-value="changeHttp" />
  <v-combobox v-if="httpMode === 'shared'" v-model="data.http_client" :items="httpTags" :label="$t('objects.tag')" :rules="[value => typeof value === 'string' && !!value.trim() || $t('collectionForms.required')]" />
  <HTTPClientForm v-if="httpMode === 'inline'" :data="data.http_client" />
  </template>
</template>
<script setup lang="ts">
import { computed } from 'vue'
import Data from '@/store/modules/data'
import CollectionFields, { type CollectionField } from './CollectionFields.vue'
import HTTPClientForm from './HTTPClientForm.vue'
const props = defineProps<{ data: Record<string, any> }>()
const fields: CollectionField[] = [
  { key: 'domain', kind: 'list', required: true }, { key: 'data_directory' }, { key: 'default_server_name' },
  { key: 'email' }, { key: 'provider' }, { key: 'account_key', kind: 'secret' },
  { key: 'disable_http_challenge', kind: 'boolean' }, { key: 'disable_tls_alpn_challenge', kind: 'boolean' },
  { key: 'alternative_http_port', kind: 'number', max: 65535 }, { key: 'alternative_tls_port', kind: 'number', max: 65535 },
  { key: 'key_type', items: ['ed25519', 'p256', 'p384', 'rsa2048', 'rsa4096'] }, { key: 'profile' },
]
const dnsCommon: CollectionField[] = [
  { key: 'ttl', kind: 'duration' }, { key: 'propagation_delay', kind: 'duration' }, { key: 'propagation_timeout', kind: 'duration' },
  { key: 'resolvers', kind: 'list' }, { key: 'override_domain' },
]
const dnsFields: Record<string, CollectionField[]> = {
  alidns: [{ key: 'access_key_id', required: true }, { key: 'access_key_secret', kind: 'secret', required: true }, { key: 'region_id' }, { key: 'security_token', kind: 'secret' }],
  cloudflare: [{ key: 'api_token', kind: 'secret', required: true }, { key: 'zone_token', kind: 'secret' }],
  acmedns: [{ key: 'username', required: true }, { key: 'password', kind: 'secret', required: true }, { key: 'subdomain', required: true }, { key: 'server_url', required: true }],
}
const originFields: CollectionField[] = [
  { key: 'domain', kind: 'list', required: true }, { key: 'data_directory' },
  { key: 'api_token', kind: 'secret' }, { key: 'origin_ca_key', kind: 'secret' },
  { key: 'request_type', items: ['origin-rsa', 'origin-ecc'] }, { key: 'requested_validity', items: [0, 7, 30, 90, 365, 730, 1095, 5475] },
]
const tailscaleTags = computed(() => (Data().endpoints ?? []).filter((endpoint: any) => endpoint.type === 'tailscale').map((endpoint: any) => endpoint.tag))
const changeType = (type: string) => {
  if (type === props.data.type) return
  const keep = new Set(type === 'acme' ? fields.map(field => field.key).concat(['external_account', 'dns01_challenge', 'http_client']) : type === 'tailscale' ? ['endpoint'] : originFields.map(field => field.key).concat(['http_client']))
  for (const key of [...fields.map(field => field.key), ...originFields.map(field => field.key), 'external_account', 'dns01_challenge', 'http_client', 'endpoint']) if (!keep.has(key)) delete props.data[key]
  props.data.type = type
}
const toggle = (key: string, enabled: boolean | null) => {
  if (enabled) props.data[key] ??= {}
  else delete props.data[key]
}
const changeDns = (provider: string) => {
  const dns = props.data.dns01_challenge
  for (const fields of Object.values(dnsFields)) for (const field of fields) delete dns[field.key]
  dns.provider = provider
}
const httpMode = computed(() => props.data.http_client == null ? 'default' : typeof props.data.http_client === 'string' ? 'shared' : 'inline')
const httpTags = computed(() => (Data().config?.http_clients ?? []).map((client: any) => client.tag).filter(Boolean))
const changeHttp = (mode: string) => {
  if (mode === httpMode.value) return
  if (mode === 'default') delete props.data.http_client
  else props.data.http_client = mode === 'shared' ? '' : {}
}
</script>
