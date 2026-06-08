<template>
  <v-card subtitle="MTProxy">
    <v-row>
      <v-col cols="12" sm="6" md="4">
        <v-select
          hide-details
          :label="$t('types.mtproxy.preferIp')"
          :items="['prefer-ipv4', 'prefer-ipv6', 'only-ipv4', 'only-ipv6']"
          v-model="data.prefer_ip">
        </v-select>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-text-field type="number" hide-details :label="$t('types.mtproxy.concurrency')" v-model.number="data.concurrency"></v-text-field>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-switch color="primary" hide-details :label="$t('types.mtproxy.autoUpdate')" v-model="data.auto_update"></v-switch>
      </v-col>
    </v-row>

    <v-card border density="compact" color="background" style="margin-top: 8px;">
      <v-card-subtitle style="padding-top: 8px;">{{ $t('types.mtproxy.domainFronting') }}</v-card-subtitle>
      <v-card-text>
        <v-row>
          <v-col cols="12" sm="4">
            <v-text-field type="number" hide-details :label="$t('out.port')" v-model.number="data.domain_fronting_port"></v-text-field>
          </v-col>
          <v-col cols="12" sm="5">
            <v-text-field hide-details :label="$t('types.mtproxy.frontingHost')" v-model="data.domain_fronting_host"></v-text-field>
          </v-col>
          <v-col cols="12" sm="3">
            <v-switch color="primary" hide-details :label="$t('types.mtproxy.proxyProtocol')" v-model="data.domain_fronting_proxy_protocol"></v-switch>
          </v-col>
        </v-row>
      </v-card-text>
    </v-card>

    <v-row style="margin-top: 4px;">
      <v-col cols="6" sm="4">
        <v-text-field hide-details :label="$t('types.mtproxy.idleTimeout')" v-model="data.idle_timeout"></v-text-field>
      </v-col>
      <v-col cols="6" sm="4">
        <v-text-field hide-details :label="$t('types.mtproxy.handshakeTimeout')" v-model="data.handshake_timeout"></v-text-field>
      </v-col>
      <v-col cols="6" sm="4">
        <v-text-field hide-details :label="$t('types.mtproxy.timeSkew')" v-model="data.tolerate_time_skewness"></v-text-field>
      </v-col>
      <v-col cols="6" sm="4">
        <v-text-field type="number" hide-details :label="$t('types.mtproxy.throttleMax')" v-model.number="data.throttle_max_connections"></v-text-field>
      </v-col>
      <v-col cols="6" sm="4">
        <v-text-field hide-details :label="$t('types.mtproxy.throttleInterval')" v-model="data.throttle_check_interval"></v-text-field>
      </v-col>
      <v-col cols="6" sm="4">
        <v-switch color="primary" hide-details :label="$t('types.mtproxy.fallbackUnknownDc')" v-model="data.allow_fallback_on_unknown_dc"></v-switch>
      </v-col>
    </v-row>

    <v-card border density="compact" color="background" style="margin-top: 8px;">
      <v-card-subtitle style="padding-top: 8px;">{{ $t('types.mtproxy.doppelganger') }}</v-card-subtitle>
      <v-card-text>
        <v-row>
          <v-col cols="12">
            <v-combobox chips multiple clearable hide-details :label="$t('types.mtproxy.doppelUrls')"
              :model-value="data.doppelganger_urls" @update:model-value="setUrls"></v-combobox>
          </v-col>
          <v-col cols="6" sm="4">
            <v-text-field type="number" hide-details :label="$t('types.mtproxy.doppelPerRaid')" v-model.number="data.doppelganger_per_raid"></v-text-field>
          </v-col>
          <v-col cols="6" sm="4">
            <v-text-field hide-details :label="$t('types.mtproxy.doppelEach')" v-model="data.doppelganger_each"></v-text-field>
          </v-col>
          <v-col cols="6" sm="4">
            <v-switch color="primary" hide-details :label="$t('types.mtproxy.drs')" v-model="data.doppelganger_drs"></v-switch>
          </v-col>
        </v-row>
      </v-card-text>
    </v-card>
  </v-card>
</template>

<script lang="ts">
export default {
  props: { data: { type: Object, required: true } },
  methods: {
    setUrls(v: string[]) {
      if (v && v.length > 0) {
        this.$props.data.doppelganger_urls = v
      } else {
        delete this.$props.data.doppelganger_urls
      }
    },
  },
}
</script>
