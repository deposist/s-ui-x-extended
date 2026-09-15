<template>
  <v-card subtitle="USB/IP Client">
    <v-row>
      <v-col cols="12" sm="6" md="4">
        <v-text-field
          :label="$t('out.addr')"
          hide-details
          v-model="data.server">
          <template #append-inner>
            <FieldHint :field-hints="fieldHints" field="server" />
          </template>
        </v-text-field>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-text-field
          :label="$t('out.port')"
          hide-details
          type="number"
          min="1"
          max="65535"
          v-model.number="data.server_port">
          <template #append-inner>
            <FieldHint :field-hints="fieldHints" field="server_port" />
          </template>
        </v-text-field>
      </v-col>
    </v-row>
    <v-card-text class="px-0">
      <v-alert v-if="(data.devices || []).length === 0" type="info" variant="tonal" density="compact">
        {{ $t('types.usbip.noDevices') }}
      </v-alert>
      <DeviceList :devices="data.devices" :field-hints="fieldHints" />
    </v-card-text>
  </v-card>
</template>

<script lang="ts">
import { USBIPClient } from '@/types/services'
import FieldHint from '@/components/FieldHint.vue'
import DeviceList from '@/components/services/usbip/DeviceList.vue'

export default {
  props: {
    data: { type: Object, required: true },
    fieldHints: { type: Object, default: () => ({}) },
  },
  computed: {
    typed: function(): USBIPClient { return <USBIPClient>this.$props.data },
  },
  components: { FieldHint, DeviceList },
}
</script>