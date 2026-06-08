<template>
  <v-row density="compact">
    <v-col cols="12" class="v-card-subtitle" style="margin-top: -5px;">{{ label }}</v-col>
    <v-col :cols="data.type == 'local' ? 12 : 4">
      <v-select
        hide-details
        :label="$t('type')"
        :items="['udp','tcp','local','tls','quic','h3']"
        @update:model-value="updateType($event)"
        density="compact"
        :class="data.type != 'local' ? 'noGutters' : ''"
        v-model="data.type">
      </v-select>
    </v-col>
    <v-col cols="5" v-if="data.type != 'local'">
      <v-combobox
        v-model="data.server"
        :items="dnsResolvers"
        :label="$t('in.addr')"
        density="compact"
        class="noGutters"
        hide-details>
      </v-combobox>
    </v-col>
    <v-col cols="3" v-if="data.type != 'local'">
      <v-text-field
        v-model.number="data.server_port"
        :label="$t('in.port')"
        density="compact"
        type="number"
        class="noGutters"
        min="1"
        hide-details>
      </v-text-field>
    </v-col>
  </v-row>
</template>

<script lang="ts">
import { dnsResolvers } from '@/types/recommended'

export default {
  props: ['data', 'label'],
  data() {
    return {
      dnsResolvers,
    }
  },
  methods: {
    updateType(t:string) {
      if (t == 'local') {
        delete this.data.server
        delete this.data.server_port
      }
    }
  }
}
</script>

<style>
.noGutters .v-field__input,
.noGutters .v-field {
  text-align: center !important;
  padding-inline-end: 0 !important;
}
</style>