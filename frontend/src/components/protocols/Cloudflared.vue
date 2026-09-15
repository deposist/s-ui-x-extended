<template>
  <v-card subtitle="Cloudflared">
    <v-row>
      <v-col cols="12">
        <v-text-field
          :label="$t('types.cloudflared.token')"
          hide-details
          v-model="data.token">
          <template #append-inner>
            <FieldHint :field-hints="fieldHints" field="token" />
          </template>
        </v-text-field>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-select
          :label="$t('types.cloudflared.protocol')"
          hide-details
          clearable
          :items="protocols"
          @click:clear="delete data.protocol"
          v-model="data.protocol">
        </v-select>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-text-field
          :label="$t('types.cloudflared.haConnections')"
          hide-details
          type="number"
          min="0"
          clearable
          @click:clear="delete data.ha_connections"
          v-model.number="data.ha_connections">
        </v-text-field>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-text-field
          :label="$t('types.cloudflared.gracePeriod')"
          hide-details
          placeholder="30s"
          clearable
          @click:clear="delete data.grace_period"
          v-model="data.grace_period">
        </v-text-field>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-select
          :label="$t('types.cloudflared.edgeIPVersion')"
          hide-details
          clearable
          :items="edgeIPVersions"
          @click:clear="delete data.edge_ip_version"
          v-model="data.edge_ip_version">
        </v-select>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-select
          :label="$t('types.cloudflared.datagramVersion')"
          hide-details
          clearable
          :items="datagramVersions"
          @click:clear="delete data.datagram_version"
          v-model="data.datagram_version">
        </v-select>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-text-field
          :label="$t('types.cloudflared.region')"
          hide-details
          clearable
          @click:clear="delete data.region"
          v-model="data.region">
        </v-text-field>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-switch
          color="primary"
          :label="$t('types.cloudflared.postQuantum')"
          hide-details
          v-model="data.post_quantum">
        </v-switch>
      </v-col>
    </v-row>
    <v-row>
      <v-col cols="12" sm="6">
        <v-switch color="primary" :label="$t('types.cloudflared.controlDialer')" hide-details v-model="optionControlDialer"></v-switch>
      </v-col>
      <v-col cols="12" sm="6">
        <v-switch color="primary" :label="$t('types.cloudflared.tunnelDialer')" hide-details v-model="optionTunnelDialer"></v-switch>
      </v-col>
    </v-row>
    <Dial v-if="data.control_dialer != undefined" :dial="data.control_dialer" :field-hints="fieldHints" />
    <Dial v-if="data.tunnel_dialer != undefined" :dial="data.tunnel_dialer" :field-hints="fieldHints" />
  </v-card>
</template>

<script lang="ts">
import { Cloudflared } from '@/types/inbounds'
import Dial from '@/components/Dial.vue'
import FieldHint from '@/components/FieldHint.vue'

export default {
  props: {
    data: { type: Object, required: true },
    fieldHints: { type: Object, default: () => ({}) },
  },
  data() {
    return {
      protocols: ['auto', 'quic', 'http2', 'h2mux'],
      edgeIPVersions: [
        { title: 'auto', value: 0 },
        { title: 'IPv4', value: 4 },
        { title: 'IPv6', value: 6 },
      ],
      datagramVersions: ['v2', 'v3'],
    }
  },
  computed: {
    typed: function(): Cloudflared { return <Cloudflared>this.$props.data },
    // The dialer blocks are optional in the core options, so they are created on
    // demand instead of always being written into the saved config.
    optionControlDialer: {
      get(): boolean { return this.data.control_dialer != undefined },
      set(v: boolean) { this.$props.data.control_dialer = v ? {} : undefined; if (!v) delete this.$props.data.control_dialer },
    },
    optionTunnelDialer: {
      get(): boolean { return this.data.tunnel_dialer != undefined },
      set(v: boolean) { this.$props.data.tunnel_dialer = v ? {} : undefined; if (!v) delete this.$props.data.tunnel_dialer },
    },
  },
  components: { Dial, FieldHint },
}
</script>