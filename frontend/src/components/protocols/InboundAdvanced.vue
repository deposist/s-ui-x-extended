<template>
  <v-row>
    <v-col cols="12" class="v-card-subtitle">{{ $t('inboundAdvanced.title') }}</v-col>
    <v-col cols="12" sm="6" md="4">
      <v-switch color="primary" hide-details :label="$t('inboundAdvanced.sniff')" v-model="data.sniff"></v-switch>
    </v-col>
    <v-col cols="12" sm="6" md="4">
      <v-switch color="primary" hide-details :label="$t('inboundAdvanced.sniffOverrideDestination')" v-model="data.sniff_override_destination"></v-switch>
    </v-col>
    <v-col cols="12" sm="6" md="4">
      <v-text-field hide-details :label="$t('inboundAdvanced.sniffTimeout')" v-model="data.sniff_timeout" placeholder="300ms"></v-text-field>
    </v-col>
    <v-col cols="12" sm="6" md="4">
      <v-switch color="primary" hide-details :label="$t('inboundAdvanced.proxyProtocol')" v-model="data.proxy_protocol"></v-switch>
    </v-col>
    <v-col cols="12" sm="6" md="4" v-if="data.proxy_protocol">
      <v-switch color="primary" hide-details :label="$t('inboundAdvanced.proxyProtocolAcceptNoHeader')" v-model="data.proxy_protocol_accept_no_header"></v-switch>
    </v-col>
    <v-col cols="12" sm="6" md="4">
      <v-select hide-details :items="domainStrategies" :label="$t('inboundAdvanced.domainStrategy')" v-model="data.domain_strategy" clearable></v-select>
    </v-col>
    <v-col cols="12" sm="6" md="4">
      <v-switch color="primary" hide-details :label="$t('inboundAdvanced.udpDisableDomainUnmapping')" v-model="data.udp_disable_domain_unmapping"></v-switch>
    </v-col>
  </v-row>
</template>

<script lang="ts">
export default {
  props: ['data'],
  data() {
    return {
      domainStrategies: ['', 'prefer_ipv4', 'prefer_ipv6', 'ipv4_only', 'ipv6_only'],
    }
  },
}
</script>
