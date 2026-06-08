<template>
  <v-row>
    <v-col cols="12" sm="6" md="4">
      <v-select
        :label="$t('transport.xhttpMode')"
        :items="['auto','packet-up','stream-up','stream-one']"
        hide-details
        v-model="transport.mode" />
    </v-col>
  </v-row>

  <XhttpBase :data="transport" />

  <v-card border density="compact" color="background" style="margin-top: 8px;">
    <v-card-text>
      <v-row>
        <v-col cols="12" sm="6" md="4" align-self="center">
          <v-switch color="primary" :label="$t('transport.xhttpDownload')" hide-details v-model="downloadEnabled" />
        </v-col>
      </v-row>
      <template v-if="downloadEnabled && transport.download">
        <v-row>
          <v-col cols="12" sm="6" md="4">
            <v-text-field :label="$t('out.addr')" hide-details v-model="transport.download.server" />
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-text-field :label="$t('out.port')" type="number" min="0" hide-details v-model.number="transport.download.server_port" />
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-text-field :label="$t('objects.detour')" hide-details v-model="transport.download.detour" />
          </v-col>
        </v-row>
        <XhttpBase :data="transport.download" />
        <OutTLS :outbound="transport.download" />
      </template>
    </v-card-text>
  </v-card>
</template>

<script lang="ts">
import XhttpBase from './XhttpBase.vue'
import OutTLS from '../tls/OutTLS.vue'
export default {
  props: ['transport'],
  mounted() {
    this.transport.mode ??= 'auto'
    // x_padding_bytes has no omitempty and the core rejects an empty/zero range,
    // so guarantee a valid default range when the transport is first created.
    this.transport.x_padding_bytes ??= '100-1000'
  },
  computed: {
    downloadEnabled: {
      get(): boolean { return this.$props.transport.download != undefined },
      set(v: boolean) {
        if (v) {
          this.$props.transport.download = { tls: {}, x_padding_bytes: '100-1000' }
        } else {
          this.$props.transport.download = undefined
        }
      }
    },
  },
  components: { XhttpBase, OutTLS },
}
</script>
