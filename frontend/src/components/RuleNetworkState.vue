<template>
  <v-row>
    <v-col cols="12" sm="6" md="4">
      <v-select
        v-model="networkTypes"
        :items="interfaceTypes"
        :label="$t('rule.networkType')"
        multiple
        chips
        hide-details
      />
    </v-col>
    <v-col cols="12" sm="6" md="4">
      <v-switch v-model="rule.network_is_expensive" color="primary" :label="$t('rule.networkExpensive')" hide-details />
    </v-col>
    <v-col cols="12" sm="6" md="4">
      <v-switch v-model="rule.network_is_constrained" color="primary" :label="$t('rule.networkConstrained')" hide-details />
    </v-col>
    <v-col cols="12" sm="6">
      <v-textarea
        v-model="wifiSSID"
        :label="$t('rule.wifiSsid')"
        rows="2"
        no-resize
        hide-details
      />
    </v-col>
    <v-col cols="12" sm="6">
      <v-textarea
        v-model="wifiBSSID"
        :label="$t('rule.wifiBssid')"
        rows="2"
        no-resize
        hide-details
      />
    </v-col>
  </v-row>
</template>

<script lang="ts">
const splitList = (value: string): string[] => value
  .split(/[\n,]/)
  .map((item) => item.trim())
  .filter((item) => item.length > 0)

export default {
  props: ['rule'],
  data() {
    return {
      interfaceTypes: ['wifi', 'cellular', 'ethernet', 'other'],
    }
  },
  computed: {
    networkTypes: {
      get(): string[] { return this.$props.rule.network_type ?? [] },
      set(value: string[]) { this.$props.rule.network_type = value.length > 0 ? value : [] },
    },
    wifiSSID: {
      get(): string { return this.$props.rule.wifi_ssid?.join('\n') ?? '' },
      set(value: string) { this.$props.rule.wifi_ssid = splitList(value) },
    },
    wifiBSSID: {
      get(): string { return this.$props.rule.wifi_bssid?.join('\n') ?? '' },
      set(value: string) { this.$props.rule.wifi_bssid = splitList(value) },
    },
  },
}
</script>
