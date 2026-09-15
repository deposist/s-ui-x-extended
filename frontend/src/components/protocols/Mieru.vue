<template>
  <v-card subtitle="Mieru">
    <v-row>
      <v-col cols="12" sm="6" md="4">
        <v-select
          hide-details
          :label="$t('types.mieru.transport')"
          :items="mieruTransport"
          v-model="data.transport">
        </v-select>
      </v-col>
      <v-col cols="12" sm="6" md="4" v-if="direction == 'out'">
        <v-select
          hide-details
          :label="$t('types.mieru.multiplexing')"
          :items="mieruMultiplexing"
          v-model="data.multiplexing">
        </v-select>
      </v-col>
      <v-col cols="12" sm="6" md="4" v-if="direction == 'in'">
        <v-switch
          color="primary"
          hide-details
          :label="$t('types.mieru.userHint')"
          v-model="data.user_hint_is_mandatory">
        </v-switch>
      </v-col>
    </v-row>
    <v-row v-if="direction == 'out'">
      <v-col cols="12" sm="6">
        <v-text-field hide-details :label="$t('types.un')" v-model="data.username"></v-text-field>
      </v-col>
      <v-col cols="12" sm="6">
        <v-text-field hide-details :label="$t('types.pw')" v-model="data.password"></v-text-field>
      </v-col>
    </v-row>
    <v-row>
      <v-col cols="12">
        <v-combobox
          chips
          multiple
          clearable
          hide-details
          :label="direction == 'in' ? $t('types.mieru.listenPorts') : $t('types.mieru.serverPorts')"
          :model-value="direction == 'in' ? data.listen_ports : data.server_ports"
          @update:model-value="setPorts">
        </v-combobox>
      </v-col>
      <v-col cols="12">
        <v-text-field
          hide-details
          :label="$t('types.mieru.trafficPattern')"
          v-model="data.traffic_pattern">
        </v-text-field>
      </v-col>
    </v-row>
    <InboundAdvanced :data="data" :field-hints="fieldHints" />
    <v-row>
      <v-col cols="12" sm="6" md="4">
        <v-text-field
          :label="$t('types.mieru.mtu')"
          hide-details
          type="number"
          min="0"
          clearable
          @click:clear="delete data.mtu"
          v-model.number="data.mtu">
          <template #append-inner>
            <FieldHint :field-hints="fieldHints" field="mtu" />
          </template>
        </v-text-field>
      </v-col>
      <v-col cols="12" sm="6" md="4" v-if="direction == 'out'">
        <v-select
          :label="$t('types.mieru.handshakeMode')"
          hide-details
          clearable
          :items="handshakeModes"
          @click:clear="delete data.handshake_mode"
          v-model="data.handshake_mode">
          <template #append-inner>
            <FieldHint :field-hints="fieldHints" field="handshake_mode" />
          </template>
        </v-select>
      </v-col>
    </v-row>
  </v-card>
</template>

<script lang="ts">
import { mieruTransport, mieruMultiplexing } from '@/types/recommended'
import InboundAdvanced from '@/components/protocols/InboundAdvanced.vue'

export default {
  props: ['direction', 'data', 'fieldHints'],
  components: {InboundAdvanced},
  data() {
    return {
      mieruTransport,
      mieruMultiplexing,
      handshakeModes: ['h1', 'h3'],
    }
  },
  methods: {
    setPorts(v: string[]) {
      const key = this.$props.direction == 'in' ? 'listen_ports' : 'server_ports'
      if (v && v.length > 0) {
        this.$props.data[key] = v
      } else {
        delete this.$props.data[key]
      }
    },
  },
}
</script>
