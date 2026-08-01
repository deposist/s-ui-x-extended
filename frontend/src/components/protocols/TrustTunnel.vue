<template>
  <v-card subtitle="TrustTunnel">
    <v-row v-if="direction == 'out'">
      <v-col cols="12" sm="6">
        <v-text-field hide-details :label="$t('types.un')" v-model="data.username"></v-text-field>
      </v-col>
      <v-col cols="12" sm="6">
        <v-text-field hide-details :label="$t('types.pw')" v-model="data.password"></v-text-field>
      </v-col>
    </v-row>
    <v-row>
      <v-col cols="12" sm="6" md="4">
        <v-select
          multiple
          chips
          hide-details
          :label="$t('network')"
          :items="['tcp', 'udp']"
          v-model="data.network">
          <template #append-inner>
            <FieldHint :field-hints="fieldHints" field="network" />
          </template>
        </v-select>
      </v-col>
      <v-col cols="6" sm="3" md="2">
        <v-switch color="primary" hide-details label="QUIC" v-model="data.quic"></v-switch>
      </v-col>
      <v-col cols="6" sm="3" md="3" v-if="direction == 'out'">
        <v-switch color="primary" hide-details :label="$t('types.trusttunnel.healthCheck')" v-model="data.health_check"></v-switch>
      </v-col>
    </v-row>
    <v-row v-if="data.quic">
      <v-col cols="12" sm="6" md="4">
        <v-select
          clearable
          hide-details
          :label="$t('types.trusttunnel.congestion')"
          :items="congestionControllers"
          v-model="data.congestion_controller">
          <template #append-inner>
            <FieldHint :field-hints="fieldHints" field="congestion_controller" />
          </template>
        </v-select>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-select
          clearable
          hide-details
          :label="$t('types.trusttunnel.bbrProfile')"
          :items="['standard', 'conservative', 'aggressive']"
          v-model="data.bbr_profile">
        </v-select>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-text-field type="number" hide-details label="CWND" v-model.number="data.cwnd"></v-text-field>
      </v-col>
    </v-row>

    <v-card v-if="direction == 'out'" border density="compact" color="background" style="margin-top: 8px;">
      <v-card-subtitle style="padding-top: 8px;">{{ $t('singbox.multiplex') }}</v-card-subtitle>
      <v-card-text>
        <v-row>
          <v-col cols="12" sm="3">
            <v-switch color="primary" hide-details :label="$t('enable')" v-model="mux.enabled"></v-switch>
          </v-col>
          <v-col cols="12" sm="3">
            <v-text-field type="number" hide-details :label="$t('singbox.maxConnections')" v-model.number="mux.max_connections"></v-text-field>
          </v-col>
          <v-col cols="12" sm="3">
            <v-text-field type="number" hide-details :label="$t('singbox.minStreams')" v-model.number="mux.min_streams"></v-text-field>
          </v-col>
          <v-col cols="12" sm="3">
            <v-text-field type="number" hide-details :label="$t('singbox.maxStreams')" v-model.number="mux.max_streams"></v-text-field>
          </v-col>
        </v-row>
      </v-card-text>
    </v-card>
    <InboundAdvanced :data="data" :field-hints="fieldHints" />
  </v-card>
</template>

<script lang="ts">
import InboundAdvanced from '@/components/protocols/InboundAdvanced.vue'
import FieldHint from '@/components/FieldHint.vue'

export default {
  props: {
    direction: { type: String },
    data: { type: Object, required: true },
    fieldHints: { type: Object, default: () => ({}) },
  },
  components: {InboundAdvanced, FieldHint},
  data() {
    return {
      congestionControllers: ['bbr', 'bbr_standard', 'bbr2', 'bbr2_variant', 'cubic', 'reno'],
    }
  },
  created() {
    if (!this.$props.data.multiplex) this.$props.data.multiplex = {}
  },
  computed: {
    mux(): any {
      return this.$props.data.multiplex
    },
  },
}
</script>
