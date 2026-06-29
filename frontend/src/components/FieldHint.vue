<template>
  <SettingInfo v-if="text" :text="text" />
</template>

<script lang="ts">
import SettingInfo from '@/components/SettingInfo.vue'

export default {
  props: {
    fieldHints: { type: Object, default: () => ({}) },
    field: { type: String, required: true },
  },
  computed: {
    text(): string {
      const hintKey = (this.$props.fieldHints as Record<string, string>)[this.$props.field]
      if (!hintKey) return ''
      const translated = this.$t(hintKey)
      return translated === hintKey ? '' : translated
    },
  },
  components: { SettingInfo },
}
</script>
