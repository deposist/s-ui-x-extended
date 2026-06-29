<template>
  <v-card :subtitle="$t('objects.tls')">
    <v-row>
      <v-col cols="12" sm="6" md="4">
        <v-select
          hide-details
          :label="$t('template')"
          :items="tlsItems"
          v-model="inbound.tls_id">
          <template #append-inner>
            <SettingInfo v-if="hint('tls_id')" :text="hint('tls_id')" />
          </template>
        </v-select>
      </v-col>
    </v-row>
  </v-card>
</template>

<script lang="ts">
import { i18n } from '@/locales'
import SettingInfo from '@/components/SettingInfo.vue'
import { tlsTemplateKind, type TlsTemplateKind } from '@/utils/tlsCompatibility'

export default {
  props: {
    inbound: { type: Object, required: true },
    tlsConfigs: { type: Array, default: () => [] },
    fieldHints: { type: Object, default: () => ({}) },
    allowedTemplateKinds: { type: Array, default: () => ['tls', 'reality'] },
  },
  methods: {
    hint(key: string): string {
      const hintKey = (this.$props.fieldHints as Record<string, string>)[key]
      if (!hintKey) return ''
      const translated = this.$t(hintKey)
      return translated === hintKey ? '' : translated
    },
    templateAllowed(tlsConfig: any): boolean {
      return (this.$props.allowedTemplateKinds as TlsTemplateKind[]).includes(tlsTemplateKind(tlsConfig))
    },
    itemForTemplate(tlsConfig: any, disabled = false): any {
      return {
        title: disabled ? `${tlsConfig.name} — ${this.$t('tls.incompatibleTemplate')}` : tlsConfig.name,
        value: tlsConfig.id,
        props: disabled ? { disabled: true } : undefined,
      }
    },
  },
  computed: {
    tlsItems(): any[] {
      const configs = (this.$props.tlsConfigs ?? []) as any[]
      const items = configs
        .filter((tlsConfig) => this.templateAllowed(tlsConfig))
        .map((tlsConfig) => this.itemForTemplate(tlsConfig))

      const selectedId = this.$props.inbound?.tls_id
      const selected = configs.find((tlsConfig) => tlsConfig.id === selectedId)
      if (selected && !this.templateAllowed(selected)) {
        items.push(this.itemForTemplate(selected, true))
      }

      return [ { title: i18n.global.t('none'), value: 0 }, ...items]
    }
  },
  components: { SettingInfo }
}
</script>
