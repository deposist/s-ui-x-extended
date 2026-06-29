<template>
  <v-select
    hide-details
    :label="$t('network')"
    :items="networks"
    v-model="Network">
    <template #append-inner>
      <SettingInfo v-if="hint" :text="hint" />
    </template>
  </v-select>
</template>

<script lang="ts">
import SettingInfo from '@/components/SettingInfo.vue'

export default {
  props: {
    data: { type: Object, required: true },
    hint: { type: String, default: '' },
  },
  data() {
    return {
      networks: [
        { title: "TCP/UDP", value: '' },
        { title: "TCP", value: 'tcp' },
        { title: "UDP", value: 'udp' },
      ],
    }
  },
  computed: {
    Network: {
      get():string { return this.$props.data.network?? '' },
      set(v:string) { this.$props.data.network = v != '' ? v : undefined }
    }
  },
  components: { SettingInfo }
}
</script>