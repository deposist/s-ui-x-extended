<template>
  <v-dialog v-model="modal.visible" max-width="1000" scrollable>
    <v-card :title="title">
      <v-card-text>
        <v-form v-if="modal.visible" ref="form" @submit.prevent="saveModal">
          <v-text-field v-model="modal.data.tag" :label="$t('objects.tag')" :rules="[tagRule]" autofocus />
          <CertificateProviderForm v-if="kind === 'certificate'" :data="modal.data" />
          <HTTPClientForm v-else-if="kind === 'http'" :data="modal.data" />
          <NetworkNamespaceForm v-else :data="modal.data" />
        </v-form>
      </v-card-text>
      <v-card-actions>
        <v-spacer />
        <v-btn @click="closeModal">{{ $t('actions.close') }}</v-btn>
        <v-btn color="primary" variant="tonal" @click="saveModal">{{ $t('actions.save') }}</v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
  <v-card-subtitle class="d-flex align-center">
    {{ title }}
    <v-spacer></v-spacer>
    <v-btn icon="mdi-plus" size="small" color="primary" :aria-label="$t('actions.add') + ': ' + title" @click="showModal(-1)" />
  </v-card-subtitle>
  <v-row v-if="items.length > 0">
    <v-col cols="12" sm="4" md="3" lg="2" v-for="(item, index) in items" :key="index">
      <v-card rounded="xl" elevation="5" min-width="200" :title="item.tag || `#${index + 1}`">
        <v-card-subtitle style="margin-top: -15px;">{{ item.type || defaultTypeLabel }}</v-card-subtitle>
        <v-divider></v-divider>
        <v-card-actions style="padding: 0;">
          <v-btn icon="mdi-file-edit" :aria-label="$t('actions.edit')" @click="showModal(index)">
            <v-icon />
            <v-tooltip activator="parent" location="top" :text="$t('actions.edit')"></v-tooltip>
          </v-btn>
          <v-btn icon="mdi-file-remove" :aria-label="$t('actions.del')" style="margin-inline-start:0;" color="warning" @click="delOverlay[index] = true">
            <v-icon />
            <v-tooltip activator="parent" location="top" :text="$t('actions.del')"></v-tooltip>
          </v-btn>
          <v-overlay v-model="delOverlay[index]" contained class="align-center justify-center">
            <v-card :title="$t('actions.del')" rounded="lg">
              <v-divider></v-divider>
              <v-card-text>{{ $t('confirm') }}</v-card-text>
              <v-card-actions>
                <v-btn color="error" variant="outlined" @click="delItem(index)">{{ $t('yes') }}</v-btn>
                <v-btn color="success" variant="outlined" @click="delOverlay[index] = false">{{ $t('no') }}</v-btn>
              </v-card-actions>
            </v-card>
          </v-overlay>
        </v-card-actions>
      </v-card>
    </v-col>
  </v-row>
  <v-alert v-else type="info" variant="tonal" density="compact">{{ $t('basic.collections.empty') }}</v-alert>
</template>

<script lang="ts" setup>
import { reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import CertificateProviderForm from './collections/CertificateProviderForm.vue'
import HTTPClientForm from './collections/HTTPClientForm.vue'
import NetworkNamespaceForm from './collections/NetworkNamespaceForm.vue'
const props = defineProps<{
  title: string
  kind: 'certificate' | 'http' | 'namespace'
  items: Record<string, any>[]
  // Template merged into a new item, e.g. { type: 'acme' } for cert providers.
  newItem: () => Record<string, any>
  defaultTypeLabel?: string
}>()

const emit = defineEmits<{
  (e: 'add', item: Record<string, any>): void
  (e: 'update', index: number, item: Record<string, any>): void
  (e: 'remove', index: number): void
}>()

const delOverlay = reactive<Record<number, boolean>>({})

const modal = reactive({
  visible: false,
  index: -1,
  data: {} as Record<string, any>,
})

const { t } = useI18n()
const form = ref<{ validate: () => Promise<{ valid: boolean }> }>()
const tagRule = (value: unknown) => {
  if (typeof value !== 'string' || !value.trim()) return t('collectionForms.required')
  return !props.items.some((item, index) => index !== modal.index && item.tag === value) || t('collectionForms.duplicateTag')
}

const showModal = (index: number) => {
  modal.index = index
  modal.data = JSON.parse(JSON.stringify(index === -1 ? props.newItem() : props.items[index]))
  modal.visible = true
}

const closeModal = () => {
  modal.visible = false
}

const saveModal = async () => {
  if (!(await form.value?.validate())?.valid) return
  const item = JSON.parse(JSON.stringify(modal.data))
  if (modal.index === -1) emit('add', item)
  else emit('update', modal.index, item)
  modal.visible = false
}

const delItem = (index: number) => {
  emit('remove', index)
  delOverlay[index] = false
}
</script>
