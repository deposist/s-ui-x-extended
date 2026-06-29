<template>
  <v-card subtitle="VLESS">
    <v-row>
      <v-col cols="12" sm="6">
        <v-text-field v-model="data.uuid" label="UUID" hide-details></v-text-field>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-select
          hide-details
          :label="$t('types.vless.flow')"
          :items="['','xtls-rprx-vision']"
          v-model="data.flow">
          <template #append-inner>
            <FieldHint :field-hints="fieldHints" field="flow" />
          </template>
        </v-select>
      </v-col>
    </v-row>
    <v-row>
      <v-col cols="12" sm="6" md="4">
        <v-select
          hide-details
          :label="$t('types.vless.udpEnc')"
          :items="['none','packetaddr','xudp']"
          v-model="packet_encoding">
          <template #append-inner>
            <FieldHint :field-hints="fieldHints" field="packet_encoding" />
          </template>
        </v-select>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <Network :data="data" :field-hints="fieldHints" />
      </v-col>
    </v-row>
  </v-card>
</template>

<script lang="ts">
import Network from '@/components/Network.vue'
import FieldHint from '@/components/FieldHint.vue'

export default {
  props: ['data', 'fieldHints'],
  data() {
    return {}
  },
  computed: {
    packet_encoding: {
      get() { return this.$props.data.packet_encoding != undefined ? this.$props.data.packet_encoding : 'none' },
      set(newValue:string) { this.$props.data.packet_encoding = newValue != "none" ? newValue : undefined }
    },
  },
  components: { Network, FieldHint }
}
</script>