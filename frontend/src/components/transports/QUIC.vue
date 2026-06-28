<template>
  <v-alert
    class="mb-4"
    density="comfortable"
    type="warning"
    variant="tonal"
  >
    QUIC transport is deprecated. Prefer TUIC or Hysteria2 unless you need legacy v2rayquic compatibility.
  </v-alert>

  <v-row>
    <v-col cols="12" sm="6" md="4">
      <v-text-field
        label="Security"
        hide-details
        v-model="quicTransport.security"
      />
    </v-col>
    <v-col cols="12" sm="6" md="4">
      <v-text-field
        label="Key"
        hide-details
        v-model="quicTransport.key"
      />
    </v-col>
    <v-col cols="12" sm="6" md="4">
      <v-text-field
        label="Header type"
        hide-details
        v-model="headerType"
      />
    </v-col>
  </v-row>
</template>

<script lang="ts">
import type { QUIC } from '../../types/transport'

export default {
  props: ['transport'],
  computed: {
    quicTransport(): QUIC {
      const transport = this.$props.transport as QUIC

      if (!transport.header) {
        transport.header = { type: '' }
      }

      return transport
    },
    headerType: {
      get(): string {
        return this.quicTransport.header?.type ?? ''
      },
      set(value: string) {
        this.quicTransport.header = { type: value }
      },
    },
  },
}
</script>
