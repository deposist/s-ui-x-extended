<template>
  <ProviderVue
    v-model="modal.visible"
    :visible="modal.visible"
    :id="modal.id"
    :data="modal.data"
    :tags="detourTags"
    @close="closeModal"
  />
  <v-row justify="center" align="center">
    <v-col cols="auto">
      <v-btn color="primary" @click="showModal(0)">{{ $t('actions.add') }}</v-btn>
    </v-col>
  </v-row>
  <v-row>
    <v-col cols="12" sm="4" md="3" lg="2" v-for="(item, index) in <any[]>providers" :key="item.tag">
      <v-card rounded="xl" elevation="5" min-width="200" :title="item.tag">
        <v-card-subtitle style="margin-top: -15px;">
          <v-row>
            <v-col>{{ item.type }}</v-col>
          </v-row>
        </v-card-subtitle>
        <v-card-text>
          <v-row v-if="item.type == 'remote'">
            <v-col>{{ $t('types.provider.url') }}</v-col>
            <v-col class="text-truncate">{{ item.url ?? '-' }}</v-col>
          </v-row>
          <v-row v-if="item.type == 'local'">
            <v-col>{{ $t('types.provider.path') }}</v-col>
            <v-col class="text-truncate">{{ item.path ?? '-' }}</v-col>
          </v-row>
          <v-row v-if="item.type == 'inline'">
            <v-col>{{ $t('types.provider.outbounds') }}</v-col>
            <v-col>{{ item.outbounds?.length ?? 0 }}</v-col>
          </v-row>
          <v-row>
            <v-col>{{ $t('types.provider.healthCheck') }}</v-col>
            <v-col>{{ $t(item.health_check?.enabled ? 'enable' : 'disable') }}</v-col>
          </v-row>
        </v-card-text>
        <v-divider></v-divider>
        <v-card-actions style="padding: 0;">
          <v-btn icon="mdi-file-edit" @click="showModal(item.id)">
            <v-icon />
            <v-tooltip activator="parent" location="top" :text="$t('actions.edit')"></v-tooltip>
          </v-btn>
          <v-btn icon="mdi-file-remove" style="margin-inline-start:0;" color="warning" @click="delOverlay[index] = true">
            <v-icon />
            <v-tooltip activator="parent" location="top" :text="$t('actions.del')"></v-tooltip>
          </v-btn>
          <v-overlay
            v-model="delOverlay[index]"
            contained
            class="align-center justify-center"
          >
            <v-card :title="$t('actions.del')" rounded="lg">
              <v-divider></v-divider>
              <v-card-text>{{ $t('confirm') }}</v-card-text>
              <v-card-actions>
                <v-btn color="error" variant="outlined" @click="delProvider(item.tag)">{{ $t('yes') }}</v-btn>
                <v-btn color="success" variant="outlined" @click="delOverlay[index] = false">{{ $t('no') }}</v-btn>
              </v-card-actions>
            </v-card>
          </v-overlay>
        </v-card-actions>
      </v-card>
    </v-col>
  </v-row>
</template>

<script lang="ts" setup>
import Data from '@/store/modules/data'
import ProviderVue from '@/layouts/modals/Provider.vue'
import { Provider } from '@/types/providers'
import { computed, ref } from 'vue'

const providers = computed((): Provider[] => {
  return <Provider[]> Data().providers
})

const detourTags = computed((): string[] => {
  return [...Data().outbounds?.map((o: any) => o.tag), ...Data().endpoints?.map((e: any) => e.tag)]
})

const modal = ref({
  visible: false,
  id: 0,
  data: "",
})

let delOverlay = ref(new Array<boolean>)

const showModal = (id: number) => {
  modal.value.id = id
  modal.value.data = id == 0 ? '' : JSON.stringify(providers.value.findLast(p => p.id == id))
  modal.value.visible = true
}

const closeModal = () => {
  modal.value.visible = false
}

const delProvider = async (tag: string) => {
  const index = providers.value.findIndex(i => i.tag == tag)
  const success = await Data().save("providers", "del", tag)
  if (success) delOverlay.value[index] = false
}
</script>
