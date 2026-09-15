<template>
  <v-card :subtitle="$t('types.quic.title')">
    <v-row>
      <v-col cols="12" sm="6">
        <v-switch
          color="primary"
          :label="$t('types.quic.enable')"
          hide-details
          v-model="optionQuic">
        </v-switch>
      </v-col>
    </v-row>
    <template v-if="optionQuic">
      <v-row>
        <v-col cols="12" sm="6" md="4">
          <v-text-field
            :label="$t('types.quic.idleTimeout')"
            hide-details
            placeholder="30s"
            clearable
            @click:clear="delete data.idle_timeout"
            v-model="data.idle_timeout">
            <template #append-inner>
              <FieldHint :field-hints="fieldHints" field="idle_timeout" />
            </template>
          </v-text-field>
        </v-col>
        <v-col cols="12" sm="6" md="4">
          <v-text-field
            :label="$t('types.quic.keepAlivePeriod')"
            hide-details
            placeholder="30s"
            clearable
            @click:clear="delete data.keep_alive_period"
            v-model="data.keep_alive_period">
            <template #append-inner>
              <FieldHint :field-hints="fieldHints" field="keep_alive_period" />
            </template>
          </v-text-field>
        </v-col>
        <v-col cols="12" sm="6" md="4">
          <v-text-field
            :label="$t('types.quic.initialPacketSize')"
            hide-details
            type="number"
            min="0"
            clearable
            @click:clear="delete data.initial_packet_size"
            v-model.number="data.initial_packet_size">
            <template #append-inner>
              <FieldHint :field-hints="fieldHints" field="initial_packet_size" />
            </template>
          </v-text-field>
        </v-col>
        <v-col cols="12" sm="6" md="4">
          <v-text-field
            :label="$t('types.quic.streamReceiveWindow')"
            hide-details
            clearable
            @click:clear="delete data.stream_receive_window"
            v-model="data.stream_receive_window">
            <template #append-inner>
              <FieldHint :field-hints="fieldHints" field="stream_receive_window" />
            </template>
          </v-text-field>
        </v-col>
        <v-col cols="12" sm="6" md="4">
          <v-text-field
            :label="$t('types.quic.connectionReceiveWindow')"
            hide-details
            clearable
            @click:clear="delete data.connection_receive_window"
            v-model="data.connection_receive_window">
            <template #append-inner>
              <FieldHint :field-hints="fieldHints" field="connection_receive_window" />
            </template>
          </v-text-field>
        </v-col>
        <v-col cols="12" sm="6" md="4">
          <v-text-field
            :label="$t('types.quic.maxConcurrentStreams')"
            hide-details
            type="number"
            min="0"
            clearable
            @click:clear="delete data.max_concurrent_streams"
            v-model.number="data.max_concurrent_streams">
            <template #append-inner>
              <FieldHint :field-hints="fieldHints" field="max_concurrent_streams" />
            </template>
          </v-text-field>
        </v-col>
        <v-col cols="12" sm="6" md="4">
          <v-switch
            color="primary"
            :label="$t('types.quic.disablePathMTUDiscovery')"
            hide-details
            v-model="data.disable_path_mtu_discovery">
          </v-switch>
        </v-col>
      </v-row>
    </template>
  </v-card>
</template>

<script lang="ts">
import FieldHint from '@/components/FieldHint.vue'

// The core embeds QUICOptions in every QUIC-based protocol (hysteria, hysteria2,
// tuic). This block is shared so the same seven knobs appear - and clear - the same
// way everywhere; the switch keeps them out of the saved config until asked for.
const quicFields = [
  'idle_timeout',
  'keep_alive_period',
  'stream_receive_window',
  'connection_receive_window',
  'max_concurrent_streams',
  'initial_packet_size',
  'disable_path_mtu_discovery',
] as const

export default {
  props: {
    data: { type: Object, required: true },
    fieldHints: { type: Object, default: () => ({}) },
  },
  computed: {
    optionQuic: {
      get(): boolean { return quicFields.some((field) => this.$props.data[field] != undefined) },
      set(v: boolean) {
        if (v) return
        for (const field of quicFields) delete this.$props.data[field]
      },
    },
  },
  components: { FieldHint },
}
</script>