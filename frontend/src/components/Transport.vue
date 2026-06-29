<template>
    <v-card :subtitle="$t('objects.transport')">
    <v-row>
      <v-col cols="12" sm="6" md="4">
        <div class="d-flex align-center ga-1">
          <v-switch color="primary" :label="$t('transport.enable')" v-model="tpEnable" hide-details></v-switch>
          <SettingInfo v-if="hint('transport_enable')" :text="hint('transport_enable')" />
        </div>
      </v-col>
      <v-col cols="12" sm="6" md="4" v-if="tpEnable">
        <v-select
          hide-details
          :label="$t('type')"
          :items="Object.keys(trspTypes).map((key,index) => ({title: key, value: Object.values(trspTypes)[index]}))"
          v-model="transportType">
          <template #append-inner>
            <SettingInfo v-if="hint('transport_type')" :text="hint('transport_type')" />
          </template>
        </v-select>
      </v-col>
    </v-row>
    <Http v-if="Transport.type == trspTypes.HTTP" :transport="Transport" />
    <WebSocket v-if="Transport.type == trspTypes.WebSocket" :transport="Transport" />
    <GRPC v-if="Transport.type == trspTypes.gRPC" :transport="Transport" />
    <HttpUpgrade v-if="Transport.type == trspTypes.HTTPUpgrade" :transport="Transport" />
    <Xhttp v-if="Transport.type == trspTypes.XHTTP" :transport="Transport" />
    <Mkcp v-if="Transport.type == trspTypes.mKCP" :transport="Transport" />
    <Quic v-if="Transport.type == trspTypes.QUIC" :transport="Transport" />
  </v-card>
</template>

<script lang="ts">
import { TrspTypes, Transport } from '@/types/transport'
import Http from './transports/Http.vue'
import WebSocket from './transports/WebSocket.vue'
import GRPC from './transports/gRPC.vue'
import HttpUpgrade from './transports/HttpUpgrade.vue'
import Xhttp from './transports/Xhttp.vue'
import Mkcp from './transports/Mkcp.vue'
import Quic from './transports/QUIC.vue'
import SettingInfo from '@/components/SettingInfo.vue'
export default {
  props: {
    data: { type: Object, required: true },
    fieldHints: { type: Object, default: () => ({}) },
  },
  data() {
    return {
      trspTypes: TrspTypes
    }
  },
  methods: {
    hint(key: string): string {
      const hintKey = (this.$props.fieldHints as Record<string, string>)[key]
      if (!hintKey) return ''
      const translated = this.$t(hintKey)
      return translated === hintKey ? '' : translated
    },
  },
  computed: {
    Transport() {
      return <Transport>this.$props.data.transport
    },
    tpEnable: {
      get() { return Object.hasOwn(this.$props.data.transport, 'type') },
      set(newValue: boolean) { this.$props.data.transport = newValue ? { type: 'http' } : {} }
    },
    transportType: {
      get() { return this.Transport.type },
      set(newValue: string) { this.$props.data.transport = { type: newValue } }
    }
  },
  components: { Http, WebSocket, GRPC, HttpUpgrade, Xhttp, Mkcp, Quic, SettingInfo }
}
</script>