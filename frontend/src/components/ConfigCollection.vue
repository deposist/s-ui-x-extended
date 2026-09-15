<template>
  <Editor
    v-model="modal.visible"
    :visible="modal.visible"
    :data="modal.data"
    :title="editorTitle"
    @close="closeModal"
    @save="saveModal"
  />
  <v-card-subtitle class="d-flex align-center">
    {{ title }}
    <v-spacer></v-spacer>
    <v-chip color="primary" density="compact" variant="elevated" @click="showModal(-1)">
      <v-icon icon="mdi-plus" />
    </v-chip>
  </v-card-subtitle>
  <v-row v-if="items.length > 0">
    <v-col cols="12" sm="4" md="3" lg="2" v-for="(item, index) in items" :key="index">
      <v-card rounded="xl" elevation="5" min-width="200" :title="item.tag || `#${index + 1}`">
        <v-card-subtitle style="margin-top: -15px;">{{ item.type || defaultTypeLabel }}</v-card-subtitle>
        <v-divider></v-divider>
        <v-card-actions style="padding: 0;">
          <v-btn icon="mdi-file-edit" @click="showModal(index)">
            <v-icon />
            <v-tooltip activator="parent" location="top" :text="$t('actions.edit')"></v-tooltip>
          </v-btn>
          <v-btn icon="mdi-file-remove" style="margin-inline-start:0;" color="warning" @click="delOverlay[index] = true">
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
import { computed, reactive } from 'vue'
import Editor from '@/components/Editor.vue'

// Read-only view of a config-blob collection. All mutations are emitted as
// events so the parent (Basics.vue) materializes the array only on a real
// edit; a pure render never dirties the form. Item bodies stay open records so
// the JSON editor round-trips every core field without loss.
const props = defineProps<{
  title: string
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
  data: '',
})

const editorTitle = computed(() => props.title)

const showModal = (index: number) => {
  modal.index = index
  modal.data = index == -1 ? JSON.stringify(props.newItem(), null, 2) : JSON.stringify(props.items[index], null, 2)
  modal.visible = true
}

const closeModal = () => {
  modal.visible = false
}

const saveModal = (data: string) => {
  let parsed: unknown
  try {
    parsed = JSON.parse(data)
  } catch {
    return // Leave the modal open so the user can fix invalid JSON instead of losing it.
  }
  if (typeof parsed !== 'object' || parsed === null || Array.isArray(parsed)) return
  if (modal.index == -1) {
    emit('add', parsed as Record<string, any>)
  } else {
    emit('update', modal.index, parsed as Record<string, any>)
  }
  modal.visible = false
}

const delItem = (index: number) => {
  emit('remove', index)
  delOverlay[index] = false
}
</script>
