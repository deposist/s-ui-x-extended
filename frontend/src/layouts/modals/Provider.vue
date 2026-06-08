<template>
  <v-dialog transition="dialog-bottom-transition" width="800">
    <v-card class="rounded-lg">
      <v-card-title>
        {{ $t('actions.' + title) + " " + $t('objects.provider') }}
      </v-card-title>
      <v-divider></v-divider>
      <v-card-text style="padding: 0 16px; overflow-y: scroll;">
        <v-container style="padding: 0;">
          <v-row>
            <v-col cols="12" sm="6" md="4">
              <v-select
                hide-details
                :label="$t('type')"
                :items="Object.keys(providerTypes).map((key,index) => ({title: key, value: Object.values(providerTypes)[index]}))"
                v-model="provider.type"
                @update:modelValue="changeType">
              </v-select>
            </v-col>
            <v-col cols="12" sm="6" md="4">
              <v-text-field v-model="provider.tag" :label="$t('objects.tag')" hide-details></v-text-field>
            </v-col>
          </v-row>
          <Inline v-if="provider.type == providerTypes.Inline" :data="provider" />
          <Local v-if="provider.type == providerTypes.Local" :data="provider" />
          <Remote v-if="provider.type == providerTypes.Remote" :data="provider" :tags="tags" />
        </v-container>
      </v-card-text>
      <v-card-actions>
        <v-spacer></v-spacer>
        <v-btn
          color="primary"
          variant="outlined"
          @click="closeModal"
        >
          {{ $t('actions.close') }}
        </v-btn>
        <v-btn
          color="primary"
          variant="tonal"
          :loading="loading"
          :disabled="loading"
          @click="saveChanges"
        >
          {{ $t('actions.save') }}
        </v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script lang="ts">
import { ProviderTypes, createProvider } from '@/types/providers'
import RandomUtil from '@/plugins/randomUtil'
import Inline from '@/components/protocols/provider/Inline.vue'
import Local from '@/components/protocols/provider/Local.vue'
import Remote from '@/components/protocols/provider/Remote.vue'
import Data from '@/store/modules/data'
export default {
  props: ['visible', 'data', 'id', 'tags'],
  emits: ['close'],
  data() {
    return {
      provider: createProvider("remote", { "tag": "" }),
      title: "add",
      loading: false,
      providerTypes: ProviderTypes,
    }
  },
  methods: {
    updateData(id: number) {
      if (id > 0) {
        const newData = JSON.parse(this.$props.data)
        this.provider = createProvider(newData.type, newData)
        this.title = "edit"
      }
      else {
        this.provider = createProvider("remote", { tag: "provider-" + RandomUtil.randomSeq(3) })
        this.title = "add"
      }
    },
    changeType() {
      // Tag change only when adding a provider
      const tag = this.$props.id > 0 ? this.provider.tag : this.provider.type + "-" + RandomUtil.randomSeq(3)
      const prevConfig = { id: this.provider.id, tag: tag }
      this.provider = createProvider(this.provider.type, prevConfig)
    },
    closeModal() {
      this.updateData(0) // reset
      this.$emit('close')
    },
    async saveChanges() {
      // Guard against double-submit (button is also :disabled while loading).
      if (!this.$props.visible || this.loading) return
      // check duplicate tag
      const isDuplicatedTag = Data().checkTag("provider", this.$props.id, this.provider.tag)
      if (isDuplicatedTag) return

      // save data
      this.loading = true
      try {
        const success = await Data().save("providers", this.$props.id == 0 ? "new" : "edit", this.provider)
        if (success) this.closeModal()
      } finally {
        this.loading = false
      }
    },
  },
  watch: {
    visible(newValue) {
      if (newValue) {
        this.updateData(this.$props.id)
      }
    },
  },
  components: { Inline, Local, Remote }
}
</script>
