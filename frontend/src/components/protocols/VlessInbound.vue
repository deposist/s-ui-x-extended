<template>
  <v-card subtitle="VLESS">
    <v-row v-if="mode === 'create'">
      <v-col cols="12" class="d-flex justify-end">
        <v-btn color="primary" variant="tonal" @click="applyRecommended">
          {{ $t('types.vless.recommendedPreset') }}
        </v-btn>
      </v-col>
    </v-row>
    <v-row>
      <v-col cols="12">
        <v-text-field
          v-model="data.decryption"
          :label="$t('types.vless.decryption')"
          :hint="hint('decryption')"
          :persistent-hint="Boolean(hint('decryption'))"
          clearable
          @click:clear="data.decryption = undefined">
          <template #append-inner>
            <SettingInfo v-if="hint('decryption')" :text="hint('decryption')" />
          </template>
        </v-text-field>
      </v-col>
    </v-row>
    <InboundAdvanced :data="data" :field-hints="fieldHints" />
  </v-card>
</template>

<script lang="ts">
import SettingInfo from '@/components/SettingInfo.vue'
import InboundAdvanced from '@/components/protocols/InboundAdvanced.vue'
import { applyVlessInboundRecommendedValues } from '@/utils/defaultRecommendations'

export default {
  props: {
    data: { type: Object, required: true },
    mode: { type: String, default: 'edit' },
    fieldHints: { type: Object, default: () => ({}) },
  },
  emits: ['apply-recommended'],
  methods: {
    hint(key: string): string {
      const hintKey = (this.$props.fieldHints as Record<string, string>)[key]
      if (!hintKey) return ''
      const translated = this.$t(hintKey)
      return translated === hintKey ? '' : translated
    },
    applyRecommended() {
      applyVlessInboundRecommendedValues(this.$props.data as any)
      this.$emit('apply-recommended')
    },
  },
  components: { SettingInfo, InboundAdvanced },
}
</script>
