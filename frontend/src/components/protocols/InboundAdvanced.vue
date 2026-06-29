<template>
  <v-row>
    <v-col cols="12" class="v-card-subtitle">{{ $t('inboundAdvanced.title') }}</v-col>
    <v-col cols="12" sm="6" md="4">
      <div class="d-flex align-center ga-1">
        <v-switch color="primary" hide-details :label="$t('inboundAdvanced.sniff')" v-model="data.sniff"></v-switch>
        <SettingInfo v-if="hint('sniff')" :text="hint('sniff')" />
      </div>
    </v-col>
    <v-col cols="12" sm="6" md="4">
      <div class="d-flex align-center ga-1">
        <v-switch color="primary" hide-details :label="$t('inboundAdvanced.sniffOverrideDestination')" v-model="data.sniff_override_destination"></v-switch>
        <SettingInfo v-if="hint('sniff_override_destination')" :text="hint('sniff_override_destination')" />
      </div>
    </v-col>
    <v-col cols="12" sm="6" md="4">
      <v-text-field hide-details :label="$t('inboundAdvanced.sniffTimeout')" v-model="data.sniff_timeout" placeholder="300ms">
        <template #append-inner>
          <SettingInfo v-if="hint('sniff_timeout')" :text="hint('sniff_timeout')" />
        </template>
      </v-text-field>
    </v-col>
    <v-col cols="12" sm="6" md="4">
      <div class="d-flex align-center ga-1">
        <v-switch color="primary" hide-details :label="$t('inboundAdvanced.proxyProtocol')" v-model="data.proxy_protocol"></v-switch>
        <SettingInfo v-if="hint('proxy_protocol')" :text="hint('proxy_protocol')" />
      </div>
    </v-col>
    <v-col cols="12" sm="6" md="4" v-if="data.proxy_protocol">
      <div class="d-flex align-center ga-1">
        <v-switch color="primary" hide-details :label="$t('inboundAdvanced.proxyProtocolAcceptNoHeader')" v-model="data.proxy_protocol_accept_no_header"></v-switch>
        <SettingInfo v-if="hint('proxy_protocol_accept_no_header')" :text="hint('proxy_protocol_accept_no_header')" />
      </div>
    </v-col>
    <v-col cols="12" sm="6" md="4">
      <v-select hide-details :items="domainStrategies" :label="$t('inboundAdvanced.domainStrategy')" v-model="data.domain_strategy" clearable>
        <template #append-inner>
          <SettingInfo v-if="hint('domain_strategy')" :text="hint('domain_strategy')" />
        </template>
      </v-select>
    </v-col>
    <v-col cols="12" sm="6" md="4">
      <div class="d-flex align-center ga-1">
        <v-switch color="primary" hide-details :label="$t('inboundAdvanced.udpDisableDomainUnmapping')" v-model="data.udp_disable_domain_unmapping"></v-switch>
        <SettingInfo v-if="hint('udp_disable_domain_unmapping')" :text="hint('udp_disable_domain_unmapping')" />
      </div>
    </v-col>
  </v-row>
</template>

<script lang="ts">
import SettingInfo from '@/components/SettingInfo.vue'

export default {
  props: {
    data: { type: Object, required: true },
    fieldHints: { type: Object, default: () => ({}) },
  },
  data() {
    return {
      domainStrategies: ['', 'prefer_ipv4', 'prefer_ipv6', 'ipv4_only', 'ipv6_only'],
    }
  },
  methods: {
    hint(key: string): string {
      const hintKey = (this.$props.fieldHints as Record<string, string>)[key]
      if (!hintKey) return ''
      const translated = this.$t(hintKey)
      return translated === hintKey ? '' : translated
    },
  },
  components: { SettingInfo },
}
</script>
