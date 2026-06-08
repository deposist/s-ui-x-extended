<template>
  <v-card subtitle="Mieru">
    <v-row>
      <v-col cols="12" sm="6" md="4">
        <v-select
          hide-details
          :label="$t('types.mieru.transport')"
          :items="['TCP', 'UDP']"
          v-model="data.transport">
        </v-select>
      </v-col>
      <v-col cols="12" sm="6" md="4" v-if="direction == 'out'">
        <v-select
          hide-details
          :label="$t('types.mieru.multiplexing')"
          :items="multiplexingOptions"
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
  </v-card>
</template>

<script lang="ts">
export default {
  props: ['direction', 'data'],
  data() {
    return {
      multiplexingOptions: [
        'MULTIPLEXING_DEFAULT',
        'MULTIPLEXING_OFF',
        'MULTIPLEXING_LOW',
        'MULTIPLEXING_MIDDLE',
        'MULTIPLEXING_HIGH',
      ],
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
