<template>
  <v-card subtitle="MASQUE">
    <v-row>
      <v-col cols="12" sm="6" md="4">
        <v-text-field hide-details :label="$t('types.masque.name')" v-model="data.name"></v-text-field>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-switch color="primary" hide-details :label="$t('types.masque.useHttp2')" v-model="data.use_http2"></v-switch>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-switch color="primary" hide-details :label="$t('types.masque.useIpv6')" v-model="data.use_ipv6"></v-switch>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-switch color="primary" hide-details :label="$t('types.wg.sysIf')" v-model="data.system"></v-switch>
      </v-col>
    </v-row>
    <v-row>
      <v-col cols="12">
        <v-combobox
          chips
          multiple
          clearable
          hide-details
          :label="$t('types.masque.allowedIps')"
          :model-value="data.allowed_ips"
          @update:model-value="setAllowedIps">
        </v-combobox>
      </v-col>
    </v-row>
    <v-row>
      <v-col cols="12" sm="6" md="4">
        <v-combobox hide-details :label="$t('types.masque.udpTimeout')" :items="durationPresets" v-model="data.udp_timeout"></v-combobox>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-combobox hide-details :label="$t('types.masque.udpKeepalive')" :items="durationPresets" v-model="data.udp_keepalive_period"></v-combobox>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-text-field type="number" hide-details :label="$t('types.masque.udpInitialPacketSize')" v-model.number="data.udp_initial_packet_size"></v-text-field>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-combobox hide-details :label="$t('types.masque.reconnectDelay')" :items="durationPresets" v-model="data.reconnect_delay"></v-combobox>
      </v-col>
    </v-row>

    <!-- Cloudflare profile -->
    <v-card border density="compact" color="background" style="margin-top: 8px;">
      <v-card-subtitle style="padding-top: 8px;">{{ $t('types.masque.profile') }}</v-card-subtitle>
      <v-card-text>
        <v-row>
          <v-col cols="12" sm="6">
            <v-text-field hide-details :label="$t('types.masque.profileId')" v-model="profile.id"></v-text-field>
          </v-col>
          <v-col cols="12" sm="6">
            <v-text-field hide-details :label="$t('types.masque.authToken')" v-model="profile.auth_token"></v-text-field>
          </v-col>
          <v-col cols="12" sm="6">
            <v-text-field hide-details :label="$t('types.masque.licenseKey')" v-model="profile.license_key"></v-text-field>
          </v-col>
          <v-col cols="12" sm="6">
            <v-text-field hide-details :label="$t('types.masque.privateKey')" v-model="profile.private_key"></v-text-field>
          </v-col>
          <v-col cols="12" sm="6">
            <v-text-field hide-details :label="$t('types.masque.detour')" v-model="profile.detour"></v-text-field>
          </v-col>
          <v-col cols="12" sm="6">
            <v-switch color="primary" hide-details :label="$t('types.masque.recreate')" v-model="profile.recreate"></v-switch>
          </v-col>
        </v-row>
      </v-card-text>
    </v-card>

    <!-- MASQUE TLS -->
    <v-card border density="compact" color="background" style="margin-top: 8px;">
      <v-card-subtitle style="padding-top: 8px;">{{ $t('objects.tls') }}</v-card-subtitle>
      <v-card-text>
        <v-row>
          <v-col cols="12" sm="6" md="4">
            <v-switch color="primary" hide-details :label="$t('tls.insecure')" v-model="tls.insecure"></v-switch>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-switch color="primary" hide-details :label="$t('tls.fragment')" v-model="tls.fragment"></v-switch>
          </v-col>
          <v-col cols="12" sm="6" md="4" v-if="tls.fragment">
            <v-switch color="primary" hide-details :label="$t('tls.recordFragment')" v-model="tls.record_fragment"></v-switch>
          </v-col>
          <v-col cols="12" sm="6" md="4" v-if="tls.fragment">
            <v-combobox hide-details :label="$t('tls.fragmentDelay')" :items="durationPresets" v-model="tls.fragment_fallback_delay"></v-combobox>
          </v-col>
        </v-row>
        <v-row>
          <v-col cols="12" md="8">
            <v-combobox chips multiple clearable hide-details :label="$t('tls.cs')"
              :model-value="tls.cipher_suites" @update:model-value="setCipherSuites"></v-combobox>
          </v-col>
        </v-row>
        <v-row>
          <v-col cols="12" md="8">
            <v-combobox chips multiple clearable hide-details :label="$t('tls.curves')" :items="tlsCurvePreferences"
              :model-value="tls.curve_preferences" @update:model-value="setCurvePreferences"></v-combobox>
          </v-col>
        </v-row>
        <v-row>
          <v-col cols="12" sm="6" md="4">
            <v-switch color="primary" hide-details :label="$t('tls.kernelTx')" v-model="tls.kernel_tx"></v-switch>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-switch color="primary" hide-details :label="$t('tls.kernelRx')" v-model="tls.kernel_rx"></v-switch>
          </v-col>
        </v-row>
      </v-card-text>
    </v-card>
  </v-card>
</template>

<script lang="ts">
import { durationPresets, tlsCurvePreferences } from '@/types/recommended'

export default {
  props: {
    data: { type: Object, required: true },
  },
  data() {
    return {
      durationPresets,
      tlsCurvePreferences,
    }
  },
  created() {
    if (!this.$props.data.profile) this.$props.data.profile = {}
    if (!this.$props.data.tls) this.$props.data.tls = {}
  },
  computed: {
    profile(): any {
      return this.$props.data.profile
    },
    tls(): any {
      return this.$props.data.tls
    },
  },
  methods: {
    setAllowedIps(v: string[]) {
      if (v && v.length > 0) {
        this.$props.data.allowed_ips = v
      } else {
        delete this.$props.data.allowed_ips
      }
    },
    setCipherSuites(v: string[]) {
      if (v && v.length > 0) {
        this.tls.cipher_suites = v
      } else {
        delete this.tls.cipher_suites
      }
    },
    setCurvePreferences(v: string[]) {
      if (v && v.length > 0) {
        this.tls.curve_preferences = v
      } else {
        delete this.tls.curve_preferences
      }
    },
  },
}
</script>
