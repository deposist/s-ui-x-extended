<template>
  <v-card subtitle="VMESS">
    <template v-if="direction === 'out'">
      <v-row>
        <v-col cols="12" sm="6">
          <v-text-field v-model="data.uuid" label="UUID" hide-details></v-text-field>
        </v-col>
        <v-col cols="12" sm="6" md="4">
          <v-text-field
            label="Alter ID"
            hide-details
            type="number"
            min=0
            v-model.number="data.alter_id">
          </v-text-field>
        </v-col>
      </v-row>
      <v-row>
        <v-col cols="12" sm="6" md="4">
          <v-select
            hide-details
            :label="$t('types.vmess.security')"
            :items="securities"
            v-model="data.security">
          </v-select>
        </v-col>
        <v-col cols="12" sm="6" md="4">
          <v-select
            hide-details
            :label="$t('types.vless.udpEnc')"
            :items="['none','packetaddr','xudp']"
            v-model="packet_encoding">
          </v-select>
        </v-col>
        <v-col cols="12" sm="6" md="4">
          <Network :data="data" />
        </v-col>
        <v-col cols="12" sm="6" md="4">
          <v-switch v-model="data.global_padding" color="primary" :label="$t('types.vmess.globalPadding')" hide-details></v-switch>
        </v-col>
        <v-col cols="12" sm="6" md="4">
          <v-switch v-model="data.authenticated_length" color="primary" :label="$t('types.vmess.authLen')" hide-details></v-switch>
        </v-col>
      </v-row>
      <InboundAdvanced :data="data" />
    </template>
    <template v-else>
      <v-alert type="info" variant="tonal" density="compact">
        VMess inbound: users managed through client system. Configure TLS/Transport/Multiplex below.
      </v-alert>
    </template>
  </v-card>
</template>

<script lang="ts">
import Network from '@/components/Network.vue'
import InboundAdvanced from '@/components/protocols/InboundAdvanced.vue'

export default {
  props: {
    data: { type: Object, required: true },
    direction: { type: String, default: 'out' },
  },
  data() {
    return {
      securities: [
        "auto",
        "none",
        "zero",
        "aes-128-gcm",
        "aes-128-ctr",
        "chacha20-poly1305",
      ]
    }
  },
  computed: {
    packet_encoding: {
      get() { return this.$props.data.packet_encoding != undefined ? this.$props.data.packet_encoding : 'none' },
      set(newValue:string) { this.$props.data.packet_encoding = newValue != "none" ? newValue : undefined }
    },
  },
  components: {Network, InboundAdvanced}
}
</script>