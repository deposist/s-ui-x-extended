<template>
  <v-alert type="info" variant="tonal" class="mb-4">{{ $t('collectionForms.namespaceHint') }}</v-alert>
  <v-select :model-value="data.type || 'default'" :items="['default', 'unshare']" :label="$t('type')" @update:model-value="changeType" />
  <CollectionFields :data="data" :fields="data.type === 'unshare' ? [{ key: 'pid_file' }] : [{ key: 'path', required: true }]" />
</template>
<script setup lang="ts">
import CollectionFields from './CollectionFields.vue'
const props = defineProps<{ data: Record<string, any> }>()
const changeType = (type: string) => {
  props.data.type = type
  delete props.data[type === 'unshare' ? 'path' : 'pid_file']
}
</script>
