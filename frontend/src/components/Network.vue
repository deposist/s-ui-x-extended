<template>
  <v-select
    hide-details
    :label="$t('network')"
    :items="networks"
    v-model="Network">
    <template #append-inner>
      <SettingInfo v-if="resolvedHint" :text="resolvedHint" />
    </template>
  </v-select>
</template>

<script lang="ts">
import SettingInfo from '@/components/SettingInfo.vue'

export default {
  props: {
    data: { type: Object, required: true },
    hint: { type: String, default: '' },
    fieldHints: { type: Object, default: () => ({}) },
    field: { type: String, default: 'network' },
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
    resolvedHint(): string {
      if (this.$props.hint) return this.$props.hint
      const hintKey = (this.$props.fieldHints as Record<string, string>)[this.$props.field]
      if (!hintKey) return ''
      const translated = this.$t(hintKey)
      return translated === hintKey ? '' : translated
    },
    Network: {
      get():string { return this.$props.data.network?? '' },
      set(v:string) { this.$props.data.network = v != '' ? v : undefined }
    }
  },
  components: { SettingInfo }
}
</script>